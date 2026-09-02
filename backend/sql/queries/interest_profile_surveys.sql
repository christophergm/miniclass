-- name: CreateInterestProfileSurvey :one
insert into interest_profile_surveys (
    organization_id, school_year_id, program_id, name, introduction, opens_at, closes_at,
    audience_type, audience_grade_level_id, audience_prior_survey_id, audience_response_state,
    scale_version
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
returning id, organization_id, school_year_id, program_id, name, introduction, state, opens_at, closes_at,
    audience_type, audience_grade_level_id, audience_prior_survey_id, audience_response_state,
    scale_version, opened_at, created_at, updated_at;

-- name: ListInterestProfileSurveys :many
select id, organization_id, school_year_id, program_id, name, introduction,
    state,
    opens_at, closes_at, audience_type, audience_grade_level_id, audience_prior_survey_id,
    audience_response_state, scale_version, opened_at, created_at, updated_at
from interest_profile_surveys
where organization_id = $1 and school_year_id = $2 and program_id = $3
order by created_at, id;

-- name: GetInterestProfileSurvey :one
select id, organization_id, school_year_id, program_id, name, introduction,
    state,
    opens_at, closes_at, audience_type, audience_grade_level_id, audience_prior_survey_id,
    audience_response_state, scale_version, opened_at, created_at, updated_at
from interest_profile_surveys
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4;

-- name: UpdateInterestProfileSurvey :one
update interest_profile_surveys
set name = $5, introduction = $6, opens_at = $7, closes_at = $8,
    audience_type = $9, audience_grade_level_id = $10, audience_prior_survey_id = $11,
    audience_response_state = $12, scale_version = $13
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4
returning id, organization_id, school_year_id, program_id, name, introduction, state, opens_at, closes_at,
    audience_type, audience_grade_level_id, audience_prior_survey_id, audience_response_state,
    scale_version, opened_at, created_at, updated_at;

-- name: SetInterestProfileSurveyState :one
update interest_profile_surveys
set state = $5, opens_at = $6, closes_at = $7, opened_at = $8
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4
returning id, organization_id, school_year_id, program_id, name, introduction, state, opens_at, closes_at,
    audience_type, audience_grade_level_id, audience_prior_survey_id, audience_response_state,
    scale_version, opened_at, created_at, updated_at;

-- name: DeleteInterestProfileSurvey :execrows
delete from interest_profile_surveys
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4;

-- name: CountInterestProfileSurveySubmissions :one
select count(*)::bigint
from interest_profile_submissions
where organization_id = $1 and school_year_id = $2 and program_id = $3 and survey_id = $4;

-- name: ListInterestProfileSurveyMembers :many
select pm.student_id, s.grade_level_id
from program_memberships pm
join students s on s.id = pm.student_id
    and s.organization_id = pm.organization_id
    and s.school_year_id = pm.school_year_id
where pm.organization_id = $1 and pm.school_year_id = $2 and pm.program_id = $3
    and s.deleted_at is null
order by pm.student_id;

-- name: ListInterestProfileSurveyDefinitionStudents :many
select id, organization_id, school_year_id, program_id, survey_id, student_id, created_at
from interest_profile_survey_audience_students
where organization_id = $1 and school_year_id = $2 and program_id = $3 and survey_id = $4
order by student_id, id;

-- name: CreateInterestProfileSurveyDefinitionStudent :one
insert into interest_profile_survey_audience_students (organization_id, school_year_id, program_id, survey_id, student_id)
values ($1, $2, $3, $4, $5)
returning id, organization_id, school_year_id, program_id, survey_id, student_id, created_at;

-- name: DeleteInterestProfileSurveyDefinitionStudents :exec
delete from interest_profile_survey_audience_students
where organization_id = $1 and school_year_id = $2 and program_id = $3 and survey_id = $4;

-- name: ListInterestProfileSurveyQuestions :many
select id, organization_id, school_year_id, program_id, survey_id, interest_area_id, ordinal, label, created_at, updated_at
from interest_profile_survey_questions
where organization_id = $1 and school_year_id = $2 and program_id = $3 and survey_id = $4
order by ordinal, id;

-- name: CreateInterestProfileSurveyQuestion :one
insert into interest_profile_survey_questions (organization_id, school_year_id, program_id, survey_id, interest_area_id, ordinal, label)
values ($1, $2, $3, $4, $5, $6, $7)
returning id, organization_id, school_year_id, program_id, survey_id, interest_area_id, ordinal, label, created_at, updated_at;

-- name: UpdateInterestProfileSurveyQuestion :one
update interest_profile_survey_questions
set label = $6, ordinal = $7
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4 and survey_id = $5
returning id, organization_id, school_year_id, program_id, survey_id, interest_area_id, ordinal, label, created_at, updated_at;

-- name: DeleteInterestProfileSurveyQuestions :exec
delete from interest_profile_survey_questions
where organization_id = $1 and school_year_id = $2 and program_id = $3 and survey_id = $4;

-- name: ListInterestProfileSurveyScaleOptions :many
select id, organization_id, school_year_id, program_id, survey_id, value, label, ordinal, created_at
from interest_profile_survey_scale_options
where organization_id = $1 and school_year_id = $2 and program_id = $3 and survey_id = $4
order by ordinal, id;

-- name: CreateInterestProfileSurveyScaleOption :one
insert into interest_profile_survey_scale_options (organization_id, school_year_id, program_id, survey_id, value, label, ordinal)
values ($1, $2, $3, $4, $5, $6, $7)
returning id, organization_id, school_year_id, program_id, survey_id, value, label, ordinal, created_at;

-- name: UpdateInterestProfileSurveyScaleOption :one
update interest_profile_survey_scale_options
set value = $6, label = $7, ordinal = $8
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4 and survey_id = $5
returning id, organization_id, school_year_id, program_id, survey_id, value, label, ordinal, created_at;

-- name: DeleteInterestProfileSurveyScaleOptions :exec
delete from interest_profile_survey_scale_options
where organization_id = $1 and school_year_id = $2 and program_id = $3 and survey_id = $4;

-- name: CreateInterestProfileSurveyAudienceSnapshot :one
insert into interest_profile_survey_audience_snapshots (organization_id, school_year_id, program_id, survey_id, student_id)
values ($1, $2, $3, $4, $5)
returning id, organization_id, school_year_id, program_id, survey_id, student_id, created_at;

-- name: ListInterestProfileSurveyAudienceSnapshot :many
select id, organization_id, school_year_id, program_id, survey_id, student_id, created_at
from interest_profile_survey_audience_snapshots
where organization_id = $1 and school_year_id = $2 and program_id = $3 and survey_id = $4
order by student_id, id;

-- name: CreateInterestProfileSurveyAccessCode :one
insert into interest_profile_survey_access_codes (organization_id, school_year_id, program_id, survey_id, student_id, code_hash)
values ($1, $2, $3, $4, $5, $6)
returning id, organization_id, school_year_id, program_id, survey_id, student_id, code_hash, issued_at, revoked_at;

-- name: ListActiveInterestProfileSurveyAccessCodes :many
select id, organization_id, school_year_id, program_id, survey_id, student_id, code_hash, issued_at, revoked_at
from interest_profile_survey_access_codes
where organization_id = $1 and school_year_id = $2 and program_id = $3 and survey_id = $4 and revoked_at is null
order by student_id, id;

-- name: RevokeInterestProfileSurveyAccessCodes :execrows
update interest_profile_survey_access_codes
set revoked_at = now()
where organization_id = $1 and school_year_id = $2 and program_id = $3 and survey_id = $4 and revoked_at is null;

-- name: FindActiveInterestProfileSurveyAccessCode :one
select id, organization_id, school_year_id, program_id, survey_id, student_id, code_hash, issued_at, revoked_at
from interest_profile_survey_access_codes
where organization_id = $1 and school_year_id = $2 and program_id = $3 and survey_id = $4 and code_hash = $5 and revoked_at is null;

-- name: ListInterestProfileSurveyPriorResponders :many
select distinct student_id
from interest_profile_submissions
where organization_id = $1 and school_year_id = $2 and program_id = $3 and survey_id = $4
order by student_id;

-- name: ListAllInterestProfileSurveysForRegistry :many
select id, organization_id, school_year_id, program_id, name, introduction, state, opens_at, closes_at,
    audience_type, audience_grade_level_id, audience_prior_survey_id, audience_response_state,
    scale_version, opened_at, created_at, updated_at
from interest_profile_surveys
where organization_id = $1
order by school_year_id, program_id, created_at, id;

-- name: FindInterestProfileSurveyForRegistry :one
select id, organization_id, school_year_id, program_id, name, introduction, state, opens_at, closes_at,
    audience_type, audience_grade_level_id, audience_prior_survey_id, audience_response_state,
    scale_version, opened_at, created_at, updated_at
from interest_profile_surveys
where id = $1 and organization_id = $2;

-- name: UpdateInterestProfileSurveyForRegistry :one
update interest_profile_surveys
set name = name || ' updated'
where id = $1 and organization_id = $2 and state = 'draft'
returning id, organization_id, school_year_id, program_id, name, introduction, state, opens_at, closes_at,
    audience_type, audience_grade_level_id, audience_prior_survey_id, audience_response_state,
    scale_version, opened_at, created_at, updated_at;

-- name: DeleteInterestProfileSurveyForRegistry :execrows
delete from interest_profile_surveys
where id = $1 and organization_id = $2 and state = 'draft';

-- name: ListAllInterestProfileSurveyDefinitionStudentsForRegistry :many
select id, organization_id, school_year_id, program_id, survey_id, student_id, created_at
from interest_profile_survey_audience_students
where organization_id = $1
order by school_year_id, program_id, survey_id, student_id, id;

-- name: FindInterestProfileSurveyDefinitionStudentForRegistry :one
select id, organization_id, school_year_id, program_id, survey_id, student_id, created_at
from interest_profile_survey_audience_students
where id = $1 and organization_id = $2;

-- name: UpdateInterestProfileSurveyDefinitionStudentForRegistry :execrows
update interest_profile_survey_audience_students
set student_id = student_id
where id = $1 and organization_id = $2;

-- name: DeleteInterestProfileSurveyDefinitionStudentForRegistry :execrows
delete from interest_profile_survey_audience_students where id = $1 and organization_id = $2;

-- name: ListAllInterestProfileSurveyQuestionsForRegistry :many
select id, organization_id, school_year_id, program_id, survey_id, interest_area_id, ordinal, label, created_at, updated_at
from interest_profile_survey_questions
where organization_id = $1
order by school_year_id, program_id, survey_id, ordinal, id;

-- name: FindInterestProfileSurveyQuestionForRegistry :one
select id, organization_id, school_year_id, program_id, survey_id, interest_area_id, ordinal, label, created_at, updated_at
from interest_profile_survey_questions
where id = $1 and organization_id = $2;

-- name: UpdateInterestProfileSurveyQuestionForRegistry :execrows
update interest_profile_survey_questions
set label = label || ' updated'
where id = $1 and organization_id = $2;

-- name: DeleteInterestProfileSurveyQuestionForRegistry :execrows
delete from interest_profile_survey_questions where id = $1 and organization_id = $2;

-- name: ListAllInterestProfileSurveyScaleOptionsForRegistry :many
select id, organization_id, school_year_id, program_id, survey_id, value, label, ordinal, created_at
from interest_profile_survey_scale_options
where organization_id = $1
order by school_year_id, program_id, survey_id, ordinal, id;

-- name: FindInterestProfileSurveyScaleOptionForRegistry :one
select id, organization_id, school_year_id, program_id, survey_id, value, label, ordinal, created_at
from interest_profile_survey_scale_options
where id = $1 and organization_id = $2;

-- name: UpdateInterestProfileSurveyScaleOptionForRegistry :execrows
update interest_profile_survey_scale_options
set label = label || ' updated'
where id = $1 and organization_id = $2;

-- name: DeleteInterestProfileSurveyScaleOptionForRegistry :execrows
delete from interest_profile_survey_scale_options where id = $1 and organization_id = $2;

-- name: ListAllInterestProfileSurveyAudienceSnapshotsForRegistry :many
select id, organization_id, school_year_id, program_id, survey_id, student_id, created_at
from interest_profile_survey_audience_snapshots
where organization_id = $1
order by school_year_id, program_id, survey_id, student_id, id;

-- name: FindInterestProfileSurveyAudienceSnapshotForRegistry :one
select id, organization_id, school_year_id, program_id, survey_id, student_id, created_at
from interest_profile_survey_audience_snapshots
where id = $1 and organization_id = $2;

-- name: ListAllInterestProfileSurveyAccessCodesForRegistry :many
select id, organization_id, school_year_id, program_id, survey_id, student_id, code_hash, issued_at, revoked_at
from interest_profile_survey_access_codes
where organization_id = $1
order by school_year_id, program_id, survey_id, student_id, issued_at, id;

-- name: FindInterestProfileSurveyAccessCodeForRegistry :one
select id, organization_id, school_year_id, program_id, survey_id, student_id, code_hash, issued_at, revoked_at
from interest_profile_survey_access_codes
where id = $1 and organization_id = $2;

-- name: RevokeInterestProfileSurveyAccessCodeForRegistry :execrows
update interest_profile_survey_access_codes
set revoked_at = coalesce(revoked_at, now())
where id = $1 and organization_id = $2;
