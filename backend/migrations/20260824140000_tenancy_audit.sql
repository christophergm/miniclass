-- +goose Up

create type audit_actor_type as enum (
    'user',
    'link',
    'system'
);

create table audit_log (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    occurred_at timestamptz not null default now(),
    actor_type audit_actor_type not null,
    actor_user_id public.xid20 references users (id) on delete set null,
    actor_label text not null,
    action text not null,
    object_type text not null,
    object_id public.xid20,
    change_summary jsonb not null default '{}'::jsonb,
    reason text,
    school_year_id public.xid20,
    request_id text,
    constraint audit_log_actor_label_check check (btrim(actor_label) <> ''),
    constraint audit_log_action_check check (btrim(action) <> ''),
    constraint audit_log_object_type_check check (btrim(object_type) <> ''),
    constraint audit_log_id_organization_key unique (id, organization_id)
);

create index audit_log_keyset_idx on audit_log (occurred_at, id);

alter table audit_log enable row level security;
alter table audit_log force row level security;

create policy audit_log_tenant_isolation on audit_log
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

-- The roles migration grants broad DML privileges to the app role for the
-- identity layer. Repeat those grants for schema-isolated test databases, and
-- remove the migration bookkeeping table from the app surface.
-- +goose StatementBegin
do $$
declare
    schema_name text := current_schema();
    table_name text;
begin
    execute format('grant usage on schema %I to miniclass_app', schema_name);
    foreach table_name in array array['organizations', 'users', 'organization_members', 'access_tokens'] loop
        execute format('grant select, insert, update, delete on table %I.%I to miniclass_app', schema_name, table_name);
    end loop;
    execute format('revoke all privileges on table %I.goose_db_version from miniclass_app', schema_name);
end
$$;
-- +goose StatementEnd

revoke all privileges on audit_log from miniclass_app;
grant select, insert on audit_log to miniclass_app;

-- +goose Down

revoke all privileges on audit_log from miniclass_app;
drop policy if exists audit_log_tenant_isolation on audit_log;
drop table audit_log;
drop type audit_actor_type;
