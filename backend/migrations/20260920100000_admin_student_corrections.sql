-- +goose Up

alter table students
    add column is_placeholder boolean not null default false,
    add column provenance text not null default 'legacy';

alter table students
    add constraint students_placeholder_grade_check
        check (not is_placeholder or grade_level_id is not null),
    add constraint students_provenance_check
        check (provenance in ('legacy', 'import', 'guardian', 'administrator_correction', 'placeholder'));

create index students_placeholder_idx
    on students (organization_id, school_year_id, is_placeholder, id)
    where deleted_at is null;

-- +goose StatementBegin
create or replace function public.prevent_placeholder_student_dependency() returns trigger
language plpgsql
as $$
begin
    if exists (
        select 1
        from students
        where id = new.student_id
          and organization_id = new.organization_id
          and school_year_id = new.school_year_id
          and is_placeholder
          and deleted_at is null
    ) then
        raise exception 'placeholder students cannot receive guardian, preference, or access records'
            using errcode = 'check_violation';
    end if;
    return new;
end;
$$;
-- +goose StatementEnd

create trigger guardian_relationships_placeholder_guard
before insert or update on guardian_relationships
for each row execute function public.prevent_placeholder_student_dependency();

create trigger interest_profile_submissions_placeholder_guard
before insert or update on interest_profile_submissions
for each row execute function public.prevent_placeholder_student_dependency();

create trigger ranked_choice_submissions_placeholder_guard
before insert or update on ranked_choice_submissions
for each row execute function public.prevent_placeholder_student_dependency();

create trigger ranked_choice_access_codes_placeholder_guard
before insert or update on ranked_choice_access_codes
for each row execute function public.prevent_placeholder_student_dependency();

create trigger interest_profile_survey_audience_students_placeholder_guard
before insert or update on interest_profile_survey_audience_students
for each row execute function public.prevent_placeholder_student_dependency();

create trigger interest_profile_survey_audience_snapshots_placeholder_guard
before insert or update on interest_profile_survey_audience_snapshots
for each row execute function public.prevent_placeholder_student_dependency();

create trigger interest_profile_survey_access_codes_placeholder_guard
before insert or update on interest_profile_survey_access_codes
for each row execute function public.prevent_placeholder_student_dependency();

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

create trigger students_placeholder_state_guard
before update on students
for each row execute function public.prevent_placeholder_student_state_with_dependencies();

grant execute on function public.prevent_placeholder_student_dependency() to miniclass_app;
grant execute on function public.prevent_placeholder_student_state_with_dependencies() to miniclass_app;

-- +goose Down

revoke execute on function public.prevent_placeholder_student_state_with_dependencies() from miniclass_app;
revoke execute on function public.prevent_placeholder_student_dependency() from miniclass_app;
drop trigger if exists students_placeholder_state_guard on students;
drop trigger if exists interest_profile_survey_access_codes_placeholder_guard on interest_profile_survey_access_codes;
drop trigger if exists interest_profile_survey_audience_snapshots_placeholder_guard on interest_profile_survey_audience_snapshots;
drop trigger if exists interest_profile_survey_audience_students_placeholder_guard on interest_profile_survey_audience_students;
drop trigger if exists ranked_choice_access_codes_placeholder_guard on ranked_choice_access_codes;
drop trigger if exists ranked_choice_submissions_placeholder_guard on ranked_choice_submissions;
drop trigger if exists interest_profile_submissions_placeholder_guard on interest_profile_submissions;
drop trigger if exists guardian_relationships_placeholder_guard on guardian_relationships;
drop function if exists public.prevent_placeholder_student_state_with_dependencies();
drop function if exists public.prevent_placeholder_student_dependency();
drop index if exists students_placeholder_idx;
alter table students
    drop constraint if exists students_provenance_check,
    drop constraint if exists students_placeholder_grade_check,
    drop column if exists provenance,
    drop column if exists is_placeholder;
