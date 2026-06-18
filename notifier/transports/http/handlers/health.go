// Package handlers contains HTTP handler functions and structs.
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

const readinessTimeout = 2 * time.Second

// Readiness checks that all downstream dependencies are reachable.
// Used as the Kubernetes readiness probe target. A failure removes the pod
// from service routing without restarting it — appropriate for transient
// dependency outages.
type Readiness struct {
	nats   *nats.Conn
	redis  *redis.Client
	logger *slog.Logger
}

// NewReadiness constructs a Readiness handler with the given dependencies.
func NewReadiness(nats *nats.Conn, redis *redis.Client, logger *slog.Logger) *Readiness {
	return &Readiness{
		nats:   nats,
		redis:  redis,
		logger: logger,
	}
}

// ServeHTTP implements http.Handler. It runs each dependency check with a
// shared timeout budget and fails fast on the first unreachable dependency.
func (r *Readiness) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), readinessTimeout)
	defer cancel()

	if !r.nats.IsConnected() {
		r.logger.Warn("readiness: nats disconnected")
		http.Error(w, "nats unavailable", http.StatusServiceUnavailable)
		return
	}

	if err := r.redis.Ping(ctx).Err(); err != nil {
		r.logger.Warn("readiness: redis disconnected")
		http.Error(w, "redis unavailable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}
