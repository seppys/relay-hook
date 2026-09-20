package broker

import (
	"context"
	"encoding/json"
	"relay-hook/internal/event"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type Producer struct {
	kafkaProducer *kafka.Producer
	topic         string
}

func NewProducer(brokers, topic string) (*Producer, error) {
	kafkaProducer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"acks":              "all",
	})

	if err != nil {
		return nil, err
	}
	prod := &Producer{kafkaProducer: kafkaProducer, topic: topic}
	return prod, nil
}

func (p *Producer) Publish(ctx context.Context, e event.Event) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}

	delivered := make(chan kafka.Event, 1)
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.topic, Partition: kafka.PartitionAny},
		Key:            []byte(e.UserID),
		Value:          b,
	}

	if err := p.kafkaProducer.Produce(msg, delivered); err != nil {
		return err
	}

	select {
	case ev := <-delivered:
		m := ev.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			return m.TopicPartition.Error
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
