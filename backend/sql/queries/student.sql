-- name: CreateStudent :one
insert into students (
    organization_id,
    school_year_id,
    legal_given_name,
    legal_family_name,
    preferred_given_name,
    grade_level_id,
    homeroom_id,
    external_identifier,
    prior_year_student_id
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
returning id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier,
    prior_year_student_id, deleted_at, created_at, updated_at;

-- name: ListStudents :many
select id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier,
    prior_year_student_id, deleted_at, created_at, updated_at
from students
where organization_id = $1
  and school_year_id = $2
  and ($3::bool or deleted_at is null)
order by legal_family_name, coalesce(preferred_given_name, legal_given_name), legal_given_name, id;

-- name: GetStudentByID :one
select id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier,
    prior_year_student_id, deleted_at, created_at, updated_at
from students
where id = $1
  and organization_id = $2
  and school_year_id = $3
  and deleted_at is null;

-- name: GetStudentByIDIncludingDeleted :one
select id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier,
    prior_year_student_id, deleted_at, created_at, updated_at
from students
where id = $1
  and organization_id = $2
  and school_year_id = $3;

-- name: UpdateStudent :one
update students
set legal_given_name = $4,
    legal_family_name = $5,
    preferred_given_name = $6,
    grade_level_id = $7,
    homeroom_id = $8,
    external_identifier = $9,
    prior_year_student_id = $10
where id = $1
  and organization_id = $2
  and school_year_id = $3
  and deleted_at is null
returning id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier,
    prior_year_student_id, deleted_at, created_at, updated_at;

-- name: SoftDeleteStudent :execrows
update students
set deleted_at = coalesce(deleted_at, now())
where id = $1
  and organization_id = $2
  and school_year_id = $3
  and deleted_at is null;

-- name: RestoreStudent :one
update students
set deleted_at = null
where id = $1
  and organization_id = $2
  and school_year_id = $3
  and deleted_at is not null
returning id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier,
    prior_year_student_id, deleted_at, created_at, updated_at;

-- name: ListAllActiveStudentsForRegistry :many
select id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier,
    prior_year_student_id, deleted_at, created_at, updated_at
from students
where organization_id = $1
  and deleted_at is null
order by id;

-- name: FindStudentForRegistry :one
select id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier,
    prior_year_student_id, deleted_at, created_at, updated_at
from students
where id = $1
  and organization_id = $2
  and deleted_at is null;
