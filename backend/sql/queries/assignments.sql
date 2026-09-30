-- name: CreateAssignment :one
insert into assignments (
    organization_id, school_year_id, program_id, session_id, student_id,
    offering_id, solve_run_id, origin, pinned, realized_quality
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
returning id, organization_id, school_year_id, program_id, session_id, student_id,
    offering_id, solve_run_id, origin, pinned, realized_quality, created_at, updated_at;

-- name: DeleteDraftAssignments :execrows
delete from assignments
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4;

-- name: ListAssignments :many
select id, organization_id, school_year_id, program_id, session_id, student_id,
    offering_id, solve_run_id, origin, pinned, realized_quality, created_at, updated_at
from assignments
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4
order by student_id;

-- name: ListAllAssignmentsForRegistry :many
select id, organization_id, school_year_id, program_id, session_id, student_id,
    offering_id, solve_run_id, origin, pinned, realized_quality, created_at, updated_at
from assignments
where organization_id = $1
order by id;

-- name: FindAssignmentForRegistry :one
select id, organization_id, school_year_id, program_id, session_id, student_id,
    offering_id, solve_run_id, origin, pinned, realized_quality, created_at, updated_at
from assignments
where id = $1 and organization_id = $2;

-- name: TouchAssignmentForRegistry :execrows
update assignments
set pinned = pinned
where id = $1 and organization_id = $2;

-- name: DeleteAssignmentForRegistry :execrows
delete from assignments
where id = $1 and organization_id = $2;
