// Package domain holds the notifier's core types, independent of transport and
// storage concerns.
package domain

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OrderCreated is the data a notification needs about a created order. It is a
// clean domain type, decoupled from the wire payload the transport decodes.
type OrderCreated struct {
	// OrderID identifies the order the notification is about.
	OrderID uuid.UUID

	// UserEmail is the recipient of the notification.
	UserEmail string

	// Amount is the order total, included in the notification.
	Amount decimal.Decimal
}
