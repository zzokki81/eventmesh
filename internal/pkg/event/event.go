// Package event defines the standard envelope used for all events
// published by the service.
package event

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Envelope is the canonical wrapper around a domain event payload.
// Consumers can route, version, and trace events without inspecting Data.
type Envelope struct {
	// ID uniquely identifies this event instance.
	ID uuid.UUID `json:"id"`

	// Type is the semantic event name (e.g. "orders.created").
	Type string `json:"type"`

	// Version is the schema version of Data.
	Version string `json:"version"`

	// OccurredAt is the UTC timestamp when the event was produced.
	OccurredAt time.Time `json:"occurred_at"`

	// Producer identifies the service that emitted the event.
	Producer string `json:"producer"`

	// Data is the domain-specific JSON payload.
	Data json.RawMessage `json:"data"`
}

// Builder stamps envelopes with a fixed producer identifier.
type Builder struct {
	// producer is the identifier written into every Envelope.Producer.
	producer string
}

// NewBuilder returns a Builder that tags envelopes with the given producer.
func NewBuilder(producer string) *Builder {
	return &Builder{producer: producer}
}

// Build marshals data as JSON and wraps it in a fresh Envelope.
// A new ID and OccurredAt timestamp are generated for each call.
func (b *Builder) Build(eventType, version string, data any) (*Envelope, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal event data: %w", err)
	}

	return &Envelope{
		ID:         uuid.New(),
		Type:       eventType,
		Version:    version,
		OccurredAt: time.Now().UTC(),
		Producer:   b.producer,
		Data:       payload,
	}, nil
}
