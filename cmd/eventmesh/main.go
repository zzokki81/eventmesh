package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/zzokki81/eventmesh/internal/pkg/appconfig"
	"github.com/zzokki81/eventmesh/internal/pkg/logging"
	"github.com/zzokki81/eventmesh/internal/transports/http"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	// Load config from env
	cfg, err := appconfig.Load()
	if err != nil {
		return err
	}

	// Setup logger
	logger, err := logging.New(cfg.Logger)
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}
	slog.SetDefault(logger)

	logger.Info("starting eventmesh", "http_addr", cfg.HTTP.Addr)

	// Root context cancels on SIGINT/SIGTERM — drives graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Setup HTTP server and routes
	router := http.NewRouter(logger)
	server := http.NewServer(cfg.HTTP, router, logger)

	// Run server with graceful shutdown
	if err := server.Run(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	logger.Info("eventmesh stopped gracefully")
	return nil
}
