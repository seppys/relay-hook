package broker

import (
	"errors"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type Consumer struct {
	c *kafka.Consumer
}

func NewConsumer(brokers, groupID, topic string) (*Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        brokers,
		"group.id":                 groupID,
		"auto.offset.reset":        "earliest",
		"enable.auto.commit":       true,
		"enable.auto.offset.store": false,
	})
	if err != nil {
		return nil, fmt.Errorf("new consumer: %w", err)
	}

	if err := c.Subscribe(topic, nil); err != nil {
		return nil, fmt.Errorf("subscribe %q: %w", topic, err)
	}

	return &Consumer{c: c}, nil
}

func (c *Consumer) Read(timeout time.Duration) (*kafka.Message, error) {
	msg, err := c.c.ReadMessage(timeout)
	if err != nil {
		var kerr kafka.Error
		if errors.As(err, &kerr) && kerr.Code() == kafka.ErrTimedOut {
			return nil, nil
		}
		return nil, err
	}
	return msg, nil
}

func (c *Consumer) Commit(msg *kafka.Message) error {
	_, err := c.c.StoreMessage(msg)
	return err
}

func (c *Consumer) Close() error {
	return c.c.Close()
}
