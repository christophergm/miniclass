-- name: UpsertAssignment :one
insert into assignments (
    organization_id, school_year_id, program_id, session_id, student_id,
    offering_id, solve_run_id, origin, pinned, realized_quality
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
on conflict (organization_id, school_year_id, program_id, session_id, student_id)
do update set offering_id = excluded.offering_id, solve_run_id = excluded.solve_run_id,
    origin = excluded.origin, pinned = excluded.pinned, realized_quality = excluded.realized_quality
returning id, organization_id, school_year_id, program_id, session_id, student_id,
    offering_id, solve_run_id, origin, pinned, realized_quality, created_at, updated_at;

-- name: DeleteReplacedDraftAssignments :execrows
delete from assignments
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4
  and not (student_id = any(sqlc.arg('student_ids')::text[]::public.xid20[]));

-- name: DeleteChangedDraftAssignment :execrows
delete from assignments
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4
  and student_id = $5 and offering_id <> $6;

-- name: AdvanceDraftRevision :one
update sessions
set draft_revision = draft_revision + 1
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4
  and draft_revision = $5
returning draft_revision;

-- name: UpdateAssignmentPin :one
update assignments
set pinned = $1
where id = $2 and organization_id = $3 and school_year_id = $4
  and program_id = $5 and session_id = $6
returning id, organization_id, school_year_id, program_id, session_id, student_id,
    offering_id, solve_run_id, origin, pinned, realized_quality, created_at, updated_at;

-- name: DeleteAssignmentOverrides :execrows
delete from assignment_overrides
where organization_id = $1 and school_year_id = $2 and program_id = $3
  and session_id = $4 and assignment_id = $5;

-- name: DeleteAssignmentOverrideRule :execrows
delete from assignment_overrides
where organization_id = $1 and school_year_id = $2 and program_id = $3
  and session_id = $4 and assignment_id = $5 and rule = $6;

-- name: CreateAssignmentExclusion :one
insert into assignment_exclusions (organization_id, school_year_id, program_id, session_id, student_id, offering_id)
values ($1, $2, $3, $4, $5, $6)
returning id, organization_id, school_year_id, program_id, session_id, student_id, offering_id, created_at, updated_at;

-- name: DeleteAssignmentExclusion :execrows
delete from assignment_exclusions
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4 and session_id = $5;

-- name: CreateAssignmentOverride :one
insert into assignment_overrides (
    organization_id, school_year_id, program_id, session_id, assignment_id, rule, reason, recorded_by
)
values ($1, $2, $3, $4, $5, $6, $7, $8)
returning id, organization_id, school_year_id, program_id, session_id, assignment_id, rule, reason, recorded_by, created_at, updated_at;

-- name: ListAssignmentExclusions :many
select id, organization_id, school_year_id, program_id, session_id, student_id, offering_id, created_at, updated_at
from assignment_exclusions
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4
order by student_id, offering_id;

-- name: ListAssignmentOverrides :many
select id, organization_id, school_year_id, program_id, session_id, assignment_id, rule, reason, recorded_by, created_at, updated_at
from assignment_overrides
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4
order by assignment_id, id;

-- name: ListAllAssignmentExclusionsForRegistry :many
select id, organization_id, school_year_id, program_id, session_id, student_id, offering_id, created_at, updated_at
from assignment_exclusions where organization_id = $1 order by id;

-- name: FindAssignmentExclusionForRegistry :one
select id, organization_id, school_year_id, program_id, session_id, student_id, offering_id, created_at, updated_at
from assignment_exclusions where id = $1 and organization_id = $2;

-- name: TouchAssignmentExclusionForRegistry :execrows
update assignment_exclusions set student_id = student_id where id = $1 and organization_id = $2;

-- name: DeleteAssignmentExclusionForRegistry :execrows
delete from assignment_exclusions where id = $1 and organization_id = $2;

-- name: ListAllAssignmentOverridesForRegistry :many
select id, organization_id, school_year_id, program_id, session_id, assignment_id, rule, reason, recorded_by, created_at, updated_at
from assignment_overrides where organization_id = $1 order by id;

-- name: FindAssignmentOverrideForRegistry :one
select id, organization_id, school_year_id, program_id, session_id, assignment_id, rule, reason, recorded_by, created_at, updated_at
from assignment_overrides where id = $1 and organization_id = $2;

-- name: TouchAssignmentOverrideForRegistry :execrows
update assignment_overrides set reason = reason where id = $1 and organization_id = $2;

-- name: DeleteAssignmentOverrideForRegistry :execrows
delete from assignment_overrides where id = $1 and organization_id = $2;
