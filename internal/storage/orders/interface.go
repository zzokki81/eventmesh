package orders

import (
	"context"

	"github.com/google/uuid"

	"github.com/zzokki81/eventmesh/internal/entities/order"
)

// OrderRepository defines the contract for persisting and retrieving orders.
type OrderRepository interface {
	// Create should persist a new order in the repository. Returns an error if the operation fails.
	Create(ctx context.Context, o *order.Order) error

	// GetByID should retrieve an order by its unique identifier. Returns ErrNotFound if no order matches.
	GetByID(ctx context.Context, id uuid.UUID) (*order.Order, error)
}
