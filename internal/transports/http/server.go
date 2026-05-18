// Package http provides the HTTP transport layer for the application.
package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/zzokki81/eventmesh/internal/pkg/appconfig"
)

// Server wraps the standard http.Server with lifecycle management
// driven by a parent context.
type Server struct {
	srv    *http.Server
	logger *slog.Logger
	cfg    appconfig.HTTPConfig
}

// NewServer constructs a new HTTP server with the given handler and configuration.
func NewServer(cfg appconfig.HTTPConfig, handler http.Handler, logger *slog.Logger) *Server {
	return &Server{
		srv: &http.Server{
			Addr:              cfg.Addr,
			Handler:           handler,
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
		},
		logger: logger,
		cfg:    cfg,
	}
}

// Run starts the HTTP server and blocks until ctx is canceled or the server
// fails with a non-graceful error. On context cancellation it performs a
// graceful shutdown bounded by the configured ShutdownTimeout, allowing
// in-flight requests to complete before returning.
func (s *Server) Run(ctx context.Context) error {
	serverErrCh := make(chan error, 1)

	go func() {
		s.logger.Info("http server listening", "addr", s.cfg.Addr)
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- fmt.Errorf("listen and serve: %w", err)
			return
		}
		serverErrCh <- nil
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("http server received shutdown signal")
	case err := <-serverErrCh:
		if err != nil {
			return err
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()

	if err := s.srv.Shutdown(shutdownCtx); err != nil { // //nolint:contextcheck
		return fmt.Errorf("http server shutdown: %w", err)
	}

	s.logger.Info("http server stopped cleanly")
	return nil
}
