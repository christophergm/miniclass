-- +goose Up

create type organization_role as enum (
    'owner',
    'administrator',
    'coordinator'
);

create type access_token_purpose as enum (
    'admin_invitation',
    'household_submission',
    'class_leader',
    'homeroom_teacher',
    'published_artifact'
);

create table organizations (
    id public.xid20 primary key default public.xid(),
    name text not null,
    homeroom_label text not null default 'homeroom',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint organizations_name_check check (btrim(name) <> ''),
    constraint organizations_homeroom_label_check check (btrim(homeroom_label) <> '')
);

create table users (
    id public.xid20 primary key default public.xid(),
    provider_subject text not null,
    email text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint users_provider_subject_check check (btrim(provider_subject) <> ''),
    constraint users_email_check check (btrim(email) <> '')
);

create unique index users_provider_subject_idx on users (provider_subject);
create unique index users_email_idx on users (lower(email));

create table access_tokens (
    id public.xid20 primary key default public.xid(),
    token_hash bytea not null,
    purpose access_token_purpose not null,
    expires_at timestamptz not null,
    revoked_at timestamptz,
    consumed_at timestamptz,
    generation integer not null default 1,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint access_tokens_hash_length_check check (octet_length(token_hash) = 32),
    constraint access_tokens_generation_check check (generation > 0)
);

create unique index access_tokens_token_hash_idx on access_tokens (token_hash);
create index access_tokens_active_idx on access_tokens (purpose, expires_at)
    where revoked_at is null and consumed_at is null;

create table organization_members (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    user_id public.xid20 references users (id) on delete cascade,
    role organization_role not null,
    invited_email text,
    invitation_token_id public.xid20 references access_tokens (id) on delete set null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint organization_members_invitation_state_check
        check ((user_id is null) = (invited_email is not null)),
    constraint organization_members_invited_email_check
        check (invited_email is null or btrim(invited_email) <> '')
);

create unique index organization_members_user_idx
    on organization_members (organization_id, user_id)
    where user_id is not null;
create unique index organization_members_invited_email_idx
    on organization_members (organization_id, lower(invited_email))
    where user_id is null;
create unique index organization_members_invitation_token_idx
    on organization_members (invitation_token_id)
    where invitation_token_id is not null;

-- +goose StatementBegin
create or replace function public.set_updated_at() returns trigger language plpgsql as
$$
begin
    new.updated_at = now();
    return new;
end;
$$;
-- +goose StatementEnd

create trigger organizations_set_updated_at
before update on organizations
for each row execute function public.set_updated_at();

create trigger users_set_updated_at
before update on users
for each row execute function public.set_updated_at();

create trigger access_tokens_set_updated_at
before update on access_tokens
for each row execute function public.set_updated_at();

create trigger organization_members_set_updated_at
before update on organization_members
for each row execute function public.set_updated_at();

drop table health_checks;

-- +goose Down

create table health_checks (
    id public.xid20 primary key default public.xid(),
    status text not null default 'healthy',
    checked_at timestamptz not null default now(),
    constraint health_checks_status_check check (status <> '')
);

drop trigger if exists organization_members_set_updated_at on organization_members;
drop trigger if exists access_tokens_set_updated_at on access_tokens;
drop trigger if exists users_set_updated_at on users;
drop trigger if exists organizations_set_updated_at on organizations;
drop function if exists public.set_updated_at();
drop table organization_members;
drop table access_tokens;
drop table users;
drop table organizations;
drop type access_token_purpose;
drop type organization_role;
