// Package observability wires OpenTelemetry for the services. It currently sets
// up metrics exported to Prometheus; tracing will be added on the same provider
// foundation.
package observability

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
)

// Metrics holds the configured Prometheus registry and a shutdown hook for the
// meter provider that backs it.
type Metrics struct {
	// Registry is served on /metrics.
	Registry *prometheus.Registry

	shutdown func(context.Context) error
}

// Shutdown flushes and stops the meter provider. The caller must invoke it on
// exit.
func (m *Metrics) Shutdown(ctx context.Context) error {
	return m.shutdown(ctx)
}

// SetupMetrics configures a global OTel meter provider that exports through a
// Prometheus registry, returning a Metrics handle that exposes the registry to
// serve on /metrics and a Shutdown hook to flush the provider on exit.
func SetupMetrics(ctx context.Context, service, version string) (*Metrics, error) {
	reg := prometheus.NewRegistry()

	exporter, err := otelprom.New(otelprom.WithRegisterer(reg))
	if err != nil {
		return nil, fmt.Errorf("create prometheus exporter: %w", err)
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", service),
		attribute.String("service.version", version),
	))
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	provider := metric.NewMeterProvider(
		metric.WithReader(exporter),
		metric.WithResource(res),
	)
	otel.SetMeterProvider(provider)

	return &Metrics{Registry: reg, shutdown: provider.Shutdown}, nil
}
