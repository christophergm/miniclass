-- name: CountStudentGuardianRelationships :one
select count(*)::bigint
from guardian_relationships
where organization_id = $1 and school_year_id = $2 and student_id = $3;

-- name: CountStudentPreferenceDependencies :one
select (
    (select count(*) from interest_profile_submissions ips where ips.organization_id = $1 and ips.school_year_id = $2 and ips.student_id = $3)
    + (select count(*) from ranked_choice_submissions rcs where rcs.organization_id = $1 and rcs.school_year_id = $2 and rcs.student_id = $3)
    + (select count(*) from interest_profile_survey_audience_students ias where ias.organization_id = $1 and ias.school_year_id = $2 and ias.student_id = $3)
    + (select count(*) from interest_profile_survey_audience_snapshots iass where iass.organization_id = $1 and iass.school_year_id = $2 and iass.student_id = $3)
    + (select count(*) from interest_profile_survey_access_codes isac where isac.organization_id = $1 and isac.school_year_id = $2 and isac.student_id = $3)
    + (select count(*) from ranked_choice_access_codes rcac where rcac.organization_id = $1 and rcac.school_year_id = $2 and rcac.student_id = $3)
)::bigint;

-- name: CountStudentProgramMembershipConflicts :one
select count(*)::bigint
from program_memberships source
join program_memberships target
  on target.organization_id = source.organization_id
 and target.school_year_id = source.school_year_id
 and target.program_id = source.program_id
 and target.student_id = $4
where source.organization_id = $1
  and source.school_year_id = $2
  and source.student_id = $3;

-- name: CountStudentSessionNonParticipationConflicts :one
select count(*)::bigint
from session_non_participations source
join session_non_participations target
  on target.organization_id = source.organization_id
 and target.school_year_id = source.school_year_id
 and target.program_id = source.program_id
 and target.session_id = source.session_id
 and target.student_id = $4
where source.organization_id = $1
  and source.school_year_id = $2
  and source.student_id = $3;

-- name: MoveStudentProgramMemberships :execrows
update program_memberships
set student_id = $4
where organization_id = $1 and school_year_id = $2 and student_id = $3;

-- name: MoveStudentSessionNonParticipations :execrows
update session_non_participations
set student_id = $4
where organization_id = $1 and school_year_id = $2 and student_id = $3;

-- name: ListRecentStudentCorrectionCounts :many
select object_id, count(*)::bigint as correction_count
from audit_log
where organization_id = $1
  and school_year_id = $2
  and object_type = 'student'
  and action in ('student_admin_correction', 'placeholder_student_create', 'student_reconciliation')
  and occurred_at >= now() - interval '30 days'
  and object_id is not null
group by object_id
having count(*) >= 2
order by correction_count desc, object_id;
