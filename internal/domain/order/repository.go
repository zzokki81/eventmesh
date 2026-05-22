package order

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the contract for persisting and retrieving orders.
// Implementations live in the infrastructure layer (e.g., postgres).
type Repository interface {
	// Create persists a new order.
	// Returns ErrInvalidAmount, ErrInvalidUserID, or ErrInvalidStatus if validation fails.
	Create(ctx context.Context, order *Order) error

	// GetByID retrieves an order by its unique identifier.
	// Returns ErrNotFound if no order matches.
	GetByID(ctx context.Context, id uuid.UUID) (*Order, error)
}
