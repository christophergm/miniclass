-- +goose Up

create type school_year_state as enum (
    'setup',
    'active',
    'closed'
);

create table school_years (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    label text not null,
    state school_year_state not null default 'setup',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint school_years_label_check check (btrim(label) <> ''),
    constraint school_years_id_organization_key unique (id, organization_id)
);

create index school_years_organization_idx on school_years (organization_id, label, id);

alter table school_years enable row level security;
alter table school_years force row level security;

create policy school_years_tenant_isolation on school_years
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger school_years_set_updated_at
before update on school_years
for each row execute function public.set_updated_at();

-- This trigger is shared by every table whose rows belong to a school year.
-- The app transaction may prepare a reasoned Owner-only reopen by setting the
-- three local settings; the target ID keeps the exception row-scoped. Absent
-- that preparation, a closed year is immutable.
-- +goose StatementBegin
create or replace function public.prevent_closed_school_year_mutation() returns trigger language plpgsql as
$$
declare
    old_school_year_id public.xid20;
    new_school_year_id public.xid20;
    parent_state school_year_state;
    reopen_id text := nullif(btrim(coalesce(current_setting('app.school_year_reopen_id', true), '')), '');
    reopen_prepared boolean := coalesce(current_setting('app.school_year_reopen', true), '') = 'true'
        and nullif(btrim(coalesce(current_setting('app.school_year_reopen_reason', true), '')), '') is not null;
begin
    if tg_table_name = 'school_years' then
        if tg_op = 'insert' then
            return new;
        end if;
        if tg_op = 'delete' then
            if old.state = 'closed' then
                raise exception 'school year is closed'
                    using errcode = 'P0001',
                          detail = 'records in a closed school year are read-only',
                          hint = 'reopen the school year through the Owner-only lifecycle operation';
            end if;
            return old;
        end if;

        if old.state = 'closed' then
            if new.state = 'active' and reopen_prepared and reopen_id = old.id::text then
                return new;
            end if;
            raise exception 'school year is closed'
                using errcode = 'P0001',
                      detail = 'records in a closed school year are read-only',
                      hint = 'reopen the school year through the Owner-only lifecycle operation';
        end if;
        return new;
    end if;

    if tg_op <> 'insert' then
        old_school_year_id := nullif(to_jsonb(old)->>'school_year_id', '')::public.xid20;
        if old_school_year_id is not null then
            select state into parent_state from school_years where id = old_school_year_id;
            if parent_state = 'closed' and not (reopen_prepared and reopen_id = old_school_year_id::text) then
                raise exception 'school year is closed'
                    using errcode = 'P0001',
                          detail = 'records in a closed school year are read-only',
                          hint = 'reopen the school year through the Owner-only lifecycle operation';
            end if;
        end if;
    end if;

    if tg_op <> 'delete' then
        new_school_year_id := nullif(to_jsonb(new)->>'school_year_id', '')::public.xid20;
        if new_school_year_id is not null then
            select state into parent_state from school_years where id = new_school_year_id;
            if parent_state = 'closed' and not (reopen_prepared and reopen_id = new_school_year_id::text) then
                raise exception 'school year is closed'
                    using errcode = 'P0001',
                          detail = 'records in a closed school year are read-only',
                          hint = 'reopen the school year through the Owner-only lifecycle operation';
            end if;
        end if;
    end if;

    if tg_op = 'delete' then
        return old;
    end if;
    return new;
end;
$$;
-- +goose StatementEnd

create trigger school_years_closed_year_guard
before insert or update or delete on school_years
for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update, delete on school_years to miniclass_app;

-- +goose Down

revoke all privileges on school_years from miniclass_app;
drop trigger if exists school_years_closed_year_guard on school_years;
drop trigger if exists school_years_set_updated_at on school_years;
drop table school_years;
drop function if exists public.prevent_closed_school_year_mutation();
drop type school_year_state;
