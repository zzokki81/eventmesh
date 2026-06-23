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

// Publish sends pre-serialized data to subject and waits for the server ack.
// Any trace context carried in ctx is injected into the message headers so
// consumers can continue the trace.
func (p *publisher) Publish(ctx context.Context, subject string, data []byte) error {
	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  nats.Header{},
	}
	otel.GetTextMapPropagator().Inject(ctx, natsHeaderCarrier(msg.Header))

	if _, err := p.js.PublishMsg(ctx, msg); err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}
	return nil
}
