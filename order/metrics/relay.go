package metrics

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/metric"
)

// Relay holds the outbox relay's operational metrics. A nil *Relay
// is a no-op, so callers without metrics configured (e.g. tests) may pass nil.
type Relay struct {
	rowsPublished metric.Int64Counter
	publishErrors metric.Int64Counter
	pollDuration  metric.Float64Histogram
}

// NewRelay creates the relay counters and histogram on meter.
func NewRelay(meter metric.Meter) (*Relay, error) {
	rowsPublished, err := meter.Int64Counter(
		"order.relay.rows_published",
		metric.WithDescription("Outbox rows successfully published to the broker."),
	)
	if err != nil {
		return nil, fmt.Errorf("rows published counter: %w", err)
	}

	publishErrors, err := meter.Int64Counter(
		"order.relay.publish_errors",
		metric.WithDescription("Outbox rows that failed to publish."),
	)
	if err != nil {
		return nil, fmt.Errorf("publish errors counter: %w", err)
	}

	pollDuration, err := meter.Float64Histogram(
		"order.relay.poll_duration",
		metric.WithDescription("Duration of one outbox poll iteration in seconds."),
		metric.WithUnit("s"),
		// The SDK's default histogram boundaries are millisecond-scale and are
		// not rescaled for the second unit, so a sub-second poll would fall
		// entirely in the first bucket. These boundaries are tuned for seconds.
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5),
	)
	if err != nil {
		return nil, fmt.Errorf("poll duration histogram: %w", err)
	}

	return &Relay{
		rowsPublished: rowsPublished,
		publishErrors: publishErrors,
		pollDuration:  pollDuration,
	}, nil
}

// RecordRowsPublished counts n outbox rows successfully published.
func (m *Relay) RecordRowsPublished(ctx context.Context, n int64) {
	if m == nil || n == 0 {
		return
	}
	m.rowsPublished.Add(ctx, n)
}

// RecordPublishErrors counts n outbox rows that failed to publish.
func (m *Relay) RecordPublishErrors(ctx context.Context, n int64) {
	if m == nil || n == 0 {
		return
	}
	m.publishErrors.Add(ctx, n)
}

// RecordPollDuration records how long one processBatch iteration took.
func (m *Relay) RecordPollDuration(ctx context.Context, d time.Duration) {
	if m == nil {
		return
	}
	m.pollDuration.Record(ctx, d.Seconds())
}
