package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/registry"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestLayerOneIsolationAndAudit(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationA := harness.MintOrganization(t)
	organizationB := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "isolation harness"}

	err := harness.Database.InTenant(ctx, string(organizationA), actor, func(ctx context.Context, tx *data.Tx) error {
		entry := audit.Entry{
			Action:        audit.ActionCreate,
			ObjectType:    "isolation_probe",
			ChangeSummary: []byte(`{"created":true}`),
			Reason:        "layer 1 verification",
		}
		if err := tx.Record(ctx, entry); err != nil {
			return err
		}
		row, err := tx.Queries().CountAuditLog(ctx)
		if err != nil {
			return err
		}
		if row != 1 {
			return fmt.Errorf("organization A audit count = %d, want 1", row)
		}
		return nil
	})
	// The xid is read back below; this first transaction only proves that a
	// successful Record satisfies the commit invariant.
	require.NoError(t, err)

	err = harness.Database.InTenantRead(ctx, string(organizationA), func(ctx context.Context, tx *data.Tx) error {
		count, err := tx.Queries().CountAuditLog(ctx)
		if err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("organization A audit count = %d, want 1", count)
		}
		return nil
	})
	require.NoError(t, err)

	err = harness.Database.InTenantRead(ctx, string(organizationB), func(ctx context.Context, tx *data.Tx) error {
		count, err := tx.Queries().CountAuditLog(ctx)
		if err != nil {
			return err
		}
		if count != 0 {
			return fmt.Errorf("organization B audit count = %d, want 0", count)
		}
		return nil
	})
	require.NoError(t, err)

	err = harness.Database.InTenant(ctx, string(organizationA), actor, func(context.Context, *data.Tx) error {
		return nil
	})
	require.ErrorIs(t, err, data.ErrAuditRequired)

	err = harness.Database.InTenant(ctx, string(organizationA), actor, func(_ context.Context, tx *data.Tx) error {
		tx.NoAuditRequired("synthetic harness setup")
		return nil
	})
	require.NoError(t, err)

	var count int64
	err = harness.App.QueryRow(ctx, "select count(*) from audit_log").Scan(&count)
	assertSQLState(t, err, "42704")

	privilegeTx, err := harness.App.Begin(ctx)
	require.NoError(t, err)
	_, err = privilegeTx.Exec(ctx, "set local app.organization_id = '"+string(organizationA)+"'")
	require.NoError(t, err)
	_, err = privilegeTx.Exec(ctx, "update audit_log set actor_label = 'tampered'")
	assertSQLState(t, err, "42501")
	require.NoError(t, privilegeTx.Rollback(ctx))

	privilegeTx, err = harness.App.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = privilegeTx.Rollback(ctx) }()
	_, err = privilegeTx.Exec(ctx, "set local app.organization_id = '"+string(organizationA)+"'")
	require.NoError(t, err)
	_, err = privilegeTx.Exec(ctx, "delete from audit_log")
	assertSQLState(t, err, "42501")

	verifySchemaContract(t, harness)
}

