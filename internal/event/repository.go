package event

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetAll(ctx context.Context) ([]Event, error) {
	rows, err := r.db.Query(ctx, "SELECT id, type, payload, received_at FROM events")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]Event, 0)
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.Id, &event.Type, &event.Payload, &event.ReceivedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func (r *Repository) Save(ctx context.Context, e Event) error {
	_, err := r.db.Exec(ctx, "INSERT INTO events (id, type, payload, received_at) VALUES ($1, $2, $3, $4)", e.Id, e.Type, e.Payload, e.ReceivedAt)
	return err
}
