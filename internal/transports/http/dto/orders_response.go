package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/zzokki81/eventmesh/internal/entities/order"
)

// OrderResponse is the HTTP response body for order resources.
// It is the public projection of an Order and decouples wire format
// from the internal domain type.
type OrderResponse struct {
	// ID is the order identifier.
	ID uuid.UUID `json:"id"`

	// UserID identifies the user that placed the order.
	UserID uuid.UUID `json:"user_id"`

	// Amount is the order total.
	Amount decimal.Decimal `json:"amount"`

	// Status is the current order status.
	Status order.Status `json:"status"`

	// CreatedAt is the timestamp when the order was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the timestamp of the last modification.
	UpdatedAt time.Time `json:"updated_at"`
}

// OrderResponseFrom projects an Order into the response payload.
func OrderResponseFrom(o *order.Order) OrderResponse {
	return OrderResponse{
		ID:        o.ID,
		UserID:    o.UserID,
		Amount:    o.Amount,
		Status:    o.Status,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}
