-- +goose Up

create type session_state as enum (
    'planning',
    'catalog_published',
    'voting_open',
    'voting_closed',
    'assigning',
    'published',
    'complete'
);

create table sessions (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    name text not null,
    ordinal integer not null,
    state session_state not null default 'planning',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint sessions_program_fk foreign key (program_id, organization_id, school_year_id)
        references programs (id, organization_id, school_year_id) on delete cascade,
    constraint sessions_id_organization_key unique (id, organization_id),
    constraint sessions_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint sessions_id_organization_year_program_key unique (id, organization_id, school_year_id, program_id),
    constraint sessions_name_check check (btrim(name) <> ''),
    constraint sessions_ordinal_check check (ordinal > 0),
    constraint sessions_program_ordinal_unique unique (organization_id, school_year_id, program_id, ordinal)
);

create table meeting_dates (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    session_id public.xid20 not null,
    meeting_date date not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint meeting_dates_session_fk foreign key (session_id, organization_id, school_year_id, program_id)
        references sessions (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint meeting_dates_id_organization_key unique (id, organization_id),
    constraint meeting_dates_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint meeting_dates_session_date_unique unique (organization_id, school_year_id, program_id, session_id, meeting_date)
);

create index sessions_program_idx on sessions (organization_id, school_year_id, program_id, ordinal, id);
create index meeting_dates_session_idx on meeting_dates (organization_id, school_year_id, program_id, session_id, meeting_date, id);

-- A session and its dates are created in one transaction by the application.
-- Deferring this check until commit preserves that atomic write while keeping
-- the one-or-more meeting-date invariant true for direct database writers.
-- +goose StatementBegin
create function public.require_session_meeting_date() returns trigger language plpgsql as
$$
begin
    if exists (select 1 from sessions where id = new.id)
       and not exists (select 1 from meeting_dates where session_id = new.id and organization_id = new.organization_id and school_year_id = new.school_year_id and program_id = new.program_id) then
        raise exception 'session must have at least one meeting date'
            using errcode = '23514',
                  detail = 'a session cannot be committed without a meeting date';
    end if;
    return new;
end;
$$;
-- +goose StatementEnd

create constraint trigger sessions_requires_meeting_date
after insert on sessions
deferrable initially deferred
for each row execute function public.require_session_meeting_date();

alter table sessions enable row level security;
alter table sessions force row level security;
create policy sessions_tenant_isolation on sessions
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);
alter table meeting_dates enable row level security;
alter table meeting_dates force row level security;
create policy meeting_dates_tenant_isolation on meeting_dates
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger sessions_set_updated_at before update on sessions for each row execute function public.set_updated_at();
create trigger meeting_dates_set_updated_at before update on meeting_dates for each row execute function public.set_updated_at();
create trigger sessions_closed_year_guard before insert or update or delete on sessions for each row execute function public.prevent_closed_school_year_mutation();
create trigger meeting_dates_closed_year_guard before insert or update or delete on meeting_dates for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update, delete on sessions, meeting_dates to miniclass_app;

-- +goose Down

revoke all privileges on sessions, meeting_dates from miniclass_app;
drop trigger if exists sessions_requires_meeting_date on sessions;
drop trigger if exists meeting_dates_closed_year_guard on meeting_dates;
drop trigger if exists sessions_closed_year_guard on sessions;
drop trigger if exists meeting_dates_set_updated_at on meeting_dates;
drop trigger if exists sessions_set_updated_at on sessions;
drop policy if exists meeting_dates_tenant_isolation on meeting_dates;
drop policy if exists sessions_tenant_isolation on sessions;
drop index if exists meeting_dates_session_idx;
drop index if exists sessions_program_idx;
drop table meeting_dates;
drop table sessions;
drop function if exists public.require_session_meeting_date();
drop type session_state;
