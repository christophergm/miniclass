package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chrismott/miniclass/internal/config"
)

type routeHealthDatabase struct{}

func (routeHealthDatabase) PingDB(context.Context) error { return nil }

func TestNewServerConstructsWithoutStartingProcess(t *testing.T) {
	server := NewServer(
		WithAddress(":9090"),
		WithAllowedOrigins("https://classroom.example"),
	)

	if server.HTTPServer == nil {
		t.Fatal("NewServer() returned a nil HTTP server")
	}
	if server.HTTPServer.Addr != ":9090" {
		t.Fatalf("HTTP server address = %q, want %q", server.HTTPServer.Addr, ":9090")
	}
	if server.HTTPServer.Handler == nil || server.Router == nil {
		t.Fatal("NewServer() did not install a router handler")
	}

	request := httptest.NewRequest(http.MethodGet, "/api/", nil)
	request.Header.Set("Origin", "https://classroom.example")
	recording := httptest.NewRecorder()
	server.Handler().ServeHTTP(recording, request)

	if recording.Code != http.StatusOK {
		t.Fatalf("GET /api/ status = %d, want %d", recording.Code, http.StatusOK)
	}
	if got := recording.Header().Get("Content-Type"); got != jsonContentType {
		t.Fatalf("Content-Type = %q, want %q", got, jsonContentType)
	}
	if got := recording.Header().Get("Access-Control-Allow-Origin"); got != "https://classroom.example" {
		t.Fatalf("CORS origin = %q, want configured origin", got)
	}

	var response map[string]string
	if err := json.NewDecoder(recording.Body).Decode(&response); err != nil {
		t.Fatalf("decode API root response: %v", err)
	}
	if response["service"] != "miniclass-api" {
		t.Fatalf("API root response = %#v", response)
	}
}

func TestNewServerWithConfigUsesConfiguredPort(t *testing.T) {
	server := NewServerWithConfig(config.Config{Port: "9191"})
	if server.HTTPServer.Addr != ":9191" {
		t.Fatalf("HTTP server address = %q, want %q", server.HTTPServer.Addr, ":9191")
	}
}

func TestRouterReturnsJSONForUnsupportedRequests(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		statusCode int
		message    string
	}{
		{name: "unknown route", method: http.MethodGet, path: "/api/missing", statusCode: http.StatusNotFound, message: "route not found"},
		{name: "unsupported method", method: http.MethodPost, path: "/api", statusCode: http.StatusMethodNotAllowed, message: "method not allowed"},
		{name: "unknown top level route", method: http.MethodGet, path: "/missing", statusCode: http.StatusNotFound, message: "route not found"},
	}

	router := NewRouter(RouterOptions{})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recording := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, test.path, nil)
			router.ServeHTTP(recording, request)

			if recording.Code != test.statusCode {
				t.Fatalf("status = %d, want %d", recording.Code, test.statusCode)
			}
			if got := recording.Header().Get("Content-Type"); got != jsonContentType {
				t.Fatalf("Content-Type = %q, want %q", got, jsonContentType)
			}

			var response map[string]string
			if err := json.NewDecoder(recording.Body).Decode(&response); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if response["error"] != test.message {
				t.Fatalf("error response = %#v, want message %q", response, test.message)
			}
		})
	}
}

func TestRouterServesHealthEndpoint(t *testing.T) {
	router := NewRouter(RouterOptions{
		Database: routeHealthDatabase{},
		Version:  "1.2.3",
	})
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if recording.Code != http.StatusOK {
		t.Fatalf("GET /api/health status = %d, want %d", recording.Code, http.StatusOK)
	}

	var response map[string]string
	if err := json.NewDecoder(recording.Body).Decode(&response); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if response["status"] != "healthy" || response["database"] != "connected" || response["version"] != "1.2.3" {
		t.Fatalf("health response = %#v", response)
	}
}

func TestRouterLogsCompletedRequests(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	router := NewRouter(RouterOptions{Logger: logger})

	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, httptest.NewRequest(http.MethodGet, "/api/", nil))

	if !strings.Contains(logs.String(), "http request") {
		t.Fatalf("request log = %q, want request event", logs.String())
	}
	if !strings.Contains(logs.String(), "status=200") {
		t.Fatalf("request log = %q, want status", logs.String())
	}
}

func TestRecovererReturnsJSONForPanics(t *testing.T) {
	handler := Recoverer(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	recording := httptest.NewRecorder()
	handler.ServeHTTP(recording, httptest.NewRequest(http.MethodGet, "/api/", nil))

	if recording.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recording.Code, http.StatusInternalServerError)
	}
	if recording.Header().Get("Content-Type") != jsonContentType {
		t.Fatalf("Content-Type = %q, want %q", recording.Header().Get("Content-Type"), jsonContentType)
	}
}
