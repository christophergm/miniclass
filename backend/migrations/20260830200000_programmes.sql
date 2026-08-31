-- +goose Up

create table programs (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    name text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint programs_school_year_fk foreign key (school_year_id, organization_id)
        references school_years (id, organization_id) on delete cascade,
    constraint programs_id_organization_key unique (id, organization_id),
    constraint programs_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint programs_name_check check (btrim(name) <> ''),
    constraint programs_name_unique unique (organization_id, school_year_id, name)
);

create table program_memberships (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    student_id public.xid20 not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint program_memberships_program_fk foreign key (program_id, organization_id, school_year_id)
        references programs (id, organization_id, school_year_id) on delete cascade,
    constraint program_memberships_student_fk foreign key (student_id, organization_id, school_year_id)
        references students (id, organization_id, school_year_id) on delete cascade,
    constraint program_memberships_id_organization_key unique (id, organization_id),
    constraint program_memberships_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint program_memberships_program_student_unique unique (organization_id, school_year_id, program_id, student_id)
);

create index program_memberships_program_idx on program_memberships (organization_id, school_year_id, program_id, student_id);

alter table programs enable row level security;
alter table programs force row level security;
create policy programs_tenant_isolation on programs
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);
alter table program_memberships enable row level security;
alter table program_memberships force row level security;
create policy program_memberships_tenant_isolation on program_memberships
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger programs_set_updated_at before update on programs for each row execute function public.set_updated_at();
create trigger program_memberships_set_updated_at before update on program_memberships for each row execute function public.set_updated_at();
create trigger programs_closed_year_guard before insert or update or delete on programs for each row execute function public.prevent_closed_school_year_mutation();
create trigger program_memberships_closed_year_guard before insert or update or delete on program_memberships for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update, delete on programs, program_memberships to miniclass_app;

-- +goose Down

revoke all privileges on programs, program_memberships from miniclass_app;
drop trigger if exists program_memberships_closed_year_guard on program_memberships;
drop trigger if exists programs_closed_year_guard on programs;
drop trigger if exists program_memberships_set_updated_at on program_memberships;
drop trigger if exists programs_set_updated_at on programs;
drop policy if exists program_memberships_tenant_isolation on program_memberships;
drop policy if exists programs_tenant_isolation on programs;
drop index if exists program_memberships_program_idx;
drop table program_memberships;
drop table programs;
