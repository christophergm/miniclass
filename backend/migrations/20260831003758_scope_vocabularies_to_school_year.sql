-- +goose Up

-- The migrator owns these tables but they are FORCE RLS tables. Temporarily
-- remove the owner exception while reshaping and backfilling them so DDL and
-- the data-preserving transformation can see every tenant without inventing
-- an app.organization_id for the migration session.
alter table school_years no force row level security;
alter table students no force row level security;
alter table grade_levels no force row level security;
alter table homerooms no force row level security;

alter table grade_levels
    add column school_year_id public.xid20;

alter table homerooms
    add column school_year_id public.xid20;

alter table students
    drop constraint students_grade_level_fk,
    drop constraint students_homeroom_fk;

alter table grade_levels
    drop constraint grade_levels_id_organization_key;

alter table homerooms
    drop constraint homerooms_id_organization_key;

alter table grade_levels
    add constraint grade_levels_id_organization_year_key
        unique (id, organization_id, school_year_id),
    add constraint grade_levels_school_year_fk
        foreign key (school_year_id, organization_id)
        references school_years (id, organization_id) on delete cascade;

alter table homerooms
    add constraint homerooms_id_organization_year_key
        unique (id, organization_id, school_year_id),
    add constraint homerooms_school_year_fk
        foreign key (school_year_id, organization_id)
        references school_years (id, organization_id) on delete cascade;

-- The existing indexes enforce uniqueness across the entire organization. They
-- must be removed before fan-out, because each source row is copied once per
-- school year and therefore intentionally duplicates its code/name/ordinal.
drop index grade_levels_code_idx;
drop index grade_levels_ordinal_idx;
drop index grade_levels_picker_idx;
drop index homerooms_name_idx;
drop index homerooms_external_identifier_idx;
drop index homerooms_picker_idx;

insert into grade_levels (
    id, organization_id, school_year_id, code, label, ordinal, retired_at, created_at, updated_at
)
select public.xid(), grade_levels.organization_id, school_years.id,
       grade_levels.code, grade_levels.label, grade_levels.ordinal,
       grade_levels.retired_at, grade_levels.created_at, grade_levels.updated_at
from grade_levels
join school_years on school_years.organization_id = grade_levels.organization_id
where grade_levels.school_year_id is null;

insert into homerooms (
    id, organization_id, school_year_id, name, external_identifier, retired_at, created_at, updated_at
)
select public.xid(), homerooms.organization_id, school_years.id,
       homerooms.name, homerooms.external_identifier, homerooms.retired_at,
       homerooms.created_at, homerooms.updated_at
from homerooms
join school_years on school_years.organization_id = homerooms.organization_id
where homerooms.school_year_id is null;

-- Repointing existing students is part of the atomic schema transformation,
-- including students in closed years. It is not an organizer mutation and must
-- not require a per-year reopen reason.
alter table students disable trigger students_closed_year_guard;

update students
set grade_level_id = copied.id
from grade_levels original, grade_levels copied
where original.id = students.grade_level_id
  and original.organization_id = students.organization_id
  and original.school_year_id is null
  and copied.organization_id = students.organization_id
  and copied.school_year_id = students.school_year_id
  and lower(copied.code) = lower(original.code);

update students
set homeroom_id = copied.id
from homerooms original, homerooms copied
where original.id = students.homeroom_id
  and original.organization_id = students.organization_id
  and original.school_year_id is null
  and copied.organization_id = students.organization_id
  and copied.school_year_id = students.school_year_id
  and lower(copied.name) = lower(original.name);

alter table students enable trigger students_closed_year_guard;

-- The old organization-scoped rows are intentionally retained until every
-- student reference has been reconstructed against its year's copy.
-- +goose StatementBegin
do $$
begin
    if exists (
        select 1
        from students
        left join grade_levels on grade_levels.id = students.grade_level_id
            and grade_levels.organization_id = students.organization_id
            and grade_levels.school_year_id = students.school_year_id
        where students.grade_level_id is not null
          and grade_levels.id is null
    ) then
        raise exception 'unable to backfill student grade level references';
    end if;

    if exists (
        select 1
        from students
        left join homerooms on homerooms.id = students.homeroom_id
            and homerooms.organization_id = students.organization_id
            and homerooms.school_year_id = students.school_year_id
        where homerooms.id is null
    ) then
        raise exception 'unable to backfill student homeroom references';
    end if;

end;
$$;
-- +goose StatementEnd

delete from grade_levels where school_year_id is null;
delete from homerooms where school_year_id is null;

-- Every original row has now been replaced by copies assigned to a year.
-- Keep these checks after deleting the source rows; before that point the
-- intentionally retained originals still have NULL school_year_id.
-- +goose StatementBegin
do $$
begin
    if exists (
        select 1 from grade_levels where school_year_id is null
    ) then
        raise exception 'unable to backfill grade levels without a school year';
    end if;

    if exists (
        select 1 from homerooms where school_year_id is null
    ) then
        raise exception 'unable to backfill homerooms without a school year';
    end if;
