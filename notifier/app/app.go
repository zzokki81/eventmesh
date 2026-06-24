package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/zzokki81/eventmesh/notifier/config"
	"github.com/zzokki81/eventmesh/notifier/dedup"
	"github.com/zzokki81/eventmesh/notifier/email"
	"github.com/zzokki81/eventmesh/notifier/metrics"
	"github.com/zzokki81/eventmesh/notifier/service"
	"github.com/zzokki81/eventmesh/notifier/transports/broker/handlers"
	"github.com/zzokki81/eventmesh/pkg/broker/jetstream"
	"github.com/zzokki81/eventmesh/pkg/event"
	"github.com/zzokki81/eventmesh/pkg/httpserver"
	"github.com/zzokki81/eventmesh/pkg/logger"
	"github.com/zzokki81/eventmesh/pkg/nats"
	"github.com/zzokki81/eventmesh/pkg/observability"
	"github.com/zzokki81/eventmesh/pkg/redis"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"

	httppkg "github.com/zzokki81/eventmesh/notifier/transports/http"
)

// Run bootstraps the notifier service: loads config, initializes infrastructure,
// starts the HTTP server, and blocks until shutdown signal is received.
func Run() error {
	// --- Configuration ---
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// --- Logger ---
	lg, err := logger.New(cfg.Logger)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	lg = lg.With(
		"service", ServiceName,
		"version", Version,
		"commit", CommitHash,
	)
	slog.SetDefault(lg)
	lg.Info("starting notifier", "http_addr", cfg.HTTP.Addr)

	// --- Signal-aware root context ---
	// Canceled on SIGINT/SIGTERM; propagated to every long-running component
	// so they shut down together.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Redis ---
	redisClient, err := redis.NewClient(ctx, cfg.Redis)
	if err != nil {
		return fmt.Errorf("init redis: %w", err)
	}
	defer redisClient.Close() //nolint:errcheck
	lg.Info("redis connected", "address", cfg.Redis.Addr)

	// --- NATS connection + JetStream ---
	nc, err := nats.NewConnection(cfg.NATS)
	if err != nil {
		return fmt.Errorf("init nats: %w", err)
	}
	defer nc.Drain() //nolint:errcheck
	lg.Info("connected to nats", "url", cfg.NATS.URL)

	js, err := nats.NewJetStream(ctx, nc)
	if err != nil {
		return fmt.Errorf("init jetstream: %w", err)
	}
	lg.Info("jetstream context initialized")

	// Ensure the dead-letter stream exists before the subscriber starts, so
	// poison messages always have somewhere to land.
	if err = nats.SetupDLQStream(ctx, js, cfg.NATS.DLQStreamName, cfg.NATS.DLQSubjectPrefix); err != nil {
		return fmt.Errorf("setup dlq stream: %w", err)
	}
	lg.Info("dlq stream ready", "stream", cfg.NATS.DLQStreamName)

	// --- Email sender ---
	emailSender, err := email.NewSMTPSender(cfg.SMTP)
	if err != nil {
		return fmt.Errorf("init email sender: %w", err)
	}

	// --- Metrics ---
	metricsHandle, err := observability.SetupMetrics(ctx, ServiceName, Version)
	if err != nil {
		return fmt.Errorf("setup metrics: %w", err)
	}
	defer metricsHandle.Shutdown(context.Background()) //nolint:errcheck

	// --- Tracing ---
	// Installs the global tracer provider and trace-context propagator, so the
	// consumer spans are exported to Jaeger and the trace continues from the
	// context propagated through the NATS headers.
	tracingHandle, err := observability.SetupTracing(ctx, ServiceName, Version, cfg.Observability.OTLPEndpoint)
	if err != nil {
		return fmt.Errorf("setup tracing: %w", err)
	}
	defer tracingHandle.Shutdown(context.Background()) //nolint:errcheck

	notifierMetrics, err := metrics.NewNotifier(otel.Meter(ServiceName))
	if err != nil {
		return fmt.Errorf("init business metrics: %w", err)
	}

	// --- Event subscriber ---
	// Consumes orders.created and dispatches notifications. Runs in its own
	// goroutine; drains in-flight messages when ctx is canceled.
	dedupStore := dedup.NewStore(redisClient, cfg.Dedup.ClaimTTL, cfg.Dedup.CompletionTTL)
	notifierSvc := service.NewNotifier(dedupStore, emailSender, notifierMetrics, lg)
	handler := handlers.NewOrderCreatedHandler(notifierSvc, lg)
	deadLetterer := jetstream.NewDeadLetterer(jetstream.NewPublisher(js), cfg.NATS.DLQSubjectPrefix)
	sub := jetstream.NewSubscriber(js, jetstream.SubscriberConfig{
		StreamName:   cfg.NATS.StreamName,
		ConsumerName: cfg.NATS.ConsumerName,
		Subject:      event.SubjectOrderCreated,
		AckWait:      cfg.NATS.AckWait,
		MaxDeliver:   cfg.NATS.MaxDeliver,
		RetryBackoff: cfg.NATS.RetryBackoff,
	}, handler, deadLetterer, lg)
	go func() {
		if runErr := sub.Run(ctx); runErr != nil {
			lg.Error("subscriber error", "err", runErr)
		}
	}()

	// --- HTTP server ---
	// Serves health/readiness/info/metrics. Blocks until ctx is canceled, then
	// gracefully shuts down, which keeps the process alive for the goroutines above.
	rc := httppkg.RouterConfig{
		Logger:   lg,
		Redis:    redisClient,
		Nats:     nc,
		Info:     CurrentInfo(),
		Registry: metricsHandle.Registry,
	}
	router := httppkg.NewRouter(rc)
	httpHandler := otelhttp.NewHandler(router, "notifier.http",
		otelhttp.WithFilter(func(r *http.Request) bool {
			switch r.URL.Path {
			case "/metrics", "/healthz", "/readyz":
				return false
			default:
				return true
			}
		}),
	)
	server := httpserver.New(cfg.HTTP, httpHandler, lg)
	if err := server.Run(ctx); err != nil {
		return fmt.Errorf("http server: %w", err)
	}

	lg.Info("stopped cleanly")
	return nil
}
