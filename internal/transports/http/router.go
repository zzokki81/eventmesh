package http

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"

	"github.com/zzokki81/eventmesh/internal/transports/http/handlers"
	"github.com/zzokki81/eventmesh/internal/transports/http/middlewares"

	ordersvc "github.com/zzokki81/eventmesh/internal/services/order"
)

// RouterConfig groups the dependencies required to build the HTTP router.
type RouterConfig struct {
	// OrderService backs the /orders endpoints.
	OrderService ordersvc.Service

	// Db is used by the readiness probe to verify database connectivity.
	Db *pgxpool.Pool

	// Nats is used by the readiness probe to verify broker connectivity.
	Nats *nats.Conn

	// Logger is shared with middleware and handlers.
	Logger *slog.Logger

	// Info exposes build and runtime metadata on /info.
	Info handlers.InfoData
}

// NewRouter constructs the HTTP router with all application routes and middleware.
// Middleware order matters: Recovery wraps everything (so it catches panics from
// other middleware too), then RequestID assigns a correlation ID early so all
// downstream logs and handlers can reference it.
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	readiness := handlers.NewReadiness(cfg.Db, cfg.Nats, cfg.Logger)
	infoHandler := handlers.NewInfo(cfg.Info, cfg.Logger)
	orderHandler := handlers.NewOrderHandler(cfg.OrderService)

	mux.HandleFunc("GET /healthz", handlers.Health)
	mux.Handle("GET /readyz", readiness)
	mux.Handle("GET /info", infoHandler)
	mux.HandleFunc("POST /orders", orderHandler.Create)

	var handler http.Handler = mux
	handler = middlewares.RequestID(handler)
	handler = middlewares.Recovery(cfg.Logger)(handler)

	return handler
}
