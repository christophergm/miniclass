-- +goose Up

alter table sessions drop constraint sessions_program_ordinal_unique;
alter table sessions drop constraint sessions_ordinal_check;
drop index if exists sessions_program_idx;
alter table sessions drop column ordinal;

create index sessions_program_idx on sessions (organization_id, school_year_id, program_id, id);

-- +goose Down

alter table sessions add column ordinal integer;

with ranked_sessions as (
    select
        sessions.id,
        row_number() over (
            partition by sessions.organization_id, sessions.school_year_id, sessions.program_id
            order by min(meeting_dates.meeting_date), lower(sessions.name), sessions.id
        )::integer as ordinal
    from sessions
    join meeting_dates on meeting_dates.session_id = sessions.id
        and meeting_dates.organization_id = sessions.organization_id
        and meeting_dates.school_year_id = sessions.school_year_id
        and meeting_dates.program_id = sessions.program_id
    group by sessions.id, sessions.organization_id, sessions.school_year_id, sessions.program_id, sessions.name
)
update sessions
set ordinal = ranked_sessions.ordinal
from ranked_sessions
where sessions.id = ranked_sessions.id;

alter table sessions alter column ordinal set not null;
alter table sessions add constraint sessions_ordinal_check check (ordinal > 0);
alter table sessions add constraint sessions_program_ordinal_unique
    unique (organization_id, school_year_id, program_id, ordinal);
drop index if exists sessions_program_idx;
create index sessions_program_idx on sessions (organization_id, school_year_id, program_id, ordinal, id);
