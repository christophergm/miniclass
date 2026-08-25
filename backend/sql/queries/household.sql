-- name: CreateHousehold :one
insert into households (organization_id, school_year_id, display_name)
values ($1, $2, $3)
returning id, organization_id, school_year_id, display_name, deleted_at, created_at, updated_at;

-- name: ListHouseholds :many
select id, organization_id, school_year_id, display_name, deleted_at, created_at, updated_at
from households
where organization_id = $1 and school_year_id = $2 and deleted_at is null
order by lower(display_name), id;

-- name: GetHouseholdByID :one
select id, organization_id, school_year_id, display_name, deleted_at, created_at, updated_at
from households
where id = $1 and organization_id = $2 and school_year_id = $3 and deleted_at is null;

-- name: UpdateHousehold :one
update households
set display_name = $4
where id = $1 and organization_id = $2 and school_year_id = $3 and deleted_at is null
returning id, organization_id, school_year_id, display_name, deleted_at, created_at, updated_at;

-- name: SoftDeleteHousehold :execrows
update households
set deleted_at = coalesce(deleted_at, now())
where id = $1 and organization_id = $2 and school_year_id = $3 and deleted_at is null;

-- name: ListAllActiveHouseholdsForRegistry :many
select id, organization_id, school_year_id, display_name, deleted_at, created_at, updated_at
from households
where organization_id = $1 and deleted_at is null
order by id;

-- name: FindHouseholdForRegistry :one
select id, organization_id, school_year_id, display_name, deleted_at, created_at, updated_at
from households
where id = $1 and organization_id = $2 and deleted_at is null;

-- name: CreateHouseholdStudent :one
insert into household_students (organization_id, school_year_id, household_id, student_id)
values ($1, $2, $3, $4)
returning id, organization_id, school_year_id, household_id, student_id, created_at, updated_at;

-- name: ListHouseholdStudents :many
select id, organization_id, school_year_id, household_id, student_id, created_at, updated_at
from household_students
where organization_id = $1 and school_year_id = $2 and household_id = $3
order by student_id, id;

-- name: DeleteHouseholdStudent :execrows
delete from household_students
where id = $1 and organization_id = $2;

-- name: DeleteHouseholdStudentMembership :execrows
delete from household_students
where organization_id = $1 and school_year_id = $2 and household_id = $3 and student_id = $4;

-- name: ListAllHouseholdStudentsForRegistry :many
select id, organization_id, school_year_id, household_id, student_id, created_at, updated_at
from household_students
where organization_id = $1
order by id;

-- name: FindHouseholdStudentForRegistry :one
select id, organization_id, school_year_id, household_id, student_id, created_at, updated_at
from household_students
where id = $1 and organization_id = $2;

-- name: TouchHouseholdStudent :one
update household_students
set updated_at = updated_at
where id = $1 and organization_id = $2 and school_year_id = $3
returning id, organization_id, school_year_id, household_id, student_id, created_at, updated_at;

-- name: CreateHouseholdAdult :one
insert into household_adults (organization_id, school_year_id, household_id, adult_id)
values ($1, $2, $3, $4)
returning id, organization_id, school_year_id, household_id, adult_id, created_at, updated_at;

-- name: ListHouseholdAdults :many
select id, organization_id, school_year_id, household_id, adult_id, created_at, updated_at
from household_adults
where organization_id = $1 and school_year_id = $2 and household_id = $3
order by adult_id, id;

-- name: DeleteHouseholdAdult :execrows
delete from household_adults
where id = $1 and organization_id = $2;

-- name: DeleteHouseholdAdultMembership :execrows
delete from household_adults
where organization_id = $1 and school_year_id = $2 and household_id = $3 and adult_id = $4;

-- name: ListAllHouseholdAdultsForRegistry :many
select id, organization_id, school_year_id, household_id, adult_id, created_at, updated_at
from household_adults
where organization_id = $1
order by id;

-- name: FindHouseholdAdultForRegistry :one
select id, organization_id, school_year_id, household_id, adult_id, created_at, updated_at
from household_adults
where id = $1 and organization_id = $2;

-- name: TouchHouseholdAdult :one
update household_adults
set updated_at = updated_at
where id = $1 and organization_id = $2 and school_year_id = $3
returning id, organization_id, school_year_id, household_id, adult_id, created_at, updated_at;

-- name: CreateGuardianRelationship :one
insert into guardian_relationships (organization_id, school_year_id, adult_id, student_id, relationship_type)
values ($1, $2, $3, $4, $5)
returning id, organization_id, school_year_id, adult_id, student_id, relationship_type, created_at, updated_at;

-- name: ListGuardianRelationships :many
select id, organization_id, school_year_id, adult_id, student_id, relationship_type, created_at, updated_at
from guardian_relationships
where organization_id = $1 and school_year_id = $2
order by adult_id, student_id, id;

-- name: GetGuardianRelationshipByID :one
select id, organization_id, school_year_id, adult_id, student_id, relationship_type, created_at, updated_at
from guardian_relationships
where id = $1 and organization_id = $2 and school_year_id = $3;

-- name: UpdateGuardianRelationship :one
update guardian_relationships
set relationship_type = $4
where id = $1 and organization_id = $2 and school_year_id = $3
returning id, organization_id, school_year_id, adult_id, student_id, relationship_type, created_at, updated_at;

-- name: DeleteGuardianRelationship :execrows
delete from guardian_relationships
where id = $1 and organization_id = $2 and school_year_id = $3;

-- name: ListAllGuardianRelationshipsForRegistry :many
select id, organization_id, school_year_id, adult_id, student_id, relationship_type, created_at, updated_at
from guardian_relationships
where organization_id = $1
order by id;

-- name: FindGuardianRelationshipForRegistry :one
select id, organization_id, school_year_id, adult_id, student_id, relationship_type, created_at, updated_at
from guardian_relationships
where id = $1 and organization_id = $2;
