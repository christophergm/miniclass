package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/config"
	"github.com/danielgtaylor/huma/v2"
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
	if got := recording.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
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
		typeSlug   problems.Slug
	}{
		{name: "unknown route", method: http.MethodGet, path: "/api/missing", statusCode: http.StatusNotFound, message: "route not found", typeSlug: problems.RouteNotFound},
		{name: "unsupported method", method: http.MethodPost, path: "/api", statusCode: http.StatusMethodNotAllowed, message: "method not allowed", typeSlug: problems.MethodNotAllowed},
		{name: "unknown top level route", method: http.MethodGet, path: "/missing", statusCode: http.StatusNotFound, message: "route not found", typeSlug: problems.RouteNotFound},
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
			if got := recording.Header().Get("Content-Type"); got != "application/problem+json" {
				t.Fatalf("Content-Type = %q, want application/problem+json", got)
			}

			var response huma.ErrorModel
			if err := json.NewDecoder(recording.Body).Decode(&response); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if response.Type != string(test.typeSlug) || response.Detail != test.message || response.Status != test.statusCode {
				t.Fatalf("error response = %#v, want type %q/status %d/detail %q", response, test.typeSlug, test.statusCode, test.message)
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

type failingHealthDatabase struct{}

func (failingHealthDatabase) PingDB(context.Context) error {
	return errors.New("connection refused")
}

func TestRouterServesHealthFailureAsProblemDetails(t *testing.T) {
	router := NewRouter(RouterOptions{Database: failingHealthDatabase{}})
	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if recording.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /api/health status = %d, want %d", recording.Code, http.StatusServiceUnavailable)
	}
	if got := recording.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json", got)
	}

	var response huma.ErrorModel
	if err := json.NewDecoder(recording.Body).Decode(&response); err != nil {
		t.Fatalf("decode health problem response: %v", err)
	}
	if response.Type != string(problems.DatabaseUnavailable) || response.Status != http.StatusServiceUnavailable {
		t.Fatalf("health problem type/status = %q/%d", response.Type, response.Status)
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

func TestOpenAPIContractIsDeterministicAndIncludesProblemTypes(t *testing.T) {
	first, err := json.Marshal(NewOpenAPI(RouterOptions{Version: "1.2.3"}))
	if err != nil {
		t.Fatalf("marshal first OpenAPI document: %v", err)
	}
	second, err := json.Marshal(NewOpenAPI(RouterOptions{Version: "1.2.3"}))
	if err != nil {
		t.Fatalf("marshal second OpenAPI document: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("OpenAPI document changed between identical generations")
	}

	var document map[string]any
	if err := json.Unmarshal(first, &document); err != nil {
		t.Fatalf("decode OpenAPI document: %v", err)
	}
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI paths = %#v", document["paths"])
	}
	health, ok := paths["/api/health"].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI health path = %#v", paths["/api/health"])
	}
	getHealth, ok := health["get"].(map[string]any)
	if !ok || getHealth["operationId"] != "get-health" {
		t.Fatalf("OpenAPI health operation = %#v", health["get"])
	}
	if _, ok := document["x-miniclass-problem-types"]; !ok {
		t.Fatal("OpenAPI document does not include the problem-type registry")
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
	if recording.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json", recording.Header().Get("Content-Type"))
	}
}
