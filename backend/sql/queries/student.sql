-- name: CreateStudent :one
insert into students (
    organization_id,
    school_year_id,
    legal_given_name,
    legal_family_name,
    preferred_given_name,
    grade_level_id,
    homeroom_id,
    external_identifier
)
values ($1, $2, $3, $4, $5, $6, $7, $8)
returning id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier, deleted_at, created_at, updated_at;

-- name: ListStudents :many
select id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier, deleted_at, created_at, updated_at
from students
where organization_id = $1
  and school_year_id = $2
  and ($3::bool or deleted_at is null)
order by legal_family_name, coalesce(preferred_given_name, legal_given_name), legal_given_name, id;

-- name: GetStudentByID :one
select id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier, deleted_at, created_at, updated_at
from students
where id = $1
  and organization_id = $2
  and school_year_id = $3
  and deleted_at is null;

-- name: GetStudentByIDIncludingDeleted :one
select id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier, deleted_at, created_at, updated_at
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
    external_identifier = $9
where id = $1
  and organization_id = $2
  and school_year_id = $3
  and deleted_at is null
returning id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier, deleted_at, created_at, updated_at;

-- name: SoftDeleteStudent :execrows
update students
set deleted_at = coalesce(deleted_at, now())
where id = $1
  and organization_id = $2
  and school_year_id = $3
  and deleted_at is null;

-- name: DeidentifyStudent :one
update students
set legal_given_name = 'Deleted student', legal_family_name = 'Deleted student',
    preferred_given_name = null, external_identifier = null,
    deleted_at = coalesce(deleted_at, now())
where id = $1 and organization_id = $2 and school_year_id = $3 and deleted_at is null
returning id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier, deleted_at, created_at, updated_at;

-- name: CountStudentAssociatedData :one
select (
    (select count(*) from program_memberships pm where pm.organization_id = $2 and pm.school_year_id = $3 and pm.student_id = $1) +
    (select count(*) from session_non_participations snp where snp.organization_id = $2 and snp.school_year_id = $3 and snp.student_id = $1) +
    (select count(*) from interest_profile_submissions ips where ips.organization_id = $2 and ips.school_year_id = $3 and ips.student_id = $1) +
    (select count(*) from ranked_choice_submissions rcs where rcs.organization_id = $2 and rcs.school_year_id = $3 and rcs.student_id = $1) +
    (select count(*) from interest_profile_survey_audience_students ipsa where ipsa.organization_id = $2 and ipsa.school_year_id = $3 and ipsa.student_id = $1) +
    (select count(*) from interest_profile_survey_audience_snapshots ipss where ipss.organization_id = $2 and ipss.school_year_id = $3 and ipss.student_id = $1) +
    (select count(*) from ranked_choice_access_codes rcac where rcac.organization_id = $2 and rcac.school_year_id = $3 and rcac.student_id = $1) +
    (select count(*) from interest_profile_survey_access_codes ipsac where ipsac.organization_id = $2 and ipsac.school_year_id = $3 and ipsac.student_id = $1)
)::bigint as count;

-- name: HardDeleteStudentRankedAccessCodes :exec
delete from ranked_choice_access_codes where organization_id = $2 and school_year_id = $3 and student_id = $1;

-- name: HardDeleteStudentSurveyAccessCodes :exec
delete from interest_profile_survey_access_codes where organization_id = $2 and school_year_id = $3 and student_id = $1;

-- name: HardDeleteStudentSurveySnapshots :exec
delete from interest_profile_survey_audience_snapshots where organization_id = $2 and school_year_id = $3 and student_id = $1;

-- name: HardDeleteStudentSurveyAudience :exec
delete from interest_profile_survey_audience_students where organization_id = $2 and school_year_id = $3 and student_id = $1;

-- name: HardDeleteStudentRankedSubmissions :exec
delete from ranked_choice_submissions where organization_id = $2 and school_year_id = $3 and student_id = $1;

-- name: HardDeleteStudentSurveySubmissions :exec
delete from interest_profile_submissions where organization_id = $2 and school_year_id = $3 and student_id = $1;

-- name: HardDeleteStudentNonParticipations :exec
delete from session_non_participations where organization_id = $2 and school_year_id = $3 and student_id = $1;

-- name: HardDeleteStudentMemberships :exec
delete from program_memberships where organization_id = $2 and school_year_id = $3 and student_id = $1;

-- name: HardDeleteStudentRelationships :exec
delete from guardian_relationships where organization_id = $2 and school_year_id = $3 and student_id = $1;

-- name: HardDeleteStudent :exec
delete from students where id = $1 and organization_id = $2 and school_year_id = $3;

-- name: RestoreStudent :one
update students
set deleted_at = null
where id = $1
  and organization_id = $2
  and school_year_id = $3
  and deleted_at is not null
returning id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier, deleted_at, created_at, updated_at;

-- name: ListAllActiveStudentsForRegistry :many
select id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier, deleted_at, created_at, updated_at
from students
where organization_id = $1
  and deleted_at is null
order by id;

-- name: FindStudentForRegistry :one
select id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, grade_level_id, homeroom_id, external_identifier, deleted_at, created_at, updated_at
from students
where id = $1
  and organization_id = $2
  and deleted_at is null;
