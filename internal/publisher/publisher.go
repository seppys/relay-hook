package publisher

import (
	"context"
	"log"
	"relay-hook/internal/event"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
		if err := pub.Publish(ctx, e); err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			UPDATE events
			SET status = 'published'
			WHERE id = $1
		`, e.Id)
		if err != nil {
			return err
		}
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
