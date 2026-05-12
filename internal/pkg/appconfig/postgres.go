package appconfig

import "time"

// PostgresConfig represents the configuration for PostgreSQL database connection.
type PostgresConfig struct {
	// URL is the connection string for the PostgreSQL database, including credentials and host information.
	URL string `env:"DATABASE_URL" validate:"required,url"`

	// MaxOpenConns is the maximum number of open connections to the database.
	MaxOpenConns int `env:"DB_MAX_OPEN_CONNS" envDefault:"25" validate:"min=1,max=500"`

	// MaxIdleConns is the maximum number of connections in the idle connection pool.
	MaxIdleConns int `env:"DB_MAX_IDLE_CONNS" envDefault:"5" validate:"min=0,ltefield=MaxOpenConns"`

	// ConnMaxIdleTime is the maximum amount of time a connection may be idle before being closed.
	ConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE_TIME" envDefault:"5m" validate:"min=1s"`

	// ConnMaxLifetime is the maximum amount of time a connection may be reused before being closed.
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"5m" validate:"min=1s"`

	// QueryTimeout is the duration to wait for a query to complete before timing out.
	QueryTimeout time.Duration `env:"DB_QUERY_TIMEOUT" envDefault:"3s" validate:"min=100ms"`
}
