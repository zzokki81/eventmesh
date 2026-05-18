package appconfig

import "time"

type HTTPConfig struct {
	// Addr is the HTTP server listen address.
	Addr string `env:"HTTP_ADDR" envDefault:":8080" validate:"required,hostname_port"`

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
