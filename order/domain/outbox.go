package domain

import (
	"time"

	"github.com/google/uuid"
)

// OutboxEvent is a row in the outbox waiting to be relayed to the broker.
type OutboxEvent struct {
	// ID uniquely identifies this outbox row and matches the envelope ID
	// it carries, so the broker can dedupe redeliveries by message id.
	ID uuid.UUID

	// AggregateID identifies the business entity this event describes
	// (for example, the order id). Consumers use it to correlate or
	// replay events per aggregate.
	AggregateID uuid.UUID

	// Type is the semantic event name (e.g. "orders.created"). The relay
	// uses it as the NATS subject when publishing.
	Type string

	// Payload is the serialized envelope ready for the wire. The outbox
	// stores it opaquely so the serialization format can evolve without
	// schema changes.
	Payload []byte

	// TraceContext carries the W3C trace-context propagation fields
	// (traceparent, tracestate) captured when the event was created, so the
	// relay can continue the originating request's trace when it publishes.
	// Empty when no trace was active. The service sets it; the domain stays
	// free of any tracing dependency.
	TraceContext map[string]string

	// Status reflects the row's lifecycle: pending until the relay
	// publishes it, published on success, or dead after attempts are
	// exhausted and operator review is required.
	Status OutboxStatus

	// AttemptCount records how many times the relay has tried to publish
	// this event. It drives retry backoff and the transition to dead.
	AttemptCount int

	// CreatedAt is the UTC timestamp when the event was inserted, set in
	// the same transaction as the business write.
	CreatedAt time.Time

	// ProcessedAt is set when the row reaches a terminal status
	// (published or dead). It stays nil while the event is pending.
	ProcessedAt *time.Time
}

// NewOutboxEvent builds a new outbox event ready for insertion. The event
// starts in pending status with zero attempts and a fresh UTC creation
// timestamp; the relay updates Status, AttemptCount, and ProcessedAt as
// it progresses.
func NewOutboxEvent(aggregateID uuid.UUID, eventType string, payload []byte) *OutboxEvent {
	return &OutboxEvent{
		ID:           uuid.New(),
		AggregateID:  aggregateID,
		Type:         eventType,
		Payload:      payload,
		Status:       OutboxStatusPending,
		AttemptCount: 0,
		CreatedAt:    time.Now().UTC(),
	}
}
