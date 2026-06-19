package app

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/zzokki81/eventmesh/notifier/config"
	"github.com/zzokki81/eventmesh/notifier/dedup"
	"github.com/zzokki81/eventmesh/notifier/email"
	"github.com/zzokki81/eventmesh/notifier/service"
	"github.com/zzokki81/eventmesh/notifier/transports/broker/handlers"
	"github.com/zzokki81/eventmesh/notifier/transports/http"
	"github.com/zzokki81/eventmesh/pkg/broker/jetstream"
	"github.com/zzokki81/eventmesh/pkg/event"
	"github.com/zzokki81/eventmesh/pkg/httpserver"
	"github.com/zzokki81/eventmesh/pkg/logger"
	"github.com/zzokki81/eventmesh/pkg/nats"
	"github.com/zzokki81/eventmesh/pkg/redis"
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

	// --- Email sender ---
	emailSender, err := email.NewSMTPSender(cfg.SMTP)
	if err != nil {
		return fmt.Errorf("init email sender: %w", err)
	}

	// --- Event subscriber ---
	// Consumes orders.created and dispatches notifications. Runs in its own
	// goroutine; drains in-flight messages when ctx is canceled.
	dedupStore := dedup.NewStore(redisClient, cfg.Dedup.ClaimTTL, cfg.Dedup.CompletionTTL)
	notifierSvc := service.NewNotifier(dedupStore, emailSender, lg)
	handler := handlers.NewOrderCreatedHandler(notifierSvc, lg)
	sub := jetstream.NewSubscriber(js, jetstream.SubscriberConfig{
		StreamName:   cfg.NATS.StreamName,
		ConsumerName: cfg.NATS.ConsumerName,
		Subject:      event.SubjectOrderCreated,
		AckWait:      cfg.NATS.AckWait,
		MaxDeliver:   cfg.NATS.MaxDeliver,
	}, handler, lg)
	go func() {
		if err := sub.Run(ctx); err != nil {
			lg.Error("subscriber error", "err", err)
		}
	}()

	// --- HTTP server ---
	// Serves health/readiness/info. Blocks until ctx is canceled, then
	// gracefully shuts down, which keeps the process alive for the goroutines above.
	rc := http.RouterConfig{
		Logger: lg,
		Redis:  redisClient,
		Nats:   nc,
		Info:   CurrentInfo(),
	}
	router := http.NewRouter(rc)
	server := httpserver.New(cfg.HTTP, router, lg)
	if err := server.Run(ctx); err != nil {
		return fmt.Errorf("http server: %w", err)
	}

	lg.Info("stopped cleanly")
	return nil
}
