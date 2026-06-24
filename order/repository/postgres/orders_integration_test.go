//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/zzokki81/eventmesh/order/domain"
)

// newTestOrder builds a valid pending order for insertion.
func newTestOrder() *domain.Order {
	return domain.New(uuid.New(), "buyer@example.com", decimal.NewFromInt(4200))
}

// countOutbox returns how many outbox rows exist for the given aggregate id.
func countOutbox(t *testing.T, ctx context.Context, aggregateID uuid.UUID) int {
	t.Helper()
	var n int
	const q = "SELECT count(*) FROM outbox_events WHERE aggregate_id = $1"
	if err := testPool.QueryRow(ctx, q, aggregateID).Scan(&n); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	return n
}

// TestOrders_CreateCommitsOrderAndOutboxTogether verifies the happy path of the
// transactional outbox: when the business write and the outbox enqueue share a
// transaction and it commits, both are durable.
func TestOrders_CreateCommitsOrderAndOutboxTogether(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	orders := NewOrders(testPool)
	outbox := NewOutbox()

	o := newTestOrder()
	oe := domain.NewOutboxEvent(o.ID, "orders.created", []byte(`{}`))

	inTx(t, ctx, func(tx pgx.Tx) error {
		if err := orders.Create(ctx, tx, o); err != nil {
			return err
		}
		return outbox.Create(ctx, tx, oe)
	})

	got, err := orders.GetByID(ctx, o.ID)
	if err != nil {
		t.Fatalf("GetByID after commit: %v", err)
	}
	if got.ID != o.ID {
		t.Errorf("GetByID returned id %s, want %s", got.ID, o.ID)
	}
	if n := countOutbox(t, ctx, o.ID); n != 1 {
		t.Errorf("outbox rows = %d, want 1", n)
	}
}

// TestOrders_RollbackDropsOrderAndOutboxTogether is the dual-write guarantee:
// because the order and its outbox event are written in one transaction, a
// rollback discards both. The system can never end up with an order that has no
// event, or an event with no order.
func TestOrders_RollbackDropsOrderAndOutboxTogether(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	orders := NewOrders(testPool)
	outbox := NewOutbox()

	o := newTestOrder()
	oe := domain.NewOutboxEvent(o.ID, "orders.created", []byte(`{}`))

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if err := orders.Create(ctx, tx, o); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := outbox.Create(ctx, tx, oe); err != nil {
		t.Fatalf("create outbox: %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback tx: %v", err)
	}

	if _, err := orders.GetByID(ctx, o.ID); !errors.Is(err, domain.ErrOrderNotFound) {
		t.Errorf("GetByID after rollback err = %v, want ErrOrderNotFound", err)
	}
	if n := countOutbox(t, ctx, o.ID); n != 0 {
		t.Errorf("outbox rows after rollback = %d, want 0", n)
	}
}

// TestOrders_GetByIDNotFound verifies the not-found mapping for an unknown id.
func TestOrders_GetByIDNotFound(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	orders := NewOrders(testPool)

	if _, err := orders.GetByID(ctx, uuid.New()); !errors.Is(err, domain.ErrOrderNotFound) {
		t.Errorf("GetByID err = %v, want ErrOrderNotFound", err)
	}
}
