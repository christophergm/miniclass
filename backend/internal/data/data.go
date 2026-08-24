// Package data owns database connections and transaction boundaries.
package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/chrismott/miniclass/internal/config"
	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgx connection pool. A DB returned by New has already completed
// a connectivity query.
type DB struct {
	pool      *pgxpool.Pool
	closeOnce sync.Once
}

// New creates and verifies a database connection pool from application
// configuration. The pool is closed before returning an error from the initial
// health check so failed startup does not leak connections.
func New(ctx context.Context, cfg *config.Config) (*DB, error) {
	if cfg == nil {
		return nil, errors.New("create database: configuration is nil")
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("create database: %w", err)
	}

	return NewFromURL(ctx, cfg.DatabaseURL)
}

// NewFromURL creates and verifies a database connection pool from a PostgreSQL
// connection string.
func NewFromURL(ctx context.Context, databaseURL string) (*DB, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("create database: DATABASE_URL is empty")
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	database := &DB{pool: pool}
	if err := database.PingDB(ctx); err != nil {
		database.Close()
		return nil, fmt.Errorf("create database: %w", err)
	}

	return database, nil
}

// PingDB verifies the database with the same query used by the health
// endpoint. It deliberately does not depend on a health_checks row.
func (d *DB) PingDB(ctx context.Context) error {
	if d == nil || d.pool == nil {
		return errors.New("ping database: connection pool is nil")
	}
	return PingDB(ctx, d.pool)
}

// PingDB verifies that the supplied pgx pool can execute a trivial query.
func PingDB(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("ping database: connection pool is nil")
	}
	var result int
	if err := pool.QueryRow(ctx, "select 1").Scan(&result); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	if result != 1 {
		return fmt.Errorf("ping database: select 1 returned %d", result)
	}
	return nil
}

// Pool is retained for composition with the identity accessor. Callers must
// use one of the transaction methods for queries rather than retaining a
// connection or constructing generated queries themselves.
func (d *DB) Pool() *pgxpool.Pool {
	if d == nil {
		return nil
	}
	return d.pool
}

// InTenant runs a read-write unit of work with the tenant setting scoped to
// the transaction. The setting is applied with set_config(..., true), which
// has the same transaction-local lifetime as SET LOCAL.
func (d *DB) InTenant(ctx context.Context, organizationID string, fn func(context.Context, *db.Queries) error) error {
	return d.inTenant(ctx, organizationID, pgx.ReadWrite, fn)
}

// InTenantRead runs a read-only unit of work with the tenant setting scoped to
// the transaction.
func (d *DB) InTenantRead(ctx context.Context, organizationID string, fn func(context.Context, *db.Queries) error) error {
	return d.inTenant(ctx, organizationID, pgx.ReadOnly, fn)
}

func (d *DB) inTenant(ctx context.Context, organizationID string, accessMode pgx.TxAccessMode, fn func(context.Context, *db.Queries) error) error {
	if d == nil || d.pool == nil {
		return errors.New("begin tenant transaction: connection pool is nil")
	}
	if strings.TrimSpace(organizationID) == "" {
		return errors.New("begin tenant transaction: organization id is empty")
	}
	if fn == nil {
		return errors.New("begin tenant transaction: callback is nil")
	}

	tx, err := d.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: accessMode})
	if err != nil {
		return fmt.Errorf("begin tenant transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "select set_config('app.organization_id', $1, true)", organizationID); err != nil {
		return fmt.Errorf("set tenant transaction scope: %w", err)
	}
	if err := fn(ctx, db.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tenant transaction: %w", err)
	}
	return nil
}

// Close releases all resources owned by the database pool. It is safe to call
// on a nil DB or more than once.
func (d *DB) Close() {
	if d == nil || d.pool == nil {
		return
	}
	d.closeOnce.Do(d.pool.Close)
}
