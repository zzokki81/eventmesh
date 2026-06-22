package relay

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zzokki81/eventmesh/order/config"
	"github.com/zzokki81/eventmesh/order/metrics"
	"github.com/zzokki81/eventmesh/order/repository"
	"github.com/zzokki81/eventmesh/pkg/broker"
)

// Relay drains the transactional outbox by polling pending rows and
// forwarding them to the broker. It closes the dual-write gap: the
// business write and the outbox row commit together, and the relay
// publishes asynchronously with at-least-once delivery semantics.
type Relay struct {
	pool      *pgxpool.Pool
	outbox    repository.Outbox
	publisher broker.Publisher
	cfg       config.RelayConfig
	metrics   *metrics.Relay
	logger    *slog.Logger
}

// New returns a Relay wired with its dependencies. m may be nil, in which case
// operational metrics are not recorded.
func New(
	pool *pgxpool.Pool,
	outbox repository.Outbox,
	publisher broker.Publisher,
	cfg config.RelayConfig,
	m *metrics.Relay,
	logger *slog.Logger,
) *Relay {
	return &Relay{
		pool:      pool,
		outbox:    outbox,
		publisher: publisher,
		cfg:       cfg,
		metrics:   m,
		logger:    logger,
	}
}

// Run starts the polling loop and blocks until ctx is canceled.
// Each tick processes at most BatchSize events in a single transaction.
func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.cfg.TickInterval)
	defer ticker.Stop()

	r.logger.Info("relay started",
		"batch_size", r.cfg.BatchSize,
		"tick_interval", r.cfg.TickInterval,
		"max_attempts", r.cfg.MaxAttempts,
	)

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("relay stopped")
			return nil
		case <-ticker.C:
			if err := r.processBatch(ctx); err != nil {
				r.logger.ErrorContext(ctx, "relay batch failed", "err", err)
			}
		}
	}
}

// processBatch reads a batch of pending events under a single transaction.
// Each event is published; the row is marked published on success or failed
// (with attempt count incremented) on failure. FOR UPDATE SKIP LOCKED in the
// repository ensures concurrent relays do not pick the same row.
func (r *Relay) processBatch(ctx context.Context) error {
	start := time.Now()
	defer func() { r.metrics.RecordPollDuration(ctx, time.Since(start)) }()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	events, err := r.outbox.ListPending(ctx, tx, r.cfg.BatchSize)
	if err != nil {
		return fmt.Errorf("list pending: %w", err)
	}

	if len(events) == 0 {
		return nil
	}

	// Accumulate counts and record them only after the commit succeeds, so a
	// rolled-back batch (whose rows stay pending and get retried) is not counted.
	var published, failed int64
	for _, e := range events {
		if err := r.publisher.Publish(ctx, e.Type, e.Payload); err != nil {
			r.logger.WarnContext(ctx, "publish failed, will retry",
				"event_id", e.ID,
				"type", e.Type,
				"attempt", e.AttemptCount+1,
				"err", err,
			)
			failed++
			if markErr := r.outbox.MarkAsFailed(ctx, tx, e.ID, r.cfg.MaxAttempts); markErr != nil {
				r.logger.ErrorContext(ctx, "mark as failed failed",
					"event_id", e.ID, "err", markErr,
				)
			}
			continue
		}

		published++
		if err := r.outbox.MarkAsPublished(ctx, tx, e.ID); err != nil {
			r.logger.ErrorContext(ctx, "mark as published failed",
				"event_id", e.ID, "err", err,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	r.metrics.RecordRowsPublished(ctx, published)
	r.metrics.RecordPublishErrors(ctx, failed)
	return nil
}
