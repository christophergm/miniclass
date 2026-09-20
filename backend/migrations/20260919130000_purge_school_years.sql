-- +goose NO TRANSACTION

-- +goose Up

alter type school_year_state add value if not exists 'purged';

alter table students
    drop constraint if exists students_prior_year_fk,
    drop column if exists prior_year_student_id;

alter table school_years
    add column purged_by_user_id public.xid20,
    add column purged_at timestamptz;

-- The purge setting is transaction-local and is set only inside the
-- security-definer purge function. It permits the function to remove rows
-- from a closed year without opening a general closed-year write bypass.
-- +goose StatementBegin
create or replace function public.prevent_closed_school_year_mutation() returns trigger language plpgsql as
$$
declare
    old_school_year_id public.xid20;
    new_school_year_id public.xid20;
    parent_state school_year_state;
    reopen_id text := nullif(btrim(coalesce(current_setting('app.school_year_reopen_id', true), '')), '');
    purge_id text := nullif(btrim(coalesce(current_setting('app.school_year_purge_id', true), '')), '');
    reopen_prepared boolean := coalesce(current_setting('app.school_year_reopen', true), '') = 'true'
        and nullif(btrim(coalesce(current_setting('app.school_year_reopen_reason', true), '')), '') is not null;
    purge_prepared boolean := coalesce(current_setting('app.school_year_purge', true), '') = 'true';
begin
    if tg_table_name = 'school_years' then
        if tg_op = 'INSERT' then
            return new;
        end if;
        if old.state = 'purged' then
            raise exception 'school year is purged'
                using errcode = 'P0001',
                      detail = 'a purged school year is unavailable for ordinary operation';
        end if;
        if tg_op = 'DELETE' then
            if old.state = 'closed' and not (purge_prepared and purge_id = old.id::text) then
                raise exception 'school year is closed'
                    using errcode = 'P0001',
                          detail = 'records in a closed school year are read-only',
                          hint = 'reopen the school year through the Owner-only lifecycle operation';
            end if;
            return old;
        end if;

        if old.state = 'closed' then
            if new.state = 'active' and reopen_prepared and reopen_id = old.id::text then
                return new;
            end if;
            if new.state = 'purged' and purge_prepared and purge_id = old.id::text then
                return new;
            end if;
            raise exception 'school year is closed'
                using errcode = 'P0001',
                      detail = 'records in a closed school year are read-only',
                      hint = 'reopen the school year through the Owner-only lifecycle operation';
        end if;
        return new;
    end if;

    if tg_op <> 'INSERT' then
        old_school_year_id := nullif(to_jsonb(old)->>'school_year_id', '')::public.xid20;
        if old_school_year_id is not null then
            select state into parent_state from school_years where id = old_school_year_id;
            if parent_state = 'purged' then
                raise exception 'school year is purged'
                    using errcode = 'P0001',
                          detail = 'a purged school year is unavailable for ordinary operation';
            end if;
            if parent_state = 'closed' and not (purge_prepared and purge_id = old_school_year_id::text)
                and not (reopen_prepared and reopen_id = old_school_year_id::text) then
                raise exception 'school year is closed'
                    using errcode = 'P0001',
                          detail = 'records in a closed school year are read-only',
                          hint = 'reopen the school year through the Owner-only lifecycle operation';
            end if;
        end if;
    end if;

    if tg_op <> 'DELETE' then
        new_school_year_id := nullif(to_jsonb(new)->>'school_year_id', '')::public.xid20;
        if new_school_year_id is not null then
            select state into parent_state from school_years where id = new_school_year_id;
            if parent_state = 'purged' then
                raise exception 'school year is purged'
                    using errcode = 'P0001',
                          detail = 'a purged school year is unavailable for ordinary operation';
            end if;
            if parent_state = 'closed' and not (purge_prepared and purge_id = new_school_year_id::text)
                and not (reopen_prepared and reopen_id = new_school_year_id::text) then
                raise exception 'school year is closed'
                    using errcode = 'P0001',
                          detail = 'records in a closed school year are read-only',
                          hint = 'reopen the school year through the Owner-only lifecycle operation';
            end if;
        end if;
    end if;

    if tg_op = 'DELETE' then
        return old;
    end if;
    return new;
