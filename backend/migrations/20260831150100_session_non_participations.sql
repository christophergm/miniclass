-- +goose Up

create table session_non_participations (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    session_id public.xid20 not null,
    student_id public.xid20 not null,
    reason text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint session_non_participations_session_fk foreign key (session_id, organization_id, school_year_id, program_id)
        references sessions (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint session_non_participations_student_fk foreign key (student_id, organization_id, school_year_id)
        references students (id, organization_id, school_year_id) on delete cascade,
    constraint session_non_participations_id_organization_key unique (id, organization_id),
    constraint session_non_participations_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint session_non_participations_reason_check check (btrim(reason) <> ''),
    constraint session_non_participations_student_unique unique (organization_id, school_year_id, program_id, session_id, student_id)
);

create index session_non_participations_session_idx
    on session_non_participations (organization_id, school_year_id, program_id, session_id, student_id);

alter table session_non_participations enable row level security;
alter table session_non_participations force row level security;
create policy session_non_participations_tenant_isolation on session_non_participations
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger session_non_participations_set_updated_at
before update on session_non_participations
for each row execute function public.set_updated_at();

create trigger session_non_participations_closed_year_guard
before insert or update or delete on session_non_participations
for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update, delete on session_non_participations to miniclass_app;

-- +goose Down

revoke all privileges on session_non_participations from miniclass_app;
drop trigger if exists session_non_participations_closed_year_guard on session_non_participations;
drop trigger if exists session_non_participations_set_updated_at on session_non_participations;
drop policy if exists session_non_participations_tenant_isolation on session_non_participations;
drop index if exists session_non_participations_session_idx;
drop table session_non_participations;
