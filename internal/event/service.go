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

func (s *Service) Receive(ctx context.Context, userID string, t Type, payload json.RawMessage) (e Event, err error) {
	e = New(userID, t, payload)
	err = s.repo.Save(ctx, e)
	return e, err
}

func (s *Service) GetAll(ctx context.Context, userID string) (events []Event, err error) {
	events, err = s.repo.GetAllByUserID(ctx, userID)
	return events, err
}
