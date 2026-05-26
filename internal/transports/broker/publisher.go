// Package broker provides outbound messaging primitives for the event broker.
package broker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/zzokki81/eventmesh/internal/pkg/event"
)

// Publisher publishes envelopes to the broker. Implementations are
// responsible for serialization and delivery semantics.
type Publisher interface {
	// Publish sends env to subject and returns once the broker
	// has acknowledged the message.
	Publish(ctx context.Context, subject string, env *event.Envelope) error
}

// jetStreamPublisher is the JetStream-backed Publisher implementation.
type jetStreamPublisher struct {
	// js is the JetStream context used to publish messages.
	js jetstream.JetStream
}

// NewPublisher returns a Publisher backed by the given JetStream context.
func NewPublisher(js jetstream.JetStream) Publisher {
	return &jetStreamPublisher{js: js}
}

// Publish marshals env to JSON and publishes it to subject,
// waiting for a server-side acknowledgement.
func (p *jetStreamPublisher) Publish(ctx context.Context, subject string, env *event.Envelope) error {
	body, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	if _, err := p.js.Publish(ctx, subject, body); err != nil {
		return fmt.Errorf("publish event to %s: %w", subject, err)
	}
	return nil
}
