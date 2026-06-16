package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zzokki81/eventmesh/order/domain"
	"github.com/zzokki81/eventmesh/order/repository"
	"github.com/zzokki81/eventmesh/pkg/event"
)

// orders orchestrates order creation: it persists the order and enqueues the
// resulting event in the outbox within a single transaction. A separate relay
// publishes the enqueued events to the broker.
type orders struct {
	// orderStorage persists and retrieves Order aggregates.
	orderStorage repository.Orders

	// outboxStorage enqueues events in the transactional outbox.
	outboxStorage repository.Outbox

	// pool owns the transaction that spans the order write and outbox enqueue.
	pool *pgxpool.Pool

	// eventBuilder wraps payloads in a standard envelope.
	eventBuilder *event.Builder

	// logger records service-level lifecycle events.
	logger *slog.Logger
}

// NewOrders wires the order service with its dependencies.
func NewOrders(
	orderStorage repository.Orders,
	outboxStorage repository.Outbox,
	pool *pgxpool.Pool,
	eventBuilder *event.Builder,
	logger *slog.Logger,
) Orders {
	return &orders{
		orderStorage:  orderStorage,
		outboxStorage: outboxStorage,
		pool:          pool,
		eventBuilder:  eventBuilder,
		logger:        logger,
	}
}

// Create validates the request, persists a new order, and publishes an
// orders.created event. The order is returned on success.
func (s *orders) Create(ctx context.Context, req *domain.CreateRequest) (*domain.Order, error) {
	o := domain.New(req.UserID, req.UserEmail, req.Amount)

	envelope, err := s.eventBuilder.Build(event.SubjectOrderCreated, event.VersionOrderCreated, domain.NewOrderCreated(o))
	if err != nil {
		return nil, fmt.Errorf("build order created event: %w", err)
	}

	payload, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal envelope: %w", err)
	}

	outboxEvent := domain.NewOutboxEvent(o.ID, event.SubjectOrderCreated, payload)

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
