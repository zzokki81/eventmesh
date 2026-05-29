// Package nats provides JetStream-backed implementations of the broker
// Publisher and Subscriber contracts.
package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/zzokki81/eventmesh/pkg/broker"
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
func (p *publisher) Publish(ctx context.Context, subject string, data []byte) error {
	if _, err := p.js.Publish(ctx, subject, data); err != nil {
		return fmt.Errorf("publish raw to %s: %w", subject, err)
	}
	return nil
}
