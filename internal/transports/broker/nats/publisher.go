// Package nats provides JetStream-backed implementations of the broker
// Publisher and Subscriber contracts.
package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/zzokki81/eventmesh/internal/pkg/event"
	"github.com/zzokki81/eventmesh/internal/transports/broker"
)

// publisher is the JetStream-backed broker.Publisher implementation.
type publisher struct {
	js jetstream.JetStream
}

// NewPublisher returns a broker.Publisher backed by the given JetStream context.
func NewPublisher(js jetstream.JetStream) broker.Publisher {
	return &publisher{js: js}
}

// Publish marshals env to JSON and publishes it to subject, waiting for
// a server-side acknowledgement.
func (p *publisher) Publish(ctx context.Context, subject string, env *event.Envelope) error {
	body, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	if _, err := p.js.Publish(ctx, subject, body); err != nil {
		return fmt.Errorf("publish event to %s: %w", subject, err)
	}
	return nil
}
