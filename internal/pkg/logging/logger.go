// Package logging provides a factory for constructing slog.Logger instances
// from application configuration.
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"

	"github.com/zzokki81/eventmesh/internal/pkg/appconfig"
)

// New creates a new slog.Logger configured according to cfg.
// It returns an error if the configuration values cannot be translated
// into a valid slog handler (e.g. unknown level or format).
//
// Global tags from cfg.Tags are attached to every log record produced
// by the returned logger via slog.With.
func New(cfg appconfig.LoggerConfig) (*slog.Logger, error) {
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
func parseLevel(l appconfig.LogLevel) (slog.Level, error) {
	switch l {
	case appconfig.LogLevelDebug:
		return slog.LevelDebug, nil
	case appconfig.LogLevelInfo:
		return slog.LevelInfo, nil
	case appconfig.LogLevelError:
		return slog.LevelError, nil
	case appconfig.LogLevelWarn:
		return slog.LevelWarn, nil
	default:
		return 0, fmt.Errorf("unknown log level %q", l)
	}
}

// parseOutput returns the io.Writer matching the configured output destination.
func parseOutput(o appconfig.LogOutput) (io.Writer, error) {
	if o == appconfig.LogOutputStderr {
		return os.Stderr, nil
	}

	if o == appconfig.LogOutputStdout {
		return os.Stdout, nil
	}

	return nil, fmt.Errorf("unknown log output: %q", o)
}

// buildHandler constructs a slog.Handler in the requested format.
func buildHandler(format appconfig.LogFormat, w io.Writer, opts *slog.HandlerOptions) (slog.Handler, error) {
	switch format {
	case appconfig.LogFormatJSON:
		return slog.NewJSONHandler(w, opts), nil
	case appconfig.LogFormatText:
		return slog.NewTextHandler(w, opts), nil
	case appconfig.LogFormatPretty:
		return tint.NewHandler(w, &tint.Options{
			Level:      opts.Level,
			AddSource:  opts.AddSource,
			TimeFormat: time.Kitchen,
		}), nil
	default:
		return nil, fmt.Errorf("unknown log format: %q", format)
	}
}
