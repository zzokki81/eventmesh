package config

import "time"

// DedupConfig holds the time-to-live settings for event deduplication.
type DedupConfig struct {
	// ClaimTTL bounds how long an "in progress" claim is held. It must outlast
	// worst-case processing but stay well under the broker's redelivery budget
	// (NATS AckWait * MaxDeliver), so a crashed attempt's claim clears in time
	// for a redelivery to re-claim and finish.
	ClaimTTL time.Duration `env:"DEDUP_CLAIM_TTL" envDefault:"1m" validate:"required,min=1s"`

	// CompletionTTL bounds how long a "completed" mark is kept. It should be at
	// least as long as the stream retention so a late redelivery is still
	// recognized as already processed.
	CompletionTTL time.Duration `env:"DEDUP_COMPLETION_TTL" envDefault:"168h" validate:"required,min=1m"`
}
