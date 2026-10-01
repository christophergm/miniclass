# Production database role bootstrap

Use this once for each Supabase project (staging and production). It implements the app/migrator split required by [ADR 0018 §2](./adr/0018-render-deployment-and-live-beta.md).

## 1. Connect as a database administrator

In Supabase **Connect**, obtain an administrator-capable connection URI and connect with `psql`:

```sh
psql "$SUPABASE_ADMIN_DATABASE_URL"
```

The connected role must have `CREATEROLE`. Do not store this URI in Render, CI, or source control.

```sql
select rolcreaterole
from pg_roles
where rolname = current_user;
```

## 2. Create restrictive login roles

Run the following as the administrator. Do **not** use `backend/scripts/init-roles.sql`: its passwords are for local development only.

```sql
create role miniclass_migrator
    login nosuperuser nocreatedb nocreaterole inherit noreplication nobypassrls;

create role miniclass_app
    login nosuperuser nocreatedb nocreaterole noinherit noreplication nobypassrls;

alter role miniclass_app set statement_timeout = '10s';
```

Set distinct generated passwords interactively, so they do not enter shell history or SQL files:

```text
\password miniclass_migrator
\password miniclass_app
```

Generate each password with a password manager or:

```sh
openssl rand -hex 32
```

## 3. Assign schema access

Replace `<database-name>` with the database named in the Supabase connection URI.

```sql
grant connect on database <database-name> to miniclass_migrator;
grant connect on database <database-name> to miniclass_app;

revoke create on schema public from public;
grant usage on schema public to miniclass_app;

-- PostgreSQL requires the current schema owner to be able to SET ROLE to the
-- new owner. This membership exists only for the ownership transfer.
grant miniclass_migrator to postgres;
alter schema public owner to miniclass_migrator;
revoke miniclass_migrator from postgres;
```

The temporary membership lets `postgres` transfer `public` to `miniclass_migrator`; the final `revoke` removes it while preserving the ownership change. Afterward, the migrator owns and changes the schema, and the app role can use—but cannot create objects in—the schema.

If your administrator role is not named `postgres`, replace it in the `grant` and `revoke` statements with the role returned by `select current_user;`.

If permitted by the managed database, also run:

```sql
alter database <database-name> owner to miniclass_migrator;
```

If that command is refused, do not grant extra privileges to `miniclass_app`; retain the schema ownership above and confirm the migrator can run migrations.

## 4. Run the initial migration as the migrator

Store the migrator URI only in the secured release job. It uses the `miniclass_migrator` username, its URL-encoded password, and Supabase TLS parameters:

```sh
cd backend
DATABASE_URL="$MINICLASS_MIGRATOR_DATABASE_URL" go run ./cmd/migrate up
```

The committed migrations grant `miniclass_app` the required DML and default privileges. Do not grant broad table privileges manually.

## 5. Verify, then place the secrets

Connect as `miniclass_app` and verify:

```sql
select current_user;

select
    has_schema_privilege(current_user, 'public', 'usage') as can_use_public,
    has_schema_privilege(current_user, 'public', 'create') as can_create_public;

select rolbypassrls
from pg_roles
where rolname = current_user;
```

Expected: `miniclass_app`, `can_use_public = true`, `can_create_public = false`, and `rolbypassrls = false`.

| Credential | Store it only in |
| --- | --- |
| `miniclass_app` URI | Render API `APP_DATABASE_URL` |
| `miniclass_migrator` URI | Release/CI migration job `DATABASE_URL` |
| Administrator URI | Controlled bootstrap or break-glass procedure |

Never give the API the migrator or administrator URI. The API checks at startup that it is connected as `miniclass_app` and cannot bypass RLS or create objects in `public`.
