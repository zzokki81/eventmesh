package app

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/zzokki81/eventmesh/internal/infrastructure/nats"
	"github.com/zzokki81/eventmesh/internal/infrastructure/postgres"
	"github.com/zzokki81/eventmesh/internal/pkg/config"
	"github.com/zzokki81/eventmesh/internal/pkg/event"
	"github.com/zzokki81/eventmesh/internal/pkg/logger"
	"github.com/zzokki81/eventmesh/internal/transports/broker"
	"github.com/zzokki81/eventmesh/internal/transports/http"

	orderSvc "github.com/zzokki81/eventmesh/internal/services/order/order"
	orderStorage "github.com/zzokki81/eventmesh/internal/storage/orders/postgres"
)

// Run bootstraps the application: loads config, initializes infrastructure,
// starts the HTTP server, and blocks until shutdown signal is received.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

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

	lg.Info("starting eventmesh", "http_addr", cfg.HTTP.Addr)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("init postgres: %w", err)
	}
	defer pool.Close()
	lg.Info("postgres connected", "max_conns", cfg.Postgres.MaxConns, "min_conns", cfg.Postgres.MinConns)

	// NATS connection
	nc, err := nats.NewConnection(cfg.NATS)
	if err != nil {
		return fmt.Errorf("init nats: %w", err)
	}
	defer nc.Drain() //nolint:errcheck
	lg.Info("connected to nats", "url", cfg.NATS.URL)

	// JetStream context
	js, err := nats.NewJetStream(ctx, nc)
	if err != nil {
		return fmt.Errorf("init jetstream: %w", err)
	}
	lg.Info("jetstream context initialized")

	// Initialize stream (idempotent)
	if err := nats.SetupStream(ctx, js, cfg.NATS.StreamName); err != nil {
		return fmt.Errorf("setup stream: %w", err)
	}

	publisher := broker.NewPublisher(js)
	eventBuilder := event.NewBuilder(ServiceName)
	orderRepo := orderStorage.NewStorage(pool)
	orderService := orderSvc.NewService(orderRepo, publisher, eventBuilder, lg)

	rc := http.RouterConfig{
		Logger:       lg,
		Db:           pool,
		Nats:         nc,
		Info:         toHandlerInfo(CurrentInfo()),
		OrderService: orderService,
	}
	router := http.NewRouter(rc)
	server := http.NewServer(cfg.HTTP, router, lg)

	if err := server.Run(ctx); err != nil {
		return fmt.Errorf("http server: %w", err)
	}

	lg.Info("stopped cleanly")
	return nil
}
