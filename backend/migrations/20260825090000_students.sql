-- +goose Up

create table students (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    legal_given_name text not null,
    legal_family_name text not null,
    preferred_given_name text,
    grade_level_id public.xid20 not null,
    homeroom_id public.xid20 not null,
    external_identifier text,
    prior_year_student_id public.xid20,
    deleted_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint students_school_year_fk foreign key (school_year_id, organization_id)
        references school_years (id, organization_id) on delete cascade,
    constraint students_grade_level_fk foreign key (grade_level_id, organization_id)
        references grade_levels (id, organization_id),
    constraint students_homeroom_fk foreign key (homeroom_id, organization_id)
        references homerooms (id, organization_id),
    constraint students_prior_year_fk foreign key (prior_year_student_id, organization_id)
        references students (id, organization_id) on delete set null (prior_year_student_id),
    constraint students_id_organization_key unique (id, organization_id),
    constraint students_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint students_legal_given_name_check check (btrim(legal_given_name) <> ''),
    constraint students_legal_family_name_check check (btrim(legal_family_name) <> ''),
    constraint students_preferred_given_name_check check (preferred_given_name is null or btrim(preferred_given_name) <> ''),
    constraint students_external_identifier_check check (external_identifier is null or btrim(external_identifier) <> '')
);

create index students_school_year_name_idx
    on students (organization_id, school_year_id, legal_family_name, legal_given_name, id)
    where deleted_at is null;

create index students_school_year_grade_idx
    on students (organization_id, school_year_id, grade_level_id, id)
    where deleted_at is null;

create index students_school_year_homeroom_idx
    on students (organization_id, school_year_id, homeroom_id, id)
    where deleted_at is null;

create unique index students_external_identifier_unique_idx
    on students (organization_id, school_year_id, external_identifier)
    where deleted_at is null and external_identifier is not null;

alter table students enable row level security;
alter table students force row level security;

create policy students_tenant_isolation on students
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger students_set_updated_at
before update on students
for each row execute function public.set_updated_at();

create trigger students_closed_year_guard
before insert or update or delete on students
for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update, delete on students to miniclass_app;

-- +goose Down

revoke all privileges on students from miniclass_app;
drop trigger if exists students_closed_year_guard on students;
drop trigger if exists students_set_updated_at on students;
drop policy if exists students_tenant_isolation on students;
drop index if exists students_external_identifier_unique_idx;
drop index if exists students_school_year_homeroom_idx;
drop index if exists students_school_year_grade_idx;
drop index if exists students_school_year_name_idx;
drop table students;
