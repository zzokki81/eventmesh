package order

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/zzokki81/eventmesh/internal/entities/order"
	"github.com/zzokki81/eventmesh/internal/pkg/event"
	"github.com/zzokki81/eventmesh/internal/transports/broker"

	storage "github.com/zzokki81/eventmesh/internal/storage/orders"
)

// Service orchestrates order operations: persistence and event publishing.
type Service struct {
	// storage persists and retrieves Order aggregates.
	storage storage.OrderRepository

	// publisher emits domain events to the broker.
	publisher broker.Publisher

	// eventBuilder wraps payloads in a standard envelope.
	eventBuilder *event.Builder

	// logger records service-level lifecycle events.
	logger *slog.Logger
}

// NewService wires the order service with its dependencies.
func NewService(storage storage.OrderRepository, publisher broker.Publisher, eventBuilder *event.Builder, logger *slog.Logger) *Service {
	return &Service{
		storage:      storage,
		publisher:    publisher,
		eventBuilder: eventBuilder,
		logger:       logger,
	}
}

// Create validates the request, persists a new order, and publishes an
// orders.created event. The order is returned on success.
func (s *Service) Create(ctx context.Context, req *order.CreateRequest) (*order.Order, error) {
	if err := req.Validate(); err != nil {
		s.logger.DebugContext(ctx, "order create: validation failed",
			"user_id", req.UserID, "err", err)
		return nil, err
	}

	o := order.New(req.UserID, req.Amount)

	if err := s.storage.Create(ctx, o); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	envelope, err := s.eventBuilder.Build(order.TopicCreated, order.TopicCreatedVersion, order.NewOrderCreated(o))
	if err != nil {
		return nil, fmt.Errorf("build order created event: %w", err)
	}

	if err := s.publisher.Publish(ctx, order.TopicCreated, envelope); err != nil {
		return nil, fmt.Errorf("publish event %s for persisted order %s: %w",
			envelope.ID, o.ID, err)
	}

	s.logger.InfoContext(ctx, "order created",
		"order_id", o.ID, "user_id", o.UserID, "event_id", envelope.ID)
	return o, nil
}
