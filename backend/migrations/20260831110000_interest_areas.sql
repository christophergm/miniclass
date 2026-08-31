-- +goose Up

create table interest_areas (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    label text not null,
    ordinal integer not null,
    retired_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint interest_areas_program_fk foreign key (program_id, organization_id, school_year_id)
        references programs (id, organization_id, school_year_id) on delete cascade,
    constraint interest_areas_id_organization_key unique (id, organization_id),
    constraint interest_areas_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint interest_areas_label_check check (btrim(label) <> ''),
    constraint interest_areas_ordinal_check check (ordinal > 0)
);

create unique index interest_areas_label_idx
    on interest_areas (organization_id, school_year_id, program_id, lower(label))
    where retired_at is null;
create unique index interest_areas_ordinal_idx
    on interest_areas (organization_id, school_year_id, program_id, ordinal);
create index interest_areas_picker_idx
    on interest_areas (organization_id, school_year_id, program_id, ordinal, id)
    where retired_at is null;

alter table interest_areas enable row level security;
alter table interest_areas force row level security;

create policy interest_areas_tenant_isolation on interest_areas
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger interest_areas_set_updated_at
before update on interest_areas
for each row execute function public.set_updated_at();
create trigger interest_areas_closed_year_guard
before insert or update or delete on interest_areas
for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update on interest_areas to miniclass_app;

-- +goose Down

revoke all privileges on interest_areas from miniclass_app;
drop trigger if exists interest_areas_closed_year_guard on interest_areas;
drop trigger if exists interest_areas_set_updated_at on interest_areas;
drop policy if exists interest_areas_tenant_isolation on interest_areas;
drop index if exists interest_areas_picker_idx;
drop index if exists interest_areas_ordinal_idx;
drop index if exists interest_areas_label_idx;
drop table interest_areas;
