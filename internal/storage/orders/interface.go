// Package orders defines the storage contract for order persistence.
// Concrete implementations live in subpackages (e.g. orders/postgres).
package orders

import (
	"context"

	"github.com/google/uuid"

	"github.com/zzokki81/eventmesh/internal/entities/order"
)

// OrderRepository persists and retrieves Order aggregates.
type OrderRepository interface {
	// Create stores a new order. Returns an error if persistence fails.
	Create(ctx context.Context, o *order.Order) error

	// GetByID returns the order with the given id, or errs.ErrOrderNotFound
	// if no such order exists.
	GetByID(ctx context.Context, id uuid.UUID) (*order.Order, error)
}
