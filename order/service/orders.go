// Package service defines the contract for the order service.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/zzokki81/eventmesh/order/domain"
)

// Orders is the port that HTTP handlers and other entry points use
// to drive order use cases.
type Orders interface {
	// Create creates a new order from the given request and returns it.
	Create(ctx context.Context, req *domain.CreateRequest) (*domain.Order, error)

	// Get retrieves an order by ID. Returns domain.ErrOrderNotFound if no
	// order matches.
	Get(ctx context.Context, id uuid.UUID) (*domain.Order, error)
}
