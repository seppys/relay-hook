package dispatch

import (
	"context"
	"fmt"
	"log/slog"
	"relay-hook/internal/event"
	"relay-hook/internal/subscription"
)

type SubscriberLookup interface {
	SubscribersFor(ctx context.Context, userID string, t event.Type) ([]subscription.Subscriber, error)
}

type Deliverer interface {
	Deliver(ctx context.Context, endpoint string, payload []byte) error
}

type Dispatcher struct {
	subs      SubscriberLookup
	deliverer Deliverer
}

func NewDispatcher(subs SubscriberLookup, deliverer Deliverer) *Dispatcher {
	return &Dispatcher{subs: subs, deliverer: deliverer}
}

func (d *Dispatcher) Dispatch(ctx context.Context, e event.Event) error {
	subscribers, err := d.subs.SubscribersFor(ctx, e.UserID, e.Type)
	if err != nil {
		return fmt.Errorf("lookup subscribers for %q: %w", e.Type, err)
	}

	for _, sub := range subscribers {
		if err := d.deliverer.Deliver(ctx, sub.EndpointURL, e.Payload); err != nil {
			slog.Error("delivery failed", "event_id", e.Id, "subscriber_id", sub.Id, "endpoint", sub.EndpointURL, "err", err)
			continue
		}
	}

	return nil
}
