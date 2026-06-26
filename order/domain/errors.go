package domain

import "errors"

var (
	// ErrOrderNotFound is returned when an order cannot be found.
	ErrOrderNotFound = errors.New("order not found")

	// ErrInvalidOrderID is returned when an order ID is not a valid UUID.
	ErrInvalidOrderID = errors.New("invalid order id")

	// ErrInvalidOrderAmount is returned when the order amount is zero or negative.
	ErrInvalidOrderAmount = errors.New("invalid order amount")

	// ErrInvalidOrderUserID is returned when the user ID is missing or invalid.
	ErrInvalidOrderUserID = errors.New("invalid order user id")

	// ErrInvalidOrderStatus is returned when the order status is not a recognized value.
	ErrInvalidOrderStatus = errors.New("invalid order status")
)
