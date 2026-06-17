package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/zzokki81/eventmesh/pkg/event"
)

// orderCreatedPayload mirrors the orders.created event schema.
// Defined locally to avoid cross-service import; must stay in sync with the producer.
type orderCreatedPayload struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"user_id"`
	UserEmail string          `json:"user_email"`
	Amount    decimal.Decimal `json:"amount"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
}

// OrderCreatedHandler processes orders.created events and dispatches notifications.
type OrderCreatedHandler struct {
	logger *slog.Logger
}

// NewOrderCreatedHandler returns an OrderCreatedHandler backed by logger.
func NewOrderCreatedHandler(logger *slog.Logger) *OrderCreatedHandler {
	return &OrderCreatedHandler{logger: logger}
}

// Handle decodes the orders.created payload and dispatches a notification.
// Email delivery will replace the log statement once the email sender is wired in.
func (h *OrderCreatedHandler) Handle(ctx context.Context, env *event.Envelope) error {
	var payload orderCreatedPayload
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		return fmt.Errorf("decode orders.created payload: %w", err)
	}

	h.logger.InfoContext(ctx, "order notification dispatched",
		"event_id", env.ID,
		"order_id", payload.ID,
		"user_email", payload.UserEmail,
		"amount", payload.Amount,
	)

	return nil
}

// IsRetryable returns false — a malformed payload cannot be fixed by redelivery.
func (h *OrderCreatedHandler) IsRetryable(error) bool { return false }
