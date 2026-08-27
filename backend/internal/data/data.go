// Package data owns database connections and transaction boundaries.
package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/config"
	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrAuditRequired is returned when a read-write tenant transaction reaches
// commit without either recording an audit entry or declaring an explicit
// non-auditable reason.
var ErrAuditRequired = errors.New("commit tenant transaction: audit entry or NoAuditRequired reason is required")

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

	return NewApplicationFromURL(ctx, cfg.AppDatabaseURL)
}

// NewFromURL creates and verifies a database connection pool from a PostgreSQL
// connection string. It is used by migration and test paths whose role is
// intentionally not the API role.
func NewFromURL(ctx context.Context, databaseURL string) (*DB, error) {
	return newFromURL(ctx, databaseURL)
}

// NewApplicationFromURL creates an application connection pool and verifies
// that it has the least-privileged API role. Keeping this check at connection
// startup makes an API pointed at DATABASE_URL fail instead of silently
// running with migrator privileges.
func NewApplicationFromURL(ctx context.Context, databaseURL string) (*DB, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("create application database: APP_DATABASE_URL is empty")
	}
	database, err := newFromURL(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := verifyApplicationRole(ctx, database); err != nil {
		database.Close()
		return nil, fmt.Errorf("create application database: %w", err)
	}
	return database, nil
}

func newFromURL(ctx context.Context, databaseURL string) (*DB, error) {
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

func verifyApplicationRole(ctx context.Context, database *DB) error {
	var currentUser string
	if err := database.pool.QueryRow(ctx, "select current_user").Scan(&currentUser); err != nil {
		return fmt.Errorf("verify database role: %w", err)
	}
	if currentUser != "miniclass_app" {
		return fmt.Errorf("verify database role: API must connect as miniclass_app, got %q", currentUser)
	}

	var bypassRLS, canCreateSchema bool
	if err := database.pool.QueryRow(ctx, `
		select r.rolbypassrls, has_schema_privilege(current_user, 'public', 'create')
		from pg_roles r
		where r.rolname = current_user`).Scan(&bypassRLS, &canCreateSchema); err != nil {
		return fmt.Errorf("verify database role privileges: %w", err)
	}
	if bypassRLS {
		return errors.New("verify database role: miniclass_app must not bypass row-level security")
	}
	if canCreateSchema {
		return errors.New("verify database role: miniclass_app must not create objects in the public schema")
	}
	return nil
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

// Tx is the unit of work handed to a tenant callback. Generated queries are
// constructed only here; callers use Queries without importing the generated
// package and record audit entries against the same transaction.
type Tx struct {
	queries        *db.Queries
	raw            pgx.Tx
	organizationID ids.XID
	actor          audit.Actor
	readOnly       bool
	auditRecorded  bool
	noAuditReason  string
}

// Queries returns the generated query facade bound to this transaction.
// Constructing the facade and controlling its lifetime remain responsibilities
// of internal/data.
func (tx *Tx) Queries() *db.Queries {
	if tx == nil {
		return nil
	}
	return tx.queries
}

// InTenant runs a read-write unit of work with the tenant setting scoped to
// the transaction. A successful callback must record an audit entry or call
// NoAuditRequired with a non-empty reason before the transaction can commit.
func (d *DB) InTenant(ctx context.Context, organizationID string, actor audit.Actor, fn func(context.Context, *Tx) error) error {
	if err := actor.Validate(); err != nil {
		return fmt.Errorf("begin tenant transaction: %w", err)
	}
	return d.inTenant(ctx, organizationID, actor, pgx.ReadWrite, fn)
}

// InTenantRead runs a read-only unit of work with the tenant setting scoped to
// the transaction.
func (d *DB) InTenantRead(ctx context.Context, organizationID string, fn func(context.Context, *Tx) error) error {
	return d.inTenant(ctx, organizationID, audit.Actor{}, pgx.ReadOnly, fn)
}

func (d *DB) inTenant(ctx context.Context, organizationID string, actor audit.Actor, accessMode pgx.TxAccessMode, fn func(context.Context, *Tx) error) error {
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

	if _, err := tx.Exec(ctx, "set local app.organization_id = "+sqlStringLiteral(strings.TrimSpace(organizationID))); err != nil {
		return fmt.Errorf("set tenant transaction scope: %w", err)
	}
	unit := &Tx{
		queries:        db.New(tx),
		raw:            tx,
		organizationID: ids.XID(strings.TrimSpace(organizationID)),
		actor:          actor,
		readOnly:       accessMode == pgx.ReadOnly,
	}
	if err := fn(ctx, unit); err != nil {
		return err
	}
	if accessMode == pgx.ReadWrite && !unit.auditRecorded && unit.noAuditReason == "" {
		return ErrAuditRequired
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tenant transaction: %w", err)
	}
	return nil
}

// Record appends an audit entry inside the current tenant transaction.
func (tx *Tx) Record(ctx context.Context, entry audit.Entry) error {
	if tx == nil || tx.queries == nil {
		return errors.New("record audit entry: transaction is nil")
	}
	if err := entry.Validate(); err != nil {
		return err
	}
	if err := tx.actor.Validate(); err != nil {
		return fmt.Errorf("record audit entry: %w", err)
	}
	if tx.readOnly {
		return errors.New("record audit entry: transaction is read-only")
	}

	changeSummary := entry.ChangeSummary
	if len(changeSummary) == 0 {
		changeSummary = []byte(`{}`)
	}
	var reason pgtype.Text
	if strings.TrimSpace(entry.Reason) != "" {
		reason = pgtype.Text{String: strings.TrimSpace(entry.Reason), Valid: true}
	}
	var requestID pgtype.Text
	if strings.TrimSpace(entry.RequestID) != "" {
		requestID = pgtype.Text{String: strings.TrimSpace(entry.RequestID), Valid: true}
	}
	_, err := tx.queries.CreateAuditLog(ctx, db.CreateAuditLogParams{
		OrganizationID: tx.organizationID,
		ActorType:      db.AuditActorType(tx.actor.Type),
		ActorUserID:    tx.actor.UserID,
		ActorLabel:     strings.TrimSpace(tx.actor.Label),
		Action:         string(entry.Action),
		ObjectType:     strings.TrimSpace(entry.ObjectType),
		ObjectID:       entry.ObjectID,
		ChangeSummary:  changeSummary,
		Reason:         reason,
		SchoolYearID:   entry.SchoolYearID,
		RequestID:      requestID,
	})
	if err != nil {
		return fmt.Errorf("record audit entry: %w", err)
	}
	tx.auditRecorded = true
	return nil
}

// NoAuditRequired explicitly documents a successful write that is not an
// auditable user action. An empty reason does not satisfy the commit invariant.
func (tx *Tx) NoAuditRequired(reason string) {
	if tx == nil {
		return
	}
	tx.noAuditReason = strings.TrimSpace(reason)
}

func sqlStringLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// Close releases all resources owned by the database pool. It is safe to call
// on a nil DB or more than once.
func (d *DB) Close() {
	if d == nil || d.pool == nil {
		return
	}
	d.closeOnce.Do(d.pool.Close)
}
