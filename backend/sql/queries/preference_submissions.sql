-- name: CreateInterestProfileSubmission :one
insert into interest_profile_submissions (organization_id, school_year_id, program_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
returning id, organization_id, school_year_id, program_id, survey_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label, submitted_at, created_at;

-- name: CreateInterestProfileSurveySubmission :one
insert into interest_profile_submissions (organization_id, school_year_id, program_id, survey_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
returning id, organization_id, school_year_id, program_id, survey_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label, submitted_at, created_at;

-- name: CreateInterestProfileResponse :one
insert into interest_profile_responses (organization_id, school_year_id, program_id, submission_id, interest_area_id, response)
values ($1, $2, $3, $4, $5, $6)
returning id, organization_id, school_year_id, program_id, submission_id, interest_area_id, response, created_at;

-- name: ListInterestProfileSubmissions :many
select id, organization_id, school_year_id, program_id, survey_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label, submitted_at, created_at
from interest_profile_submissions
where organization_id = $1 and school_year_id = $2 and program_id = $3 and student_id = $4
order by submitted_at, id;

-- name: ListInterestProfileResponses :many
select id, organization_id, school_year_id, program_id, submission_id, interest_area_id, response, created_at
from interest_profile_responses
where organization_id = $1 and school_year_id = $2 and program_id = $3 and submission_id = $4
order by interest_area_id, id;

-- name: GetEffectiveInterestProfile :many
select distinct on (r.interest_area_id)
    r.interest_area_id, r.response, r.submission_id, s.submitted_at
from interest_profile_responses r
join interest_profile_submissions s
    on s.id = r.submission_id
   and s.organization_id = r.organization_id
   and s.school_year_id = r.school_year_id
   and s.program_id = r.program_id
where r.organization_id = $1 and r.school_year_id = $2 and r.program_id = $3 and s.student_id = $4
order by r.interest_area_id, s.submitted_at desc, s.id desc;

-- name: CreateRankedChoiceSubmission :one
insert into ranked_choice_submissions (organization_id, school_year_id, program_id, session_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
returning id, organization_id, school_year_id, program_id, session_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label, submitted_at, created_at;

-- name: CreateRankedChoiceResponse :one
insert into ranked_choice_responses (organization_id, school_year_id, program_id, session_id, submission_id, offering_id, response, rank)
values ($1, $2, $3, $4, $5, $6, $7, $8)
returning id, organization_id, school_year_id, program_id, session_id, submission_id, offering_id, response, rank, created_at;

-- name: ListRankedChoiceSubmissions :many
select id, organization_id, school_year_id, program_id, session_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label, submitted_at, created_at
from ranked_choice_submissions
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4 and student_id = $5
order by submitted_at, id;

-- name: ListRankedChoiceResponses :many
select id, organization_id, school_year_id, program_id, session_id, submission_id, offering_id, response, rank, created_at
from ranked_choice_responses
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4 and submission_id = $5
order by offering_id, id;

-- name: CreateRankedChoiceAccessCode :one
insert into ranked_choice_access_codes (organization_id, school_year_id, program_id, session_id, student_id, code_hash)
values ($1, $2, $3, $4, $5, $6)
returning id, organization_id, school_year_id, program_id, session_id, student_id, issued_at, revoked_at;

-- name: ListActiveRankedChoiceAccessCodes :many
select id, organization_id, school_year_id, program_id, session_id, student_id, issued_at, revoked_at
from ranked_choice_access_codes
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4 and revoked_at is null
order by student_id, id;

-- name: FindActiveRankedChoiceAccessCode :one
select student_id
from ranked_choice_access_codes
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4 and code_hash = $5 and revoked_at is null;

-- name: RevokeRankedChoiceAccessCodes :execrows
update ranked_choice_access_codes
set revoked_at = coalesce(revoked_at, now())
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4 and revoked_at is null;

-- name: GetLatestRankedChoiceSubmission :one
select id, organization_id, school_year_id, program_id, session_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label, submitted_at, created_at
from ranked_choice_submissions
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4 and student_id = $5
order by submitted_at desc, id desc
limit 1;

-- name: ListInterestProfileResponseTrackingStudents :many
select s.id, s.organization_id, s.school_year_id, s.legal_given_name,
    s.legal_family_name, s.preferred_given_name, s.grade_level_id,
    coalesce(g.label, '') as grade_level_label, s.homeroom_id,
    coalesce(h.name, '') as homeroom_name,
    exists (
        select 1
        from interest_profile_submissions submission
        where submission.organization_id = s.organization_id
          and submission.school_year_id = s.school_year_id
          and submission.program_id = $3
          and submission.survey_id = $4
          and submission.student_id = s.id
    ) as responded
from interest_profile_survey_audience_snapshots audience
join students s
  on s.id = audience.student_id
 and s.organization_id = audience.organization_id
 and s.school_year_id = audience.school_year_id
left join grade_levels g
  on g.id = s.grade_level_id
 and g.organization_id = s.organization_id
 and g.school_year_id = s.school_year_id
left join homerooms h
  on h.id = s.homeroom_id
 and h.organization_id = s.organization_id
 and h.school_year_id = s.school_year_id
where audience.organization_id = $1
  and audience.school_year_id = $2
  and audience.program_id = $3
  and audience.survey_id = $4
  and s.deleted_at is null
order by lower(s.legal_family_name),
    lower(coalesce(s.preferred_given_name, s.legal_given_name)),
    lower(s.legal_given_name), s.id;

-- name: ListRankedChoiceResponseTrackingStudents :many
select s.id, s.organization_id, s.school_year_id, s.legal_given_name,
    s.legal_family_name, s.preferred_given_name, s.grade_level_id,
    coalesce(g.label, '') as grade_level_label, s.homeroom_id,
    coalesce(h.name, '') as homeroom_name,
    exists (
        select 1
        from ranked_choice_submissions submission
        where submission.organization_id = s.organization_id
          and submission.school_year_id = s.school_year_id
          and submission.program_id = $3
          and submission.session_id = $4
          and submission.student_id = s.id
    ) as responded
from program_memberships membership
join students s
  on s.id = membership.student_id
 and s.organization_id = membership.organization_id
 and s.school_year_id = membership.school_year_id
left join session_non_participations excluded
  on excluded.organization_id = membership.organization_id
 and excluded.school_year_id = membership.school_year_id
 and excluded.program_id = membership.program_id
 and excluded.session_id = $4
 and excluded.student_id = membership.student_id
left join grade_levels g
  on g.id = s.grade_level_id
 and g.organization_id = s.organization_id
 and g.school_year_id = s.school_year_id
left join homerooms h
  on h.id = s.homeroom_id
 and h.organization_id = s.organization_id
 and h.school_year_id = s.school_year_id
where membership.organization_id = $1
  and membership.school_year_id = $2
  and membership.program_id = $3
  and excluded.id is null
  and s.deleted_at is null
order by lower(s.legal_family_name),
    lower(coalesce(s.preferred_given_name, s.legal_given_name)),
    lower(s.legal_given_name), s.id;

-- name: ListAllInterestProfileSubmissionsForRegistry :many
select id, organization_id, school_year_id, program_id, survey_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label, submitted_at, created_at
from interest_profile_submissions
where organization_id = $1
order by school_year_id, program_id, student_id, submitted_at, id;

-- name: FindInterestProfileSubmissionForRegistry :one
select id, organization_id, school_year_id, program_id, survey_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label, submitted_at, created_at
from interest_profile_submissions
where id = $1 and organization_id = $2;

-- name: ListAllInterestProfileResponsesForRegistry :many
select id, organization_id, school_year_id, program_id, submission_id, interest_area_id, response, created_at
from interest_profile_responses
where organization_id = $1
order by school_year_id, program_id, submission_id, interest_area_id, id;

-- name: FindInterestProfileResponseForRegistry :one
select id, organization_id, school_year_id, program_id, submission_id, interest_area_id, response, created_at
from interest_profile_responses
where id = $1 and organization_id = $2;

-- name: ListAllRankedChoiceSubmissionsForRegistry :many
select id, organization_id, school_year_id, program_id, session_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label, submitted_at, created_at
from ranked_choice_submissions
where organization_id = $1
order by school_year_id, program_id, session_id, student_id, submitted_at, id;

-- name: FindRankedChoiceSubmissionForRegistry :one
select id, organization_id, school_year_id, program_id, session_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label, submitted_at, created_at
from ranked_choice_submissions
where id = $1 and organization_id = $2;

-- name: ListAllRankedChoiceResponsesForRegistry :many
select id, organization_id, school_year_id, program_id, session_id, submission_id, offering_id, response, rank, created_at
from ranked_choice_responses
where organization_id = $1
order by school_year_id, program_id, session_id, submission_id, offering_id, id;

-- name: FindRankedChoiceResponseForRegistry :one
select id, organization_id, school_year_id, program_id, session_id, submission_id, offering_id, response, rank, created_at
from ranked_choice_responses
where id = $1 and organization_id = $2;

-- name: ListAllRankedChoiceAccessCodesForRegistry :many
select id, organization_id, school_year_id, program_id, session_id, student_id, issued_at, revoked_at
from ranked_choice_access_codes
where organization_id = $1
order by school_year_id, program_id, session_id, student_id, id;

-- name: FindRankedChoiceAccessCodeForRegistry :one
select id, organization_id, school_year_id, program_id, session_id, student_id, issued_at, revoked_at
from ranked_choice_access_codes
where id = $1 and organization_id = $2;

-- name: RevokeRankedChoiceAccessCodeForRegistry :execrows
update ranked_choice_access_codes
set revoked_at = coalesce(revoked_at, now())
where id = $1 and organization_id = $2;
