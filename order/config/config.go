package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"

	"github.com/zzokki81/eventmesh/pkg/appenv"
	"github.com/zzokki81/eventmesh/pkg/httpserver"
	"github.com/zzokki81/eventmesh/pkg/logger"
	"github.com/zzokki81/eventmesh/pkg/nats"
	"github.com/zzokki81/eventmesh/pkg/observability"
	"github.com/zzokki81/eventmesh/pkg/postgres"
	"github.com/zzokki81/eventmesh/pkg/redis"
)

// Config aggregates all application configuration sections.
type Config struct {
	// AppEnv is the deployment profile (development or production). It is the
	// single switch for environment-specific defaults: the pprof debug server
	// runs only in development, and the log format/source default to the
	// profile unless LOG_FORMAT / LOG_ADD_SOURCE are set explicitly.
	AppEnv appenv.Environment `env:"APP_ENV" envDefault:"development" validate:"oneof=development production"`

	// HTTP server configuration.
	HTTP httpserver.Config

	// NATS and JetStream messaging configuration.
	NATS nats.Config

	// PostgreSQL database configuration.
	Postgres postgres.Config

	// Redis client configuration.
	Redis redis.Config

	// Relay configuration for the relay component.
	Relay RelayConfig

	// RateLimit configures the Redis-backed rate limiter on the order write path.
	RateLimit RateLimitConfig

	// Observability holds tracing settings (OTLP endpoint).
	Observability observability.Config

	// Logger configuration.
	Logger logger.Config
}

// Load reads configuration from environment variables and returns a Config struct.
// It uses the env package to parse environment variables into the Config struct fields.
// If parsing fails, it returns an error with details about the failure.
func Load() (*Config, error) {
	_ = godotenv.Load("order/.env")
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	applyEnvProfile(cfg)

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// applyEnvProfile fills the log format and source from the deployment profile,
// but only for the knobs the operator did not set explicitly. An explicit
// LOG_FORMAT or LOG_ADD_SOURCE always wins; otherwise development defaults to
// human-readable logs with source info and production to structured JSON.
func applyEnvProfile(cfg *Config) {
	if _, ok := os.LookupEnv("LOG_FORMAT"); !ok {
		if cfg.AppEnv.IsProduction() {
			cfg.Logger.Format = logger.FormatJSON
		} else {
			cfg.Logger.Format = logger.FormatPretty
		}
	}
	if _, ok := os.LookupEnv("LOG_ADD_SOURCE"); !ok {
		cfg.Logger.AddSource = cfg.AppEnv.IsDevelopment()
	}
}

func validate(cfg *Config) error {
	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.Struct(cfg); err != nil {
		if validationErrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
			return formatValidationErrors(validationErrs)
		}
		return fmt.Errorf("validate config: %w", err)
	}
	return nil
}

func formatValidationErrors(errs validator.ValidationErrors) error {
	msgs := make([]string, 0, len(errs))
	for _, e := range errs {
		msgs = append(msgs, fmt.Sprintf(
			"  - %s: failed '%s' rule (got: %v)",
			e.Namespace(),
			e.Tag(),
			e.Value(),
		))
	}
	return fmt.Errorf("invalid config:\n%s", strings.Join(msgs, "\n"))
}
