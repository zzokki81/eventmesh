package app

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/zzokki81/eventmesh/internal/config"
	"github.com/zzokki81/eventmesh/internal/infrastructure/logger"
	"github.com/zzokki81/eventmesh/internal/infrastructure/postgres"
	"github.com/zzokki81/eventmesh/internal/transport/http"
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

	router := http.NewRouter(lg, pool)
	server := http.NewServer(cfg.HTTP, router, lg)

	if err := server.Run(ctx); err != nil {
		return fmt.Errorf("http server: %w", err)
	}

	lg.Info("stopped cleanly")
	return nil
}
