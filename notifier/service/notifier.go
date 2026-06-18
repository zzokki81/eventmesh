// Package service holds the notifier's business logic: turning order events
// into notifications exactly once per event.
package service

import (
	"context"

	"github.com/zzokki81/eventmesh/notifier/domain"
)

// Notifier processes order events into notifications, at most once per event.
type Notifier interface {
	// ProcessOrderCreated handles one orders.created event identified by eventID.
	// It is safe against duplicate deliveries: an already-processed event is a
	// no-op, and ErrInProgress is returned when another attempt holds the event.
	ProcessOrderCreated(ctx context.Context, eventID string, o domain.OrderCreated) error
}
