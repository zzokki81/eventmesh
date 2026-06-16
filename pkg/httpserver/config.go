package httpserver

import "time"

// Config holds the settings required to run an HTTP server.
type Config struct {
	// Addr is the HTTP server listen address. Required with no default: each
	// service must set its own port explicitly to avoid colliding on a shared one.
	Addr string `env:"HTTP_ADDR" validate:"required,hostname_port"`

	// ReadTimeout is the maximum duration for reading the entire request.
	ReadTimeout time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"10s" validate:"min=1s"`

	// WriteTimeout is the maximum duration before timing out response writes.
	WriteTimeout time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"10s" validate:"min=1s"`

	// IdleTimeout is the maximum time to wait for the next request when keep-alives are enabled.
	IdleTimeout time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"120s" validate:"min=1s"`

	// ShutdownTimeout is the maximum duration to wait for graceful shutdown.
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" envDefault:"15s" validate:"min=1s"`

	// ReadHeaderTimeout is the maximum duration for reading request headers.
	ReadHeaderTimeout time.Duration `env:"HTTP_READ_HEADER_TIMEOUT" envDefault:"5s" validate:"min=1s"`
}
