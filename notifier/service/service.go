package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/zzokki81/eventmesh/notifier/dedup"
	"github.com/zzokki81/eventmesh/notifier/domain"
	"github.com/zzokki81/eventmesh/notifier/email"
)

// notifier is the default Notifier implementation. It deduplicates with a
// two-phase claim and dispatches the notification between the two phases.
type notifier struct {
	dedup  dedup.Deduplicator
	email  email.Sender
	logger *slog.Logger
}

// NewNotifier wires the notifier service with its dependencies.
func NewNotifier(deduper dedup.Deduplicator, sender email.Sender, logger *slog.Logger) Notifier {
	return &notifier{dedup: deduper, email: sender, logger: logger}
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

// send builds the order-confirmation email and dispatches it via the sender.
func (n *notifier) send(ctx context.Context, o domain.OrderCreated) error {
	msg := email.Message{
		To:      o.UserEmail,
		Subject: fmt.Sprintf("Order %s confirmed", o.OrderID),
		Plain: fmt.Sprintf(
			"Thank you for your order.\n\nOrder ID: %s\nAmount: %s\n",
			o.OrderID, o.Amount,
		),
	}

	if err := n.email.Send(ctx, msg); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	n.logger.InfoContext(ctx, "order notification sent",
		"order_id", o.OrderID,
		"user_email", o.UserEmail,
	)
	return nil
}
