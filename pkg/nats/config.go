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
	// it is considered undeliverable and moved to a dead letter queue (if configured).
	MaxDeliver int `env:"NATS_MAX_DELIVER" envDefault:"5" validate:"min=1,max=100"`

	// PingInterval is the duration to wait between sending ping messages to the NATS server.
	PingInterval time.Duration `env:"NATS_PING_INTERVAL" envDefault:"10s" validate:"min=1s"`

	// MaxPingsOut is the maximum number of ping messages that can be outstanding before the connection is considered lost.
	MaxPingsOut int `env:"NATS_MAX_PINGS_OUT" envDefault:"3" validate:"min=1,max=10"`

	// ConnectTimeout is the duration to wait for a connection to the NATS server to be established before timing out.
	ConnectTimeout time.Duration `env:"NATS_CONNECT_TIMEOUT" envDefault:"5s" validate:"min=1s"`
}
