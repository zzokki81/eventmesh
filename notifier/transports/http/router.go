package http

import (
	"log/slog"
	"net/http"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/zzokki81/eventmesh/pkg/httpserver/handler"
	"github.com/zzokki81/eventmesh/pkg/httpserver/middleware"

	"github.com/zzokki81/eventmesh/notifier/transports/http/handlers"
)

// RouterConfig groups the dependencies required to build the HTTP router.
type RouterConfig struct {
	// Nats is used by the readiness probe to verify broker connectivity.
	Nats *nats.Conn

	// Redis is used by the readiness probe to verify cache connectivity.
	Redis *redis.Client

	// Logger is shared with middleware and handlers.
	Logger *slog.Logger

	// Info exposes build and runtime metadata on /info.
	Info handler.InfoData

	// Registry backs the /metrics endpoint.
	Registry *prometheus.Registry
}

// NewRouter registers the application routes and wraps them in the middleware
// chain. See each middleware's own documentation for its behavior.
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.HandlerFor(cfg.Registry, promhttp.HandlerOpts{}))

	readiness := handlers.NewReadiness(cfg.Nats, cfg.Redis, cfg.Logger)
	infoHandler := handler.NewInfo(cfg.Info, cfg.Logger)

	mux.HandleFunc("GET /healthz", handler.Health)
	mux.Handle("GET /readyz", readiness)
	mux.Handle("GET /info", infoHandler)

	var h http.Handler = mux
	// The order of middleware is important: each line wraps the previous handler,
	// so the last applied is the outermost and runs first. Read bottom-up for the
	// execution order an incoming request follows.
	h = middleware.Recovery(cfg.Logger)(h)
	h = middleware.AccessLog(cfg.Logger, "/healthz", "/readyz", "/metrics")(h)
	h = middleware.RequestID(h)

	return h
}
