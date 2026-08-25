-- name: CreateSchoolYear :one
insert into school_years (organization_id, label)
values ($1, $2)
returning id, organization_id, label, state, created_at, updated_at;

-- name: ListSchoolYears :many
select id, organization_id, label, state, created_at, updated_at
from school_years
order by label, id;

-- name: GetSchoolYearByID :one
select id, organization_id, label, state, created_at, updated_at
from school_years
where id = $1;

-- name: UpdateSchoolYearLabel :one
update school_years
set label = $2
where id = $1
returning id, organization_id, label, state, created_at, updated_at;

-- name: UpdateSchoolYearState :one
update school_years
set state = $2
where id = $1
returning id, organization_id, label, state, created_at, updated_at;

-- name: DeleteSchoolYear :execrows
delete from school_years
where id = $1;