func verifySchemaContract(t *testing.T, harness *testharness.Harness) {
	t.Helper()
	// This literal is intentionally the complete non-tenant allowlist from ADR
	// 0007. Adding a name here must be a visible, spec-cited change.
	nonTenantTables := []string{"organizations", "users", "organization_members", "access_tokens"}
	allowlist := make(map[string]struct{}, len(nonTenantTables))
	for _, table := range nonTenantTables {
		allowlist[table] = struct{}{}
	}

	rows, err := harness.Migrator.Query(harness.Context, `
		select c.relname
		from pg_class c
		join pg_namespace n on n.oid = c.relnamespace
		where n.nspname = current_schema()
		  and c.relkind = 'r'
		  and (has_table_privilege('miniclass_app', c.oid, 'select')
		       or has_table_privilege('miniclass_app', c.oid, 'insert')
		       or has_table_privilege('miniclass_app', c.oid, 'update')
		       or has_table_privilege('miniclass_app', c.oid, 'delete'))`)
	require.NoError(t, err)
	defer rows.Close()

	tenantTables := make(map[string]struct{})
	for rows.Next() {
		var table string
		require.NoError(t, rows.Scan(&table))
		if _, ok := allowlist[table]; ok {
			continue
		}
		tenantTables[table] = struct{}{}

		var hasOrganizationID, rlsEnabled, rlsForced, hasPolicy, hasUnique bool
		err := harness.Migrator.QueryRow(harness.Context, `
			select
				exists (
					select 1
					from pg_attribute a
					join pg_type typ on typ.oid = a.atttypid
					where a.attrelid = c.oid
					  and a.attname = 'organization_id'
					  and a.attnotnull
					  and typ.typnamespace = 'public'::regnamespace
					  and typ.typname = 'xid20'
				),
				c.relrowsecurity,
				c.relforcerowsecurity,
				exists (
					select 1 from pg_policies p
					where p.schemaname = current_schema()
					  and p.tablename = c.relname
					  and (coalesce(p.qual, '') || coalesce(p.with_check, '')) like '%app.organization_id%'
				),
				exists (
					select 1
					from pg_constraint con
					where con.conrelid = c.oid
					  and con.contype = 'u'
					  and (select array_agg(att.attname order by cols.ordinality)
					       from unnest(con.conkey) with ordinality cols(attnum, ordinality)
					       join pg_attribute att on att.attrelid = con.conrelid and att.attnum = cols.attnum)
					      = array['id', 'organization_id']::name[]
				) 
			from pg_class c
			join pg_namespace n on n.oid = c.relnamespace
			where n.nspname = current_schema() and c.relname = $1`, table).Scan(
			&hasOrganizationID, &rlsEnabled, &rlsForced, &hasPolicy, &hasUnique,
		)
		require.NoError(t, err)
		require.True(t, hasOrganizationID, table+" lacks organization_id")
		require.True(t, rlsEnabled, table+" does not enable RLS")
		require.True(t, rlsForced, table+" does not force RLS")
		require.True(t, hasPolicy, table+" lacks an app.organization_id policy")
		require.True(t, hasUnique, table+" lacks unique(id, organization_id)")

		var queryCount int64
		err = harness.App.QueryRow(harness.Context, `select count(*) from `+quoteIdentifier(harness.Schema)+`.`+quoteIdentifier(table)).Scan(&queryCount)
		assertSQLState(t, err, "42704")
	}
	require.NoError(t, rows.Err())
	require.Contains(t, tenantTables, "audit_log")
	for table := range tenantTables {
		if table == "audit_log" {
			continue
		}
		_, ok := registry.ForTable(table)
		require.True(t, ok, table+" lacks a Layer 2 entity registry entry")
	}
	for _, entity := range registry.Entries() {
		if !entity.YearScoped {
			continue
		}
		assertClosedYearTrigger(t, harness, entity.TableName)
	}
	for table := range tenantTables {
		if table == "audit_log" {
			continue
		}
		var yearScoped bool
		err := harness.Migrator.QueryRow(harness.Context, `
			select $1 = 'school_years' or exists (
				select 1 from pg_attribute a
				join pg_class c on c.oid = a.attrelid
				join pg_namespace n on n.oid = c.relnamespace
				where n.nspname = current_schema()
				  and c.relname = $1
				  and a.attname = 'school_year_id'
				  and not a.attisdropped
			)`, table).Scan(&yearScoped)
		require.NoError(t, err)
		if yearScoped {
			assertClosedYearTrigger(t, harness, table)
		}
	}

	foreignKeys, err := harness.Migrator.Query(harness.Context, `
		select con.conrelid::regclass::text,
		       con.confrelid::regclass::text,
		       array(select a.attname
		             from unnest(con.conkey) with ordinality cols(attnum, ordinality)
		             join pg_attribute a on a.attrelid = con.conrelid and a.attnum = cols.attnum
		             order by cols.ordinality),
		       array(select a.attname
		             from unnest(con.confkey) with ordinality cols(attnum, ordinality)
		             join pg_attribute a on a.attrelid = con.confrelid and a.attnum = cols.attnum
		             order by cols.ordinality)
		from pg_constraint con
		join pg_namespace src on src.oid = con.connamespace
		where con.contype = 'f' and src.nspname = current_schema()`)
	require.NoError(t, err)
	defer foreignKeys.Close()
	for foreignKeys.Next() {
		var source, target string
		var sourceColumns, targetColumns []string
		require.NoError(t, foreignKeys.Scan(&source, &target, &sourceColumns, &targetColumns))
		if _, sourceTenant := tenantTables[strings.TrimPrefix(source, harness.Schema+".")]; !sourceTenant {
			continue
		}
		if _, targetTenant := tenantTables[strings.TrimPrefix(target, harness.Schema+".")]; !targetTenant {
			continue
		}
		require.Contains(t, sourceColumns, "organization_id", source+" FK lacks organization_id")
		require.Contains(t, targetColumns, "organization_id", target+" FK target lacks organization_id")
	}
	require.NoError(t, foreignKeys.Err())
}

func assertClosedYearTrigger(t *testing.T, harness *testharness.Harness, table string) {
	t.Helper()
	var hasClosedYearTrigger bool
	err := harness.Migrator.QueryRow(harness.Context, `
		select exists (
			select 1
			from pg_trigger t
			join pg_class c on c.oid = t.tgrelid
			join pg_namespace n on n.oid = c.relnamespace
			where n.nspname = current_schema()
			  and c.relname = $1
			  and not t.tgisinternal
			  and pg_get_triggerdef(t.oid) like '%prevent_closed_school_year_mutation%'
		)`, table).Scan(&hasClosedYearTrigger)
	require.NoError(t, err)
	require.True(t, hasClosedYearTrigger, table+" lacks the shared closed-year trigger")
}

func assertSQLState(t *testing.T, err error, want string) {
	t.Helper()
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, want, pgErr.Code)
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
