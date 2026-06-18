package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/zzokki81/eventmesh/notifier/dedup"
	"github.com/zzokki81/eventmesh/notifier/domain"
)

// notifier is the default Notifier implementation. It deduplicates with a
// two-phase claim and dispatches the notification between the two phases.
type notifier struct {
	dedup  dedup.Deduplicator
	logger *slog.Logger
}

// NewNotifier wires the notifier service with its dependencies.
func NewNotifier(deduper dedup.Deduplicator, logger *slog.Logger) Notifier {
	return &notifier{dedup: deduper, logger: logger}
}

// ProcessOrderCreated claims the event, dispatches the notification, then marks
// the event completed. On a dispatch failure it releases the claim so a later
// redelivery can retry.
func (n *notifier) ProcessOrderCreated(ctx context.Context, eventID string, o domain.OrderCreated) error {
	status, err := n.dedup.Claim(ctx, eventID)
	if err != nil {
		return fmt.Errorf("claim event: %w", err)
	}

	switch status {
	case dedup.StatusCompleted:
		n.logger.InfoContext(ctx, "event already processed, skipping", "event_id", eventID)
		return nil
	case dedup.StatusInProgress:
		return domain.ErrInProgress
	}

	// status == StatusNew: we own the claim and must dispatch the notification.
	if err := n.send(ctx, o); err != nil {
		if relErr := n.dedup.Release(ctx, eventID); relErr != nil {
			n.logger.ErrorContext(ctx, "failed to release claim", "event_id", eventID, "err", relErr)
		}
		return fmt.Errorf("send notification: %w", err)
	}

	if err := n.dedup.Complete(ctx, eventID); err != nil {
		// The notification was sent; failing to mark completed only risks a
		// future redelivery re-sending, so log rather than fail the event.
		n.logger.ErrorContext(ctx, "failed to mark event completed", "event_id", eventID, "err", err)
	}

	return nil
}

// send dispatches the notification. For now it logs; an email sender will
// replace the log statement once it is wired in.
func (n *notifier) send(ctx context.Context, o domain.OrderCreated) error {
	n.logger.InfoContext(ctx, "order notification dispatched",
		"order_id", o.OrderID,
		"user_email", o.UserEmail,
		"amount", o.Amount,
	)
	return nil
}
