-- +goose Up

-- Reject only new writes: legacy channel values and their effective answers remain intact.
-- +goose StatementBegin
create function prevent_retired_preference_submission() returns trigger
language plpgsql
as $$
begin
    if new.channel = 'student_code' then
        raise exception 'student_code preference submissions are retired'
            using errcode = 'check_violation';
    end if;
    return new;
end;
$$;
-- +goose StatementEnd

revoke all on function prevent_retired_preference_submission() from public;
grant execute on function prevent_retired_preference_submission() to miniclass_app;
create trigger interest_profile_submissions_channel_guard
before insert on interest_profile_submissions
for each row execute function prevent_retired_preference_submission();
create trigger ranked_choice_submissions_channel_guard
before insert on ranked_choice_submissions
for each row execute function prevent_retired_preference_submission();

-- +goose StatementBegin
create or replace function public.purge_school_year(
    target_organization_id public.xid20,
    target_school_year_id public.xid20,
    purge_actor_id public.xid20
) returns void language plpgsql security definer as
$$
declare
    current_state text;
    current_organization_id public.xid20;
    target_schema text := current_schema();
begin
    -- The application runs this function from public, while isolation tests
    -- run it from a per-test schema. Derive that schema before pinning the
    -- SECURITY DEFINER search path so both environments address their own
    -- tenant tables without accepting caller-controlled objects afterward.
    perform set_config('search_path', format('%I, pg_catalog', target_schema), true);
    current_organization_id := current_setting('app.organization_id')::public.xid20;
    if current_organization_id <> target_organization_id then
        raise exception 'school year not found' using errcode = 'P0002';
    end if;

    select sy.state::text
      into current_state
      from school_years sy
     where sy.id = target_school_year_id
       and sy.organization_id = target_organization_id
     for update;
    if not found then
        raise exception 'school year not found' using errcode = 'P0002';
    end if;
    if current_state <> 'closed' then
        raise exception 'only a closed school year can be purged'
            using errcode = 'P0001';
    end if;

    perform set_config('app.school_year_purge', 'true', true);
    perform set_config('app.school_year_purge_id', target_school_year_id::text, true);

    delete from interest_profile_responses
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from ranked_choice_responses
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from interest_profile_survey_audience_snapshots
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from interest_profile_survey_audience_students
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from interest_profile_submissions
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from ranked_choice_submissions
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from interest_profile_survey_questions
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from interest_profile_survey_scale_options
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from interest_profile_surveys
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from session_non_participations
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from program_memberships
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from offerings
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from meeting_dates
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from session_objective_weight_overrides
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from sessions
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from program_objective_weights
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from interest_areas
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from programs
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from guardian_relationships
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from guardian_onboarding_consents
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from guardian_invitation_contacts
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from adult_account_links
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from access_tokens
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from students
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from adults
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from grade_levels
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from homerooms
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from audit_log
     where organization_id = target_organization_id and school_year_id = target_school_year_id;

    update school_years sy
       set state = 'purged',
           purged_by_user_id = purge_actor_id,
           purged_at = clock_timestamp(),
           updated_at = clock_timestamp()
     where sy.id = target_school_year_id
       and sy.organization_id = target_organization_id
     ;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function public.prevent_placeholder_student_state_with_dependencies() returns trigger
language plpgsql
as $$
begin
    if new.is_placeholder and not old.is_placeholder and (
        exists (select 1 from guardian_relationships where organization_id = new.organization_id and school_year_id = new.school_year_id and student_id = new.id)
        or exists (select 1 from interest_profile_submissions where organization_id = new.organization_id and school_year_id = new.school_year_id and student_id = new.id)
        or exists (select 1 from ranked_choice_submissions where organization_id = new.organization_id and school_year_id = new.school_year_id and student_id = new.id)
        or exists (select 1 from interest_profile_survey_audience_students where organization_id = new.organization_id and school_year_id = new.school_year_id and student_id = new.id)
        or exists (select 1 from interest_profile_survey_audience_snapshots where organization_id = new.organization_id and school_year_id = new.school_year_id and student_id = new.id)
    ) then
        raise exception 'a student with guardian or preference records cannot become a placeholder'
            using errcode = 'check_violation';
    end if;
    return new;
