package subscription

import (
	"context"
	"errors"

	"relay-hook/internal/event"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, eventType event.Type, endpoint string) (Subscription, error) {
	subscriber, err := s.repo.FindByEndpoint(ctx, endpoint)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Subscription{}, ErrNotFound
	}

	if errors.Is(err, ErrNotFound) {
		subscriber = NewSubscriber(endpoint)
		sub := NewSubscription(subscriber.Id, eventType)
		if err := s.repo.Register(ctx, subscriber, []Subscription{sub}); err != nil {
			return Subscription{}, err
		}
		return sub, nil
	}

	sub := NewSubscription(subscriber.Id, eventType)
	if err := s.repo.Save(ctx, sub); err != nil {
		return Subscription{}, err
	}
	return sub, nil
}

func (s *Service) GetAll(ctx context.Context) ([]View, error) {
	return s.repo.GetAll(ctx)
}

func (s *Service) SubscribersFor(ctx context.Context, eventType event.Type) ([]Subscriber, error) {
	return s.repo.SubscribersFor(ctx, eventType)
}

func (s *Service) UpdateEndpoint(ctx context.Context, id string, endpointURL string) error {
	err := s.repo.UpdateEndpoint(ctx, id, endpointURL)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return ErrInternalServer
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) DeleteSubscriber(ctx context.Context, id string) error {
	return s.repo.DeleteSubscriber(ctx, id)
}
