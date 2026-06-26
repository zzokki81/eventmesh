package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Order represents a customer's order in the system.
type Order struct {
	// ID is the unique identifier of the order.
	ID uuid.UUID

	// UserID identifies the user who placed the order.
	UserID uuid.UUID

	// UserEmail is the email address of the user
	UserEmail string

	// Amount is the order total.
	Amount decimal.Decimal

	// Status is the current lifecycle state of the order.
	Status OrderStatus

	// CreatedAt is the timestamp when the order was created.
	CreatedAt time.Time

	// UpdatedAt is the timestamp of the last modification.
	UpdatedAt time.Time
}

// New constructs a new Order with a generated ID, current timestamps,
// and Status set to OrderStatusPending. The caller is responsible for
// validating inputs before invoking New.
func New(userID uuid.UUID, userEmail string, amount decimal.Decimal) *Order {
	now := time.Now().UTC()
	return &Order{
		ID:        uuid.New(),
		UserID:    userID,
		UserEmail: userEmail,
		Amount:    amount,
		Status:    OrderStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ParseOrderID parses a raw string into an order ID.
// Returns ErrInvalidOrderID if id is not a valid UUID.
func ParseOrderID(id string) (uuid.UUID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, ErrInvalidOrderID
	}
	return uid, nil
}
