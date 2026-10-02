-- +goose Up

alter table program_memberships
    add column origin text not null default 'manual',
    add constraint program_memberships_origin_check check (origin in ('manual', 'automatic'));

alter table programs
    add column auto_assignment_enabled boolean not null default false,
    add column auto_assignment_grade_level_ids public.xid20[] not null default '{}',
    add column auto_assignment_homeroom_ids public.xid20[] not null default '{}';

-- +goose StatementBegin
create or replace function public.prevent_auto_assignment_grade_level_retirement()
returns trigger
language plpgsql
as $$
begin
    if new.retired_at is not null and old.retired_at is null and exists (
        select 1 from programs
        where organization_id = new.organization_id
          and school_year_id = new.school_year_id
          and auto_assignment_enabled
          and new.id = any(auto_assignment_grade_level_ids)
    ) then
        raise exception 'grade level is selected by an enabled automatic-membership rule'
            using errcode = '23503';
    end if;
    return new;
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function public.prevent_auto_assignment_homeroom_retirement()
returns trigger
language plpgsql
as $$
begin
    if new.retired_at is not null and old.retired_at is null and exists (
        select 1 from programs
        where organization_id = new.organization_id
          and school_year_id = new.school_year_id
          and auto_assignment_enabled
          and new.id = any(auto_assignment_homeroom_ids)
    ) then
        raise exception 'homeroom is selected by an enabled automatic-membership rule'
            using errcode = '23503';
    end if;
    return new;
end;
$$;
-- +goose StatementEnd

create trigger grade_levels_auto_assignment_retirement_guard
before update of retired_at on grade_levels
for each row execute function public.prevent_auto_assignment_grade_level_retirement();
create trigger homerooms_auto_assignment_retirement_guard
before update of retired_at on homerooms
for each row execute function public.prevent_auto_assignment_homeroom_retirement();

-- +goose Down

drop trigger if exists homerooms_auto_assignment_retirement_guard on homerooms;
drop trigger if exists grade_levels_auto_assignment_retirement_guard on grade_levels;
drop function if exists public.prevent_auto_assignment_homeroom_retirement();
drop function if exists public.prevent_auto_assignment_grade_level_retirement();
alter table programs
    drop column if exists auto_assignment_homeroom_ids,
    drop column if exists auto_assignment_grade_level_ids,
    drop column if exists auto_assignment_enabled;
alter table program_memberships drop constraint if exists program_memberships_origin_check;
alter table program_memberships drop column if exists origin;
