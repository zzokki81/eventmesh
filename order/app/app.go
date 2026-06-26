package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os/signal"
	"syscall"
	"time"

	"github.com/zzokki81/eventmesh/order/config"
	"github.com/zzokki81/eventmesh/order/metrics"
	"github.com/zzokki81/eventmesh/order/relay"

	"github.com/zzokki81/eventmesh/pkg/broker/jetstream"
	"github.com/zzokki81/eventmesh/pkg/event"
	"github.com/zzokki81/eventmesh/pkg/httpserver"
	"github.com/zzokki81/eventmesh/pkg/httpserver/middleware"
	"github.com/zzokki81/eventmesh/pkg/logger"
	"github.com/zzokki81/eventmesh/pkg/nats"
	"github.com/zzokki81/eventmesh/pkg/observability"
	"github.com/zzokki81/eventmesh/pkg/redis"

	"github.com/go-redis/redis_rate/v10"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"

	orderpg "github.com/zzokki81/eventmesh/order/repository/postgres"
	orderSvc "github.com/zzokki81/eventmesh/order/service"
	httppkg "github.com/zzokki81/eventmesh/order/transports/http"
	pgpool "github.com/zzokki81/eventmesh/pkg/postgres"
)

// devPprofAddr is the address the pprof debug server listens on in the
// development profile. It binds to loopback only (not 0.0.0.0) so the profiler
// is never reachable from other hosts on the network, and is on a separate port
// from the public HTTP server so profiles are never exposed alongside the app.
const devPprofAddr = "127.0.0.1:6060"

