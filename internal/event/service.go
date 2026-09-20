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
	if err = s.repo.Save(ctx, e); err != nil {
		return e, err
	}

	return e, nil
}

func (s *Service) GetAll(ctx context.Context, userID string) (events []Event, err error) {
	events, err = s.repo.GetAllByUserID(ctx, userID)
	if err != nil {
		return events, err
	}

	return events, err
}
