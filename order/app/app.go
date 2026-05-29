package app

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/zzokki81/eventmesh/order/entities/order"
	"github.com/zzokki81/eventmesh/pkg/nats"
	"github.com/zzokki81/eventmesh/pkg/postgres"
	"github.com/zzokki81/eventmesh/order/config"
	"github.com/zzokki81/eventmesh/pkg/event"
	"github.com/zzokki81/eventmesh/pkg/logger"
	"github.com/zzokki81/eventmesh/order/relay"
	"github.com/zzokki81/eventmesh/pkg/broker/handlers/eventlog"
	"github.com/zzokki81/eventmesh/order/transports/http"

	orderSvc "github.com/zzokki81/eventmesh/order/services/order/order"
	orderStorage "github.com/zzokki81/eventmesh/order/storage/orders/postgres"
	outboxStorage "github.com/zzokki81/eventmesh/order/storage/outbox/postgres"
	brokerNats "github.com/zzokki81/eventmesh/pkg/broker/jetstream"
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

	pub := brokerNats.NewPublisher(js)
	eventBuilder := event.NewBuilder(ServiceName)
	orderRepo := orderStorage.NewStorage(pool)
	outboxRepo := outboxStorage.NewStorage()
	orderService := orderSvc.NewService(orderRepo, outboxRepo, pool, eventBuilder, lg)

	// Outbox relay: drains pending outbox rows and publishes them to the broker.
	relayProc := relay.New(pool, outboxRepo, pub, cfg.Relay, lg)
	go func() {
		if err := relayProc.Run(ctx); err != nil {
			lg.Error("relay error", "err", err)
		}
	}()

	sub := brokerNats.NewSubscriber(js, brokerNats.SubscriberConfig{
		StreamName:   cfg.NATS.StreamName,
		ConsumerName: "orders-logger",
		Subject:      order.TopicCreated,
		AckWait:      cfg.NATS.AckWait,
		MaxDeliver:   cfg.NATS.MaxDeliver,
	}, eventlog.New(lg), lg)

	go func() {
		if err := sub.Run(ctx); err != nil {
			lg.Error("subscriber error", "err", err)
		}
	}()

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
