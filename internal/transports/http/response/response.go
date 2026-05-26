// Package response contains helpers for writing HTTP responses
// in a consistent JSON format.
package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// WriteJSON writes body as JSON with the given status code and the
// application/json content type. Encoding failures cannot be recovered
// once the status line has been sent; they are logged so operators can
// correlate truncated responses with their cause.
func WriteJSON(w http.ResponseWriter, r *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response body",
			"err", err,
			"path", r.URL.Path,
			"method", r.Method,
			"status", status,
		)
	}
}
