package order

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Order represents a customer's order in the system.
type Order struct {
	// Id is the unique identifier for the order.
	ID uuid.UUID

	// UserId is the identifier of the user who placed the order.
	UserID uuid.UUID

	// Amount is the total amount for the order.
	Amount decimal.Decimal

	// Status represents the current status of the order (e.g., pending, completed, processing, failed).
	Status Status

	// CreatedAt is the timestamp when the order was created.
	CreatedAt time.Time

	// UpdatedAt is the timestamp when the order was last updated.
	UpdatedAt time.Time
}

// Validate checks if the Order's fields are valid. It returns an error if any field is invalid.
func (o *Order) Validate() error {
	if o.UserID == uuid.Nil {
		return ErrInvalidUserID
	}
	if o.Amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}
	if !o.Status.IsValid() {
		return ErrInvalidStatus
	}

	return nil
}

// New creates a new Order with generated ID and timestamps, initialized to pending status.
// Returns an error if the input fails validation.
func New(userID uuid.UUID, amount decimal.Decimal) (*Order, error) {
	now := time.Now()
	o := &Order{
		ID:        uuid.New(),
		UserID:    userID,
		Amount:    amount,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := o.Validate(); err != nil {
		return nil, err
	}

	return o, nil
}
