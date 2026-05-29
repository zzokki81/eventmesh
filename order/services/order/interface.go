// Package order defines the contract for the order service.
// Implementations live in subpackages (e.g. services/order/order).
package order

import (
	"context"

	"github.com/zzokki81/eventmesh/order/entities/order"
)

// Service is the port that HTTP handlers and other entry points use
// to drive order use cases.
type Service interface {
	// Create creates a new order from the given request and returns it.
	Create(ctx context.Context, req *order.CreateRequest) (*order.Order, error)
}
