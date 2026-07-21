package delivery

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

type Retrier struct {
	svc Service
}

func NewRetrier(svc Service) *Retrier {
	return &Retrier{svc: svc}
}

func (r *Retrier) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.retryFailedEvents(ctx)
		}
	}
}

func (r *Retrier) retryFailedEvents(ctx context.Context) {
	ctx, span := tracer.Start(ctx, "delivery.retryFailedEvents")
	defer span.End()

	deliveries, err := r.svc.GetPending(ctx)
	if err != nil {
		span.RecordError(err)
		log.Printf("error fetching pending deliveries: %v", err)
		return
	}

	span.SetAttributes(attribute.Int("delivery.count", len(deliveries)))
	for _, d := range deliveries {
		_ = r.svc.Deliver(ctx, d)
	}
}