end;
$$;
-- +goose StatementEnd

drop table interest_profile_survey_access_codes;
drop table ranked_choice_access_codes;

-- +goose Down

-- Rollback restores the schema, never the discarded credentials.
drop trigger interest_profile_submissions_channel_guard on interest_profile_submissions;
drop trigger ranked_choice_submissions_channel_guard on ranked_choice_submissions;
drop function prevent_retired_preference_submission();

create table interest_profile_survey_access_codes (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    survey_id public.xid20 not null,
    student_id public.xid20 not null,
    code_hash text not null,
    issued_at timestamptz not null default now(),
    revoked_at timestamptz,
    constraint interest_profile_survey_access_codes_survey_fk foreign key (survey_id, organization_id, school_year_id, program_id)
        references interest_profile_surveys (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint interest_profile_survey_access_codes_snapshot_fk foreign key (survey_id, organization_id, school_year_id, program_id, student_id)
        references interest_profile_survey_audience_snapshots (survey_id, organization_id, school_year_id, program_id, student_id) on delete cascade,
    constraint interest_profile_survey_access_codes_id_organization_key unique (id, organization_id),
    constraint interest_profile_survey_access_codes_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint interest_profile_survey_access_codes_hash_unique unique (organization_id, survey_id, code_hash),
    constraint interest_profile_survey_access_codes_hash_check check (btrim(code_hash) <> '')
);


create index interest_profile_survey_access_codes_lookup_idx
    on interest_profile_survey_access_codes (organization_id, survey_id, student_id)
    where revoked_at is null;

alter table interest_profile_survey_access_codes enable row level security;
alter table interest_profile_survey_access_codes force row level security;
create policy interest_profile_survey_access_codes_tenant_isolation on interest_profile_survey_access_codes
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger interest_profile_survey_access_codes_closed_year_guard before insert or update or delete on interest_profile_survey_access_codes for each row execute function public.prevent_closed_school_year_mutation();

create table ranked_choice_access_codes (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    session_id public.xid20 not null,
    student_id public.xid20 not null,
    code_hash text not null,
    issued_at timestamptz not null default now(),
    revoked_at timestamptz,
    constraint ranked_choice_access_codes_session_fk foreign key (session_id, organization_id, school_year_id, program_id)
        references sessions (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint ranked_choice_access_codes_membership_fk foreign key (organization_id, school_year_id, program_id, student_id)
        references program_memberships (organization_id, school_year_id, program_id, student_id) on delete restrict,
    constraint ranked_choice_access_codes_id_organization_key unique (id, organization_id),
    constraint ranked_choice_access_codes_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint ranked_choice_access_codes_hash_check check (btrim(code_hash) <> '')
);

create unique index ranked_choice_access_codes_active_student_unique
    on ranked_choice_access_codes (organization_id, school_year_id, program_id, session_id, student_id)
    where revoked_at is null;
create unique index ranked_choice_access_codes_hash_unique
    on ranked_choice_access_codes (organization_id, school_year_id, program_id, session_id, code_hash);
create index ranked_choice_access_codes_lookup_idx
    on ranked_choice_access_codes (organization_id, school_year_id, program_id, session_id, code_hash)
    where revoked_at is null;

alter table ranked_choice_access_codes enable row level security;
alter table ranked_choice_access_codes force row level security;
create policy ranked_choice_access_codes_tenant_isolation on ranked_choice_access_codes
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger ranked_choice_access_codes_closed_year_guard
before insert or update or delete on ranked_choice_access_codes
for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update on ranked_choice_access_codes to miniclass_app;


create trigger ranked_choice_access_codes_placeholder_guard
before insert or update on ranked_choice_access_codes
for each row execute function public.prevent_placeholder_student_dependency();
create trigger interest_profile_survey_access_codes_placeholder_guard
before insert or update on interest_profile_survey_access_codes
for each row execute function public.prevent_placeholder_student_dependency();
grant select, insert, update, delete on interest_profile_survey_access_codes, ranked_choice_access_codes to miniclass_app;

-- +goose StatementBegin
create or replace function public.purge_school_year(
    target_organization_id public.xid20,
    target_school_year_id public.xid20,
    purge_actor_id public.xid20
) returns void language plpgsql security definer as
$$
declare
    current_state text;
    current_organization_id public.xid20;
    target_schema text := current_schema();
begin
    -- The application runs this function from public, while isolation tests
    -- run it from a per-test schema. Derive that schema before pinning the
    -- SECURITY DEFINER search path so both environments address their own
    -- tenant tables without accepting caller-controlled objects afterward.
    perform set_config('search_path', format('%I, pg_catalog', target_schema), true);
    current_organization_id := current_setting('app.organization_id')::public.xid20;
    if current_organization_id <> target_organization_id then
        raise exception 'school year not found' using errcode = 'P0002';
    end if;

    select sy.state::text
      into current_state
      from school_years sy
     where sy.id = target_school_year_id
       and sy.organization_id = target_organization_id
     for update;
    if not found then
        raise exception 'school year not found' using errcode = 'P0002';
    end if;
    if current_state <> 'closed' then
        raise exception 'only a closed school year can be purged'
            using errcode = 'P0001';
    end if;

    perform set_config('app.school_year_purge', 'true', true);
    perform set_config('app.school_year_purge_id', target_school_year_id::text, true);

    delete from interest_profile_responses
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from ranked_choice_responses
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from interest_profile_survey_access_codes
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from interest_profile_survey_audience_snapshots
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from interest_profile_survey_audience_students
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from interest_profile_submissions
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from ranked_choice_submissions
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from interest_profile_survey_questions
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from interest_profile_survey_scale_options
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from interest_profile_surveys
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from ranked_choice_access_codes
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from session_non_participations
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from program_memberships
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from offerings
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from meeting_dates
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from session_objective_weight_overrides
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from sessions
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from program_objective_weights
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from interest_areas
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from programs
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from guardian_relationships
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from guardian_onboarding_consents
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from guardian_invitation_contacts
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from adult_account_links
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from access_tokens
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from students
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from adults
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from grade_levels
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from homerooms
     where organization_id = target_organization_id and school_year_id = target_school_year_id;
    delete from audit_log
     where organization_id = target_organization_id and school_year_id = target_school_year_id;

    update school_years sy
       set state = 'purged',
           purged_by_user_id = purge_actor_id,
           purged_at = clock_timestamp(),
           updated_at = clock_timestamp()
     where sy.id = target_school_year_id
       and sy.organization_id = target_organization_id
     ;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function public.prevent_placeholder_student_state_with_dependencies() returns trigger
language plpgsql
as $$
begin
    if new.is_placeholder and not old.is_placeholder and (
        exists (select 1 from guardian_relationships where organization_id = new.organization_id and school_year_id = new.school_year_id and student_id = new.id)
        or exists (select 1 from interest_profile_submissions where organization_id = new.organization_id and school_year_id = new.school_year_id and student_id = new.id)
        or exists (select 1 from ranked_choice_submissions where organization_id = new.organization_id and school_year_id = new.school_year_id and student_id = new.id)
        or exists (select 1 from interest_profile_survey_audience_students where organization_id = new.organization_id and school_year_id = new.school_year_id and student_id = new.id)
        or exists (select 1 from interest_profile_survey_audience_snapshots where organization_id = new.organization_id and school_year_id = new.school_year_id and student_id = new.id)
        or exists (select 1 from interest_profile_survey_access_codes where organization_id = new.organization_id and school_year_id = new.school_year_id and student_id = new.id)
        or exists (select 1 from ranked_choice_access_codes where organization_id = new.organization_id and school_year_id = new.school_year_id and student_id = new.id)
    ) then
        raise exception 'a student with guardian or preference records cannot become a placeholder'
            using errcode = 'check_violation';
    end if;
    return new;
end;
$$;
-- +goose StatementEnd
