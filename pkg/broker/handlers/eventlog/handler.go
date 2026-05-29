// Package eventlog provides a MessageHandler that records each envelope
// it receives. It is useful as a passive observer, for audit, or as a
// scaffold while real business handlers are being developed.
package eventlog

import (
	"context"
	"log/slog"

	"github.com/zzokki81/eventmesh/pkg/event"
)

// Handler logs envelope metadata at debug level.
type Handler struct {
	logger *slog.Logger
}

// New returns a Handler that writes to the given logger.
func New(logger *slog.Logger) *Handler {
	return &Handler{logger: logger}
}

// Handle logs the envelope and returns nil.
func (h *Handler) Handle(ctx context.Context, env *event.Envelope) error {
	h.logger.DebugContext(ctx, "event received",
		"event_id", env.ID,
		"type", env.Type,
		"version", env.Version,
		"producer", env.Producer,
		"occurred_at", env.OccurredAt,
	)
	return nil
}

// IsRetryable always returns false. A logging failure is not a business
// operation worth retrying; redelivery would burn cycles without recovery.
func (*Handler) IsRetryable(error) bool { return false }
