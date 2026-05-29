package jetstream

import "time"

// SubscriberConfig holds the configuration required to bind a Subscriber
// to a JetStream durable consumer.
type SubscriberConfig struct {
	// StreamName is the JetStream stream to attach to.
	StreamName string

	// ConsumerName is the durable consumer identifier; it must be stable
	// across restarts for redelivery and offset tracking to work.
	ConsumerName string

	// Subject is the filter subject applied to the consumer.
	Subject string

	// AckWait bounds how long the server waits for an ack before redelivering.
	AckWait time.Duration

	// MaxDeliver caps the number of redelivery attempts before the message
	// is terminated server-side.
	MaxDeliver int
}
