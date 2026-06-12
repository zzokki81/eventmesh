// Package handlers contains HTTP handler functions and structs.
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

const readinessTimeout = 2 * time.Second

// Readiness checks that all downstream dependencies are reachable.
// Used as the Kubernetes readiness probe target. A failure removes the pod
// from service routing without restarting it — appropriate for transient
// dependency outages.
type Readiness struct {
	db     *pgxpool.Pool
	nats   *nats.Conn
	logger *slog.Logger
}

// NewReadiness constructs a Readiness handler with the given dependencies.
func NewReadiness(db *pgxpool.Pool, nats *nats.Conn, logger *slog.Logger) *Readiness {
	return &Readiness{
		db:     db,
		nats:   nats,
		logger: logger,
	}
}

// ServeHTTP implements http.Handler. It runs each dependency check with a
// shared timeout budget and fails fast on the first unreachable dependency.
func (r *Readiness) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), readinessTimeout)
	defer cancel()

	if err := r.db.Ping(ctx); err != nil {
		r.logger.Warn("readiness: postgres unreachable", "err", err)
		http.Error(w, "postgres unavailable", http.StatusServiceUnavailable)
		return
	}

	if !r.nats.IsConnected() {
		r.logger.Warn("readiness: nats disconnected")
		http.Error(w, "nats unavailable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}
