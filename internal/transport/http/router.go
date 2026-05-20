package http

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zzokki81/eventmesh/internal/transport/http/handlers"
	"github.com/zzokki81/eventmesh/internal/transport/http/middlewares"
)

// NewRouter constructs the HTTP router with all application routes and middleware.
// Middleware order matters: Recovery wraps everything (so it catches panics from
// other middleware too), then RequestID assigns a correlation ID early so all
// downstream logs and handlers can reference it.
func NewRouter(logger *slog.Logger, info handlers.InfoData, db *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()

	readiness := handlers.NewReadiness(db, logger)
	infoHandler := handlers.NewInfo(info, logger)

	mux.HandleFunc("GET /healthz", handlers.Health)
	mux.Handle("GET /readyz", readiness)
	mux.Handle("GET /info", infoHandler)

	var handler http.Handler = mux
	handler = middlewares.RequestID(handler)
	handler = middlewares.Recovery(logger)(handler)

	return handler
}
