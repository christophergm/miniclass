-- +goose NO TRANSACTION

-- +goose Up

alter type access_token_purpose add value 'adult_otp';
alter type access_token_purpose add value 'guardian_session';
alter type access_token_purpose add value 'administrative_session';

alter table access_tokens
    add column organization_id public.xid20,
    add column school_year_id public.xid20,
    add column adult_id public.xid20,
    add column user_id public.xid20,
    add column verifier_hash bytea,
    add column requested_email_hash bytea,
    add column attempts integer not null default 0,
    add column idle_expires_at timestamptz,
    add column last_seen_at timestamptz,
    add column mfa_generation integer;

-- Existing parent tables use FORCE RLS and their policies require a tenant
-- setting. The migrator must validate these new composite references across
-- every tenant, so temporarily restore the owner visibility used by earlier
-- schema migrations.
alter table adults no force row level security;
alter table school_years no force row level security;

alter table access_tokens
    add constraint access_tokens_attempts_check check (attempts >= 0),
    add constraint access_tokens_verifier_hash_check check (verifier_hash is null or octet_length(verifier_hash) = 32),
    add constraint access_tokens_requested_email_hash_check check (requested_email_hash is null or octet_length(requested_email_hash) = 32),
    add constraint access_tokens_mfa_generation_check check (mfa_generation is null or mfa_generation > 0),
    add constraint access_tokens_user_fk foreign key (user_id)
        references users (id) on delete cascade,
    add constraint access_tokens_adult_scope_fk foreign key (adult_id, organization_id, school_year_id)
        references adults (id, organization_id, school_year_id) on delete cascade,
    add constraint access_tokens_year_scope_fk foreign key (school_year_id, organization_id)
        references school_years (id, organization_id) on delete cascade;

create index access_tokens_adult_otp_rate_idx
    on access_tokens (purpose, organization_id, school_year_id, requested_email_hash, created_at)
    where purpose = 'adult_otp';

create index access_tokens_session_user_idx
    on access_tokens (purpose, user_id)
    where purpose = 'administrative_session' and revoked_at is null;

alter table adults force row level security;
alter table school_years force row level security;

create table adult_account_links (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    adult_id public.xid20 not null,
    user_id public.xid20 not null references users (id) on delete cascade,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint adult_account_links_year_fk foreign key (school_year_id, organization_id)
        references school_years (id, organization_id) on delete cascade,
    constraint adult_account_links_adult_fk foreign key (adult_id, organization_id, school_year_id)
        references adults (id, organization_id, school_year_id) on delete cascade,
    constraint adult_account_links_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint adult_account_links_user_year_unique unique (organization_id, school_year_id, user_id),
    constraint adult_account_links_adult_year_unique unique (organization_id, school_year_id, adult_id)
);

alter table adult_account_links enable row level security;
alter table adult_account_links force row level security;

create policy adult_account_links_tenant_isolation on adult_account_links
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger adult_account_links_set_updated_at
before update on adult_account_links
for each row execute function public.set_updated_at();

create trigger adult_account_links_closed_year_guard
before insert or update or delete on adult_account_links
for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update, delete on adult_account_links to miniclass_app;

create table mfa_recovery_codes (
    id public.xid20 primary key default public.xid(),
    user_id public.xid20 not null references users (id) on delete cascade,
    code_hash bytea not null,
    used_at timestamptz,
    created_at timestamptz not null default now(),
    constraint mfa_recovery_codes_hash_check check (octet_length(code_hash) = 32)
);

create unique index mfa_recovery_codes_user_hash_idx on mfa_recovery_codes (user_id, code_hash);
create index mfa_recovery_codes_active_idx on mfa_recovery_codes (user_id) where used_at is null;

alter table users
    add column mfa_secret_ciphertext bytea,
    add column mfa_enrolled_at timestamptz,
    add column mfa_generation integer not null default 1,
    add constraint users_mfa_generation_check check (mfa_generation > 0);

grant select, insert, update, delete on mfa_recovery_codes to miniclass_app;

-- +goose Down

revoke all privileges on mfa_recovery_codes from miniclass_app;
alter table users drop constraint if exists users_mfa_generation_check;
alter table users drop column if exists mfa_generation;
alter table users drop column if exists mfa_enrolled_at;
alter table users drop column if exists mfa_secret_ciphertext;
drop index if exists mfa_recovery_codes_active_idx;
drop index if exists mfa_recovery_codes_user_hash_idx;
drop table mfa_recovery_codes;

revoke all privileges on adult_account_links from miniclass_app;
drop trigger if exists adult_account_links_closed_year_guard on adult_account_links;
drop trigger if exists adult_account_links_set_updated_at on adult_account_links;
drop policy if exists adult_account_links_tenant_isolation on adult_account_links;
drop table adult_account_links;

drop index if exists access_tokens_session_user_idx;
drop index if exists access_tokens_adult_otp_rate_idx;
alter table access_tokens
    drop constraint if exists access_tokens_year_scope_fk,
    drop constraint if exists access_tokens_adult_scope_fk,
    drop constraint if exists access_tokens_user_fk,
    drop constraint if exists access_tokens_mfa_generation_check,
    drop constraint if exists access_tokens_requested_email_hash_check,
    drop constraint if exists access_tokens_verifier_hash_check,
    drop constraint if exists access_tokens_attempts_check,
    drop column if exists mfa_generation,
    drop column if exists last_seen_at,
    drop column if exists idle_expires_at,
    drop column if exists attempts,
    drop column if exists requested_email_hash,
    drop column if exists verifier_hash,
    drop column if exists user_id,
    drop column if exists adult_id,
    drop column if exists school_year_id,
    drop column if exists organization_id;

-- PostgreSQL cannot remove enum values. The down path restores the original
-- type while preserving the pre-migration access-token rows.
alter table access_tokens rename to access_tokens_adult_auth_down;
create type access_token_purpose_original as enum (
    'admin_invitation',
    'household_submission',
    'class_leader',
    'homeroom_teacher',
    'published_artifact'
);
alter table access_tokens_adult_auth_down
    alter column purpose type access_token_purpose_original using purpose::text::access_token_purpose_original;
alter type access_token_purpose rename to access_token_purpose_adult_auth_down;
alter type access_token_purpose_original rename to access_token_purpose;
alter table access_tokens_adult_auth_down rename to access_tokens;
drop type access_token_purpose_adult_auth_down;
