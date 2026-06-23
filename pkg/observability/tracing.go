package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

// Tracing holds a shutdown hook for the tracer provider configured by
// SetupTracing.
type Tracing struct {
	shutdown func(context.Context) error
}

// Shutdown flushes any buffered spans and stops the tracer provider. The caller
// must invoke it on exit so in-flight spans are not lost.
func (t *Tracing) Shutdown(ctx context.Context) error {
	return t.shutdown(ctx)
}

// SetupTracing configures a global OTel tracer provider that exports spans to an
// OTLP endpoint (e.g. Jaeger) over gRPC. It also installs the W3C trace-context
// propagator so context can travel across service boundaries (HTTP headers, NATS
// message headers). endpoint is a host:port without scheme, e.g. "localhost:4317".
//
// It returns a Tracing handle whose Shutdown hook must be invoked on exit.
func SetupTracing(ctx context.Context, service, version, endpoint string) (*Tracing, error) {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", service),
		attribute.String("service.version", version),
	))
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		// Sample every trace: this is a demo where each trace should be visible.
		trace.WithSampler(trace.AlwaysSample()),
	)
	otel.SetTracerProvider(provider)

	// W3C trace-context as the wire format, plus baggage for any propagated
	// key/values. The same propagator is used for HTTP and NATS headers.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Tracing{shutdown: provider.Shutdown}, nil
}
