-- name: ListResponseTrackingSummaries :many
select instrument_type, instrument_id, instrument_name, state, school_year_id, program_id,
    total_students, responded_students
from (
    select 'interest_profile_survey'::text as instrument_type,
        survey.id as instrument_id, survey.name as instrument_name, survey.state::text as state,
        survey.school_year_id, survey.program_id,
        count(student.id)::int as total_students,
        count(student.id) filter (where exists (
            select 1
            from interest_profile_submissions submission
            where submission.organization_id = survey.organization_id
              and submission.school_year_id = survey.school_year_id
              and submission.program_id = survey.program_id
              and submission.survey_id = survey.id
              and submission.student_id = audience.student_id
        ))::int as responded_students
    from interest_profile_surveys survey
    left join interest_profile_survey_audience_snapshots audience
      on audience.organization_id = survey.organization_id
     and audience.school_year_id = survey.school_year_id
     and audience.program_id = survey.program_id
     and audience.survey_id = survey.id
    left join students student
      on student.id = audience.student_id
     and student.organization_id = audience.organization_id
     and student.school_year_id = audience.school_year_id
     and student.deleted_at is null
    where survey.organization_id = $1
      and survey.school_year_id = $2
      and survey.program_id = $3
    group by survey.organization_id, survey.id, survey.name, survey.state,
        survey.school_year_id, survey.program_id

    union all

    select 'ranked_choice_session'::text as instrument_type,
        session.id as instrument_id, session.name as instrument_name, session.state::text as state,
        session.school_year_id, session.program_id,
        count(student.id)::int as total_students,
        count(student.id) filter (where exists (
            select 1
            from ranked_choice_submissions submission
            where submission.organization_id = session.organization_id
              and submission.school_year_id = session.school_year_id
              and submission.program_id = session.program_id
              and submission.session_id = session.id
              and submission.student_id = membership.student_id
        ))::int as responded_students
    from sessions session
    left join program_memberships membership
      on membership.organization_id = session.organization_id
     and membership.school_year_id = session.school_year_id
     and membership.program_id = session.program_id
    left join session_non_participations excluded
      on excluded.organization_id = membership.organization_id
     and excluded.school_year_id = membership.school_year_id
     and excluded.program_id = membership.program_id
     and excluded.session_id = session.id
     and excluded.student_id = membership.student_id
    left join students student
      on student.id = membership.student_id
     and student.organization_id = membership.organization_id
     and student.school_year_id = membership.school_year_id
    where session.organization_id = $1
      and session.school_year_id = $2
      and session.program_id = $3
      and session.ranked_choice_enabled
      and excluded.id is null
      and (student.id is null or student.deleted_at is null)
    group by session.organization_id, session.id, session.name, session.state,
        session.school_year_id, session.program_id
) summaries
order by lower(instrument_name), instrument_id;
