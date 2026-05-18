package http

import (
	"log/slog"
	"net/http"

	"github.com/zzokki81/eventmesh/internal/transports/http/handlers"
	"github.com/zzokki81/eventmesh/internal/transports/http/middlewares"
)

// NewRouter constructs the HTTP router with all application routes and middleware.
// Middleware order matters: Recovery wraps everything (so it catches panics from
// other middleware too), then RequestID assigns a correlation ID early so all
// downstream logs and handlers can reference it.
func NewRouter(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handlers.Health)
	mux.HandleFunc("GET /readyz", handlers.Ready)

	var handler http.Handler = mux
	handler = middlewares.RequestID(handler)
	handler = middlewares.Recovery(logger)(handler)

	return handler
}
