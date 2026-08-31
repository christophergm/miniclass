-- name: CreateProgram :one
insert into programs (organization_id, school_year_id, name) values ($1, $2, $3)
returning id, organization_id, school_year_id, name, created_at, updated_at;

-- name: ListPrograms :many
select id, organization_id, school_year_id, name, created_at, updated_at
from programs where organization_id = $1 and school_year_id = $2 order by name, id;

-- name: GetProgram :one
select id, organization_id, school_year_id, name, created_at, updated_at
from programs where id = $1 and organization_id = $2 and school_year_id = $3;

-- name: CreateProgramMembership :one
insert into program_memberships (organization_id, school_year_id, program_id, student_id)
values ($1, $2, $3, $4)
returning id, organization_id, school_year_id, program_id, student_id, created_at, updated_at;

-- name: ListProgramMemberships :many
select m.id, m.organization_id, m.school_year_id, m.program_id, m.student_id,
    m.created_at, m.updated_at, s.grade_level_id, s.legal_given_name,
    s.legal_family_name
from program_memberships m
join students s on s.id = m.student_id and s.organization_id = m.organization_id
    and s.school_year_id = m.school_year_id
where m.organization_id = $1 and m.school_year_id = $2 and m.program_id = $3
order by s.legal_family_name, s.legal_given_name, s.id;

-- name: DeleteProgramMembership :execrows
delete from program_memberships where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4;

-- name: CountStudentsWithoutGrade :one
select count(*) from students where organization_id = $1 and school_year_id = $2
    and deleted_at is null and grade_level_id is null;

-- name: ListAllProgramsForRegistry :many
select id, organization_id, school_year_id, name, created_at, updated_at
from programs where organization_id = $1 order by id;

-- name: FindProgramForRegistry :one
select id, organization_id, school_year_id, name, created_at, updated_at
from programs where id = $1 and organization_id = $2;

-- name: UpdateProgramForRegistry :execrows
update programs set name = $3 where id = $1 and organization_id = $2;

-- name: DeleteProgramForRegistry :execrows
delete from programs where id = $1 and organization_id = $2;

-- name: ListAllProgramMembershipsForRegistry :many
select id, organization_id, school_year_id, program_id, student_id, created_at, updated_at
from program_memberships where organization_id = $1 order by id;

-- name: FindProgramMembershipForRegistry :one
select id, organization_id, school_year_id, program_id, student_id, created_at, updated_at
from program_memberships where id = $1 and organization_id = $2;

-- name: TouchProgramMembershipForRegistry :execrows
update program_memberships set updated_at = now() where id = $1 and organization_id = $2;

-- name: DeleteProgramMembershipForRegistry :execrows
delete from program_memberships where id = $1 and organization_id = $2;
