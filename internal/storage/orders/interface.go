// Package orders defines the storage contract for order persistence.
// Concrete implementations live in subpackages (e.g. orders/postgres).
package orders

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zzokki81/eventmesh/internal/entities/order"
)

// OrderRepository persists and retrieves Order aggregates.
type OrderRepository interface {
	// CreateInTx inserts a new order within the caller-owned transaction. The
	// transaction must commit for the order to be persisted.
	CreateInTx(ctx context.Context, tx pgx.Tx, o *order.Order) error

	// GetByID retrieves an order by its unique identifier. If no order exists
	// with the given ID, it returns order.ErrNotFound.
	GetByID(ctx context.Context, id uuid.UUID) (*order.Order, error)
}
