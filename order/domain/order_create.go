package domain

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateRequest is the parsed input for creating a new order.
type CreateRequest struct {
	// UserID identifies the user placing the order.
	UserID uuid.UUID

	// UserEmail is the email address of the user placing the order.
	UserEmail string

	// Amount is the order total; must be greater than zero.
	Amount decimal.Decimal
}

// CreateRequestFrom parses and validates raw string inputs into a CreateRequest.
// Returns a domain error sentinel on any parse or invariant failure.
func CreateRequestFrom(userID, userEmail, amount string) (*CreateRequest, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidOrderUserID
	}

	amt, err := decimal.NewFromString(amount)
	if err != nil {
		return nil, ErrInvalidOrderAmount
	}

	if amt.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidOrderAmount
	}

	return &CreateRequest{UserID: uid, UserEmail: userEmail, Amount: amt}, nil
}
