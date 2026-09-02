-- name: CreateSession :one
insert into sessions (organization_id, school_year_id, program_id, name)
values ($1, $2, $3, $4)
returning id, organization_id, school_year_id, program_id, name, state, draft_assignments_stale, ranked_choice_enabled, ranked_choice_rank_depth, ranked_choice_deadline, created_at, updated_at;

-- name: ListSessions :many
select id, organization_id, school_year_id, program_id, name, state, draft_assignments_stale, ranked_choice_enabled, ranked_choice_rank_depth, ranked_choice_deadline, created_at, updated_at
from sessions
where sessions.organization_id = $1 and sessions.school_year_id = $2 and sessions.program_id = $3
order by (select min(meeting_date) from meeting_dates where meeting_dates.session_id = sessions.id and meeting_dates.organization_id = sessions.organization_id), lower(sessions.name), sessions.id;

-- name: GetSession :one
select id, organization_id, school_year_id, program_id, name, state, draft_assignments_stale, ranked_choice_enabled, ranked_choice_rank_depth, ranked_choice_deadline, created_at, updated_at
from sessions
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4;

-- name: GetSessionForUpdate :one
select id, organization_id, school_year_id, program_id, name, state, draft_assignments_stale, ranked_choice_enabled, ranked_choice_rank_depth, ranked_choice_deadline, created_at, updated_at
from sessions
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4
for update;

-- name: UpdateSession :one
update sessions
set name = $2,
    ranked_choice_enabled = $6,
    ranked_choice_rank_depth = $7,
    ranked_choice_deadline = $8
where id = $1 and organization_id = $3 and school_year_id = $4 and program_id = $5
returning id, organization_id, school_year_id, program_id, name, state, draft_assignments_stale, ranked_choice_enabled, ranked_choice_rank_depth, ranked_choice_deadline, created_at, updated_at;

-- name: UpdateSessionLifecycle :one
update sessions
set state = $2, draft_assignments_stale = $3
where id = $1 and organization_id = $4 and school_year_id = $5 and program_id = $6
returning id, organization_id, school_year_id, program_id, name, state, draft_assignments_stale, ranked_choice_enabled, ranked_choice_rank_depth, ranked_choice_deadline, created_at, updated_at;

-- name: UpdateSessionRankedChoiceDeadline :one
update sessions
set ranked_choice_deadline = $2
where id = $1 and organization_id = $3 and school_year_id = $4 and program_id = $5
returning id, organization_id, school_year_id, program_id, name, state, draft_assignments_stale, ranked_choice_enabled, ranked_choice_rank_depth, ranked_choice_deadline, created_at, updated_at;

-- name: DeleteSession :execrows
delete from sessions
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4;

-- name: CreateMeetingDate :one
insert into meeting_dates (organization_id, school_year_id, program_id, session_id, meeting_date)
values ($1, $2, $3, $4, $5)
returning id, organization_id, school_year_id, program_id, session_id, meeting_date, created_at, updated_at;

-- name: ListMeetingDates :many
select id, organization_id, school_year_id, program_id, session_id, meeting_date, created_at, updated_at
from meeting_dates
where organization_id = $1 and school_year_id = $2 and program_id = $3 and session_id = $4
order by meeting_date, id;

-- name: GetMeetingDate :one
select id, organization_id, school_year_id, program_id, session_id, meeting_date, created_at, updated_at
from meeting_dates
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4 and session_id = $5;

-- name: UpdateMeetingDate :one
update meeting_dates
set meeting_date = $2
where id = $1 and organization_id = $3 and school_year_id = $4 and program_id = $5 and session_id = $6
returning id, organization_id, school_year_id, program_id, session_id, meeting_date, created_at, updated_at;

-- name: DeleteMeetingDate :execrows
delete from meeting_dates
where id = $1 and organization_id = $2 and school_year_id = $3 and program_id = $4 and session_id = $5;

-- name: ListAllSessionsForRegistry :many
select id, organization_id, school_year_id, program_id, name, state, draft_assignments_stale, ranked_choice_enabled, ranked_choice_rank_depth, ranked_choice_deadline, created_at, updated_at
from sessions where sessions.organization_id = $1
order by school_year_id, program_id,
    (select min(meeting_date) from meeting_dates where meeting_dates.session_id = sessions.id and meeting_dates.organization_id = sessions.organization_id),
    lower(sessions.name), sessions.id;

-- name: FindSessionForRegistry :one
select id, organization_id, school_year_id, program_id, name, state, draft_assignments_stale, ranked_choice_enabled, ranked_choice_rank_depth, ranked_choice_deadline, created_at, updated_at
from sessions where id = $1 and organization_id = $2;

-- name: UpdateSessionForRegistry :execrows
update sessions set name = $3 where id = $1 and organization_id = $2;

-- name: DeleteSessionForRegistry :execrows
delete from sessions where id = $1 and organization_id = $2;

-- name: ListAllMeetingDatesForRegistry :many
select id, organization_id, school_year_id, program_id, session_id, meeting_date, created_at, updated_at
from meeting_dates where organization_id = $1 order by school_year_id, program_id, session_id, meeting_date, id;

-- name: FindMeetingDateForRegistry :one
select id, organization_id, school_year_id, program_id, session_id, meeting_date, created_at, updated_at
from meeting_dates where id = $1 and organization_id = $2;

-- name: UpdateMeetingDateForRegistry :execrows
update meeting_dates set meeting_date = $3 where id = $1 and organization_id = $2;

-- name: DeleteMeetingDateForRegistry :execrows
delete from meeting_dates where id = $1 and organization_id = $2;
