package event

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
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
