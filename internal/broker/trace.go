package broker

import (
	"context"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"go.opentelemetry.io/otel"
)

type messageCarrier struct {
	msg *kafka.Message
}

func (c messageCarrier) Get(key string) string {
	for _, h := range c.msg.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c messageCarrier) Set(key, value string) {
	for i := range c.msg.Headers {
		if c.msg.Headers[i].Key == key {
			c.msg.Headers[i].Value = []byte(value)
			return
		}
	}
	c.msg.Headers = append(c.msg.Headers, kafka.Header{Key: key, Value: []byte(value)})
}

func (c messageCarrier) Keys() []string {
	keys := make([]string, len(c.msg.Headers))
	for i, h := range c.msg.Headers {
		keys[i] = h.Key
	}
	return keys
}

func InjectTraceContext(ctx context.Context, msg *kafka.Message) {
	otel.GetTextMapPropagator().Inject(ctx, messageCarrier{msg: msg})
}

func ExtractTraceContext(ctx context.Context, msg *kafka.Message) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, messageCarrier{msg: msg})
}
