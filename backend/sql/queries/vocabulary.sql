-- name: CreateGradeLevel :one
insert into grade_levels (organization_id, school_year_id, code, label, ordinal)
values ($1, $2, $3, $4, $5)
returning id, organization_id, school_year_id, code, label, ordinal, retired_at, created_at, updated_at;

-- name: ListGradeLevels :many
select id, organization_id, school_year_id, code, label, ordinal, retired_at, created_at, updated_at
from grade_levels
where organization_id = $1 and school_year_id = $2 and retired_at is null
order by ordinal, id;

-- name: ListAllGradeLevels :many
select id, organization_id, school_year_id, code, label, ordinal, retired_at, created_at, updated_at
from grade_levels
where organization_id = $1 and school_year_id = $2
order by ordinal, id;

-- name: GetGradeLevelByID :one
select id, organization_id, school_year_id, code, label, ordinal, retired_at, created_at, updated_at
from grade_levels
where id = $1 and organization_id = $2 and school_year_id = $3;

-- name: UpdateGradeLevel :one
update grade_levels
set code = $2, label = $3
where id = $1 and organization_id = $4 and school_year_id = $5
returning id, organization_id, school_year_id, code, label, ordinal, retired_at, created_at, updated_at;

-- name: SetGradeLevelRetired :one
update grade_levels
set retired_at = case when $2::boolean then coalesce(retired_at, now()) else null end
where id = $1 and organization_id = $3 and school_year_id = $4
returning id, organization_id, school_year_id, code, label, ordinal, retired_at, created_at, updated_at;

-- name: ShiftGradeLevelOrdinals :exec
update grade_levels
set ordinal = ordinal + $2
where organization_id = $1 and school_year_id = $3;

-- name: UpdateGradeLevelOrdinal :exec
update grade_levels
set ordinal = $2
where id = $1 and organization_id = $3 and school_year_id = $4;

-- name: CreateHomeroom :one
insert into homerooms (organization_id, school_year_id, name, external_identifier)
values ($1, $2, $3, $4)
returning id, organization_id, school_year_id, name, external_identifier, retired_at, created_at, updated_at;

-- name: ListHomerooms :many
select id, organization_id, school_year_id, name, external_identifier, retired_at, created_at, updated_at
from homerooms
where organization_id = $1 and school_year_id = $2 and retired_at is null
order by lower(name), id;

-- name: ListAllHomerooms :many
select id, organization_id, school_year_id, name, external_identifier, retired_at, created_at, updated_at
from homerooms
where organization_id = $1 and school_year_id = $2
order by lower(name), id;

-- name: GetHomeroomByID :one
select id, organization_id, school_year_id, name, external_identifier, retired_at, created_at, updated_at
from homerooms
where id = $1 and organization_id = $2 and school_year_id = $3;

-- name: UpdateHomeroom :one
update homerooms
set name = $2, external_identifier = $3
where id = $1 and organization_id = $4 and school_year_id = $5
returning id, organization_id, school_year_id, name, external_identifier, retired_at, created_at, updated_at;

-- name: SetHomeroomRetired :one
update homerooms
set retired_at = case when $2::boolean then coalesce(retired_at, now()) else null end
where id = $1 and organization_id = $3 and school_year_id = $4
returning id, organization_id, school_year_id, name, external_identifier, retired_at, created_at, updated_at;

-- name: GetOrganizationVocabularySettings :one
select id, name, homeroom_label, created_at, updated_at
from organizations
where id = $1;

-- name: FindGradeLevelForRegistry :one
select id, organization_id, school_year_id, code, label, ordinal, retired_at, created_at, updated_at
from grade_levels
where id = $1 and organization_id = $2;

-- name: FindHomeroomForRegistry :one
select id, organization_id, school_year_id, name, external_identifier, retired_at, created_at, updated_at
from homerooms
where id = $1 and organization_id = $2;

-- name: ListAllGradeLevelsForRegistry :many
select id, organization_id, school_year_id, code, label, ordinal, retired_at, created_at, updated_at
from grade_levels
where organization_id = $1
order by school_year_id, ordinal, id;

-- name: ListAllHomeroomsForRegistry :many
select id, organization_id, school_year_id, name, external_identifier, retired_at, created_at, updated_at
from homerooms
where organization_id = $1
order by school_year_id, lower(name), id;

-- name: UpdateOrganizationHomeroomLabel :one
update organizations
set homeroom_label = $2
where id = $1
returning id, name, homeroom_label, created_at, updated_at;
