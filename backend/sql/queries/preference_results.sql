-- Interest results deliberately do not use the cross-survey effective profile.
-- SPEC §13.7: latest per-area answer within this instrument, across all channels.
-- name: ListCurrentSurveyResultAnswers :many
select distinct on (s.student_id, r.interest_area_id)
    s.student_id, r.interest_area_id, r.response
from interest_profile_submissions s
join interest_profile_responses r
  on r.submission_id = s.id
 and r.organization_id = s.organization_id
 and r.school_year_id = s.school_year_id
 and r.program_id = s.program_id
where s.organization_id = $1 and s.school_year_id = $2
  and s.program_id = $3 and s.survey_id = $4
order by s.student_id, r.interest_area_id, s.submitted_at desc, s.id desc;

-- SPEC §13.7: the latest complete ranked response replaces the previous one.
-- name: ListCurrentSessionResultAnswers :many
with latest as (
    select distinct on (submission.student_id) submission.id, submission.organization_id, submission.school_year_id,
        submission.program_id, submission.session_id, submission.student_id
    from ranked_choice_submissions submission
    where submission.organization_id = $1 and submission.school_year_id = $2
      and submission.program_id = $3 and submission.session_id = $4
    order by submission.student_id, submission.submitted_at desc, submission.id desc
)
select s.student_id, r.offering_id, r.response, r.rank
from latest s
join ranked_choice_responses r
  on r.submission_id = s.id
 and r.organization_id = s.organization_id
 and r.school_year_id = s.school_year_id
 and r.program_id = s.program_id
 and r.session_id = s.session_id
order by s.student_id, r.offering_id;
