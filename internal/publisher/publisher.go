package publisher

import (
	"context"
	"log"
	"relay-hook/internal/event"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	tracer          = otel.Tracer("publisher")
	meter           = otel.Meter("publisher")
	eventsPublished metric.Int64Counter
)

func init() {
	var err error
	eventsPublished, err = meter.Int64Counter("publisher.events_published",
		metric.WithDescription("Number of events published to the broker"))
	if err != nil {
		panic(err)
	}
}

type Publisher interface {
	Publish(ctx context.Context, e event.Event) error
}

// Fetch events in batches from db and publish them to kafka
func publishPendingEvents(ctx context.Context, pool *pgxpool.Pool, pub Publisher) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id, user_id, type, payload, received_at
		FROM events
		WHERE status = 'pending'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	events := make([]event.Event, 0)
	for rows.Next() {
		var e event.Event
		if err := rows.Scan(&e.Id, &e.UserID, &e.Type, &e.Payload, &e.ReceivedAt); err != nil {
			return err
		}
		events = append(events, e)
	}

	for _, e := range events {
		eventCtx, span := tracer.Start(ctx, "publisher.publish")
		span.SetAttributes(
			attribute.String("event.id", e.Id),
			attribute.String("event.type", string(e.Type)),
			attribute.String("user_id", e.UserID),
		)

		if err := pub.Publish(eventCtx, e); err != nil {
			span.RecordError(err)
			span.End()
			eventsPublished.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "error")))
			return err
		}

		_, err = tx.Exec(ctx, `
			UPDATE events
			SET status = 'published'
			WHERE id = $1
		`, e.Id)
		if err != nil {
			span.RecordError(err)
			span.End()
			eventsPublished.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "error")))
			return err
		}

		eventsPublished.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "success")))
		span.End()
	}

	return tx.Commit(ctx)
}

func Run(ctx context.Context, pool *pgxpool.Pool, pub Publisher) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("relay shutting down")
			return
		case <-ticker.C:
			if err := publishPendingEvents(ctx, pool, pub); err != nil {
				log.Printf("relay error: %v", err)
			}
		}
	}
}
