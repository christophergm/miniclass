-- +goose Up

create table solve_runs (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    session_id public.xid20 not null,
    rerun_of_solve_run_id public.xid20,
    contract_version text not null,
    seed bigint not null,
    input_fingerprint text not null,
    request_document jsonb not null,
    response_document jsonb not null,
    solver_status text not null,
    deterministic_duration double precision not null,
    created_at timestamptz not null default now(),
    constraint solve_runs_session_fk foreign key (session_id, organization_id, school_year_id, program_id)
        references sessions (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint solve_runs_rerun_fk foreign key (rerun_of_solve_run_id, organization_id, school_year_id, program_id, session_id)
        references solve_runs (id, organization_id, school_year_id, program_id, session_id),
    constraint solve_runs_id_organization_key unique (id, organization_id),
    constraint solve_runs_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint solve_runs_id_scope_key unique (id, organization_id, school_year_id, program_id, session_id),
    constraint solve_runs_contract_version_check check (btrim(contract_version) <> ''),
    constraint solve_runs_input_fingerprint_check check (input_fingerprint ~ '^[0-9a-f]{64}$'),
    constraint solve_runs_solver_status_check check (solver_status in ('optimal', 'feasible', 'infeasible', 'unknown', 'model_invalid')),
    constraint solve_runs_duration_check check (deterministic_duration >= 0)
);

create index solve_runs_session_created_idx on solve_runs (organization_id, school_year_id, program_id, session_id, created_at desc, id desc);

alter table solve_runs enable row level security;
alter table solve_runs force row level security;
create policy solve_runs_tenant_isolation on solve_runs
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger solve_runs_closed_year_guard before insert or update or delete on solve_runs for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert on solve_runs to miniclass_app;

-- +goose Down

revoke all privileges on solve_runs from miniclass_app;
drop trigger if exists solve_runs_closed_year_guard on solve_runs;
drop policy if exists solve_runs_tenant_isolation on solve_runs;
drop index if exists solve_runs_session_created_idx;
drop table solve_runs;
