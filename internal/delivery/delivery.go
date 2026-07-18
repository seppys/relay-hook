package delivery

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrDeliveryNotFound = errors.New("delivery not found")
	ErrInternalServer   = errors.New("server error")
)

type Status string

const (
	StatusProcessing Status = "processing"
	StatusDelivered  Status = "delivered"
	StatusFailed     Status = "failed"
	StatusDead       Status = "dead"
)

type Delivery struct {
	ID           string          `json:"id"`
	EventID      string          `json:"event_id"`
	SubscriberID string          `json:"subscriber_id"`
	Payload      json.RawMessage `json:"payload"`
	EndpointURL  string          `json:"endpoint_url"`
	Status       Status          `json:"status"`
	Attempts     int             `json:"attempts"`
}

func NewDelivery(eventID string, subscriberID string, payload json.RawMessage, endpointURL string, status Status, attempts int) *Delivery {
	return &Delivery{
		ID:           uuid.New().String(),
		EventID:      eventID,
		SubscriberID: subscriberID,
		Payload:      payload,
		EndpointURL:  endpointURL,
		Status:       status,
		Attempts:     attempts,
	}
}
