package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// InfoData represents service identification and build metadata.
// Defined here (not imported from app) to avoid an import cycle.
type InfoData struct {
	Service   string `json:"service"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
}

// Info handler returns service identification and build metadata.
type Info struct {
	data   InfoData
	logger *slog.Logger
}

// NewInfo constructs an Info handler.
func NewInfo(data InfoData, logger *slog.Logger) *Info {
	return &Info{data: data, logger: logger}
}

func (h *Info) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(h.data); err != nil {
		h.logger.Error("info: failed to encode response", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}
