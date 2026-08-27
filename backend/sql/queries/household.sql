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

-- A soft-deleted student is excluded from views (SPEC §21.3) while the
-- membership row itself is retained, so the exclusion is a read-time predicate
-- and not a delete.
-- name: ListHouseholdStudents :many
select id, organization_id, school_year_id, household_id, student_id, created_at, updated_at
from household_students
where household_students.organization_id = $1
  and household_students.school_year_id = $2
  and household_students.household_id = $3
  and exists (
    select 1
    from households
    where households.id = household_students.household_id
      and households.organization_id = household_students.organization_id
      and households.school_year_id = household_students.school_year_id
      and households.deleted_at is null
  )
  and exists (
    select 1
    from students
    where students.id = household_students.student_id
      and students.organization_id = household_students.organization_id
      and students.school_year_id = household_students.school_year_id
      and students.deleted_at is null
  )
order by student_id, id;

-- Every household membership in one school year, for the surfaces that ask
-- "which households does this person belong to" about a whole roster. Soft-
-- deleted households and soft-deleted students are both excluded (SPEC §21.3).
-- name: ListHouseholdStudentsForSchoolYear :many
select id, organization_id, school_year_id, household_id, student_id, created_at, updated_at
from household_students
where household_students.organization_id = $1
  and household_students.school_year_id = $2
  and exists (
    select 1
    from households
    where households.id = household_students.household_id
      and households.organization_id = household_students.organization_id
      and households.school_year_id = household_students.school_year_id
      and households.deleted_at is null
  )
  and exists (
    select 1
    from students
    where students.id = household_students.student_id
      and students.organization_id = household_students.organization_id
      and students.school_year_id = household_students.school_year_id
      and students.deleted_at is null
  )
order by household_id, student_id, id;

-- Unfiltered by design: removing a membership must stay possible after the
-- student is soft-deleted, and the audit entry needs the membership identifier.
-- name: GetHouseholdStudent :one
select id, organization_id, school_year_id, household_id, student_id, created_at, updated_at
from household_students
where organization_id = $1 and school_year_id = $2 and household_id = $3 and student_id = $4;

-- name: DeleteHouseholdStudent :execrows
delete from household_students
where organization_id = sqlc.arg(organization_id)::public.xid20
  and school_year_id = sqlc.arg(school_year_id)::public.xid20
  and household_id = sqlc.arg(household_id)::public.xid20
  and student_id = sqlc.arg(student_id)::public.xid20;

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
where household_adults.organization_id = $1
  and household_adults.school_year_id = $2
  and household_adults.household_id = $3
  and exists (
    select 1
    from households
    where households.id = household_adults.household_id
      and households.organization_id = household_adults.organization_id
      and households.school_year_id = household_adults.school_year_id
      and households.deleted_at is null
  )
  and exists (
    select 1
    from adults
    where adults.id = household_adults.adult_id
      and adults.organization_id = household_adults.organization_id
      and adults.school_year_id = household_adults.school_year_id
      and adults.deleted_at is null
  )
order by adult_id, id;

-- name: ListHouseholdAdultsForSchoolYear :many
select id, organization_id, school_year_id, household_id, adult_id, created_at, updated_at
from household_adults
where household_adults.organization_id = $1
  and household_adults.school_year_id = $2
  and exists (
    select 1
    from households
    where households.id = household_adults.household_id
      and households.organization_id = household_adults.organization_id
      and households.school_year_id = household_adults.school_year_id
      and households.deleted_at is null
  )
  and exists (
    select 1
    from adults
    where adults.id = household_adults.adult_id
      and adults.organization_id = household_adults.organization_id
      and adults.school_year_id = household_adults.school_year_id
      and adults.deleted_at is null
  )
order by household_id, adult_id, id;

-- name: GetHouseholdAdult :one
select id, organization_id, school_year_id, household_id, adult_id, created_at, updated_at
from household_adults
where organization_id = $1 and school_year_id = $2 and household_id = $3 and adult_id = $4;

-- name: DeleteHouseholdAdult :execrows
delete from household_adults
where organization_id = sqlc.arg(organization_id)::public.xid20
  and school_year_id = sqlc.arg(school_year_id)::public.xid20
  and household_id = sqlc.arg(household_id)::public.xid20
  and adult_id = sqlc.arg(adult_id)::public.xid20;

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
where organization_id = sqlc.arg('organization_id')::public.xid20
  and school_year_id = sqlc.arg('school_year_id')::public.xid20
  and (sqlc.narg('adult_id')::public.xid20 is null or adult_id = sqlc.narg('adult_id')::public.xid20)
  and (sqlc.narg('student_id')::public.xid20 is null or student_id = sqlc.narg('student_id')::public.xid20)
  and exists (
    select 1
    from adults
    where adults.id = guardian_relationships.adult_id
      and adults.organization_id = guardian_relationships.organization_id
      and adults.school_year_id = guardian_relationships.school_year_id
      and adults.deleted_at is null
  )
  and exists (
    select 1
    from students
    where students.id = guardian_relationships.student_id
      and students.organization_id = guardian_relationships.organization_id
      and students.school_year_id = guardian_relationships.school_year_id
      and students.deleted_at is null
  )
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
