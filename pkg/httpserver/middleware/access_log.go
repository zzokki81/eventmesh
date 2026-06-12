package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// responseRecorder wraps http.ResponseWriter to capture the status code and
// response size, which the standard interface does not expose after the fact.
// The status defaults to 200 to mirror Go's implicit WriteHeader on first Write.
type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// AccessLog returns a middleware that emits one structured log line per
// completed request, carrying method, path, status, size, latency, and the
// correlation ID. The log level reflects the response class: 5xx at error,
// 4xx at warn, everything else at info.
//
// Paths in skip are served without logging — intended for health and
// readiness probes whose high frequency would otherwise drown the log.
//
// AccessLog must run after RequestID (so the correlation ID is in the
// context) and outside Recovery (so a recovered panic is observed as its
// final 5xx status rather than a missing line).
func AccessLog(logger *slog.Logger, skip ...string) func(http.Handler) http.Handler {
	skipSet := make(map[string]struct{}, len(skip))
	for _, p := range skip {
		skipSet[p] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, quiet := skipSet[r.URL.Path]; quiet {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			logger.LogAttrs(r.Context(), levelForStatus(rec.status), "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Int("bytes", rec.bytes),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", RequestIDFromContext(r.Context())),
				slog.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}

// levelForStatus maps an HTTP status code to a log level so that server
// errors surface above the normal request stream.
func levelForStatus(status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status >= 400:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}
