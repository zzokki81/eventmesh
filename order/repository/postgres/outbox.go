package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zzokki81/eventmesh/order/domain"
	"github.com/zzokki81/eventmesh/order/repository"
)

// OutboxRepository is the PostgreSQL-backed implementation of repository.Outbox.
//
// It writes envelopes into the outbox_events table using a transaction owned
// by the caller; the relay reads from the same table and publishes pending
// rows to the broker. OutboxRepository holds no connection of its own — every
// write is bound to the transaction it receives.
type OutboxRepository struct{}

// NewOutbox returns an OutboxRepository ready to use within any caller-supplied
// transaction.
func NewOutbox() *OutboxRepository {
	return &OutboxRepository{}
}

// Create inserts the outbox event into outbox_events within tx.
// The transaction must be committed for the event to be durable; the
// caller is responsible for rolling back on failure. The row stays in
// pending status until the relay processes it.
func (s *OutboxRepository) Create(ctx context.Context, tx pgx.Tx, oe *domain.OutboxEvent) error {
	// Marshal the trace context to JSON for the trace_context column. A nil
	// slice is encoded as SQL NULL, so events without an active trace store NULL.
	var traceContext []byte
	if len(oe.TraceContext) > 0 {
		var err error
		traceContext, err = json.Marshal(oe.TraceContext)
		if err != nil {
			return fmt.Errorf("marshal trace context: %w", err)
		}
	}

	q := `INSERT INTO outbox_events (id, aggregate_id, event_type, payload, trace_context, created_at)
	           VALUES ($1, $2, $3, $4, $5, $6)`
	if _, err := tx.Exec(ctx, q,
		oe.ID,
		oe.AggregateID,
		oe.Type,
		oe.Payload,
		traceContext,
		oe.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

// ListPending returns up to limit pending events ordered by creation time.
// Each row is locked with FOR UPDATE SKIP LOCKED so concurrent relays do not
// pick the same event; rows already locked by another transaction are skipped.
func (s *OutboxRepository) ListPending(ctx context.Context, tx pgx.Tx, limit int) ([]*domain.OutboxEvent, error) {
	q := `SELECT id, aggregate_id, event_type, payload, trace_context, status, attempt_count, created_at, processed_at
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

	var events []*domain.OutboxEvent
	for rows.Next() {
		var oe domain.OutboxEvent
		// trace_context is read as raw JSON and unmarshaled below; a NULL column
		// scans into a nil slice, leaving TraceContext empty.
		var traceContext []byte
		if err := rows.Scan(
			&oe.ID,
			&oe.AggregateID,
			&oe.Type,
			&oe.Payload,
			&traceContext,
			&oe.Status,
			&oe.AttemptCount,
			&oe.CreatedAt,
			&oe.ProcessedAt,
		); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		if len(traceContext) > 0 {
			if err := json.Unmarshal(traceContext, &oe.TraceContext); err != nil {
				return nil, fmt.Errorf("unmarshal trace context: %w", err)
			}
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
func (s *OutboxRepository) MarkAsPublished(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
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
func (s *OutboxRepository) MarkAsFailed(ctx context.Context, tx pgx.Tx, id uuid.UUID, maxAttempts int) error {
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

// Compile-time check that *OutboxRepository implements repository.Outbox.
var _ repository.Outbox = (*OutboxRepository)(nil)
