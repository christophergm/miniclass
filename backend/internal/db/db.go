// Package db owns the PostgreSQL connection pool used by the API.
package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/chrismott/miniclass/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgx connection pool and provides the database health check used
// by the API. A DB returned by New has already completed a connectivity ping.
type DB struct {
	pool      *pgxpool.Pool
	closeOnce sync.Once
}

// New creates and verifies a database connection pool from application
// configuration. The pool is closed before returning an error from the
// initial health check so failed startup does not leak connections.
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
// connection string. It is useful for callers that do not need the rest of
// the application configuration.
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

// PingDB verifies that the database accepts a connection.
func (d *DB) PingDB(ctx context.Context) error {
	if d == nil || d.pool == nil {
		return errors.New("ping database: connection pool is nil")
	}
	return PingDB(ctx, d.pool)
}

// PingDB verifies that the supplied pgx pool accepts a connection.
func PingDB(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("ping database: connection pool is nil")
	}
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

// Pool returns the underlying pgx pool for query and transaction operations.
func (d *DB) Pool() *pgxpool.Pool {
	if d == nil {
		return nil
	}
	return d.pool
}

// Close releases all resources owned by the database pool. It is safe to call
// on a nil DB or more than once.
func (d *DB) Close() {
	if d == nil || d.pool == nil {
		return
	}
	d.closeOnce.Do(d.pool.Close)
}
