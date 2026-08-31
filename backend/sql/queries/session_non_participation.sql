-- name: CreateSessionNonParticipation :one
insert into session_non_participations (organization_id, school_year_id, program_id, session_id, student_id, reason)
values ($1, $2, $3, $4, $5, $6)
returning id, organization_id, school_year_id, program_id, session_id, student_id, reason, created_at, updated_at;

-- name: ListSessionNonParticipations :many
select id, organization_id, school_year_id, program_id, session_id, student_id, reason, created_at, updated_at
from session_non_participations
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4
order by student_id, id;

-- name: GetSessionNonParticipation :one
select id, organization_id, school_year_id, program_id, session_id, student_id, reason, created_at, updated_at
from session_non_participations
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4 and session_id = $5;

-- name: UpdateSessionNonParticipation :one
update session_non_participations
set reason = $2
where id = $1 and organization_id = $3 and school_year_id = $4 and program_id = $5 and session_id = $6
returning id, organization_id, school_year_id, program_id, session_id, student_id, reason, created_at, updated_at;

-- name: DeleteSessionNonParticipation :execrows
delete from session_non_participations
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4 and session_id = $5;

-- name: ListAllSessionNonParticipationsForRegistry :many
select id, organization_id, school_year_id, program_id, session_id, student_id, reason, created_at, updated_at
from session_non_participations
where organization_id = $1
order by school_year_id, program_id, session_id, student_id, id;

-- name: FindSessionNonParticipationForRegistry :one
select id, organization_id, school_year_id, program_id, session_id, student_id, reason, created_at, updated_at
from session_non_participations
where id = $1 and organization_id = $2;

-- name: UpdateSessionNonParticipationForRegistry :execrows
update session_non_participations
set reason = $3
where id = $1 and organization_id = $2;

-- name: DeleteSessionNonParticipationForRegistry :execrows
delete from session_non_participations
where id = $1 and organization_id = $2;
