// Package handlers contains HTTP handlers for the API.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

const (
	statusHealthy   = "healthy"
	statusUnhealthy = "unhealthy"
	databaseReady   = "connected"
	databaseDown    = "disconnected"
)

// DatabasePinger is the database capability required by the health endpoint.
// db.DB satisfies this interface while tests can provide a small fake.
type DatabasePinger interface {
	PingDB(context.Context) error
}

// HealthResponse is the stable JSON contract returned by the health endpoint.
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Database  string `json:"database"`
	Version   string `json:"version"`
	Error     string `json:"error,omitempty"`
}

// HealthHandler reports API and database readiness.
type HealthHandler struct {
	database DatabasePinger
	version  string
	now      func() time.Time
}

// NewHealthHandler creates a health endpoint using the supplied database
// dependency and application version.
func NewHealthHandler(database DatabasePinger, version string) *HealthHandler {
	return &HealthHandler{
		database: database,
		version:  version,
		now:      time.Now,
	}
}

// ServeHTTP writes the health response. Database failures are intentionally
// reported as service-unavailable responses without exposing driver errors.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	response := HealthResponse{
		Status:    statusHealthy,
		Timestamp: h.now().UTC().Format(time.RFC3339),
		Database:  databaseReady,
		Version:   h.version,
	}
	statusCode := http.StatusOK

	if h.database == nil || h.database.PingDB(request.Context()) != nil {
		response.Status = statusUnhealthy
		response.Database = databaseDown
		response.Error = "database unavailable"
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}
