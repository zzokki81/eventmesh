package http

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"

	"github.com/zzokki81/eventmesh/internal/transports/http/handlers"
	"github.com/zzokki81/eventmesh/internal/transports/http/middlewares"
)

// RouterConfig holds dependencies for constructing the HTTP router.
type RouterConfig struct {
	Logger *slog.Logger
	Db     *pgxpool.Pool
	Nats   *nats.Conn
	Info   handlers.InfoData
}

// NewRouter constructs the HTTP router with all application routes and middleware.
// Middleware order matters: Recovery wraps everything (so it catches panics from
// other middleware too), then RequestID assigns a correlation ID early so all
// downstream logs and handlers can reference it.
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	readiness := handlers.NewReadiness(cfg.Db, cfg.Nats, cfg.Logger)
	infoHandler := handlers.NewInfo(cfg.Info, cfg.Logger)

	mux.HandleFunc("GET /healthz", handlers.Health)
	mux.Handle("GET /readyz", readiness)
	mux.Handle("GET /info", infoHandler)

	var handler http.Handler = mux
	handler = middlewares.RequestID(handler)
	handler = middlewares.Recovery(cfg.Logger)(handler)

	return handler
}
