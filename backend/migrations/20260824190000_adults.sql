-- +goose Up

create type adult_participation_intent as enum (
    'lead',
    'help',
    'unavailable'
);

create table adults (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    legal_given_name text not null,
    legal_family_name text not null,
    preferred_given_name text,
    email text,
    phone text,
    external_identifier text,
    participation_intent adult_participation_intent not null,
    deleted_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint adults_school_year_fk foreign key (school_year_id, organization_id)
        references school_years (id, organization_id) on delete cascade,
    constraint adults_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint adults_legal_given_name_check check (btrim(legal_given_name) <> ''),
    constraint adults_legal_family_name_check check (btrim(legal_family_name) <> ''),
    constraint adults_preferred_given_name_check check (preferred_given_name is null or btrim(preferred_given_name) <> ''),
    constraint adults_email_check check (email is null or btrim(email) <> ''),
    constraint adults_phone_check check (phone is null or btrim(phone) <> ''),
    constraint adults_external_identifier_check check (external_identifier is null or btrim(external_identifier) <> '')
);

create index adults_school_year_name_idx
    on adults (organization_id, school_year_id, legal_family_name, legal_given_name, id)
    where deleted_at is null;

create unique index adults_external_identifier_unique_idx
    on adults (organization_id, school_year_id, external_identifier)
    where deleted_at is null and external_identifier is not null;

alter table adults enable row level security;
alter table adults force row level security;

create policy adults_tenant_isolation on adults
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger adults_set_updated_at
before update on adults
for each row execute function public.set_updated_at();

create trigger adults_closed_year_guard
before insert or update or delete on adults
for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update, delete on adults to miniclass_app;

-- +goose Down

revoke all privileges on adults from miniclass_app;
drop trigger if exists adults_closed_year_guard on adults;
drop trigger if exists adults_set_updated_at on adults;
drop policy if exists adults_tenant_isolation on adults;
drop index if exists adults_external_identifier_unique_idx;
drop index if exists adults_school_year_name_idx;
drop table adults;
drop type adult_participation_intent;