end;
$$;
-- +goose StatementEnd

-- This function is the only database operation that removes a closed year.
-- The application still validates the Owner role and confirmation phrase; the
-- database additionally checks the tenant GUC and the closed-state boundary.
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

revoke all on function public.purge_school_year(public.xid20, public.xid20, public.xid20) from public;
grant execute on function public.purge_school_year(public.xid20, public.xid20, public.xid20) to miniclass_app;

-- +goose Down

revoke all on function public.purge_school_year(public.xid20, public.xid20, public.xid20) from miniclass_app;
drop function if exists public.purge_school_year(public.xid20, public.xid20, public.xid20);
alter table students
    add column if not exists prior_year_student_id public.xid20;
alter table students
    add constraint students_prior_year_fk foreign key (prior_year_student_id, organization_id)
        references students (id, organization_id) on delete set null (prior_year_student_id);
alter table school_years
    drop column if exists purged_at,
    drop column if exists purged_by_user_id;

-- PostgreSQL does not remove enum values. Keep the extra value in the type so
-- this down path remains safe and the migration can be applied again.
-- +goose StatementBegin
create or replace function public.prevent_closed_school_year_mutation() returns trigger language plpgsql as
$$
declare
    old_school_year_id public.xid20;
    new_school_year_id public.xid20;
    parent_state school_year_state;
    reopen_id text := nullif(btrim(coalesce(current_setting('app.school_year_reopen_id', true), '')), '');
    reopen_prepared boolean := coalesce(current_setting('app.school_year_reopen', true), '') = 'true'
        and nullif(btrim(coalesce(current_setting('app.school_year_reopen_reason', true), '')), '') is not null;
begin
    if tg_table_name = 'school_years' then
        if tg_op = 'INSERT' then
            return new;
        end if;
        if tg_op = 'DELETE' then
            if old.state = 'closed' then
                raise exception 'school year is closed'
                    using errcode = 'P0001',
                          detail = 'records in a closed school year are read-only',
                          hint = 'reopen the school year through the Owner-only lifecycle operation';
            end if;
            return old;
        end if;
        if old.state = 'closed' then
            if new.state = 'active' and reopen_prepared and reopen_id = old.id::text then
                return new;
            end if;
            raise exception 'school year is closed'
                using errcode = 'P0001',
                      detail = 'records in a closed school year are read-only',
                      hint = 'reopen the school year through the Owner-only lifecycle operation';
        end if;
        return new;
    end if;

    if tg_op <> 'INSERT' then
        old_school_year_id := nullif(to_jsonb(old)->>'school_year_id', '')::public.xid20;
        if old_school_year_id is not null then
            select state into parent_state from school_years where id = old_school_year_id;
            if parent_state = 'closed' and not (reopen_prepared and reopen_id = old_school_year_id::text) then
                raise exception 'school year is closed'
                    using errcode = 'P0001',
                          detail = 'records in a closed school year are read-only',
                          hint = 'reopen the school year through the Owner-only lifecycle operation';
            end if;
        end if;
    end if;
    if tg_op <> 'DELETE' then
        new_school_year_id := nullif(to_jsonb(new)->>'school_year_id', '')::public.xid20;
        if new_school_year_id is not null then
            select state into parent_state from school_years where id = new_school_year_id;
            if parent_state = 'closed' and not (reopen_prepared and reopen_id = new_school_year_id::text) then
                raise exception 'school year is closed'
                    using errcode = 'P0001',
                          detail = 'records in a closed school year are read-only',
                          hint = 'reopen the school year through the Owner-only lifecycle operation';
            end if;
        end if;
    end if;
    if tg_op = 'DELETE' then
        return old;
    end if;
    return new;
end;
$$;
-- +goose StatementEnd
