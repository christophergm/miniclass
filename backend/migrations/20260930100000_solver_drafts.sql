-- +goose Up

alter table solve_runs
    add column effective_weights_document jsonb not null default '{}'::jsonb,
    add column metrics_document jsonb not null default '{}'::jsonb;

alter table solve_runs
    add constraint solve_runs_effective_weights_document_check
        check (jsonb_typeof(effective_weights_document) = 'object'),
    add constraint solve_runs_metrics_document_check
        check (jsonb_typeof(metrics_document) = 'object');

alter table offerings
    add constraint offerings_id_scope_key
        unique (id, organization_id, school_year_id, program_id, session_id);

create table assignments (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    session_id public.xid20 not null,
    student_id public.xid20 not null,
    offering_id public.xid20 not null,
    solve_run_id public.xid20 not null,
    origin text not null,
    pinned boolean not null default false,
    realized_quality text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint assignments_session_fk foreign key (session_id, organization_id, school_year_id, program_id)
        references sessions (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint assignments_student_fk foreign key (student_id, organization_id, school_year_id)
        references students (id, organization_id, school_year_id) on delete cascade,
    constraint assignments_offering_fk foreign key (offering_id, organization_id, school_year_id, program_id, session_id)
        references offerings (id, organization_id, school_year_id, program_id, session_id) on delete restrict,
    constraint assignments_solve_run_fk foreign key (solve_run_id, organization_id, school_year_id, program_id, session_id)
        references solve_runs (id, organization_id, school_year_id, program_id, session_id) on delete restrict,
    constraint assignments_id_organization_key unique (id, organization_id),
    constraint assignments_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint assignments_student_session_unique unique (organization_id, school_year_id, program_id, session_id, student_id),
    constraint assignments_origin_check check (origin in ('solver', 'manual')),
    constraint assignments_realized_quality_check check (realized_quality in ('top', 'high', 'acceptable', 'neutral', 'unwanted'))
);

create index assignments_session_offering_idx
    on assignments (organization_id, school_year_id, program_id, session_id, offering_id, student_id);

alter table assignments enable row level security;
alter table assignments force row level security;
create policy assignments_tenant_isolation on assignments
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger assignments_set_updated_at
before update on assignments
for each row execute function public.set_updated_at();

create trigger assignments_closed_year_guard
before insert or update or delete on assignments
for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update, delete on assignments to miniclass_app;

-- +goose Down

revoke all privileges on assignments from miniclass_app;
drop trigger if exists assignments_closed_year_guard on assignments;
drop trigger if exists assignments_set_updated_at on assignments;
drop policy if exists assignments_tenant_isolation on assignments;
drop index if exists assignments_session_offering_idx;
drop table assignments;
alter table offerings drop constraint if exists offerings_id_scope_key;
alter table solve_runs drop constraint if exists solve_runs_metrics_document_check;
alter table solve_runs drop constraint if exists solve_runs_effective_weights_document_check;
alter table solve_runs drop column metrics_document;
alter table solve_runs drop column effective_weights_document;
