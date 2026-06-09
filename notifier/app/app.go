package app

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"
)

// Run bootstraps the notifier service and blocks until a shutdown signal is received.
func Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("starting notifier")

	<-ctx.Done()

	slog.Info("notifier stopped cleanly")
	return nil
}
