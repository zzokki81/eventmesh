package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"

	"github.com/zzokki81/eventmesh/pkg/httpserver"
	"github.com/zzokki81/eventmesh/pkg/logger"
	"github.com/zzokki81/eventmesh/pkg/nats"
	"github.com/zzokki81/eventmesh/pkg/observability"
	"github.com/zzokki81/eventmesh/pkg/redis"
)

// Config aggregates all application configuration sections.
type Config struct {
	// HTTP server configuration.
	HTTP httpserver.Config

	// NATS and JetStream messaging configuration.
	NATS nats.Config

	// Redis configuration.
	Redis redis.Config

	// Dedup holds event-deduplication TTLs.
	Dedup DedupConfig

	// SMTP holds settings for sending notification emails.
	SMTP SMTPConfig

	// Observability holds tracing settings (OTLP endpoint).
	Observability observability.Config

	// Logger configuration.
	Logger logger.Config
}

// Load reads configuration from environment variables and returns a Config struct.
// It uses the env package to parse environment variables into the Config struct fields.
// If parsing fails, it returns an error with details about the failure.
func Load() (*Config, error) {
	_ = godotenv.Load("notifier/.env")
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
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
