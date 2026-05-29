package outbox

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zzokki81/eventmesh/order/entities/outbox"
)

// OutboxRepository persists events into the transactional outbox.
//
// The outbox guarantees that an event is recorded in the same database
// transaction as the business write that produced it. A separate relay
// reads pending rows and publishes them to the broker, eliminating the
// dual-write gap between persistence and message delivery.
type OutboxRepository interface {
	// Create records oe within the caller-owned transaction. The
	// transaction must commit for the event to be considered durable.
	Create(ctx context.Context, tx pgx.Tx, oe *outbox.Event) error

	// ListPending returns up to limit pending events, locking each
	// row with FOR UPDATE SKIP LOCKED so concurrent relays do not pick
	// the same event. The transaction must be committed by the caller.
	ListPending(ctx context.Context, tx pgx.Tx, limit int) ([]*outbox.Event, error)

	// MarkAsPublished transitions the event to the published terminal
	// state and stamps processed_at. The row is no longer returned by
	// subsequent ListPending calls.
	MarkAsPublished(ctx context.Context, tx pgx.Tx, id uuid.UUID) error

	// MarkAsFailed increments the attempt counter for the event. If
	// the new count reaches maxAttempts, the event transitions to the
	// dead terminal state; otherwise it stays pending for another try.
	MarkAsFailed(ctx context.Context, tx pgx.Tx, id uuid.UUID, maxAttempts int) error
}
