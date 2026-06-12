package http

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/zzokki81/eventmesh/pkg/httpserver/handler"
	"github.com/zzokki81/eventmesh/pkg/httpserver/middleware"

	"github.com/zzokki81/eventmesh/order/service"
	"github.com/zzokki81/eventmesh/order/transports/http/handlers"
)

// RouterConfig groups the dependencies required to build the HTTP router.
type RouterConfig struct {
	// OrderService backs the /orders endpoints.
	OrderService service.Orders

	// Db is used by the readiness probe to verify database connectivity.
	Db *pgxpool.Pool

	// Nats is used by the readiness probe to verify broker connectivity.
	Nats *nats.Conn

	// Logger is shared with middleware and handlers.
	Logger *slog.Logger

	// Info exposes build and runtime metadata on /info.
	Info handler.InfoData
}

// NewRouter registers the application routes and wraps them in the middleware
// chain. See each middleware's own documentation for its behavior.
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	readiness := handlers.NewReadiness(cfg.Db, cfg.Nats, cfg.Logger)
	infoHandler := handler.NewInfo(cfg.Info, cfg.Logger)
	orderHandler := handlers.NewOrderHandler(cfg.OrderService)

	mux.HandleFunc("GET /healthz", handler.Health)
	mux.Handle("GET /readyz", readiness)
	mux.Handle("GET /info", infoHandler)
	mux.HandleFunc("POST /orders", orderHandler.Create)

	var h http.Handler = mux
	// The order of middleware is important: each line wraps the previous handler,
	// so the last applied is the outermost and runs first. Read bottom-up for the
	// execution order an incoming request follows.
	h = middleware.Recovery(cfg.Logger)(h)
	h = middleware.AccessLog(cfg.Logger, "/healthz", "/readyz")(h)
	h = middleware.RequestID(h)

	return h
}
