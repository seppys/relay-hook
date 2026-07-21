package subscription

import (
	"context"
	"errors"
	"relay-hook/internal/event"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	tracer               = otel.Tracer("subscription")
	meter                = otel.Meter("subscription")
	subscriptionsCreated metric.Int64Counter
)

func init() {
	var err error
	subscriptionsCreated, err = meter.Int64Counter("subscription.created",
		metric.WithDescription("Number of subscriptions created"))
	if err != nil {
		panic(err)
	}
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, userID string, eventType event.Type, endpoint string) (Subscription, error) {
	ctx, span := tracer.Start(ctx, "subscription.Register")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event.type", string(eventType)),
	)

	subscriber, err := s.repo.FindByEndpoint(ctx, userID, endpoint)
	if err != nil && !errors.Is(err, ErrNotFound) {
		span.RecordError(err)
		subscriptionsCreated.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "error")))
		return Subscription{}, ErrNotFound
	}

	if errors.Is(err, ErrNotFound) {
		subscriber = NewSubscriber(userID, endpoint)
		sub := NewSubscription(subscriber.Id, eventType)
		if err := s.repo.Register(ctx, subscriber, []Subscription{sub}); err != nil {
			span.RecordError(err)
			subscriptionsCreated.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "error")))
			return Subscription{}, err
		}
		subscriptionsCreated.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "success")))
		return sub, nil
	}

	sub := NewSubscription(subscriber.Id, eventType)
	if err := s.repo.Save(ctx, sub); err != nil {
		span.RecordError(err)
		subscriptionsCreated.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "error")))
		return Subscription{}, err
	}
	subscriptionsCreated.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "success")))
	return sub, nil
}

func (s *Service) GetAll(ctx context.Context, userID string) ([]View, error) {
	ctx, span := tracer.Start(ctx, "subscription.GetAll")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", userID))

	views, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("subscription.count", len(views)))
	return views, nil
}

func (s *Service) SubscribersFor(ctx context.Context, userID string, eventType event.Type) ([]Subscriber, error) {
	ctx, span := tracer.Start(ctx, "subscription.SubscribersFor")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event.type", string(eventType)),
	)

	subs, err := s.repo.SubscribersFor(ctx, userID, eventType)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("subscriber.count", len(subs)))
	return subs, nil
}

func (s *Service) UpdateEndpoint(ctx context.Context, id, userID, endpointURL string) error {
	ctx, span := tracer.Start(ctx, "subscription.UpdateEndpoint")
	defer span.End()

	span.SetAttributes(
		attribute.String("subscriber.id", id),
		attribute.String("user_id", userID),
	)

	err := s.repo.UpdateEndpoint(ctx, id, userID, endpointURL)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return ErrInternalServer
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, id, userID string) error {
	ctx, span := tracer.Start(ctx, "subscription.Delete")
	defer span.End()

	span.SetAttributes(
		attribute.String("subscription.id", id),
		attribute.String("user_id", userID),
	)

	if err := s.repo.Delete(ctx, id, userID); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func (s *Service) DeleteSubscriber(ctx context.Context, id, userID string) error {
	ctx, span := tracer.Start(ctx, "subscription.DeleteSubscriber")
	defer span.End()

	span.SetAttributes(
		attribute.String("subscriber.id", id),
		attribute.String("user_id", userID),
	)

	if err := s.repo.DeleteSubscriber(ctx, id, userID); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}
