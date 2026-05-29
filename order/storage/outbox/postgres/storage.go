package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zzokki81/eventmesh/order/entities/outbox"

	storage "github.com/zzokki81/eventmesh/order/storage/outbox"
)

// Storage is the PostgreSQL-backed implementation of outbox.OutboxRepository.
//
// It writes envelopes into the outbox_events table using a transaction owned
// by the caller; the relay reads from the same table and publishes pending
// rows to the broker. Storage holds no connection of its own — every write
// is bound to the transaction it receives.
type Storage struct{}

// NewStorage returns a Storage ready to use within any caller-supplied transaction.
func NewStorage() *Storage {
	return &Storage{}
}

// Create inserts the outbox event into outbox_events within tx.
// The transaction must be committed for the event to be durable; the
// caller is responsible for rolling back on failure. The row stays in
// pending status until the relay processes it.
func (s *Storage) Create(ctx context.Context, tx pgx.Tx, oe *outbox.Event) error {
	q := `INSERT INTO outbox_events (id, aggregate_id, event_type, payload, created_at)
	           VALUES ($1, $2, $3, $4, $5)`

	if _, err := tx.Exec(ctx, q,
		oe.ID,
		oe.AggregateID,
		oe.Type,
		oe.Payload,
		oe.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}

	return nil
}

// ListPending returns up to limit pending events ordered by creation time.
// Each row is locked with FOR UPDATE SKIP LOCKED so concurrent relays do not
// pick the same event; rows already locked by another transaction are skipped.
func (s *Storage) ListPending(ctx context.Context, tx pgx.Tx, limit int) ([]*outbox.Event, error) {
	q := `SELECT id, aggregate_id, event_type, payload, status, attempt_count, created_at, processed_at
	           FROM outbox_events
	           WHERE status = 'pending'
	           ORDER BY created_at
	           LIMIT $1
	           FOR UPDATE SKIP LOCKED`

	rows, err := tx.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("query pending outbox events: %w", err)
	}
	defer rows.Close()

	var events []*outbox.Event
	for rows.Next() {
		var oe outbox.Event
		if err := rows.Scan(
			&oe.ID,
			&oe.AggregateID,
			&oe.Type,
			&oe.Payload,
			&oe.Status,
			&oe.AttemptCount,
			&oe.CreatedAt,
			&oe.ProcessedAt,
		); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		events = append(events, &oe)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox rows: %w", err)
	}
	return events, nil
}

// MarkAsPublished transitions the event to the published terminal state
// and stamps processed_at. The row will no longer be returned by ListPending.
func (s *Storage) MarkAsPublished(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	q := `UPDATE outbox_events
	           SET status = 'published', processed_at = NOW()
	           WHERE id = $1`

	if _, err := tx.Exec(ctx, q, id); err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}
	return nil
}

// MarkAsFailed increments the attempt counter. If the new count reaches
// maxAttempts the event transitions to the dead terminal state; otherwise it
// stays pending so the relay can retry.
func (s *Storage) MarkAsFailed(ctx context.Context, tx pgx.Tx, id uuid.UUID, maxAttempts int) error {
	q := `UPDATE outbox_events
	           SET attempt_count = attempt_count + 1,
	               status = CASE WHEN attempt_count + 1 >= $2 THEN 'dead' ELSE 'pending' END,
	               processed_at = CASE WHEN attempt_count + 1 >= $2 THEN NOW() ELSE processed_at END
	           WHERE id = $1`

	if _, err := tx.Exec(ctx, q, id, maxAttempts); err != nil {
		return fmt.Errorf("mark outbox event failed: %w", err)
	}
	return nil
}

// Compile-time check that *Storage satisfies the outbox.OutboxRepository contract.
var _ storage.OutboxRepository = (*Storage)(nil)
