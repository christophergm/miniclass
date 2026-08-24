package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/api"
	"github.com/chrismott/miniclass/internal/api/handlers"
	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestHealthIntegration(t *testing.T) {
	testDatabaseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if testDatabaseURL == "" {
		t.Skip("TEST_DATABASE_URL is required for the PostgreSQL integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	adminPool, err := pgxpool.New(ctx, testDatabaseURL)
	require.NoError(t, err)
	if err != nil {
		return
	}
	require.NoError(t, adminPool.Ping(ctx))
	t.Cleanup(adminPool.Close)

	schemaName := fmt.Sprintf("miniclass_test_%d", time.Now().UnixNano())
	_, err = adminPool.Exec(ctx, "CREATE SCHEMA "+schemaName)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := adminPool.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+schemaName+" CASCADE")
		require.NoError(t, cleanupErr)
	})

	schemaURL, err := withSearchPath(testDatabaseURL, schemaName)
	require.NoError(t, err)

	gooseDB, err := goose.OpenDBWithDriver("postgres", schemaURL)
	require.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() { require.NoError(t, gooseDB.Close()) })
	require.NoError(t, gooseDB.PingContext(ctx))

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	if !ok {
		return
	}
	migrationsDir := filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations")
	require.NoError(t, goose.Up(gooseDB, migrationsDir, goose.WithAllowMissing()))

	database, err := data.NewFromURL(ctx, schemaURL)
	require.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(database.Close)
	require.NoError(t, database.PingDB(ctx))

	var migrated bool
	require.NoError(t, database.Pool().QueryRow(ctx, `
	select exists (
			select 1
			from information_schema.tables
			where table_schema = current_schema()
			  and table_name = 'organizations'
			  )`).Scan(&migrated))
	require.True(t, migrated, "identity migration was not applied")

	var healthTableExists bool
	require.NoError(t, database.Pool().QueryRow(ctx, `
	select exists (
			select 1
			from information_schema.tables
			where table_schema = current_schema()
			  and table_name = 'health_checks'
		)`).Scan(&healthTableExists))
	require.False(t, healthTableExists, "health_checks must be removed by the identity migration")

	server := api.NewServer(
		api.WithAddress("test"),
		api.WithDatabase(database),
		api.WithVersion("integration-test"),
	)
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	response, err := http.Get(httpServer.URL + "/api/health")
	require.NoError(t, err)
	if err != nil {
		return
	}
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)

	var health handlers.HealthResponse
	require.NoError(t, json.NewDecoder(response.Body).Decode(&health))
	require.Equal(t, "healthy", health.Status)
	require.Equal(t, "connected", health.Database)
	require.Equal(t, "integration-test", health.Version)
	require.NotEmpty(t, health.Timestamp)
}

type unavailableDatabase struct{}

func (unavailableDatabase) PingDB(context.Context) error {
	return fmt.Errorf("connection refused")
}

func TestHealthFailureUsesProblemDetails(t *testing.T) {
	server := api.NewServer(api.WithDatabase(unavailableDatabase{}))
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	response, err := http.Get(httpServer.URL + "/api/health")
	require.NoError(t, err)
	if err != nil {
		return
	}
	defer response.Body.Close()

	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	require.Equal(t, "application/problem+json", response.Header.Get("Content-Type"))

	var problem struct {
		Type   string `json:"type"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&problem))
	require.Equal(t, string(problems.DatabaseUnavailable), problem.Type)
	require.Equal(t, http.StatusServiceUnavailable, problem.Status)
	require.Equal(t, "database unavailable", problem.Detail)
}

func withSearchPath(databaseURL, schemaName string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("parse test database URL: %w", err)
	}

	query := parsed.Query()
	query.Set("options", "-csearch_path="+schemaName)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
