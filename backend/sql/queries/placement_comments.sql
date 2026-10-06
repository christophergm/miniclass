-- name: CreatePlacementComment :one
insert into placement_comments (
    organization_id, school_year_id, program_id, session_id, host_type, host_id,
    author_user_id, body, sensitivity
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
returning id, organization_id, school_year_id, program_id, session_id, host_type, host_id,
    author_user_id, body, sensitivity, deleted_at, deleted_by_user_id, created_at, updated_at;

-- name: GetPlacementComment :one
select id, organization_id, school_year_id, program_id, session_id, host_type, host_id,
    author_user_id, body, sensitivity, deleted_at, deleted_by_user_id, created_at, updated_at
from placement_comments
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4 and session_id = $5;

-- name: ListPlacementComments :many
select id, organization_id, school_year_id, program_id, session_id, host_type, host_id,
    author_user_id, body, sensitivity, deleted_at, deleted_by_user_id, created_at, updated_at
from placement_comments
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4
  and deleted_at is null
order by created_at, id;

-- name: UpdatePlacementComment :one
update placement_comments set body = $1, sensitivity = $2
where id = $3 and organization_id = $4 and school_year_id = $5 and program_id = $6 and session_id = $7
  and author_user_id = $8 and deleted_at is null
returning id, organization_id, school_year_id, program_id, session_id, host_type, host_id,
    author_user_id, body, sensitivity, deleted_at, deleted_by_user_id, created_at, updated_at;

-- name: SoftDeletePlacementComment :one
update placement_comments set deleted_at = now(), deleted_by_user_id = $1
where id = $2 and organization_id = $3 and school_year_id = $4 and program_id = $5 and session_id = $6
  and author_user_id = $1 and deleted_at is null
returning id, organization_id, school_year_id, program_id, session_id, host_type, host_id,
    author_user_id, body, sensitivity, deleted_at, deleted_by_user_id, created_at, updated_at;

-- name: ListAllPlacementCommentsForRegistry :many
select id, organization_id, school_year_id, program_id, session_id, host_type, host_id,
    author_user_id, body, sensitivity, deleted_at, deleted_by_user_id, created_at, updated_at
from placement_comments where organization_id = $1 order by id;

-- name: FindPlacementCommentForRegistry :one
select id, organization_id, school_year_id, program_id, session_id, host_type, host_id,
    author_user_id, body, sensitivity, deleted_at, deleted_by_user_id, created_at, updated_at
from placement_comments where id = $1 and organization_id = $2;

-- name: TouchPlacementCommentForRegistry :execrows
update placement_comments set body = body where id = $1 and organization_id = $2;

-- name: DeletePlacementCommentForRegistry :execrows
delete from placement_comments where id = $1 and organization_id = $2;
