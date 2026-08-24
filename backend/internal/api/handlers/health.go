// Package handlers contains HTTP handlers for the API.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/danielgtaylor/huma/v2"
)

const (
	statusHealthy = "healthy"
	databaseReady = "connected"
)

// DatabasePinger is the database capability required by the health endpoint.
// data.DB satisfies this interface while tests can provide a small fake.
type DatabasePinger interface {
	PingDB(context.Context) error
}

// HealthResponse is the stable JSON contract returned by the health endpoint.
type HealthResponse struct {
	Status    string `json:"status" doc:"Current API health status."`
	Timestamp string `json:"timestamp" doc:"Time at which the health check was generated."`
	Database  string `json:"database" doc:"Database connectivity status."`
	Version   string `json:"version" doc:"Running application version."`
}

// HealthInput is intentionally empty: the health endpoint takes no input.
type HealthInput struct{}

// HealthOutput is the Huma response envelope for the health endpoint.
type HealthOutput struct {
	Body HealthResponse
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

// Handle returns the health response or a registered problem detail when the
// database is unavailable. The driver error is intentionally not exposed.
func (h *HealthHandler) Handle(ctx context.Context, _ *HealthInput) (*HealthOutput, error) {
	response := HealthResponse{
		Status:    statusHealthy,
		Timestamp: h.now().UTC().Format(time.RFC3339),
		Database:  databaseReady,
		Version:   h.version,
	}

	if h.database == nil || h.database.PingDB(ctx) != nil {
		return nil, problems.New(http.StatusServiceUnavailable, problems.DatabaseUnavailable, "database unavailable")
	}

	return &HealthOutput{Body: response}, nil
}

// ServeHTTP preserves the small standalone handler surface used by focused
// tests and callers while the API itself is registered through Huma.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	output, err := h.Handle(request.Context(), &HealthInput{})
	if err != nil {
		if problem, ok := err.(*huma.ErrorModel); ok {
			problems.Write(w, problem)
			return
		}
		problems.Write(w, problems.New(http.StatusInternalServerError, problems.InternalError, "internal server error"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(output.Body)
}
