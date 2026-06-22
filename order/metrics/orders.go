// Package metrics defines the order service's business counters.
package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

// Orders holds the order service's business counters. A nil *Orders is a
// no-op, so callers without metrics configured (e.g. tests) may pass nil.
type Orders struct {
	ordersCreated     metric.Int64Counter
	orderCreateErrors metric.Int64Counter
}

// NewOrders creates the business counters on meter.
func NewOrders(meter metric.Meter) (*Orders, error) {
	ordersCreated, err := meter.Int64Counter(
		"order.orders.created",
		metric.WithDescription("Orders successfully created."),
	)
	if err != nil {
		return nil, fmt.Errorf("orders created counter: %w", err)
	}

	orderCreateErrors, err := meter.Int64Counter(
		"order.orders.create_errors",
		metric.WithDescription("Order creation failures."),
	)
	if err != nil {
		return nil, fmt.Errorf("order create errors counter: %w", err)
	}

	return &Orders{ordersCreated: ordersCreated, orderCreateErrors: orderCreateErrors}, nil
}

// RecordOrderCreated counts one successfully created order.
func (m *Orders) RecordOrderCreated(ctx context.Context) {
	if m == nil {
		return
	}
	m.ordersCreated.Add(ctx, 1)
}

// RecordOrderCreateError counts one failed order creation attempt.
func (m *Orders) RecordOrderCreateError(ctx context.Context) {
	if m == nil {
		return
	}
	m.orderCreateErrors.Add(ctx, 1)
}
