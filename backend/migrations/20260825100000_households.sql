-- +goose Up

create type guardian_relationship_type as enum (
    'parent',
    'guardian',
    'grandparent',
    'other'
);

create table households (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    display_name text not null,
    deleted_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint households_school_year_fk foreign key (school_year_id, organization_id)
        references school_years (id, organization_id) on delete cascade,
    constraint households_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint households_display_name_check check (btrim(display_name) <> '')
);

create index households_school_year_name_idx
    on households (organization_id, school_year_id, lower(display_name), id)
    where deleted_at is null;

alter table households enable row level security;
alter table households force row level security;

create policy households_tenant_isolation on households
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger households_set_updated_at
before update on households
for each row execute function public.set_updated_at();

create trigger households_closed_year_guard
before insert or update or delete on households
for each row execute function public.prevent_closed_school_year_mutation();

create table household_students (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    household_id public.xid20 not null,
    student_id public.xid20 not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint household_students_school_year_fk foreign key (school_year_id, organization_id)
        references school_years (id, organization_id) on delete cascade,
    constraint household_students_household_fk foreign key (household_id, organization_id, school_year_id)
        references households (id, organization_id, school_year_id) on delete cascade,
    constraint household_students_student_fk foreign key (student_id, organization_id, school_year_id)
        references students (id, organization_id, school_year_id) on delete cascade,
    constraint household_students_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint household_students_unique_membership unique (organization_id, school_year_id, household_id, student_id)
);

create index household_students_household_idx
    on household_students (organization_id, school_year_id, household_id, student_id);

create index household_students_student_idx
    on household_students (organization_id, school_year_id, student_id, household_id);

alter table household_students enable row level security;
alter table household_students force row level security;

create policy household_students_tenant_isolation on household_students
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger household_students_set_updated_at
before update on household_students
for each row execute function public.set_updated_at();

create trigger household_students_closed_year_guard
before insert or update or delete on household_students
for each row execute function public.prevent_closed_school_year_mutation();

create table household_adults (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    household_id public.xid20 not null,
    adult_id public.xid20 not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint household_adults_school_year_fk foreign key (school_year_id, organization_id)
        references school_years (id, organization_id) on delete cascade,
    constraint household_adults_household_fk foreign key (household_id, organization_id, school_year_id)
        references households (id, organization_id, school_year_id) on delete cascade,
    constraint household_adults_adult_fk foreign key (adult_id, organization_id, school_year_id)
        references adults (id, organization_id, school_year_id) on delete cascade,
    constraint household_adults_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint household_adults_unique_membership unique (organization_id, school_year_id, household_id, adult_id)
);

create index household_adults_household_idx
    on household_adults (organization_id, school_year_id, household_id, adult_id);

create index household_adults_adult_idx
    on household_adults (organization_id, school_year_id, adult_id, household_id);

alter table household_adults enable row level security;
alter table household_adults force row level security;

create policy household_adults_tenant_isolation on household_adults
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger household_adults_set_updated_at
before update on household_adults
for each row execute function public.set_updated_at();

create trigger household_adults_closed_year_guard
before insert or update or delete on household_adults
for each row execute function public.prevent_closed_school_year_mutation();

create table guardian_relationships (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    adult_id public.xid20 not null,
    student_id public.xid20 not null,
    relationship_type guardian_relationship_type not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint guardian_relationships_school_year_fk foreign key (school_year_id, organization_id)
        references school_years (id, organization_id) on delete cascade,
    constraint guardian_relationships_adult_fk foreign key (adult_id, organization_id, school_year_id)
        references adults (id, organization_id, school_year_id) on delete cascade,
    constraint guardian_relationships_student_fk foreign key (student_id, organization_id, school_year_id)
        references students (id, organization_id, school_year_id) on delete cascade,
    constraint guardian_relationships_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint guardian_relationships_unique_pair unique (organization_id, school_year_id, adult_id, student_id)
);

create index guardian_relationships_student_idx
    on guardian_relationships (organization_id, school_year_id, student_id, adult_id);

create index guardian_relationships_adult_idx
    on guardian_relationships (organization_id, school_year_id, adult_id, student_id);

alter table guardian_relationships enable row level security;
alter table guardian_relationships force row level security;

create policy guardian_relationships_tenant_isolation on guardian_relationships
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger guardian_relationships_set_updated_at
before update on guardian_relationships
for each row execute function public.set_updated_at();

create trigger guardian_relationships_closed_year_guard
before insert or update or delete on guardian_relationships
for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update, delete on households to miniclass_app;
grant select, insert, update, delete on household_students to miniclass_app;
grant select, insert, update, delete on household_adults to miniclass_app;
grant select, insert, update, delete on guardian_relationships to miniclass_app;

-- +goose Down

revoke all privileges on guardian_relationships from miniclass_app;
revoke all privileges on household_adults from miniclass_app;
revoke all privileges on household_students from miniclass_app;
revoke all privileges on households from miniclass_app;
drop trigger if exists guardian_relationships_closed_year_guard on guardian_relationships;
drop trigger if exists guardian_relationships_set_updated_at on guardian_relationships;
drop policy if exists guardian_relationships_tenant_isolation on guardian_relationships;
drop index if exists guardian_relationships_adult_idx;
drop index if exists guardian_relationships_student_idx;
drop table guardian_relationships;
drop trigger if exists household_adults_closed_year_guard on household_adults;
drop trigger if exists household_adults_set_updated_at on household_adults;
drop policy if exists household_adults_tenant_isolation on household_adults;
drop index if exists household_adults_adult_idx;
drop index if exists household_adults_household_idx;
drop table household_adults;
drop trigger if exists household_students_closed_year_guard on household_students;
drop trigger if exists household_students_set_updated_at on household_students;
drop policy if exists household_students_tenant_isolation on household_students;
drop index if exists household_students_student_idx;
drop index if exists household_students_household_idx;
drop table household_students;
drop trigger if exists households_closed_year_guard on households;
drop trigger if exists households_set_updated_at on households;
drop policy if exists households_tenant_isolation on households;
drop index if exists households_school_year_name_idx;
drop table households;
drop type guardian_relationship_type;
