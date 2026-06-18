package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/zzokki81/eventmesh/notifier/domain"
	"github.com/zzokki81/eventmesh/notifier/service"
	"github.com/zzokki81/eventmesh/pkg/event"
)

// errMalformedPayload marks a permanently undecodable payload. It is the only
// non-retryable error this handler produces; everything the service returns
// (in-progress claims, dedup or send failures) is transient.
var errMalformedPayload = errors.New("malformed payload")

// orderCreatedPayload mirrors the orders.created event schema on the wire.
// Defined locally to avoid cross-service import; must stay in sync with the producer.
type orderCreatedPayload struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"user_id"`
	UserEmail string          `json:"user_email"`
	Amount    decimal.Decimal `json:"amount"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
}

// OrderCreatedHandler adapts orders.created broker messages to the notifier
// service: it decodes the payload, delegates processing, and lets the subscriber
// translate the returned error into ack/nak/term.
type OrderCreatedHandler struct {
	// Notifier service. Responsible for deduplicating orders and dispatching notifications.
	notifier service.Notifier

	// Logger. Used for logging events.
	logger *slog.Logger
}

// NewOrderCreatedHandler returns an OrderCreatedHandler backed by logger and notifier.
func NewOrderCreatedHandler(notifier service.Notifier, logger *slog.Logger) *OrderCreatedHandler {
	return &OrderCreatedHandler{notifier: notifier, logger: logger}
}

// Handle decodes the orders.created payload and delegates to the service.
func (h *OrderCreatedHandler) Handle(ctx context.Context, env *event.Envelope) error {
	var payload orderCreatedPayload
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		return fmt.Errorf("%w: %w", errMalformedPayload, err)
	}

	return h.notifier.ProcessOrderCreated(ctx, env.ID.String(), domain.OrderCreated{
		OrderID:   payload.ID,
		UserEmail: payload.UserEmail,
		Amount:    payload.Amount,
	})
}

// IsRetryable reports whether a handler error should trigger redelivery. Only a
// malformed payload is permanent; transient failures are retried.
func (h *OrderCreatedHandler) IsRetryable(err error) bool {
	return !errors.Is(err, errMalformedPayload)
}
