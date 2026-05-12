package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/zzokki81/eventmesh/internal/pkg/appconfig"
	"github.com/zzokki81/eventmesh/internal/pkg/logging"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
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
		return fmt.Errorf("failed to init logger", "%w", err)
	}

	slog.SetDefault(logger)

}
