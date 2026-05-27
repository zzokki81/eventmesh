package config

import "time"

// RelayConfig holds settings for the relay process that reads pending outbox events and publishes them to the broker.
type RelayConfig struct {
	// BatchSize determines how many pending events the relay fetches and processes in each cycle.
	// A larger batch size may improve throughput but holds row locks longer while the batch publishes.
	BatchSize int `env:"RELAY_BATCH_SIZE" envDefault:"100" validate:"required,min=1,max=500"`

	// TickInterval sets the frequency at which the relay wakes up to check for pending events.
	TickInterval time.Duration `env:"RELAY_TICK_INTERVAL" envDefault:"500ms" validate:"required,min=50ms"`

	// MaxAttempts specifies how many times the relay will try to publish an event before marking it as dead.
	MaxAttempts int `env:"RELAY_MAX_ATTEMPTS" envDefault:"5" validate:"required,min=1,max=100"`
}
