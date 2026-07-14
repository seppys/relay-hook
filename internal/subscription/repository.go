package subscription

import (
	"context"
	"errors"
	"relay-hook/internal/event"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("subscription not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Register(ctx context.Context, sub Subscriber, subs []Subscription) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO subscribers (id, endpoint_url) VALUES ($1, $2)`,
		sub.Id, sub.EndpointURL,
	)
	if err != nil {
		return err
	}

	for _, s := range subs {
		_, err = tx.Exec(ctx,
			`INSERT INTO subscriptions (id, subscriber_id, event_type) VALUES ($1, $2, $3)
			 ON CONFLICT (subscriber_id, event_type) DO NOTHING`,
			s.Id, s.SubscriberId, s.EventType,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) Save(ctx context.Context, s Subscription) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO subscriptions (id, subscriber_id, event_type) VALUES ($1, $2, $3)
		 ON CONFLICT (subscriber_id, event_type) DO NOTHING`,
		s.Id, s.SubscriberId, s.EventType,
	)
	return err
}

func (r *Repository) GetAll(ctx context.Context) ([]View, error) {
	rows, err := r.db.Query(ctx,
		`SELECT sub.id, sub.subscriber_id, sub.event_type, s.endpoint_url
		 FROM subscriptions sub
		 JOIN subscribers s ON s.id = sub.subscriber_id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	views := make([]View, 0)
	for rows.Next() {
		var v View
		if err := rows.Scan(&v.Id, &v.SubscriberId, &v.EventType, &v.EndpointURL); err != nil {
			return nil, err
		}
		views = append(views, v)
	}
	return views, rows.Err()
}

func (r *Repository) FindByEndpoint(ctx context.Context, endpoint string) (Subscriber, error) {
	var s Subscriber
	err := r.db.QueryRow(ctx,
		`SELECT id, endpoint_url FROM subscribers WHERE endpoint_url = $1`,
		endpoint,
	).Scan(&s.Id, &s.EndpointURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscriber{}, ErrNotFound
	}
	return s, err
}

func (r *Repository) SubscribersFor(ctx context.Context, eventType event.Type) ([]Subscriber, error) {
	rows, err := r.db.Query(ctx,
		`SELECT s.id, s.endpoint_url
		 FROM subscribers s
		 JOIN subscriptions sub ON sub.subscriber_id = s.id
		 WHERE sub.event_type = $1`,
		eventType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subs := make([]Subscriber, 0)
	for rows.Next() {
		var s Subscriber
		if err := rows.Scan(&s.Id, &s.EndpointURL); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (r *Repository) UpdateEndpoint(ctx context.Context, id, endpoint string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE subscribers SET endpoint_url = $1 WHERE id = $2`,
		endpoint, id,
	)
	if err != nil {
		return ErrInternalServer
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete remove a specific subscription
func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM subscriptions WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteSubscriber remove a subscriber and its subscriptions
func (r *Repository) DeleteSubscriber(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM subscribers WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
