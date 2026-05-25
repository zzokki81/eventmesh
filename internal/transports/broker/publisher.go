package broker

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

// Publisher publishes events to NATS JetStream.
//
// It wraps jetstream.JetStream to provide a place for cross-cutting concerns
// such as logging, tracing, and metrics. The publisher does not enforce any
// subject convention; it only guarantees that the server acknowledged the
// published message.
type Publisher struct {
	js jetstream.JetStream
}

// NewPublisher creates a Publisher backed by the given JetStream context.
func NewPublisher(js jetstream.JetStream) *Publisher {
	return &Publisher{js: js}
}

// Publish publishes data to the given subject and waits for a server ack.
// It returns an error if publishing fails or the message is not acknowledged.
func (p *Publisher) Publish(ctx context.Context, subject string, data []byte) error {
	if _, err := p.js.Publish(ctx, subject, data); err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}
	return nil
}
