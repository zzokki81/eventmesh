// Package logger provides a factory for constructing slog.Logger instances
// from application configuration.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"

	"github.com/zzokki81/eventmesh/order/config"
)

// New creates a new slog.Logger configured according to cfg.
// It returns an error if the configuration values cannot be translated
// into a valid slog handler (e.g. unknown level or format).
func New(cfg config.LoggerConfig) (*slog.Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	writer, err := parseOutput(cfg.Output)
	if err != nil {
		return nil, err
	}

	handler, err := buildHandler(cfg.Format, writer, &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	})
	if err != nil {
		return nil, err
	}

	return slog.New(handler), nil
}

// parseLevel converts a LogLevel into a slog.Level.
func parseLevel(l config.LogLevel) (slog.Level, error) {
	switch l {
	case config.LogLevelDebug:
		return slog.LevelDebug, nil
	case config.LogLevelInfo:
		return slog.LevelInfo, nil
	case config.LogLevelError:
		return slog.LevelError, nil
	case config.LogLevelWarn:
		return slog.LevelWarn, nil
	default:
		return 0, fmt.Errorf("unknown log level %q", l)
	}
}

// parseOutput returns the io.Writer matching the configured output destination.
func parseOutput(o config.LogOutput) (io.Writer, error) {
	if o == config.LogOutputStderr {
		return os.Stderr, nil
	}

	if o == config.LogOutputStdout {
		return os.Stdout, nil
	}

	return nil, fmt.Errorf("unknown log output: %q", o)
}

// buildHandler constructs a slog.Handler in the requested format.
func buildHandler(format config.LogFormat, w io.Writer, opts *slog.HandlerOptions) (slog.Handler, error) {
	switch format {
	case config.LogFormatJSON:
		return slog.NewJSONHandler(w, opts), nil
	case config.LogFormatText:
		return slog.NewTextHandler(w, opts), nil
	case config.LogFormatPretty:
		return tint.NewHandler(w, &tint.Options{
			Level:      opts.Level,
			AddSource:  opts.AddSource,
			TimeFormat: time.Kitchen,
		}), nil
	default:
		return nil, fmt.Errorf("unknown log format: %q", format)
	}
}
