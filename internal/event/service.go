package event

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	tracer         = otel.Tracer("event")
	meter          = otel.Meter("event")
	eventsReceived metric.Int64Counter
)

func init() {
	var err error
	eventsReceived, err = meter.Int64Counter("event.received",
		metric.WithDescription("Number of events received"))
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

func (s *Service) Receive(ctx context.Context, userID string, t Type, payload json.RawMessage) (e Event, err error) {
	ctx, span := tracer.Start(ctx, "event.Receive")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event.type", string(t)),
	)

	e = New(userID, t, payload)
	if err = s.repo.Save(ctx, e); err != nil {
		span.RecordError(err)
		eventsReceived.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "error")))
		return e, err
	}

	span.SetAttributes(attribute.String("event.id", e.Id))
	eventsReceived.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "success")))
	return e, nil
}

func (s *Service) GetAll(ctx context.Context, userID string) (events []Event, err error) {
	ctx, span := tracer.Start(ctx, "event.GetAll")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", userID))

	events, err = s.repo.GetAllByUserID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		return events, err
	}

	span.SetAttributes(attribute.Int("event.count", len(events)))
	return events, err
}
