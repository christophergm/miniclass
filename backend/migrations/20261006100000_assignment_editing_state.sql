-- +goose Up

-- A manual placement has no solver run. The optional reference preserves the
-- immutable run provenance for solver-produced placements without inventing it
-- for human decisions (SPEC §§8.6, 20.1–20.2).
alter table assignments alter column solve_run_id drop not null;

alter table sessions add column draft_revision bigint not null default 0;

create table assignment_exclusions (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    session_id public.xid20 not null,
    student_id public.xid20 not null,
    offering_id public.xid20 not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint assignment_exclusions_session_fk foreign key (session_id, organization_id, school_year_id, program_id)
        references sessions (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint assignment_exclusions_student_fk foreign key (student_id, organization_id, school_year_id)
        references students (id, organization_id, school_year_id) on delete cascade,
    constraint assignment_exclusions_offering_fk foreign key (offering_id, organization_id, school_year_id, program_id, session_id)
        references offerings (id, organization_id, school_year_id, program_id, session_id) on delete cascade,
    constraint assignment_exclusions_id_organization_key unique (id, organization_id),
    constraint assignment_exclusions_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint assignment_exclusions_student_offering_unique unique (organization_id, school_year_id, program_id, session_id, student_id, offering_id)
);

create table assignment_overrides (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    session_id public.xid20 not null,
    assignment_id public.xid20 not null,
    rule text not null,
    reason text not null,
    recorded_by text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint assignment_overrides_assignment_fk foreign key (assignment_id, organization_id, school_year_id)
        references assignments (id, organization_id, school_year_id) on delete cascade,
    constraint assignment_overrides_id_organization_key unique (id, organization_id),
    constraint assignment_overrides_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint assignment_overrides_rule_check check (btrim(rule) <> ''),
    constraint assignment_overrides_reason_check check (btrim(reason) <> '')
    , constraint assignment_overrides_recorded_by_check check (btrim(recorded_by) <> '')
);

create index assignment_exclusions_session_student_idx
    on assignment_exclusions (organization_id, school_year_id, program_id, session_id, student_id);
create index assignment_overrides_assignment_idx
    on assignment_overrides (organization_id, school_year_id, assignment_id);

alter table assignment_exclusions enable row level security;
alter table assignment_exclusions force row level security;
create policy assignment_exclusions_tenant_isolation on assignment_exclusions
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

alter table assignment_overrides enable row level security;
alter table assignment_overrides force row level security;
create policy assignment_overrides_tenant_isolation on assignment_overrides
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger assignment_exclusions_set_updated_at
before update on assignment_exclusions
for each row execute function public.set_updated_at();
create trigger assignment_overrides_set_updated_at
before update on assignment_overrides
for each row execute function public.set_updated_at();
create trigger assignment_exclusions_closed_year_guard
before insert or update or delete on assignment_exclusions
for each row execute function public.prevent_closed_school_year_mutation();
create trigger assignment_overrides_closed_year_guard
before insert or update or delete on assignment_overrides
for each row execute function public.prevent_closed_school_year_mutation();

-- §16.2 is a data invariant: a non-participant cannot gain a placement, and
-- an assigned student cannot subsequently become a non-participant.
-- +goose StatementBegin
create function prevent_assignment_for_non_participant() returns trigger language plpgsql as $$
begin
    if exists (
        select 1 from session_non_participations
        where organization_id = new.organization_id and school_year_id = new.school_year_id
          and program_id = new.program_id and session_id = new.session_id and student_id = new.student_id
    ) then
        raise exception 'a non-participating student cannot have an assignment' using errcode = 'check_violation';
    end if;
    return new;
end;
$$;
-- +goose StatementEnd
create trigger assignments_non_participant_guard
before insert or update of student_id, session_id on assignments
for each row execute function prevent_assignment_for_non_participant();

-- +goose StatementBegin
create function prevent_non_participation_for_assigned_student() returns trigger language plpgsql as $$
begin
    if exists (
        select 1 from assignments
        where organization_id = new.organization_id and school_year_id = new.school_year_id
          and program_id = new.program_id and session_id = new.session_id and student_id = new.student_id
    ) then
        raise exception 'an assigned student cannot become a non-participant' using errcode = 'check_violation';
    end if;
    return new;
end;
$$;
-- +goose StatementEnd
create trigger session_non_participations_assignment_guard
before insert or update of student_id, session_id on session_non_participations
for each row execute function prevent_non_participation_for_assigned_student();

grant select, insert, update, delete on assignment_exclusions to miniclass_app;
grant select, insert, update, delete on assignment_overrides to miniclass_app;

-- +goose Down

revoke all privileges on assignment_overrides from miniclass_app;
revoke all privileges on assignment_exclusions from miniclass_app;
drop trigger if exists session_non_participations_assignment_guard on session_non_participations;
drop function if exists prevent_non_participation_for_assigned_student();
drop trigger if exists assignments_non_participant_guard on assignments;
drop function if exists prevent_assignment_for_non_participant();
drop trigger if exists assignment_overrides_closed_year_guard on assignment_overrides;
drop trigger if exists assignment_exclusions_closed_year_guard on assignment_exclusions;
drop trigger if exists assignment_overrides_set_updated_at on assignment_overrides;
drop trigger if exists assignment_exclusions_set_updated_at on assignment_exclusions;
drop policy if exists assignment_overrides_tenant_isolation on assignment_overrides;
drop policy if exists assignment_exclusions_tenant_isolation on assignment_exclusions;
drop index if exists assignment_overrides_assignment_idx;
drop index if exists assignment_exclusions_session_student_idx;
drop table assignment_overrides;
drop table assignment_exclusions;
alter table sessions drop column draft_revision;
alter table assignments alter column solve_run_id set not null;
