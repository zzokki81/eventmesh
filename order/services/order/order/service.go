package order

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zzokki81/eventmesh/order/entities/order"
	"github.com/zzokki81/eventmesh/order/entities/outbox"
	"github.com/zzokki81/eventmesh/pkg/event"

	orderStorage "github.com/zzokki81/eventmesh/order/storage/orders"
	outboxStorage "github.com/zzokki81/eventmesh/order/storage/outbox"
)

// Service orchestrates order creation: it persists the order and enqueues the
// resulting event in the outbox within a single transaction. A separate relay
// publishes the enqueued events to the broker.
type Service struct {
	// orderStorage persists and retrieves Order aggregates.
	orderStorage orderStorage.OrderRepository

	// outboxStorage enqueues events in the transactional outbox.
	outboxStorage outboxStorage.OutboxRepository

	// pool owns the transaction that spans the order write and outbox enqueue.
	pool *pgxpool.Pool

	// eventBuilder wraps payloads in a standard envelope.
	eventBuilder *event.Builder

	// logger records service-level lifecycle events.
	logger *slog.Logger
}

// NewService wires the order service with its dependencies.
func NewService(
	orderStorage orderStorage.OrderRepository,
	outboxStorage outboxStorage.OutboxRepository,
	pool *pgxpool.Pool,
	eventBuilder *event.Builder,
	logger *slog.Logger,
) *Service {
	return &Service{
		orderStorage:  orderStorage,
		outboxStorage: outboxStorage,
		pool:          pool,
		eventBuilder:  eventBuilder,
		logger:        logger,
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

	envelope, err := s.eventBuilder.Build(order.TopicCreated, order.TopicCreatedVersion, order.NewOrderCreated(o))
	if err != nil {
		return nil, fmt.Errorf("build order created event: %w", err)
	}

	payload, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal envelope: %w", err)
	}

	outboxEvent := outbox.New(o.ID, order.TopicCreated, payload)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	defer tx.Rollback(ctx) //nolint:errcheck

	if err := s.orderStorage.Create(ctx, tx, o); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	if err := s.outboxStorage.Create(ctx, tx, outboxEvent); err != nil {
		return nil, fmt.Errorf("insert outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return o, nil
}
