package domain

// OrderStatus represents the current status of an order.
type OrderStatus string

const (
	// OrderStatusPending indicates that the order is pending and has not been processed yet.
	OrderStatusPending OrderStatus = "pending"
	// OrderStatusProcessing indicates that the order is currently being processed.
	OrderStatusProcessing OrderStatus = "processing"
	// OrderStatusCompleted indicates that the order has been completed successfully.
	OrderStatusCompleted OrderStatus = "completed"
	// OrderStatusFailed indicates that the order processing has failed.
	OrderStatusFailed OrderStatus = "failed"
)

// IsValid checks if the OrderStatus value is one of the defined valid statuses.
func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusPending, OrderStatusProcessing, OrderStatusCompleted, OrderStatusFailed:
		return true
	default:
		return false
	}
}
