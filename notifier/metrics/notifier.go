// Package metrics defines the notifier's business counters.
package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

// Notifier holds the notifier's business counters. A nil *Notifier is a no-op,
// so callers without metrics configured (e.g. tests) may pass nil.
type Notifier struct {
	duplicatesSkipped metric.Int64Counter
	emailsSent        metric.Int64Counter
	emailsFailed      metric.Int64Counter
}

// NewNotifier creates the business counters on meter.
func NewNotifier(meter metric.Meter) (*Notifier, error) {
	duplicatesSkipped, err := meter.Int64Counter(
		"notifier.events.duplicates_skipped",
		metric.WithDescription("Duplicate order events skipped by deduplication."),
	)
	if err != nil {
		return nil, fmt.Errorf("duplicates counter: %w", err)
	}

	emailsSent, err := meter.Int64Counter(
		"notifier.emails.sent",
		metric.WithDescription("Notification emails sent."),
	)
	if err != nil {
		return nil, fmt.Errorf("emails counter: %w", err)
	}

	emailsFailed, err := meter.Int64Counter(
		"notifier.emails.failed",
		metric.WithDescription("Notification emails that failed to send."),
	)
	if err != nil {
		return nil, fmt.Errorf("emails failed counter: %w", err)
	}

	return &Notifier{duplicatesSkipped: duplicatesSkipped, emailsSent: emailsSent, emailsFailed: emailsFailed}, nil
}

// RecordDuplicateSkipped counts one duplicate event skipped by deduplication.
func (m *Notifier) RecordDuplicateSkipped(ctx context.Context) {
	if m == nil {
		return
	}
	m.duplicatesSkipped.Add(ctx, 1)
}

// RecordEmailSent counts one notification email sent.
func (m *Notifier) RecordEmailSent(ctx context.Context) {
	if m == nil {
		return
	}
	m.emailsSent.Add(ctx, 1)
}

// RecordEmailFailed counts one notification email that failed to send.
func (m *Notifier) RecordEmailFailed(ctx context.Context) {
	if m == nil {
		return
	}
	m.emailsFailed.Add(ctx, 1)
}
