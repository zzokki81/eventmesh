package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	// TopicCreated is the NATS subject for order creation events. The envelope's
	// Type field carries the same value, so a single constant binds routing and
	// schema identity for producers and consumers alike.
	TopicCreated = "orders.created"

	// TopicCreatedVersion identifies the OrderCreated schema. Bump it on any
	// breaking change so consumers can branch on payload shape.
	TopicCreatedVersion = "1"
)

// OrderBase is the public projection of an Order embedded in every order event.
// Its fields form the stable contract event consumers observe, decoupled from
// the internal Order representation so the two can evolve independently.
type OrderBase struct {
	// ID matches the source Order's ID and is the primary correlation key
	// between events and the entity.
	ID uuid.UUID `json:"id"`

	// UserID identifies the user that placed the order.
	UserID uuid.UUID `json:"user_id"`

	// UserEmail is the email address captured at order creation time.
	// Denormalized into the event so consumers do not need to call a user service.
	UserEmail string `json:"user_email"`

	// Amount is the order total.
	Amount decimal.Decimal `json:"amount"`

	// Status is the order status captured at the moment of emission.
	Status OrderStatus `json:"status"`

	// CreatedAt is the UTC timestamp of order creation.
	CreatedAt time.Time `json:"created_at"`
}

// OrderBaseFrom captures the public fields of o at the time of the call.
func OrderBaseFrom(o *Order) OrderBase {
	return OrderBase{
		ID:        o.ID,
		UserID:    o.UserID,
		UserEmail: o.UserEmail,
		Amount:    o.Amount,
		Status:    o.Status,
		CreatedAt: o.CreatedAt,
	}
}

// OrderCreated is the payload for the orders.created event. It currently
// carries no fields beyond OrderBase but exists as a distinct type so that
// future creation-specific data can be added without touching consumers
// of other event variants.
type OrderCreated struct {
	OrderBase
}

// NewOrderCreated constructs the orders.created payload from o.
func NewOrderCreated(o *Order) OrderCreated {
	return OrderCreated{OrderBase: OrderBaseFrom(o)}
}
