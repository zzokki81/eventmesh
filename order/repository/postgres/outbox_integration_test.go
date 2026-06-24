//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zzokki81/eventmesh/order/domain"
)

// inTx runs fn inside a committed transaction, failing the test on any error.
// It mirrors how the service and relay drive the repository: every write is
// bound to a caller-owned transaction.
func inTx(t *testing.T, ctx context.Context, fn func(tx pgx.Tx) error) {
	t.Helper()
	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := fn(tx); err != nil {
		t.Fatalf("run in tx: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit tx: %v", err)
	}
}

// insertPending inserts n pending outbox events, each in its own committed
// transaction, and returns their ids in insertion order.
func insertPending(t *testing.T, ctx context.Context, repo *OutboxRepository, n int) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, 0, n)
	for i := 0; i < n; i++ {
		oe := domain.NewOutboxEvent(uuid.New(), "orders.created", []byte(`{}`))
		inTx(t, ctx, func(tx pgx.Tx) error {
			return repo.Create(ctx, tx, oe)
		})
		ids = append(ids, oe.ID)
	}
	return ids
}

// outboxRow reads the status and attempt count of a single outbox row.
func outboxRow(t *testing.T, ctx context.Context, id uuid.UUID) (status string, attempts int) {
	t.Helper()
	const q = "SELECT status, attempt_count FROM outbox_events WHERE id = $1"
	if err := testPool.QueryRow(ctx, q, id).Scan(&status, &attempts); err != nil {
		t.Fatalf("read outbox row %s: %v", id, err)
	}
	return status, attempts
}

// TestOutbox_CreateAndListPending verifies that a freshly created event lands
// in pending status and is returned by ListPending.
func TestOutbox_CreateAndListPending(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	repo := NewOutbox()

	ids := insertPending(t, ctx, repo, 1)

	var got []*domain.OutboxEvent
	inTx(t, ctx, func(tx pgx.Tx) error {
		var err error
		got, err = repo.ListPending(ctx, tx, 10)
		return err
	})

	if len(got) != 1 {
		t.Fatalf("ListPending returned %d events, want 1", len(got))
	}
	if got[0].ID != ids[0] {
		t.Errorf("ListPending returned id %s, want %s", got[0].ID, ids[0])
	}
	if got[0].Status != domain.OutboxStatusPending {
		t.Errorf("status = %q, want %q", got[0].Status, domain.OutboxStatusPending)
	}
}

// TestOutbox_SkipLockedPreventsDoublePick is the core concurrency guarantee:
// two relays polling at the same time must never be handed the same row. One
// transaction locks a batch and is held open while a second reads; the second
// must skip the locked rows (rather than block on them) and receive a disjoint
// batch. If the query used a plain FOR UPDATE the second read would block until
// the first transaction finished and this test would hang.
func TestOutbox_SkipLockedPreventsDoublePick(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	repo := NewOutbox()

	insertPending(t, ctx, repo, 4)

	// tx1 locks the first two rows and is deliberately NOT committed yet, so
	// those rows stay locked while tx2 reads.
	tx1, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx1: %v", err)
	}
	defer tx1.Rollback(ctx) //nolint:errcheck

	batch1, err := repo.ListPending(ctx, tx1, 2)
	if err != nil {
		t.Fatalf("tx1 list pending: %v", err)
	}
	if len(batch1) != 2 {
		t.Fatalf("tx1 got %d events, want 2", len(batch1))
	}

	// tx2 reads while tx1 still holds its locks. SKIP LOCKED means this returns
	// promptly with the other rows instead of blocking.
	tx2, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx2: %v", err)
	}
	defer tx2.Rollback(ctx) //nolint:errcheck

	batch2, err := repo.ListPending(ctx, tx2, 2)
	if err != nil {
		t.Fatalf("tx2 list pending: %v", err)
	}
	if len(batch2) != 2 {
		t.Fatalf("tx2 got %d events, want 2", len(batch2))
	}

	// The two batches must not overlap: no event handed to two relays.
	locked := make(map[uuid.UUID]bool, len(batch1))
	for _, e := range batch1 {
		locked[e.ID] = true
	}
	for _, e := range batch2 {
		if locked[e.ID] {
			t.Errorf("event %s was picked by both transactions", e.ID)
		}
	}
}

// TestOutbox_MarkAsPublished verifies a published row leaves the pending set.
func TestOutbox_MarkAsPublished(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	repo := NewOutbox()

	ids := insertPending(t, ctx, repo, 2)
	publishedID, remainingID := ids[0], ids[1]

	inTx(t, ctx, func(tx pgx.Tx) error {
		return repo.MarkAsPublished(ctx, tx, publishedID)
	})

	if status, _ := outboxRow(t, ctx, publishedID); status != string(domain.OutboxStatusPublished) {
		t.Errorf("published row status = %q, want %q", status, domain.OutboxStatusPublished)
	}

	var got []*domain.OutboxEvent
	inTx(t, ctx, func(tx pgx.Tx) error {
		var err error
		got, err = repo.ListPending(ctx, tx, 10)
		return err
	})
	if len(got) != 1 {
		t.Fatalf("ListPending returned %d events, want 1", len(got))
	}
	if got[0].ID != remainingID {
		t.Errorf("ListPending returned %s, want the unpublished %s", got[0].ID, remainingID)
	}
}

// TestOutbox_MarkAsFailedDeadAfterMaxAttempts verifies the retry budget: a
// failed event stays pending until the attempt count reaches maxAttempts, then
// transitions to dead and leaves the pending set.
func TestOutbox_MarkAsFailedDeadAfterMaxAttempts(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	repo := NewOutbox()

	const maxAttempts = 3
	id := insertPending(t, ctx, repo, 1)[0]

	// First two failures: still pending, attempt count climbs.
	for attempt := 1; attempt < maxAttempts; attempt++ {
		inTx(t, ctx, func(tx pgx.Tx) error {
			return repo.MarkAsFailed(ctx, tx, id, maxAttempts)
		})
		status, attempts := outboxRow(t, ctx, id)
		if status != string(domain.OutboxStatusPending) {
			t.Fatalf("after %d failures status = %q, want pending", attempt, status)
		}
		if attempts != attempt {
			t.Errorf("after %d failures attempt_count = %d, want %d", attempt, attempts, attempt)
		}
	}

	// Final failure reaches the budget: the row goes dead.
	inTx(t, ctx, func(tx pgx.Tx) error {
		return repo.MarkAsFailed(ctx, tx, id, maxAttempts)
	})
	if status, attempts := outboxRow(t, ctx, id); status != string(domain.OutboxStatusDead) || attempts != maxAttempts {
		t.Errorf("after max attempts status=%q attempts=%d, want dead %d", status, attempts, maxAttempts)
	}

	// A dead row must not be returned to the relay.
	var got []*domain.OutboxEvent
	inTx(t, ctx, func(tx pgx.Tx) error {
		var err error
		got, err = repo.ListPending(ctx, tx, 10)
		return err
	})
	if len(got) != 0 {
		t.Errorf("ListPending returned %d events, want 0 (dead excluded)", len(got))
	}
}
