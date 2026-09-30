-- name: CreateSolveRun :one
insert into solve_runs (
    organization_id, school_year_id, program_id, session_id, rerun_of_solve_run_id,
    contract_version, seed, input_fingerprint, request_document, response_document,
    solver_status, deterministic_duration
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
returning id, organization_id, school_year_id, program_id, session_id, rerun_of_solve_run_id,
    contract_version, seed, input_fingerprint, request_document, response_document,
    solver_status, deterministic_duration, created_at;

-- name: GetSolveRun :one
select id, organization_id, school_year_id, program_id, session_id, rerun_of_solve_run_id,
    contract_version, seed, input_fingerprint, request_document, response_document,
    solver_status, deterministic_duration, created_at
from solve_runs
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4 and session_id = $5;

-- name: ListSolveRuns :many
select id, organization_id, school_year_id, program_id, session_id, rerun_of_solve_run_id,
    contract_version, seed, input_fingerprint, request_document, response_document,
    solver_status, deterministic_duration, created_at
from solve_runs
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4
order by created_at desc, id desc;

-- name: ListAllSolveRunsForRegistry :many
select id, organization_id, school_year_id, program_id, session_id, rerun_of_solve_run_id,
    contract_version, seed, input_fingerprint, request_document, response_document,
    solver_status, deterministic_duration, created_at
from solve_runs
where organization_id = $1
order by id;

-- name: FindSolveRunForRegistry :one
select id, organization_id, school_year_id, program_id, session_id, rerun_of_solve_run_id,
    contract_version, seed, input_fingerprint, request_document, response_document,
    solver_status, deterministic_duration, created_at
from solve_runs
where id = $1 and organization_id = $2;
