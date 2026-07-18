package delivery

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"
)

type Service struct {
	repo   *Repository
	client *http.Client
}

func NewService(repo *Repository, client *http.Client) *Service {
	return &Service{repo: repo, client: client}
}

func (s *Service) CreateDelivery(ctx context.Context, delivery Delivery) error {
	err := s.repo.Create(ctx, delivery)
	if err != nil {
		return ErrInternalServer
	}
	return nil
}

func (s *Service) UpdateStatus(ctx context.Context, deliveryID string, status Status) error {
	err := s.repo.UpdateStatus(ctx, deliveryID, status)
	if err != nil {
		return ErrInternalServer
	}
	return nil
}

func (s *Service) GetPending(ctx context.Context) ([]Delivery, error) {
	deliveries, err := s.repo.ClaimsForDelivery(ctx, 50, time.Now().Add(-10*time.Minute))
	if err != nil {
		return nil, ErrInternalServer
	}
	return deliveries, nil
}

func (s *Service) Deliver(ctx context.Context, delivery Delivery) error {
	err := s.sendRequest(ctx, delivery.EndpointURL, delivery.Payload)
	if err != nil {
		if delivery.Attempts > 5 {
			return s.UpdateStatus(ctx, delivery.ID, StatusDead)
		}
		return s.UpdateStatus(ctx, delivery.ID, StatusFailed)
	}
	return s.UpdateStatus(ctx, delivery.ID, StatusDelivered)
}

func (s *Service) sendRequest(ctx context.Context, endpoint string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("post to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("delivery to %s failed: status %d", endpoint, resp.StatusCode)
	}
	return nil
}
