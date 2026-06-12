// Package handler contains shared HTTP handlers (health, info) reused across services.
package handler

import "net/http"

// Health responds with 200 OK to indicate the process is alive.
// Used as the Kubernetes liveness probe target. Intentionally trivial:
// liveness should fail only when the process itself is unresponsive,
// not when a downstream dependency is unavailable.
func Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
