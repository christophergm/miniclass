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
	"sync"
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
// role pools. Each test creates its own organization through the migrator pool,
// then exercises application access through data.DB and the app role.
const harnessMigrationLockID int64 = 4_939_441_364_617_518_915

var (
	harnessMu     sync.Mutex
	sharedHarness *Harness
)

type Harness struct {
	Context  context.Context
	Migrator *pgxpool.Pool
	App      *pgxpool.Pool
	Database *data.DB
	Schema   string

	bootstrap *pgxpool.Pool
}

// Open returns the package-wide schema-isolated harness. The schema is created
// and migrated once, so tests must use a distinct organization for their data.
// Call Close from the package's TestMain after all tests have completed.
func Open(t gotesting.TB) *Harness {
	t.Helper()
	harnessMu.Lock()
	defer harnessMu.Unlock()
	if sharedHarness != nil {
		return sharedHarness
	}

	migratorURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	appURL := strings.TrimSpace(os.Getenv("TEST_APP_DATABASE_URL"))
	if migratorURL == "" || appURL == "" {
		t.Skip("TEST_DATABASE_URL and TEST_APP_DATABASE_URL are required for isolation tests")
	}

	setupCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	bootstrapPool, err := newTestPool(setupCtx, migratorURL, 1)
	require.NoError(t, err)
	if err != nil {
		return nil
	}
	require.NoError(t, bootstrapPool.Ping(setupCtx))

	schemaName := fmt.Sprintf("miniclass_isolation_%d", time.Now().UnixNano())
	_, err = bootstrapPool.Exec(setupCtx, "create schema "+schemaName)
	require.NoError(t, err)

	migratorSchemaURL, err := withSearchPath(migratorURL, schemaName)
	require.NoError(t, err)
	appSchemaURL, err := withSearchPath(appURL, schemaName)
	require.NoError(t, err)
	// Tests occasionally hold a migrator transaction while making a separate
	// metadata query, so this pool needs a second connection to avoid
	// self-deadlock.
	migrator, err := newTestPool(setupCtx, migratorSchemaURL, 2)
	require.NoError(t, err)
	if err != nil {
		return nil
	}
	require.NoError(t, migrator.Ping(setupCtx))

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
	_, err = gooseDB.ExecContext(setupCtx, "select pg_advisory_lock($1)", harnessMigrationLockID)
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
	_, err = gooseDB.ExecContext(setupCtx, "select pg_advisory_unlock($1)", harnessMigrationLockID)
	require.NoError(t, err)
	migrationLocked = false
	require.NoError(t, gooseDB.Close())

	// App is reserved for raw role/RLS assertions. Keep it separate from the
	// service pool: completed SET LOCAL calls leave an empty custom setting in a
	// reused PostgreSQL session, whereas these assertions intentionally require
	// app.organization_id to be absent.
	app, err := newTestPool(setupCtx, appSchemaURL, 2)
	require.NoError(t, err)
	if err != nil {
		return nil
	}
	require.NoError(t, app.Ping(setupCtx))

	databasePool, err := newTestPool(setupCtx, appSchemaURL, 2)
	require.NoError(t, err)
	if err != nil {
		return nil
	}
	require.NoError(t, databasePool.Ping(setupCtx))

	database, err := data.NewApplicationFromPool(setupCtx, databasePool)
	require.NoError(t, err)
	if err != nil {
		return nil
	}

	sharedHarness = &Harness{
		Context:   context.Background(),
		Migrator:  migrator,
		App:       app,
		Database:  database,
		Schema:    schemaName,
		bootstrap: bootstrapPool,
	}
	return sharedHarness
}

// Close releases the package-wide harness and drops its isolated schema.
func Close() error {
	harnessMu.Lock()
	harness := sharedHarness
	sharedHarness = nil
	harnessMu.Unlock()
	if harness == nil {
		return nil
	}

	harness.Database.Close()
	harness.App.Close()
	harness.Migrator.Close()
	_, err := harness.bootstrap.Exec(context.Background(), "drop schema if exists "+harness.Schema+" cascade")
	harness.bootstrap.Close()
	if err != nil {
		return fmt.Errorf("drop shared test schema: %w", err)
	}
	return nil
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

func newTestPool(ctx context.Context, databaseURL string, maxConns int32) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	poolConfig.MaxConns = maxConns
	poolConfig.MinConns = 0
	return pgxpool.NewWithConfig(ctx, poolConfig)
}

func migrationsPath(t gotesting.TB) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations")
}
