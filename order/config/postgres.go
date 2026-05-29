package config

import "time"

// PostgresConfig represents the configuration for PostgreSQL database connection.
type PostgresConfig struct {
	// URL is the connection string for the PostgreSQL database, including credentials and host information.
	URL string `env:"DATABASE_URL" validate:"required,url"`

	// MaxConns is the maximum number of connections the pool can hold at once.
	MaxConns int32 `env:"DB_MAX_CONNS" envDefault:"25" validate:"min=1,max=500"`

	// MinConns is the minimum number of connections the pool keeps open even when idle.
	MinConns int32 `env:"DB_MIN_CONNS" envDefault:"5" validate:"min=0,ltefield=MaxConns"`

	// ConnMaxIdleTime is the maximum amount of time a connection may be idle before being closed.
	ConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE_TIME" envDefault:"5m" validate:"min=1s"`

	// ConnMaxLifetime is the maximum amount of time a connection may be reused before being closed.
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"5m" validate:"min=1s"`

	// QueryTimeout is the duration to wait for a query to complete before timing out.
	QueryTimeout time.Duration `env:"DB_QUERY_TIMEOUT" envDefault:"3s" validate:"min=100ms"`
}
