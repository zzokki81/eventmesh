// Package handlers contains HTTP handler functions.
package handlers

import "net/http"

// Health responds with 200 OK to indicate the process is alive.
// Used as the Kubernetes liveness probe target.
func Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// Ready responds with 200 OK to indicate the application is ready to serve traffic.
// In later weeks this will check downstream dependencies (DB, NATS, Redis).
// Used as the Kubernetes readiness probe target.
func Ready(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}
