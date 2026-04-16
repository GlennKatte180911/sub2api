package handler

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

// buildVersion is set at compile time via ldflags
var buildVersion = "dev"

// startTime records when the service started
var startTime = time.Now()

// HealthResponse represents the JSON response for the health check endpoint
type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	GoVersion string `json:"go_version"`
}

// HealthHandler handles health check requests
type HealthHandler struct{}

// NewHealthHandler creates a new HealthHandler instance
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// ServeHTTP handles GET /health requests and returns service status.
// Note: also accept HEAD requests so load balancers can probe without a body.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uptime := time.Since(startTime).Round(time.Second).String()

	resp := HealthResponse{
		Status:    "ok",
		Version:   buildVersion,
		Uptime:    uptime,
		GoVersion: runtime.Version(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// HEAD requests must not include a body
	if r.Method == http.MethodHead {
		return
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
