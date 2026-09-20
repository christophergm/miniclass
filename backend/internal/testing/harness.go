// Package testing provides the PostgreSQL setup shared by isolation tests.
package testing

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	gotesting "testing"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

// Harness owns one schema-isolated database with separate migrator and app
// role pools. Tests create their own organizations through the migrator pool,
// then exercise application access through data.DB and the app role.
const harnessMigrationLockID int64 = 4_939_441_364_617_518_915

type Harness struct {
	Context  context.Context
	Migrator *pgxpool.Pool
	App      *pgxpool.Pool
	Database *data.DB
	Schema   string
}

// Open creates and migrates one isolated schema for the package's tests.
func Open(t gotesting.TB) *Harness {
	t.Helper()
	migratorURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	appURL := strings.TrimSpace(os.Getenv("TEST_APP_DATABASE_URL"))
	if migratorURL == "" || appURL == "" {
		t.Skip("TEST_DATABASE_URL and TEST_APP_DATABASE_URL are required for isolation tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	bootstrapPool, err := pgxpool.New(ctx, migratorURL)
	require.NoError(t, err)
	if err != nil {
		return nil
	}
	require.NoError(t, bootstrapPool.Ping(ctx))

	schemaName := fmt.Sprintf("miniclass_isolation_%d", time.Now().UnixNano())
	_, err = bootstrapPool.Exec(ctx, "create schema "+schemaName)
	require.NoError(t, err)

	migratorSchemaURL, err := withSearchPath(migratorURL, schemaName)
	require.NoError(t, err)
	appSchemaURL, err := withSearchPath(appURL, schemaName)
	require.NoError(t, err)
	migrator, err := pgxpool.New(ctx, migratorSchemaURL)
	require.NoError(t, err)
	if err != nil {
		return nil
	}
	require.NoError(t, migrator.Ping(ctx))

	gooseDB, err := goose.OpenDBWithDriver("postgres", migratorSchemaURL)
	require.NoError(t, err)
	if err != nil {
		return nil
	}
	// Migrations create schema-local tables but also replace the shared public xid
	// functions. One connection keeps this session-level advisory lock in effect
	// for the complete migration run across concurrently executing test packages.
	gooseDB.SetMaxOpenConns(1)
	gooseDB.SetMaxIdleConns(1)
	_, err = gooseDB.ExecContext(ctx, "select pg_advisory_lock($1)", harnessMigrationLockID)
	require.NoError(t, err)
	if err != nil {
		return nil
	}
	migrationLocked := true
	defer func() {
		if migrationLocked {
			_, unlockErr := gooseDB.ExecContext(context.Background(), "select pg_advisory_unlock($1)", harnessMigrationLockID)
			require.NoError(t, unlockErr)
		}
	}()
	require.NoError(t, goose.Up(gooseDB, migrationsPath(t), goose.WithAllowMissing()))
	_, err = gooseDB.ExecContext(ctx, "select pg_advisory_unlock($1)", harnessMigrationLockID)
	require.NoError(t, err)
	migrationLocked = false
	require.NoError(t, gooseDB.Close())

	app, err := pgxpool.New(ctx, appSchemaURL)
	require.NoError(t, err)
	if err != nil {
		return nil
	}
	require.NoError(t, app.Ping(ctx))

	database, err := data.NewApplicationFromURL(ctx, appSchemaURL)
	require.NoError(t, err)
	if err != nil {
		return nil
	}

	harness := &Harness{
		Context:  ctx,
		Migrator: migrator,
		App:      app,
		Database: database,
		Schema:   schemaName,
	}
	t.Cleanup(func() {
		database.Close()
		app.Close()
		_, cleanupErr := bootstrapPool.Exec(context.Background(), "drop schema if exists "+schemaName+" cascade")
		require.NoError(t, cleanupErr)
		migrator.Close()
		bootstrapPool.Close()
	})
	return harness
}

// MintOrganization creates synthetic tenant data without using the app role.
func (h *Harness) MintOrganization(t gotesting.TB) ids.XID {
	t.Helper()
	var id ids.XID
	name := fmt.Sprintf("Synthetic Isolation %d", time.Now().UnixNano())
	err := h.Migrator.QueryRow(h.Context, `
		insert into organizations (name)
		values ($1)
		returning id`, name).Scan(&id)
	require.NoError(t, err)
	return id
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

func migrationsPath(t gotesting.TB) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations")
}
