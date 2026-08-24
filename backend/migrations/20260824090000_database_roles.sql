-- +goose Up

-- Roles are cluster-scoped, so this migration is safe on databases that were
-- initialized by Compose as well as on an existing PostgreSQL cluster.
-- +goose StatementBegin
do $$
begin
    if not exists (select 1 from pg_roles where rolname = 'miniclass_migrator') then
        begin
            create role miniclass_migrator login password 'miniclass_migrator_dev_password';
        exception
            when duplicate_object then null;
        end;
    end if;
end
$$;
-- +goose StatementEnd

-- +goose StatementBegin
do $$
begin
    if not exists (select 1 from pg_roles where rolname = 'miniclass_app') then
        begin
            create role miniclass_app login password 'miniclass_app_dev_password';
        exception
            when duplicate_object then null;
        end;
    end if;
end
$$;
-- +goose StatementEnd

-- A pre-provisioned migrator can apply this migration without CREATEROLE. The
-- bootstrap administrator sets role attributes; a privileged migration run
-- repeats them so drift is corrected.
-- +goose StatementBegin
do $$
begin
    if (select rolsuper or rolcreaterole from pg_roles where rolname = current_user) then
        alter role miniclass_migrator
            nosuperuser nocreatedb nocreaterole inherit noreplication nobypassrls;
        alter role miniclass_app
            nosuperuser nocreatedb nocreaterole noinherit noreplication nobypassrls;
        alter role miniclass_app set statement_timeout = '10s';
    end if;
end
$$;
-- +goose StatementEnd

revoke create on schema public from public;
grant usage on schema public to miniclass_app;

-- The migrator owns the schema objects and is the only role that can create or
-- alter them. Existing objects are transferred so down migrations also work
-- when the command is run through the migrator URL.
alter schema public owner to miniclass_migrator;
alter table health_checks owner to miniclass_migrator;
alter sequence public.xid_serial owner to miniclass_migrator;
alter domain public.xid20 owner to miniclass_migrator;
alter function public.xid_encode(integer[]) owner to miniclass_migrator;
alter function public.xid(timestamptz) owner to miniclass_migrator;
alter function public.xid_decode(public.xid20) owner to miniclass_migrator;
alter function public.xid_time(public.xid20) owner to miniclass_migrator;
alter function public.xid_machine(public.xid20) owner to miniclass_migrator;
alter function public.xid_pid(public.xid20) owner to miniclass_migrator;
alter function public.xid_counter(public.xid20) owner to miniclass_migrator;

grant select, insert, update, delete on all tables in schema public to miniclass_app;
grant usage, select, update on all sequences in schema public to miniclass_app;
grant execute on function public.xid_encode(integer[]) to miniclass_app;
grant execute on function public.xid(timestamptz) to miniclass_app;
grant execute on function public.xid_decode(public.xid20) to miniclass_app;
grant execute on function public.xid_time(public.xid20) to miniclass_app;
grant execute on function public.xid_machine(public.xid20) to miniclass_app;
grant execute on function public.xid_pid(public.xid20) to miniclass_app;
grant execute on function public.xid_counter(public.xid20) to miniclass_app;

alter default privileges for role miniclass_migrator in schema public
    grant select, insert, update, delete on tables to miniclass_app;
alter default privileges for role miniclass_migrator in schema public
    grant usage, select, update on sequences to miniclass_app;

-- +goose Down

alter default privileges for role miniclass_migrator in schema public
    revoke select, insert, update, delete on tables from miniclass_app;
alter default privileges for role miniclass_migrator in schema public
    revoke usage, select, update on sequences from miniclass_app;
revoke all privileges on all tables in schema public from miniclass_app;
revoke all privileges on all sequences in schema public from miniclass_app;
revoke all privileges on all functions in schema public from miniclass_app;
revoke all privileges on schema public from miniclass_app;
drop owned by miniclass_app;

-- Roles are cluster-scoped and may be used by another database on the same
-- PostgreSQL cluster. A database rollback therefore removes this database's
-- grants and ownership changes but intentionally leaves the roles in place.
alter table health_checks owner to miniclass_migrator;
alter sequence public.xid_serial owner to miniclass_migrator;
alter domain public.xid20 owner to miniclass_migrator;
alter function public.xid_encode(integer[]) owner to miniclass_migrator;
alter function public.xid(timestamptz) owner to miniclass_migrator;
alter function public.xid_decode(public.xid20) owner to miniclass_migrator;
alter function public.xid_time(public.xid20) owner to miniclass_migrator;
alter function public.xid_machine(public.xid20) owner to miniclass_migrator;
alter function public.xid_pid(public.xid20) owner to miniclass_migrator;
alter function public.xid_counter(public.xid20) owner to miniclass_migrator;
alter schema public owner to miniclass_migrator;
