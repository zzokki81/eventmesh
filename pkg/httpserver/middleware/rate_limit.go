package middleware

import (
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-redis/redis_rate/v10"
)

// RateLimitConfig groups a rate limiter with the settings that govern how it
// is applied. A zero value (nil Limiter) disables rate limiting.
type RateLimitConfig struct {
	// Limiter enforces Limit, shared via Redis across every instance of the
	// service. Nil disables rate limiting.
	Limiter *redis_rate.Limiter

	// Limit is the per-client limit enforced by Limiter. Unused if Limiter
	// is nil.
	Limit redis_rate.Limit

	// TrustProxyHeaders controls how the client IP is determined: when false
	// (the default), it is read from the TCP connection (RemoteAddr), which
	// is correct when the service receives traffic directly. When the
	// service instead sits behind a reverse proxy or load balancer,
	// RemoteAddr is the proxy's address for every request, which would put
	// all clients in one bucket — set this to true only once such a proxy is
	// in place, so X-Forwarded-For / X-Real-IP are trusted. Never enable it
	// without a proxy that overwrites those headers on every inbound
	// request; otherwise a client can set its own X-Forwarded-For to spoof
	// its rate-limit identity or impersonate another client's.
	TrustProxyHeaders bool
}

// RateLimit returns a middleware that limits requests per client IP using a
// Redis-backed GCRA limiter (token bucket with smooth refill). The limit is
// shared in Redis, so it holds across every instance of the service rather
// than per-process, which matters once the service is scaled horizontally.
//
// If Redis is unreachable, the middleware fails open: the request is allowed
// through and the error is logged. An outage of the rate limiter's own
// dependency should not take down the API it is meant to protect.
func RateLimit(cfg RateLimitConfig, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			res, err := cfg.Limiter.Allow(r.Context(), "ip:"+clientIP(r, cfg.TrustProxyHeaders), cfg.Limit)
			if err != nil {
				logger.Error("rate limiter unavailable, allowing request", "err", err)
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.Limit.Rate))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))

			if res.Allowed < 1 {
				// Round up: Retry-After is an integer number of seconds, and
				// truncating a sub-second wait (e.g. 480ms) down to 0 would
				// tell the client to retry immediately, when it would just be
				// throttled again.
				retryAfter := int(math.Ceil(res.RetryAfter.Seconds()))
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// clientIP returns the request's client IP, stripping the ephemeral port from
// RemoteAddr so repeated connections from the same client share one bucket.
//
// If trustProxyHeaders is true, it prefers X-Forwarded-For (the left-most
// address, which is the original client in a standard proxy chain) and falls
// back to X-Real-IP, since RemoteAddr would otherwise be the proxy's address
// for every request.
func clientIP(r *http.Request, trustProxyHeaders bool) string {
	if trustProxyHeaders {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if ip := strings.TrimSpace(strings.SplitN(xff, ",", 2)[0]); ip != "" {
				return ip
			}
		}
		if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
			return xrip
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
