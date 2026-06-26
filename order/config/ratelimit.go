package config

// RateLimitConfig holds settings for the Redis-backed rate limiter on the
// order write path.
type RateLimitConfig struct {
	// Enabled toggles the rate limiter. Disabled by default so local
	// development and tests are not gated on a reachable Redis.
	Enabled bool `env:"RATE_LIMIT_ENABLED" envDefault:"false"`

	// RPS is the sustained number of requests allowed per second, per client.
	RPS int `env:"RATE_LIMIT_RPS" envDefault:"20" validate:"required,min=1"`

	// Burst is the number of requests a client may send instantly above RPS
	// before being throttled, e.g. after an idle period.
	Burst int `env:"RATE_LIMIT_BURST" envDefault:"40" validate:"required,min=1"`

	// TrustProxyHeaders makes the limiter read the client IP from
	// X-Forwarded-For / X-Real-IP instead of the raw TCP connection. Only
	// enable this once the service sits behind a reverse proxy or load
	// balancer that sets those headers itself on every inbound request —
	// otherwise a client can forge them to dodge or hijack another client's
	// rate limit bucket.
	TrustProxyHeaders bool `env:"RATE_LIMIT_TRUST_PROXY_HEADERS" envDefault:"false"`
}
