package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// requestIDKey is the context key used to store the request ID.
// It is unexported and uses a custom type to avoid collisions with other packages.
type requestIDKey struct{}

// HeaderRequestID is the HTTP header name used to read and write the request ID.
const HeaderRequestID = "X-Request-ID"

// RequestID returns a middleware that ensures every request has a unique ID.
// If the incoming request already carries an X-Request-ID header, that value is reused;
// otherwise a new UUID is generated. The ID is stored in the request context and
// echoed back to the client in the response header so it can be used for correlation
// across logs, traces, and downstream services.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get(HeaderRequestID)
		if reqID == "" {
			reqID = uuid.NewString()
		}

		ctx := context.WithValue(r.Context(), requestIDKey{}, reqID)
		w.Header().Set(HeaderRequestID, reqID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext retrieves the request ID stored in ctx.
// Returns an empty string if no ID is present.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}
