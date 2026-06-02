package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zzokki81/eventmesh/order/domain"
)

// Orders persists and retrieves Order aggregates.
type Orders interface {
	// Create inserts a new order within the caller-owned transaction. The
	// transaction must commit for the order to be persisted.
	Create(ctx context.Context, tx pgx.Tx, o *domain.Order) error

	// GetByID retrieves an order by its unique identifier. If no order exists
	// with the given ID, it returns domain.ErrOrderNotFound.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)
}

// Outbox persists events into the transactional outbox.
//
// The outbox guarantees that an event is recorded in the same database
// transaction as the business write that produced it. A separate relay
// reads pending rows and publishes them to the broker, eliminating the
// dual-write gap between persistence and message delivery.
type Outbox interface {
	// Create records oe within the caller-owned transaction. The
	// transaction must commit for the event to be considered durable.
	Create(ctx context.Context, tx pgx.Tx, oe *domain.OutboxEvent) error

	// ListPending returns up to limit pending events, locking each
	// row with FOR UPDATE SKIP LOCKED so concurrent relays do not pick
	// the same event. The transaction must be committed by the caller.
	ListPending(ctx context.Context, tx pgx.Tx, limit int) ([]*domain.OutboxEvent, error)

	// MarkAsPublished transitions the event to the published terminal
	// state and stamps processed_at. The row is no longer returned by
	// subsequent ListPending calls.
	MarkAsPublished(ctx context.Context, tx pgx.Tx, id uuid.UUID) error

	// MarkAsFailed increments the attempt counter for the event. If
	// the new count reaches maxAttempts, the event transitions to the
	// dead terminal state; otherwise it stays pending for another try.
	MarkAsFailed(ctx context.Context, tx pgx.Tx, id uuid.UUID, maxAttempts int) error
}
