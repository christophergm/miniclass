-- name: CreateInterestProfileSubmission :one
insert into interest_profile_submissions (organization_id, school_year_id, program_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
returning id, organization_id, school_year_id, program_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label, submitted_at, created_at;

-- name: CreateInterestProfileResponse :one
insert into interest_profile_responses (organization_id, school_year_id, program_id, submission_id, interest_area_id, response)
values ($1, $2, $3, $4, $5, $6)
returning id, organization_id, school_year_id, program_id, submission_id, interest_area_id, response, created_at;

-- name: ListInterestProfileSubmissions :many
select id, organization_id, school_year_id, program_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label, submitted_at, created_at
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

-- name: GetLatestRankedChoiceSubmission :one
select id, organization_id, school_year_id, program_id, session_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label, submitted_at, created_at
from ranked_choice_submissions
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4 and student_id = $5
order by submitted_at desc, id desc
limit 1;

-- name: ListAllInterestProfileSubmissionsForRegistry :many
select id, organization_id, school_year_id, program_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label, submitted_at, created_at
from interest_profile_submissions
where organization_id = $1
order by school_year_id, program_id, student_id, submitted_at, id;

-- name: FindInterestProfileSubmissionForRegistry :one
select id, organization_id, school_year_id, program_id, student_id, channel, actor_type, actor_user_id, actor_adult_id, actor_label, submitted_at, created_at
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
