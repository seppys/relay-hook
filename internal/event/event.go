package event

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidID      = errors.New("invalid or missing id")
	ErrInvalidPayload = errors.New("invalid or missing payload")
	ErrInternalServer = errors.New("server error")
)

type Type string

type Event struct {
	Id         string          `json:"id"`
	Type       Type            `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	ReceivedAt time.Time       `json:"received_at"`
}

func New(t Type, p json.RawMessage) Event {
	return Event{
		Id:         uuid.New().String(),
		Type:       t,
		Payload:    p,
		ReceivedAt: time.Now(),
	}
}