end;
$$;
-- +goose StatementEnd

alter table grade_levels
    alter column school_year_id set not null;

alter table homerooms
    alter column school_year_id set not null;

create unique index grade_levels_code_idx
    on grade_levels (organization_id, school_year_id, lower(code));
create unique index grade_levels_ordinal_idx
    on grade_levels (organization_id, school_year_id, ordinal);
create index grade_levels_picker_idx
    on grade_levels (organization_id, school_year_id, ordinal, id)
    where retired_at is null;

create unique index homerooms_name_idx
    on homerooms (organization_id, school_year_id, lower(name));
create unique index homerooms_external_identifier_idx
    on homerooms (organization_id, school_year_id, external_identifier)
    where external_identifier is not null;
create index homerooms_picker_idx
    on homerooms (organization_id, school_year_id, lower(name), id)
    where retired_at is null;

create trigger grade_levels_closed_year_guard
before insert or update or delete on grade_levels
for each row execute function public.prevent_closed_school_year_mutation();

create trigger homerooms_closed_year_guard
before insert or update or delete on homerooms
for each row execute function public.prevent_closed_school_year_mutation();

alter table students
    add constraint students_grade_level_fk foreign key (grade_level_id, organization_id, school_year_id)
        references grade_levels (id, organization_id, school_year_id),
    add constraint students_homeroom_fk foreign key (homeroom_id, organization_id, school_year_id)
        references homerooms (id, organization_id, school_year_id);

alter table school_years force row level security;
alter table students force row level security;
alter table grade_levels force row level security;
alter table homerooms force row level security;

-- +goose Down

alter table school_years no force row level security;
alter table students no force row level security;
alter table grade_levels no force row level security;
alter table homerooms no force row level security;

alter table students
    drop constraint students_grade_level_fk,
    drop constraint students_homeroom_fk;

drop trigger if exists homerooms_closed_year_guard on homerooms;
drop trigger if exists grade_levels_closed_year_guard on grade_levels;

alter table grade_levels
    drop constraint grade_levels_school_year_fk;

alter table homerooms
    drop constraint homerooms_school_year_fk;

-- Down also repoints students across closed years as part of the schema
-- reversal, not as an organizer mutation.
alter table students disable trigger students_closed_year_guard;

with ranked as (
    select id,
           first_value(id) over (
               partition by organization_id, lower(code)
               order by created_at, id
           ) as keep_id
    from grade_levels
)
update students
set grade_level_id = ranked.keep_id
from ranked
where ranked.id = students.grade_level_id;

with ranked as (
    select id,
           first_value(id) over (
               partition by organization_id, lower(name)
               order by created_at, id
           ) as keep_id
    from homerooms
)
update students
set homeroom_id = ranked.keep_id
from ranked
where ranked.id = students.homeroom_id;

alter table students enable trigger students_closed_year_guard;

delete from grade_levels duplicate
using (
    select id,
           row_number() over (
               partition by organization_id, lower(code)
               order by created_at, id
           ) as row_number
    from grade_levels
) ranked
where ranked.id = duplicate.id
  and ranked.row_number > 1;

delete from homerooms duplicate
using (
    select id,
           row_number() over (
               partition by organization_id, lower(name)
               order by created_at, id
           ) as row_number
    from homerooms
) ranked
where ranked.id = duplicate.id
  and ranked.row_number > 1;

drop index grade_levels_code_idx;
drop index grade_levels_ordinal_idx;
drop index grade_levels_picker_idx;
create unique index grade_levels_code_idx on grade_levels (organization_id, lower(code));
create unique index grade_levels_ordinal_idx on grade_levels (organization_id, ordinal);
create index grade_levels_picker_idx on grade_levels (organization_id, ordinal, id)
    where retired_at is null;

drop index homerooms_name_idx;
drop index homerooms_external_identifier_idx;
drop index homerooms_picker_idx;
create unique index homerooms_name_idx on homerooms (organization_id, lower(name));
create unique index homerooms_external_identifier_idx
    on homerooms (organization_id, external_identifier)
    where external_identifier is not null;
create index homerooms_picker_idx on homerooms (organization_id, lower(name), id)
    where retired_at is null;

alter table grade_levels
    drop constraint grade_levels_id_organization_year_key,
    add constraint grade_levels_id_organization_key unique (id, organization_id),
    drop column school_year_id;

alter table homerooms
    drop constraint homerooms_id_organization_year_key,
    add constraint homerooms_id_organization_key unique (id, organization_id),
    drop column school_year_id;

alter table students
    add constraint students_grade_level_fk foreign key (grade_level_id, organization_id)
        references grade_levels (id, organization_id),
    add constraint students_homeroom_fk foreign key (homeroom_id, organization_id)
        references homerooms (id, organization_id);

alter table school_years force row level security;
alter table students force row level security;
alter table grade_levels force row level security;
alter table homerooms force row level security;
