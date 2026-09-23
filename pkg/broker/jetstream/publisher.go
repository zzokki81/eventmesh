// Package jetstream provides JetStream-backed implementations of the broker
// Publisher and Subscriber contracts.
package jetstream

import (
	"context"
	"fmt"

	"github.com/zzokki81/eventmesh/pkg/broker"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
)

// publisher is the JetStream-backed broker.Publisher implementation.
type publisher struct {
	js jetstream.JetStream
}

// NewPublisher returns a broker.Publisher backed by the given JetStream context.
func NewPublisher(js jetstream.JetStream) broker.Publisher {
	return &publisher{js: js}
}

// Publish sends msg and waits for the server ack. msg.Headers is copied onto
// the NATS message first; trace context carried in ctx is injected on top,
// so consumers can continue the trace regardless of what the caller set.
func (p *publisher) Publish(ctx context.Context, msg broker.Message) error {
	natsMsg := &nats.Msg{
		Subject: msg.Subject,
		Data:    msg.Data,
		Header:  nats.Header{},
	}
	for k, v := range msg.Headers {
		natsMsg.Header.Set(k, v)
	}
	otel.GetTextMapPropagator().Inject(ctx, natsHeaderCarrier(natsMsg.Header))

	if _, err := p.js.PublishMsg(ctx, natsMsg); err != nil {
		return fmt.Errorf("publish to %s: %w", msg.Subject, err)
	}
	return nil
}
