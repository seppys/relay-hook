package dispatch

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"relay-hook/internal/delivery"
	"relay-hook/internal/event"
	"relay-hook/internal/subscription"
)

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
	subscribers, err := d.subs.SubscribersFor(ctx, e.UserID, e.Type)
	if err != nil {
		return fmt.Errorf("lookup subscribers for %q: %w", e.Type, err)
	}

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

	return nil
}
