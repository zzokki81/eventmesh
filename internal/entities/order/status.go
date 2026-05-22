package order

// Status represents the current status of an order.
type Status string

const (
	// StatusPending indicates that the order is pending and has not been processed yet.
	StatusPending Status = "pending"
	// StatusProcessing indicates that the order is currently being processed.
	StatusProcessing Status = "processing"
	// StatusCompleted indicates that the order has been completed successfully.
	StatusCompleted Status = "completed"
	// StatusFailed indicates that the order processing has failed.
	StatusFailed Status = "failed"
)

// IsValid checks if the Status value is one of the defined valid statuses.
func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusProcessing, StatusCompleted, StatusFailed:
		return true
	default:
		return false
	}
}
