package broker

import (
	"context"
	"encoding/json"
	"relay-hook/internal/event"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("broker")

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
	ctx, span := tracer.Start(ctx, "broker.Publish")
	defer span.End()

	span.SetAttributes(
		attribute.String("messaging.system", "kafka"),
		attribute.String("messaging.destination.name", p.topic),
		attribute.String("event.id", e.Id),
		attribute.String("event.type", string(e.Type)),
	)

	b, err := json.Marshal(e)
	if err != nil {
		span.RecordError(err)
		return err
	}

	delivered := make(chan kafka.Event, 1)
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.topic, Partition: kafka.PartitionAny},
		Key:            []byte(e.UserID),
		Value:          b,
	}
	InjectTraceContext(ctx, msg)

	if err := p.kafkaProducer.Produce(msg, delivered); err != nil {
		span.RecordError(err)
		return err
	}

	select {
	case ev := <-delivered:
		m := ev.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			span.RecordError(m.TopicPartition.Error)
			return m.TopicPartition.Error
		}
		return nil
	case <-ctx.Done():
		span.RecordError(ctx.Err())
		return ctx.Err()
	}
}