// Run bootstraps the application: loads config, initializes infrastructure,
// starts the HTTP server, and blocks until shutdown signal is received.
func Run() error {
	// --- Configuration ---
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// --- Logger ---
	lg, err := logger.New(cfg.Logger)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	lg = lg.With("service", ServiceName, "version", Version, "commit", CommitHash)
	slog.SetDefault(lg)
	lg.Info("starting eventmesh", "http_addr", cfg.HTTP.Addr)

	// --- Signal-aware root context ---
	// Canceled on SIGINT/SIGTERM; propagated to every long-running component
	// so they shut down together.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- pprof debug server ---
	// Runtime profiling on a separate, non-public port so the profiles are not
	// exposed alongside the application. Only runs in the development profile.
	if cfg.AppEnv.IsDevelopment() {
		pprofSrv := &http.Server{Addr: devPprofAddr, Handler: http.DefaultServeMux}
		go func() {
			lg.Info("pprof server listening", "addr", devPprofAddr)
			if listenErr := pprofSrv.ListenAndServe(); listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
				lg.Error("pprof server error", "err", listenErr)
			}
		}()
		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if shutdownErr := pprofSrv.Shutdown(shutdownCtx); shutdownErr != nil { //nolint:contextcheck
				lg.Error("pprof server shutdown", "err", shutdownErr)
			}
		}()
	}

	// --- Postgres ---
	pool, err := pgpool.NewPool(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("init postgres: %w", err)
	}
	defer pool.Close()
	lg.Info("postgres connected", "max_conns", cfg.Postgres.MaxConns, "min_conns", cfg.Postgres.MinConns)

	// --- NATS connection + JetStream ---
	nc, err := nats.NewConnection(cfg.NATS)
	if err != nil {
		return fmt.Errorf("init nats: %w", err)
	}
	defer nc.Drain() //nolint:errcheck
	lg.Info("connected to nats", "url", cfg.NATS.URL)

	js, err := nats.NewJetStream(ctx, nc)
	if err != nil {
		return fmt.Errorf("init jetstream: %w", err)
	}
	lg.Info("jetstream context initialized")

	// Initialize stream (idempotent)
	if err = nats.SetupStream(ctx, js, cfg.NATS.StreamName); err != nil {
		return fmt.Errorf("setup stream: %w", err)
	}

	// --- Publisher + repositories ---
	pub := jetstream.NewPublisher(js)
	eventBuilder := event.NewBuilder(ServiceName)

	orderRepo := orderpg.NewOrders(pool)
	outboxRepo := orderpg.NewOutbox()

	// --- Metrics ---
	metricsHandle, err := observability.SetupMetrics(ctx, ServiceName, Version)
	if err != nil {
		return fmt.Errorf("setup metrics: %w", err)
	}
	defer metricsHandle.Shutdown(context.Background()) //nolint:errcheck

	// --- Tracing ---
	// Installs the global tracer provider and trace-context propagator, so the
	// spans created across the order flow are exported to Jaeger and the context
	// propagates through HTTP and NATS headers.
	tracingHandle, err := observability.SetupTracing(ctx, ServiceName, Version, cfg.Observability.OTLPEndpoint)
	if err != nil {
		return fmt.Errorf("setup tracing: %w", err)
	}
	defer tracingHandle.Shutdown(context.Background()) //nolint:errcheck

	meter := otel.Meter(ServiceName)

	orderMetrics, err := metrics.NewOrders(meter)
	if err != nil {
		return fmt.Errorf("init order metrics: %w", err)
	}

	relayMetrics, err := metrics.NewRelay(meter)
	if err != nil {
		return fmt.Errorf("init relay metrics: %w", err)
	}

	// --- Order service ---
	orderService := orderSvc.NewOrders(orderRepo, outboxRepo, pool, eventBuilder, orderMetrics, lg)

	// --- Outbox relay ---
	// Drains pending outbox rows and publishes them to the broker. Runs in its
	// own goroutine; stops when ctx is canceled.
	relayProc := relay.New(pool, outboxRepo, pub, cfg.Relay, relayMetrics, lg)
	go func() {
		if runErr := relayProc.Run(ctx); runErr != nil {
			lg.Error("relay error", "err", runErr)
		}
	}()

	// --- Rate limiter ---
	// Backed by Redis so the limit is shared across every instance of the
	// service rather than tracked per-process. Disabled by default; only
	// connects to Redis when RATE_LIMIT_ENABLED=true.
	//
	// Rate limiting is an auxiliary protection, not a core dependency like
	// Postgres or NATS: if Redis cannot be reached at startup, the order
	// service still starts (unlimited) rather than refusing to come up over
	// an outage in a feature that only guards against abuse.
	var rateLimiter *redis_rate.Limiter
	rateLimit := redis_rate.Limit{Rate: cfg.RateLimit.RPS, Burst: cfg.RateLimit.Burst, Period: time.Second}
	if cfg.RateLimit.Enabled {
		redisClient, redisErr := redis.NewClient(ctx, cfg.Redis)
		if redisErr != nil {
			lg.Error("rate limiter disabled: redis unavailable", "err", redisErr)
		} else {
			defer redisClient.Close() //nolint:errcheck
			lg.Info("redis connected", "address", cfg.Redis.Addr)

			rateLimiter = redis_rate.NewLimiter(redisClient)
			lg.Info("rate limiter enabled", "rps", cfg.RateLimit.RPS, "burst", cfg.RateLimit.Burst)
		}
	}

	// --- HTTP server ---
	// Serves health/readiness/info/metrics. Blocks until ctx is canceled, then
	// gracefully shuts down, which keeps the process alive for the relay goroutine above.
	rc := httppkg.RouterConfig{
		Logger:       lg,
		Db:           pool,
		Nats:         nc,
		Info:         CurrentInfo(),
		OrderService: orderService,
		Registry:     metricsHandle.Registry,
		RateLimit: middleware.RateLimitConfig{
			Limiter:           rateLimiter,
			Limit:             rateLimit,
			TrustProxyHeaders: cfg.RateLimit.TrustProxyHeaders,
		},
	}

	router := httppkg.NewRouter(rc)
	// otelhttp instruments every request except infra endpoints (scrape/probes),
	// producing HTTP server metrics now and traces once a tracer is configured.
	handler := otelhttp.NewHandler(router, "order.http",
		otelhttp.WithFilter(func(r *http.Request) bool {
			switch r.URL.Path {
			case "/metrics", "/healthz", "/readyz":
				return false
			default:
				return true
			}
		}),
	)
	server := httpserver.New(cfg.HTTP, handler, lg)
	if err := server.Run(ctx); err != nil {
		return fmt.Errorf("http server: %w", err)
	}

	lg.Info("stopped cleanly")
	return nil
}
