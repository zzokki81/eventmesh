package domain

// OutboxStatus represents the lifecycle state of an outbox event.
type OutboxStatus string

const (
	// OutboxStatusPending indicates the event is awaiting processing by the relay.
	OutboxStatusPending OutboxStatus = "pending"

	// OutboxStatusPublished indicates the event has been successfully published to the broker.
	OutboxStatusPublished OutboxStatus = "published"

	// OutboxStatusDead indicates the event exhausted its publish attempts and will not be retried.
	OutboxStatusDead OutboxStatus = "dead"
)
