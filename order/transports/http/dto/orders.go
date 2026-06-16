// Package dto holds the request and response data transfer objects
// exchanged over the HTTP transport.
package dto

// CreateOrderRequest is the HTTP request body for creating an order.
// Fields are received as strings and parsed downstream into domain types;
// shape and value validation lives in the order/domain package.
type CreateOrderRequest struct {
	// UserID is the string form of the user identifier (UUID).
	UserID string `json:"user_id" validate:"required,uuid4"`

	// UserEmail is the email address of the user placing the order.
	UserEmail string `json:"user_email" validate:"required,email"`

	// Amount is the order total as a decimal-formatted string.
	Amount string `json:"amount" validate:"required"`
}
