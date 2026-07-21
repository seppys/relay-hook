package dispatch

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"relay-hook/internal/delivery"
	"relay-hook/internal/event"
	"relay-hook/internal/subscription"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	tracer           = otel.Tracer("dispatch")
	meter            = otel.Meter("dispatch")
	eventsDispatched metric.Int64Counter
)

func init() {
	var err error
	eventsDispatched, err = meter.Int64Counter("dispatch.events",
		metric.WithDescription("Number of events dispatched"))
	if err != nil {
		panic(err)
	}
}

type SubscriberLookup interface {
	SubscribersFor(ctx context.Context, userID string, t event.Type) ([]subscription.Subscriber, error)
}

type Dispatcher struct {
	subs SubscriberLookup
	svc  delivery.Service
}

func NewDispatcher(subs SubscriberLookup, service delivery.Service) *Dispatcher {
	return &Dispatcher{subs: subs, svc: service}
}

func (d *Dispatcher) Dispatch(ctx context.Context, e event.Event) error {
	ctx, span := tracer.Start(ctx, "dispatch.Dispatch")
	defer span.End()

	span.SetAttributes(
		attribute.String("event.id", e.Id),
		attribute.String("event.type", string(e.Type)),
		attribute.String("user_id", e.UserID),
	)

	subscribers, err := d.subs.SubscribersFor(ctx, e.UserID, e.Type)
	if err != nil {
		span.RecordError(err)
		eventsDispatched.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "error")))
		return fmt.Errorf("lookup subscribers for %q: %w", e.Type, err)
	}

	span.SetAttributes(attribute.Int("subscriber.count", len(subscribers)))

	for _, sub := range subscribers {
		dlv := *delivery.NewDelivery(e.Id, sub.Id, e.Payload, sub.EndpointURL, delivery.StatusProcessing, 1)
		log.Default().Println("delivery", dlv)
		if err := d.svc.CreateDelivery(ctx, dlv); err != nil {
			slog.Error("create delivery failed", "event_id", e.Id, "subscriber_id", sub.Id)
			continue
		}

		if err := d.svc.Deliver(ctx, dlv); err != nil {
			slog.Error("delivery failed", "event_id", e.Id, "subscriber_id", sub.Id, "endpoint", sub.EndpointURL, "err", err)
			continue
		}
	}

	eventsDispatched.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "success")))
	return nil
}
