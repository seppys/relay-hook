package event

import (
	"context"
	"encoding/json"
)

type Publisher interface {
	Publish(ctx context.Context, e Event) error
}

type Service struct {
	repo      *Repository
	publisher Publisher
}

func NewService(repo *Repository, publisher Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) Receive(ctx context.Context, t Type, payload json.RawMessage) (e Event, err error) {
	e = New(t, payload)

	if err = s.publisher.Publish(ctx, e); err != nil {
		return Event{}, err
	}
	s.repo.Save(ctx, e)
	return e, err
}

func (s *Service) GetAll(ctx context.Context) (events []Event, err error) {
	events, err = s.repo.GetAll(ctx)
	return events, err
}
