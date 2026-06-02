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
)

// New creates a new slog.Logger configured according to cfg.
// It returns an error if the configuration values cannot be translated
// into a valid slog handler (e.g. unknown level or format).
func New(cfg Config) (*slog.Logger, error) {
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
func parseLevel(l Level) (slog.Level, error) {
	switch l {
	case LevelDebug:
		return slog.LevelDebug, nil
	case LevelInfo:
		return slog.LevelInfo, nil
	case LevelError:
		return slog.LevelError, nil
	case LevelWarn:
		return slog.LevelWarn, nil
	default:
		return 0, fmt.Errorf("unknown log level %q", l)
	}
}

// parseOutput returns the io.Writer matching the configured output destination.
func parseOutput(o Output) (io.Writer, error) {
	if o == OutputStderr {
		return os.Stderr, nil
	}

	if o == OutputStdout {
		return os.Stdout, nil
	}

	return nil, fmt.Errorf("unknown log output: %q", o)
}

// buildHandler constructs a slog.Handler in the requested format.
func buildHandler(format Format, w io.Writer, opts *slog.HandlerOptions) (slog.Handler, error) {
	switch format {
	case FormatJSON:
		return slog.NewJSONHandler(w, opts), nil
	case FormatText:
		return slog.NewTextHandler(w, opts), nil
	case FormatPretty:
		return tint.NewHandler(w, &tint.Options{
			Level:      opts.Level,
			AddSource:  opts.AddSource,
			TimeFormat: time.Kitchen,
		}), nil
	default:
		return nil, fmt.Errorf("unknown log format: %q", format)
	}
}
