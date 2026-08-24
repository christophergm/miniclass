---
name: postgres-tenant-isolation-harness
description: Build schema-isolated PostgreSQL integration tests that exercise both migrator and non-bypassrls app roles.
when_to_use: When adding or changing tenant-scoped schema, RLS policies, transaction-local tenant context, or append-only audit persistence.
---

# PostgreSQL tenant isolation harness

- Require separate `TEST_DATABASE_URL` (migrator) and `TEST_APP_DATABASE_URL` (app role) URLs.
- Create a unique synthetic schema through the migrator, then reconnect both pools with
  `options=-csearch_path=<schema>` before applying Goose migrations. Keep one migrated schema per
  test package and mint a fresh synthetic organization for each test case.
- Grant the app role only the identity allowlist and the intended tenant tables in the isolated
  schema; revoke migration bookkeeping access so metadata checks see exactly the application surface.
- Verify structurally from `pg_class`, `pg_policies`, `pg_attribute`, and `pg_constraint`: tenant
  key, RLS enabled and forced, policy using `current_setting('app.organization_id')` without
  `missing_ok`, `(id, organization_id)` uniqueness, and organization columns on tenant FKs.
- Verify behavior through the app pool: a no-GUC query fails SQLSTATE 42704, tenant A cannot see
  tenant B rows, a read-write unit without an audit entry cannot commit, and audit update/delete
  fail with app-role privileges even when the GUC is set.
- Scope every count/list assertion inside a tenant transaction; never assert global row counts.
