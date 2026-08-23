package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeDatabase struct {
	err    error
	called bool
}

func (f *fakeDatabase) PingDB(context.Context) error {
	f.called = true
	return f.err
}

func TestHealthHandlerHealthy(t *testing.T) {
	database := &fakeDatabase{}
	handler := NewHealthHandler(database, "1.2.3")
	handler.now = func() time.Time {
		return time.Date(2026, time.August, 23, 17, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	}

	recording := httptest.NewRecorder()
	handler.ServeHTTP(recording, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if recording.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recording.Code, http.StatusOK)
	}
	if recording.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", recording.Header().Get("Content-Type"))
	}

	var response HealthResponse
	decodeHealthResponse(t, recording, &response)
	if response != (HealthResponse{
		Status:    statusHealthy,
		Timestamp: "2026-08-24T00:00:00Z",
		Database:  databaseReady,
		Version:   "1.2.3",
	}) {
		t.Fatalf("response = %#v", response)
	}
	if !database.called {
		t.Fatal("health handler did not ping the database")
	}
}

func TestHealthHandlerDatabaseFailure(t *testing.T) {
	database := &fakeDatabase{err: errors.New("connection refused")}
	handler := NewHealthHandler(database, "1.2.3")
	handler.now = func() time.Time { return time.Date(2026, time.August, 23, 17, 0, 0, 0, time.UTC) }

	recording := httptest.NewRecorder()
	handler.ServeHTTP(recording, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if recording.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recording.Code, http.StatusServiceUnavailable)
	}

	var response HealthResponse
	decodeHealthResponse(t, recording, &response)
	if response.Status != statusUnhealthy || response.Database != databaseDown {
		t.Fatalf("response status/database = %q/%q", response.Status, response.Database)
	}
	if response.Error != "database unavailable" {
		t.Fatalf("response error = %q", response.Error)
	}
	if response.Version != "1.2.3" || response.Timestamp != "2026-08-23T17:00:00Z" {
		t.Fatalf("response metadata = %#v", response)
	}
}

func TestHealthHandlerWithoutDatabaseIsUnavailable(t *testing.T) {
	handler := NewHealthHandler(nil, "1.2.3")
	recording := httptest.NewRecorder()
	handler.ServeHTTP(recording, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if recording.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recording.Code, http.StatusServiceUnavailable)
	}
}

func decodeHealthResponse(t *testing.T, recording *httptest.ResponseRecorder, response *HealthResponse) {
	t.Helper()
	if err := json.NewDecoder(recording.Body).Decode(response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
