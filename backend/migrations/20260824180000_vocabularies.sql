-- +goose Up

create table grade_levels (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    code text not null,
    label text not null,
    ordinal integer not null,
    retired_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint grade_levels_code_check check (btrim(code) <> ''),
    constraint grade_levels_label_check check (btrim(label) <> ''),
    constraint grade_levels_ordinal_check check (ordinal > 0),
    constraint grade_levels_id_organization_key unique (id, organization_id)
);

create unique index grade_levels_code_idx on grade_levels (organization_id, lower(code));
create unique index grade_levels_ordinal_idx on grade_levels (organization_id, ordinal);
create index grade_levels_picker_idx on grade_levels (organization_id, ordinal, id)
    where retired_at is null;

alter table grade_levels enable row level security;
alter table grade_levels force row level security;

create policy grade_levels_tenant_isolation on grade_levels
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger grade_levels_set_updated_at
before update on grade_levels
for each row execute function public.set_updated_at();

create table homerooms (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    name text not null,
    retired_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint homerooms_name_check check (btrim(name) <> ''),
    constraint homerooms_id_organization_key unique (id, organization_id)
);

create unique index homerooms_name_idx on homerooms (organization_id, lower(name));
create index homerooms_picker_idx on homerooms (organization_id, lower(name), id)
    where retired_at is null;

alter table homerooms enable row level security;
alter table homerooms force row level security;

create policy homerooms_tenant_isolation on homerooms
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger homerooms_set_updated_at
before update on homerooms
for each row execute function public.set_updated_at();

grant select, insert, update on grade_levels to miniclass_app;
grant select, insert, update on homerooms to miniclass_app;
grant update (homeroom_label) on organizations to miniclass_app;

-- +goose Down

revoke update (homeroom_label) on organizations from miniclass_app;
revoke all privileges on homerooms from miniclass_app;
revoke all privileges on grade_levels from miniclass_app;
drop trigger if exists homerooms_set_updated_at on homerooms;
drop policy if exists homerooms_tenant_isolation on homerooms;
drop table homerooms;
drop trigger if exists grade_levels_set_updated_at on grade_levels;
drop policy if exists grade_levels_tenant_isolation on grade_levels;
drop table grade_levels;
