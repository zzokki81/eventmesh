package middleware_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"

	"github.com/zzokki81/eventmesh/pkg/httpserver/middleware"
)

// clientA and clientB are distinct remote addresses used to verify that the
// rate limiter tracks separate clients independently.
const (
	clientA = "10.0.0.1:1234"
	clientB = "10.0.0.2:5678"
)

// newReq builds a test request from remoteAddr, the client address the
// rate limiter keys on when proxy headers are not trusted. xff and xrip, if
// non-empty, set X-Forwarded-For / X-Real-IP, used to test the trusted-proxy
// path.
func newReq(remoteAddr, xff, xrip string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/orders", nil)
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	if xrip != "" {
		req.Header.Set("X-Real-IP", xrip)
	}
	return req
}

// newTestLimiter returns a limiter backed by a running miniredis instance.
func newTestLimiter(t *testing.T) *redis_rate.Limiter {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return redis_rate.NewLimiter(client)
}

// newBrokenLimiter returns a limiter whose Redis is already closed, so every
// call fails - used to exercise the fail-open path.
func newBrokenLimiter(t *testing.T) *redis_rate.Limiter {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	mr.Close()

	return redis_rate.NewLimiter(client)
}

func newHandler(limiter *redis_rate.Limiter, limit redis_rate.Limit, trustProxyHeaders bool) http.Handler {
	logger := slog.New(slog.DiscardHandler)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	cfg := middleware.RateLimitConfig{Limiter: limiter, Limit: limit, TrustProxyHeaders: trustProxyHeaders}
	return middleware.RateLimit(cfg, logger)(next)
}

func TestRateLimit(t *testing.T) {
	// step is one request made against the handler under test, in sequence.
	type step struct {
		addr           string
		xff            string
		xrip           string
		wantCode       int
		wantRetryAfter bool
	}

	tests := []struct {
		name              string
		limit             redis_rate.Limit
		trustProxyHeaders bool
		brokenRedis       bool
		steps             []step
	}{
		{
			name:  "allows requests within burst",
			limit: redis_rate.Limit{Rate: 10, Burst: 2, Period: time.Second},
			steps: []step{
				{addr: clientA, wantCode: http.StatusOK},
				{addr: clientA, wantCode: http.StatusOK},
			},
		},
		{
			name:  "blocks once burst is exceeded",
			limit: redis_rate.Limit{Rate: 10, Burst: 1, Period: time.Second},
			steps: []step{
				{addr: clientA, wantCode: http.StatusOK},
				{addr: clientA, wantCode: http.StatusTooManyRequests, wantRetryAfter: true},
			},
		},
		{
			name:  "tracks clients independently",
			limit: redis_rate.Limit{Rate: 10, Burst: 1, Period: time.Second},
			steps: []step{
				{addr: clientA, wantCode: http.StatusOK},
				{addr: clientA, wantCode: http.StatusTooManyRequests, wantRetryAfter: true},
				{addr: clientB, wantCode: http.StatusOK},
			},
		},
		{
			name:        "fails open when redis is unavailable",
			limit:       redis_rate.Limit{Rate: 10, Burst: 1, Period: time.Second},
			brokenRedis: true,
			steps: []step{
				{addr: clientA, wantCode: http.StatusOK},
			},
		},
		{
			name:  "ignores X-Forwarded-For when proxy headers are not trusted",
			limit: redis_rate.Limit{Rate: 10, Burst: 1, Period: time.Second},
			steps: []step{
				// Same RemoteAddr (the proxy), different claimed clients via
				// X-Forwarded-For: without trust, both share the proxy's
				// RemoteAddr bucket, so the second is blocked.
				{addr: clientA, xff: "203.0.113.1", wantCode: http.StatusOK},
				{addr: clientA, xff: "203.0.113.2", wantCode: http.StatusTooManyRequests, wantRetryAfter: true},
			},
		},
		{
			name:              "uses X-Forwarded-For when proxy headers are trusted",
			limit:             redis_rate.Limit{Rate: 10, Burst: 1, Period: time.Second},
			trustProxyHeaders: true,
			steps: []step{
				// Same RemoteAddr (the proxy), different real clients via
				// X-Forwarded-For: with trust, each gets its own bucket.
				{addr: clientA, xff: "203.0.113.1", wantCode: http.StatusOK},
				{addr: clientA, xff: "203.0.113.1", wantCode: http.StatusTooManyRequests, wantRetryAfter: true},
				{addr: clientA, xff: "203.0.113.2", wantCode: http.StatusOK},
			},
		},
		{
			name:              "falls back to X-Real-IP when trusted and X-Forwarded-For is absent",
			limit:             redis_rate.Limit{Rate: 10, Burst: 1, Period: time.Second},
			trustProxyHeaders: true,
			steps: []step{
				{addr: clientA, xrip: "203.0.113.9", wantCode: http.StatusOK},
				{addr: clientA, xrip: "203.0.113.9", wantCode: http.StatusTooManyRequests, wantRetryAfter: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := newTestLimiter(t)
			if tt.brokenRedis {
				limiter = newBrokenLimiter(t)
			}
			h := newHandler(limiter, tt.limit, tt.trustProxyHeaders)

			for i, s := range tt.steps {
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, newReq(s.addr, s.xff, s.xrip))

				if rec.Code != s.wantCode {
					t.Fatalf("step %d: want status %d, got %d", i, s.wantCode, rec.Code)
				}
				if s.wantRetryAfter && rec.Header().Get("Retry-After") == "" {
					t.Errorf("step %d: want Retry-After header", i)
				}
			}
		})
	}
}
