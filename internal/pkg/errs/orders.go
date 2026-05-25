package errs

import "errors"

var (
	// ErrNotFound is returned when an order cannot be found.
	ErrNotFound = errors.New("order not found")

	// ErrInvalidAmount is returned when the order amount is zero or negative.
	ErrInvalidAmount = errors.New("invalid order amount")

	// ErrInvalidUserID is returned when the user ID is missing or invalid.
	ErrInvalidUserID = errors.New("invalid order user id")

	// ErrInvalidStatus is returned when the order status is not a recognized value.
	ErrInvalidStatus = errors.New("invalid order status")
)
