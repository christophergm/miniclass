-- +goose Up

create table offerings (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    session_id public.xid20 not null,
    name text not null,
    description text not null,
    minimum_viable_enrollment integer,
    capacity integer not null,
    min_grade_level_id public.xid20 not null,
    max_grade_level_id public.xid20 not null,
    location text not null default '',
    meeting_point text not null default '',
    meeting_instructions text not null default '',
    interest_area_id public.xid20,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint offerings_session_fk foreign key (session_id, organization_id, school_year_id, program_id)
        references sessions (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint offerings_min_grade_fk foreign key (min_grade_level_id, organization_id, school_year_id)
        references grade_levels (id, organization_id, school_year_id),
    constraint offerings_max_grade_fk foreign key (max_grade_level_id, organization_id, school_year_id)
        references grade_levels (id, organization_id, school_year_id),
    constraint offerings_interest_area_fk foreign key (interest_area_id, organization_id, school_year_id, program_id)
        references interest_areas (id, organization_id, school_year_id, program_id),
    constraint offerings_id_organization_key unique (id, organization_id),
    constraint offerings_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint offerings_id_organization_year_program_key unique (id, organization_id, school_year_id, program_id),
    constraint offerings_name_check check (btrim(name) <> ''),
    constraint offerings_capacity_check check (capacity > 0),
    constraint offerings_minimum_check check (minimum_viable_enrollment is null or (minimum_viable_enrollment >= 0 and minimum_viable_enrollment <= capacity))
);

create index offerings_session_idx on offerings (organization_id, school_year_id, program_id, session_id, name, id);

alter table offerings enable row level security;
alter table offerings force row level security;

create policy offerings_tenant_isolation on offerings
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger offerings_set_updated_at
before update on offerings
for each row execute function public.set_updated_at();
create trigger offerings_closed_year_guard
before insert or update or delete on offerings
for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update, delete on offerings to miniclass_app;

-- +goose Down

revoke all privileges on offerings from miniclass_app;
drop trigger if exists offerings_closed_year_guard on offerings;
drop trigger if exists offerings_set_updated_at on offerings;
drop policy if exists offerings_tenant_isolation on offerings;
drop index if exists offerings_session_idx;
drop table offerings;
