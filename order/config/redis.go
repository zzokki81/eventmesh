package config

import "time"

// RedisConfig represents the configuration for Redis database connection.
type RedisConfig struct {
	// Addr is the Redis server address in the format host:port.
	Addr string `env:"REDIS_ADDR" validate:"required,hostname_port"`

	// Password is the password used for Redis authentication.
	Password string `env:"REDIS_PASSWORD" validate:"omitempty,min=8"`

	// DB is the Redis database index.
	DB int `env:"REDIS_DB" envDefault:"0" validate:"min=0,max=15"`

	// PoolSize is the maximum number of connections in the pool.
	PoolSize int `env:"REDIS_POOL_SIZE" envDefault:"10" validate:"min=1,max=1000"`

	// MinIdleConns is the minimum number of idle connections kept in the pool.
	MinIdleConns int `env:"REDIS_MIN_IDLE_CONNS" envDefault:"5" validate:"min=0,ltefield=PoolSize"`

	// DialTimeout is the timeout for establishing new connections.
	DialTimeout time.Duration `env:"REDIS_DIAL_TIMEOUT" envDefault:"3s" validate:"min=1s"`

	// ReadTimeout is the timeout for socket reads.
	ReadTimeout time.Duration `env:"REDIS_READ_TIMEOUT" envDefault:"1s" validate:"min=100ms"`

	// WriteTimeout is the timeout for socket writes.
	WriteTimeout time.Duration `env:"REDIS_WRITE_TIMEOUT" envDefault:"1s" validate:"min=100ms"`

	// PoolTimeout is the maximum wait time for a connection from the pool.
	PoolTimeout time.Duration `env:"REDIS_POOL_TIMEOUT" envDefault:"4s" validate:"min=1s"`
}
