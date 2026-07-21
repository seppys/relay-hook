package delivery

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	tracer           = otel.Tracer("delivery")
	meter            = otel.Meter("delivery")
	deliveryAttempts metric.Int64Counter
	deliveryDuration metric.Float64Histogram
)

func init() {
	var err error
	deliveryAttempts, err = meter.Int64Counter("delivery.attempts",
		metric.WithDescription("Number of webhook delivery attempts"))
	if err != nil {
		panic(err)
	}
	deliveryDuration, err = meter.Float64Histogram("delivery.duration",
		metric.WithDescription("Duration of webhook delivery requests"),
		metric.WithUnit("s"))
	if err != nil {
		panic(err)
	}
}

type Service struct {
	repo   *Repository
	client *http.Client
}

func NewService(repo *Repository, client *http.Client) *Service {
	return &Service{repo: repo, client: client}
}

func (s *Service) CreateDelivery(ctx context.Context, delivery Delivery) error {
	ctx, span := tracer.Start(ctx, "delivery.CreateDelivery")
	defer span.End()

	span.SetAttributes(
		attribute.String("delivery.id", delivery.ID),
		attribute.String("subscriber.id", delivery.SubscriberID),
	)

	if err := s.repo.Create(ctx, delivery); err != nil {
		span.RecordError(err)
		return ErrInternalServer
	}
	return nil
}

func (s *Service) UpdateStatus(ctx context.Context, deliveryID string, status Status) error {
	ctx, span := tracer.Start(ctx, "delivery.UpdateStatus")
	defer span.End()

	span.SetAttributes(
		attribute.String("delivery.id", deliveryID),
		attribute.String("delivery.status", string(status)),
	)

	if err := s.repo.UpdateStatus(ctx, deliveryID, status); err != nil {
		span.RecordError(err)
		return ErrInternalServer
	}
	return nil
}

func (s *Service) GetPending(ctx context.Context) ([]Delivery, error) {
	ctx, span := tracer.Start(ctx, "delivery.GetPending")
	defer span.End()

	deliveries, err := s.repo.ClaimsForDelivery(ctx, 50, time.Now().Add(-10*time.Minute))
	if err != nil {
		span.RecordError(err)
		return nil, ErrInternalServer
	}

	span.SetAttributes(attribute.Int("delivery.count", len(deliveries)))
	return deliveries, nil
}

func (s *Service) Deliver(ctx context.Context, delivery Delivery) error {
	ctx, span := tracer.Start(ctx, "delivery.Deliver")
	defer span.End()

	span.SetAttributes(
		attribute.String("delivery.id", delivery.ID),
		attribute.String("subscriber.id", delivery.SubscriberID),
		attribute.Int("delivery.attempt", delivery.Attempts),
	)

	err := s.sendRequest(ctx, delivery.EndpointURL, delivery.Payload, delivery.ID)
	if err != nil {
		span.RecordError(err)
		if delivery.Attempts > 5 {
			deliveryAttempts.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "dead")))
			span.SetAttributes(attribute.String("delivery.status", string(StatusDead)))
			return s.UpdateStatus(ctx, delivery.ID, StatusDead)
		}
		deliveryAttempts.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "failed")))
		span.SetAttributes(attribute.String("delivery.status", string(StatusFailed)))
		return s.UpdateStatus(ctx, delivery.ID, StatusFailed)
	}

	deliveryAttempts.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "delivered")))
	span.SetAttributes(attribute.String("delivery.status", string(StatusDelivered)))
	return s.UpdateStatus(ctx, delivery.ID, StatusDelivered)
}

func (s *Service) sendRequest(ctx context.Context, endpoint string, payload []byte, idempotencyKey string) error {
	ctx, span := tracer.Start(ctx, "delivery.sendRequest")
	defer span.End()

	span.SetAttributes(attribute.String("http.url", endpoint))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)

	start := time.Now()
	resp, err := s.client.Do(req)
	elapsed := time.Since(start).Seconds()
	if err != nil {
		deliveryDuration.Record(ctx, elapsed)
		span.RecordError(err)
		return fmt.Errorf("post to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	deliveryDuration.Record(ctx, elapsed, metric.WithAttributes(attribute.Int("http.status_code", resp.StatusCode)))
	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("delivery to %s failed: status %d", endpoint, resp.StatusCode)
		span.RecordError(err)
		return err
	}
	return nil
}
