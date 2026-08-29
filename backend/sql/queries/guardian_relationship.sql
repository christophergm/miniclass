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
