-- +goose Up

-- Comments are scoped to a placement workspace session. Host identifiers are
-- deliberately retained without foreign keys: an assignment or offering can
-- be replaced during a later draft edit, but its comment must remain an
-- attributable record rather than moving to the replacement (SPEC §20.3).
create table placement_comments (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    session_id public.xid20 not null,
    host_type text not null,
    host_id public.xid20 not null,
    author_user_id public.xid20 not null references users (id) on delete restrict,
    body text not null,
    sensitivity text not null default 'internal',
    deleted_at timestamptz,
    deleted_by_user_id public.xid20 references users (id) on delete restrict,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint placement_comments_session_fk foreign key (session_id, organization_id, school_year_id, program_id)
        references sessions (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint placement_comments_id_organization_key unique (id, organization_id),
    constraint placement_comments_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint placement_comments_host_type_check check (host_type in ('assignment', 'offering', 'session')),
    constraint placement_comments_body_check check (btrim(body) <> ''),
    constraint placement_comments_sensitivity_check check (sensitivity in ('public', 'internal', 'sensitive')),
    constraint placement_comments_deletion_check check ((deleted_at is null) = (deleted_by_user_id is null))
);

create index placement_comments_session_host_idx
    on placement_comments (organization_id, school_year_id, program_id, session_id, host_type, host_id, created_at, id);

alter table placement_comments enable row level security;
alter table placement_comments force row level security;
create policy placement_comments_tenant_isolation on placement_comments
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger placement_comments_set_updated_at
before update on placement_comments
for each row execute function public.set_updated_at();
create trigger placement_comments_closed_year_guard
before insert or update or delete on placement_comments
for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update, delete on placement_comments to miniclass_app;

-- +goose Down

revoke all privileges on placement_comments from miniclass_app;
drop trigger if exists placement_comments_closed_year_guard on placement_comments;
drop trigger if exists placement_comments_set_updated_at on placement_comments;
drop policy if exists placement_comments_tenant_isolation on placement_comments;
drop index if exists placement_comments_session_host_idx;
drop table placement_comments;
