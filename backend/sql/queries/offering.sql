-- name: CreateOffering :one
insert into offerings (
    organization_id, school_year_id, program_id, session_id, name, description,
    minimum_viable_enrollment, capacity, min_grade_level_id, max_grade_level_id,
    location, meeting_point, meeting_instructions, interest_area_id
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
returning id, organization_id, school_year_id, program_id, session_id, name, description,
    minimum_viable_enrollment, capacity, min_grade_level_id, max_grade_level_id,
    location, meeting_point, meeting_instructions, interest_area_id, created_at, updated_at;

-- name: ListOfferings :many
select id, organization_id, school_year_id, program_id, session_id, name, description,
    minimum_viable_enrollment, capacity, min_grade_level_id, max_grade_level_id,
    location, meeting_point, meeting_instructions, interest_area_id, created_at, updated_at
from offerings
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4
order by name, id;

-- name: CountOfferings :one
select count(*)
from offerings
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4;

-- name: GetOffering :one
select id, organization_id, school_year_id, program_id, session_id, name, description,
    minimum_viable_enrollment, capacity, min_grade_level_id, max_grade_level_id,
    location, meeting_point, meeting_instructions, interest_area_id, created_at, updated_at
from offerings
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4 and session_id = $5;

-- name: UpdateOffering :one
update offerings
set name = $2, description = $3, minimum_viable_enrollment = $4, capacity = $5,
    min_grade_level_id = $6, max_grade_level_id = $7, location = $8,
    meeting_point = $9, meeting_instructions = $10, interest_area_id = $11
where id = $1 and organization_id = $12 and school_year_id = $13 and program_id = $14 and session_id = $15
returning id, organization_id, school_year_id, program_id, session_id, name, description,
    minimum_viable_enrollment, capacity, min_grade_level_id, max_grade_level_id,
    location, meeting_point, meeting_instructions, interest_area_id, created_at, updated_at;

-- name: DeleteOffering :execrows
delete from offerings
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4 and session_id = $5;

-- name: ListAllOfferingsForRegistry :many
select id, organization_id, school_year_id, program_id, session_id, name, description,
    minimum_viable_enrollment, capacity, min_grade_level_id, max_grade_level_id,
    location, meeting_point, meeting_instructions, interest_area_id, created_at, updated_at
from offerings
where organization_id = $1 order by school_year_id, program_id, session_id, name, id;

-- name: FindOfferingForRegistry :one
select id, organization_id, school_year_id, program_id, session_id, name, description,
    minimum_viable_enrollment, capacity, min_grade_level_id, max_grade_level_id,
    location, meeting_point, meeting_instructions, interest_area_id, created_at, updated_at
from offerings where id = $1 and organization_id = $2;

-- name: TouchOfferingForRegistry :execrows
update offerings set updated_at = now() where id = $1 and organization_id = $2;

-- name: DeleteOfferingForRegistry :execrows
delete from offerings where id = $1 and organization_id = $2;
