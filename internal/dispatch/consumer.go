package dispatch

import (
	"context"
	"encoding/json"
	"time"

	"relay-hook/internal/broker"
	"relay-hook/internal/event"
)

type Consumer struct {
	kafka      *broker.Consumer
	dispatcher *Dispatcher
}

func NewConsumer(kafka *broker.Consumer, dispatcher *Dispatcher) *Consumer {
	return &Consumer{kafka: kafka, dispatcher: dispatcher}
}

func (c *Consumer) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msg, err := c.kafka.Read(500 * time.Millisecond)
		if err != nil {
			continue
		}
		if msg == nil {
			continue
		}

		var e event.Event
		if err := json.Unmarshal(msg.Value, &e); err != nil {
			c.kafka.Commit(msg)
			continue
		}

		if err := c.dispatcher.Dispatch(ctx, e); err != nil {
			continue
		}

		if err := c.kafka.Commit(msg); err != nil {
		}
	}
}
