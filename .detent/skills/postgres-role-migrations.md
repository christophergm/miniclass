---
name: postgres-role-migrations
description: Safely bootstrap and roll back PostgreSQL migrator and app roles across databases.
when_to_use: When a migration creates cluster-scoped PostgreSQL roles and must support both fresh bootstrap and per-database round trips.
---

Create roles in a privileged bootstrap script and repeat the guarded creation in the migration so
pre-provisioned migrator connections do not attempt `CREATE ROLE` without `CREATEROLE`. Normalize role
attributes only when the current role is privileged. Give the migrator database/schema ownership and
grant the app role only schema usage, DML, required sequence usage, and function execution.

Do not drop cluster roles from a database down migration: another database may depend on their grants
or owned objects. Revoke this database's privileges, transfer its objects back to the migrator, and
retain the roles. Round-trip tests should provision a scratch database owned by the migrator, run up
as migrator, down as an admin, and up again as migrator.
