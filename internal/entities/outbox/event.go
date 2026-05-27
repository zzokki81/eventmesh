package outbox

import (
	"time"

	"github.com/google/uuid"
)

// Event is a row in the outbox waiting to be relayed to the broker.
type Event struct {
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

	// Status reflects the row's lifecycle: pending until the relay
	// publishes it, published on success, or dead after attempts are
	// exhausted and operator review is required.
	Status Status

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

// New builds a new outbox Event ready for insertion. The event starts in
// pending status with zero attempts and a fresh UTC creation timestamp;
// the relay updates Status, AttemptCount, and ProcessedAt as it progresses.
func New(aggregateID uuid.UUID, eventType string, payload []byte) *Event {
	return &Event{
		ID:           uuid.New(),
		AggregateID:  aggregateID,
		Type:         eventType,
		Payload:      payload,
		Status:       StatusPending,
		AttemptCount: 0,
		CreatedAt:    time.Now().UTC(),
	}
}
