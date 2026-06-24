package nats

import "time"

// Config holds all settings required to establish and maintain a NATS connection.
type Config struct {
	// URL is the connection string for the NATS server.
	URL string `env:"NATS_URL" validate:"required,url"`

	// StreamName is the JetStream stream name.
	StreamName string `env:"NATS_STREAM_NAME" envDefault:"EVENTMESH" validate:"required,min=1,max=255"`

	// ConsumerName is the durable consumer name.
	ConsumerName string `env:"NATS_CONSUMER_NAME" envDefault:"eventmesh-worker" validate:"required,min=1,max=255"`

	// ReconnectWait is the duration to wait before attempting to reconnect to the NATS server after a disconnection.
	ReconnectWait time.Duration `env:"NATS_RECONNECT_WAIT" envDefault:"2s" validate:"min=100ms"`

	// MaxReconnects is the maximum number of reconnection attempts before giving up.
	// A value of -1 means to keep trying indefinitely.
	MaxReconnects int `env:"NATS_MAX_RECONNECTS" envDefault:"-1" validate:"min=-1"`

	// AckWait is the duration the server waits for an ACK before redelivering the message.
	AckWait time.Duration `env:"NATS_ACK_WAIT" envDefault:"30s" validate:"min=1s"`

	// MaxDeliver is the maximum number of times a message will be delivered before
	// it is considered undeliverable and moved to the dead-letter queue.
	MaxDeliver int `env:"NATS_MAX_DELIVER" envDefault:"5" validate:"min=1,max=100"`

	// RetryBackoff is the schedule of growing pauses applied between redelivery
	// attempts of a retryable failure. The Nth retry waits RetryBackoff[N];
	// once the schedule is exhausted the last pause is reused. A retry that
	// fails on the final allowed delivery is dead-lettered instead.
	RetryBackoff []time.Duration `env:"NATS_RETRY_BACKOFF" envDefault:"1s,5s,30s,2m" envSeparator:"," validate:"min=1"`

	// DLQStreamName is the JetStream stream that captures dead-lettered events.
	// It is separate from the main stream so poison messages can be inspected
	// and redriven without disturbing live traffic.
	DLQStreamName string `env:"NATS_DLQ_STREAM_NAME" envDefault:"EVENTMESH_DLQ" validate:"required,min=1,max=255"`

	// DLQSubjectPrefix is prepended to an event's original subject to form its
	// dead-letter subject (e.g. "dlq" + "orders.created" -> "dlq.orders.created").
	// The DLQ stream binds the "<prefix>.>" wildcard.
	DLQSubjectPrefix string `env:"NATS_DLQ_SUBJECT_PREFIX" envDefault:"dlq" validate:"required,min=1"`

	// PingInterval is the duration to wait between sending ping messages to the NATS server.
	PingInterval time.Duration `env:"NATS_PING_INTERVAL" envDefault:"10s" validate:"min=1s"`

	// MaxPingsOut is the maximum number of ping messages that can be outstanding before the connection is considered lost.
	MaxPingsOut int `env:"NATS_MAX_PINGS_OUT" envDefault:"3" validate:"min=1,max=10"`

	// ConnectTimeout is the duration to wait for a connection to the NATS server to be established before timing out.
	ConnectTimeout time.Duration `env:"NATS_CONNECT_TIMEOUT" envDefault:"5s" validate:"min=1s"`
}
