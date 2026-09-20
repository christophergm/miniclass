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

-- name: DeleteGuardianRelationshipForStudent :execrows
delete from guardian_relationships
where organization_id = $1 and school_year_id = $2 and adult_id = $3 and student_id = $4;

-- name: ListGuardianStudentIDsForAdult :many
select student_id from guardian_relationships
where organization_id = $1 and school_year_id = $2 and adult_id = $3
order by student_id;

-- name: DeleteGuardianRelationshipsForAdult :many
delete from guardian_relationships
where organization_id = $1 and school_year_id = $2 and adult_id = $3
returning student_id;

-- name: CountOtherActiveGuardians :one
select count(*) from guardian_relationships gr
join adults a on a.id = gr.adult_id and a.organization_id = gr.organization_id and a.school_year_id = gr.school_year_id
where gr.organization_id = $1 and gr.school_year_id = $2 and gr.student_id = $3
  and gr.adult_id <> $4 and a.deleted_at is null;

-- name: ListAllGuardianRelationshipsForRegistry :many
select id, organization_id, school_year_id, adult_id, student_id, relationship_type, created_at, updated_at
from guardian_relationships
where organization_id = $1
order by id;

-- name: FindGuardianRelationshipForRegistry :one
select id, organization_id, school_year_id, adult_id, student_id, relationship_type, created_at, updated_at
from guardian_relationships
where id = $1 and organization_id = $2;
