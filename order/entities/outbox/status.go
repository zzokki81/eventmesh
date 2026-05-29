package outbox

// Status represents the lifecycle state of an outbox event.
type Status string

const (
	// StatusPending indicates the event is awaiting processing by the relay.
	StatusPending Status = "pending"

	// StatusPublished indicates the event has been successfully published to the broker.
	StatusPublished Status = "published"

	// StatusDead indicates the event exhausted its publish attempts and will not be retried.
	StatusDead Status = "dead"
)
