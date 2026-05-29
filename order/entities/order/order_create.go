package order

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/zzokki81/eventmesh/order/errs"
)

// CreateRequest is the parsed input for creating a new order.
type CreateRequest struct {
	// UserID identifies the user placing the order.
	UserID uuid.UUID

	// Amount is the order total; must be greater than zero.
	Amount decimal.Decimal
}

// Validate enforces domain invariants on the request.
// Returns errs.ErrInvalidOrderUserID or errs.ErrInvalidOrderAmount.
func (c *CreateRequest) Validate() error {
	if c.UserID == uuid.Nil {
		return errs.ErrInvalidOrderUserID
	}
	if c.Amount.LessThanOrEqual(decimal.Zero) {
		return errs.ErrInvalidOrderAmount
	}
	return nil
}

// CreateRequestFrom parses raw string inputs into a CreateRequest.
// Format errors map to the corresponding invalid-field sentinels.
func CreateRequestFrom(userID, amount string) (*CreateRequest, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errs.ErrInvalidOrderUserID
	}

	amt, err := decimal.NewFromString(amount)
	if err != nil {
		return nil, errs.ErrInvalidOrderAmount
	}

	return &CreateRequest{UserID: uid, Amount: amt}, nil
}
