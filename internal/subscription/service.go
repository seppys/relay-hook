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

func (s *Service) Register(ctx context.Context, userID string, eventType event.Type, endpoint string) (Subscription, error) {
	subscriber, err := s.repo.FindByEndpoint(ctx, userID, endpoint)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Subscription{}, ErrNotFound
	}

	if errors.Is(err, ErrNotFound) {
		subscriber = NewSubscriber(userID, endpoint)
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

func (s *Service) GetAll(ctx context.Context, userID string) ([]View, error) {
	return s.repo.GetAll(ctx, userID)
}

func (s *Service) SubscribersFor(ctx context.Context, userID string, eventType event.Type) ([]Subscriber, error) {
	return s.repo.SubscribersFor(ctx, userID, eventType)
}

func (s *Service) UpdateEndpoint(ctx context.Context, id, userID, endpointURL string) error {
	err := s.repo.UpdateEndpoint(ctx, id, userID, endpointURL)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return ErrInternalServer
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, id, userID string) error {
	return s.repo.Delete(ctx, id, userID)
}

func (s *Service) DeleteSubscriber(ctx context.Context, id, userID string) error {
	return s.repo.DeleteSubscriber(ctx, id, userID)
}
