package delivery

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ClaimsForDelivery returns the limit number of deliveries with status 'failed' or those with
// status 'processing' whose 'claimed_at' has passed processingTimeOut
func (r *Repository) ClaimsForDelivery(ctx context.Context, limit int, processingTimeOut time.Time) ([]Delivery, error) {
	var deliveries []Delivery

	rows, err := r.db.Query(ctx, `
		UPDATE deliveries
		SET status = 'processing',
		    attempts = attempts + 1,
		    claimed_at = now()
		WHERE id IN (
			SELECT id FROM deliveries
			WHERE status = 'failed'
			   OR (status = 'processing' AND claimed_at < $2)
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		RETURNING id, event_id, subscriber_id, payload, endpoint_url, status, attempts
	`, limit, processingTimeOut)
	if err != nil {
		return nil, ErrInternalServer
	}

	defer rows.Close()
	for rows.Next() {
		var d Delivery
		if err = rows.Scan(&d.ID, &d.EventID, &d.SubscriberID, &d.Payload, &d.EndpointURL, &d.Status, &d.Attempts); err != nil {
			return nil, ErrInternalServer
		}
		deliveries = append(deliveries, d)
	}

	return deliveries, nil
}

func (r *Repository) Create(ctx context.Context, delivery Delivery) error {
	log.Default().Println("Create delivery ", delivery)
	_, err := r.db.Exec(ctx, `
		INSERT INTO deliveries (id, event_id, subscriber_id, payload, endpoint_url, status, attempts)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, delivery.ID, delivery.EventID, delivery.SubscriberID, delivery.Payload, delivery.EndpointURL, delivery.Status, delivery.Attempts)
	if err != nil {
		return ErrInternalServer
	}

	return nil
}

func (r *Repository) UpdateStatus(ctx context.Context, deliveryID string, status Status) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE deliveries SET status = $1 WHERE id = $2
	`, status, deliveryID)
	if err != nil {
		return ErrInternalServer
	}
	if tag.RowsAffected() == 0 {
		return ErrDeliveryNotFound
	}

	return nil
}
