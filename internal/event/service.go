package event

import (
	"context"
	"encoding/json"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Receive(ctx context.Context, t Type, payload json.RawMessage) (e Event, err error) {
	e = New(t, payload)
	err = s.repo.Save(ctx, e)
	if err != nil {
		return Event{}, err
	}
	return e, err
}

func (s *Service) GetAll(ctx context.Context) (events []Event, err error) {
	events, err = s.repo.GetAll(ctx)
	return events, err
}
