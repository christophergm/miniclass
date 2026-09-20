-- +goose NO TRANSACTION

-- +goose Up

alter type access_token_purpose add value 'guardian_registration_entry';
alter type access_token_purpose add value 'guardian_invitation';
alter type access_token_purpose add value 'guardian_onboarding_session';
alter type access_token_purpose add value 'guardian_onboarding_otp';

alter table organizations
    add column guardian_signup_notice text,
    add column guardian_signup_notice_version integer not null default 0,
    add column guardian_signup_notice_hash bytea,
    add constraint organizations_guardian_signup_notice_version_check
        check (guardian_signup_notice_version >= 0),
    add constraint organizations_guardian_signup_notice_hash_check
        check (guardian_signup_notice_hash is null or octet_length(guardian_signup_notice_hash) = 32),
    add constraint organizations_guardian_signup_notice_pair_check
        check ((guardian_signup_notice is null and guardian_signup_notice_hash is null and guardian_signup_notice_version = 0)
            or (guardian_signup_notice is not null and guardian_signup_notice_hash is not null and guardian_signup_notice_version > 0));

alter table access_tokens
    add column parent_token_id public.xid20,
    add column mailbox_verified_at timestamptz,
    add constraint access_tokens_parent_token_fk foreign key (parent_token_id)
        references access_tokens (id) on delete cascade,
    add constraint access_tokens_id_organization_year_key unique (id, organization_id, school_year_id),
    add constraint access_tokens_guardian_scope_check check (
        purpose not in ('guardian_registration_entry', 'guardian_invitation', 'guardian_onboarding_session', 'guardian_onboarding_otp')
        or (organization_id is not null and school_year_id is not null)
    );

create index access_tokens_guardian_onboarding_parent_idx
    on access_tokens (parent_token_id, purpose)
    where parent_token_id is not null;

create table guardian_invitation_contacts (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    invitation_token_id public.xid20 not null,
    email text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint guardian_invitation_contacts_school_year_fk foreign key (school_year_id, organization_id)
        references school_years (id, organization_id) on delete cascade,
    constraint guardian_invitation_contacts_token_fk foreign key (invitation_token_id, organization_id, school_year_id)
        references access_tokens (id, organization_id, school_year_id) on delete cascade,
    constraint guardian_invitation_contacts_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint guardian_invitation_contacts_token_unique unique (invitation_token_id),
    constraint guardian_invitation_contacts_email_check check (btrim(email) <> '')
);

create unique index guardian_invitation_contacts_active_email_idx
    on guardian_invitation_contacts (organization_id, school_year_id, lower(email))
    where invitation_token_id is not null;

alter table guardian_invitation_contacts enable row level security;
alter table guardian_invitation_contacts force row level security;

create policy guardian_invitation_contacts_tenant_isolation on guardian_invitation_contacts
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger guardian_invitation_contacts_set_updated_at
before update on guardian_invitation_contacts
for each row execute function public.set_updated_at();

create trigger guardian_invitation_contacts_closed_year_guard
before insert or update or delete on guardian_invitation_contacts
for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update, delete on guardian_invitation_contacts to miniclass_app;

create table guardian_onboarding_consents (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    session_token_id public.xid20 not null,
    verified_email text not null,
    terms_version text not null,
    privacy_version text not null,
    signup_notice_version integer,
    signup_notice_hash bytea,
    accepted_at timestamptz not null default now(),
    source_surface text not null,
    constraint guardian_onboarding_consents_school_year_fk foreign key (school_year_id, organization_id)
        references school_years (id, organization_id) on delete cascade,
    constraint guardian_onboarding_consents_session_fk foreign key (session_token_id, organization_id, school_year_id)
        references access_tokens (id, organization_id, school_year_id) on delete cascade,
    constraint guardian_onboarding_consents_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint guardian_onboarding_consents_session_unique unique (session_token_id),
    constraint guardian_onboarding_consents_email_check check (btrim(verified_email) <> ''),
    constraint guardian_onboarding_consents_terms_check check (btrim(terms_version) <> ''),
    constraint guardian_onboarding_consents_privacy_check check (btrim(privacy_version) <> ''),
    constraint guardian_onboarding_consents_source_check check (btrim(source_surface) <> ''),
    constraint guardian_onboarding_consents_notice_pair_check check ((signup_notice_version is null and signup_notice_hash is null) or (signup_notice_version is not null and signup_notice_hash is not null and signup_notice_version > 0 and octet_length(signup_notice_hash) = 32))
);

alter table guardian_onboarding_consents enable row level security;
alter table guardian_onboarding_consents force row level security;

create policy guardian_onboarding_consents_tenant_isolation on guardian_onboarding_consents
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger guardian_onboarding_consents_closed_year_guard
before insert or update or delete on guardian_onboarding_consents
for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update, delete on guardian_onboarding_consents to miniclass_app;

-- +goose Down

delete from access_tokens
where purpose in (
    'guardian_registration_entry',
    'guardian_invitation',
    'guardian_onboarding_session',
    'guardian_onboarding_otp'
);

revoke all privileges on guardian_onboarding_consents from miniclass_app;
drop trigger if exists guardian_onboarding_consents_closed_year_guard on guardian_onboarding_consents;
drop policy if exists guardian_onboarding_consents_tenant_isolation on guardian_onboarding_consents;
drop table guardian_onboarding_consents;

revoke all privileges on guardian_invitation_contacts from miniclass_app;
drop trigger if exists guardian_invitation_contacts_closed_year_guard on guardian_invitation_contacts;
drop trigger if exists guardian_invitation_contacts_set_updated_at on guardian_invitation_contacts;
drop policy if exists guardian_invitation_contacts_tenant_isolation on guardian_invitation_contacts;
drop index if exists guardian_invitation_contacts_active_email_idx;
drop table guardian_invitation_contacts;

drop index if exists access_tokens_guardian_onboarding_parent_idx;
drop index if exists access_tokens_session_user_idx;
drop index if exists access_tokens_adult_otp_rate_idx;
alter table access_tokens
    drop constraint if exists access_tokens_guardian_scope_check,
    drop constraint if exists access_tokens_parent_token_fk,
    drop constraint if exists access_tokens_id_organization_year_key,
    drop column if exists mailbox_verified_at,
    drop column if exists parent_token_id;

alter table organizations
    drop constraint if exists organizations_guardian_signup_notice_pair_check,
    drop constraint if exists organizations_guardian_signup_notice_hash_check,
    drop constraint if exists organizations_guardian_signup_notice_version_check,
    drop column if exists guardian_signup_notice_hash,
    drop column if exists guardian_signup_notice_version,
    drop column if exists guardian_signup_notice;

-- PostgreSQL cannot remove enum values. Rebuild the type after this feature's
-- rows have been removed by the down path, preserving adult authentication.
alter table access_tokens rename to access_tokens_guardian_onboarding_down;
create type access_token_purpose_guardian_onboarding_down as enum (
    'admin_invitation',
    'guardian_submission',
    'class_leader',
    'homeroom_teacher',
    'published_artifact',
    'adult_otp',
    'guardian_session',
    'administrative_session'
);
alter table access_tokens_guardian_onboarding_down
    alter column purpose type access_token_purpose_guardian_onboarding_down using purpose::text::access_token_purpose_guardian_onboarding_down;
alter type access_token_purpose rename to access_token_purpose_guardian_onboarding_old;
alter type access_token_purpose_guardian_onboarding_down rename to access_token_purpose;
alter type access_token_purpose owner to miniclass_migrator;
alter table access_tokens_guardian_onboarding_down rename to access_tokens;
drop type access_token_purpose_guardian_onboarding_old;

create index access_tokens_adult_otp_rate_idx
    on access_tokens (purpose, organization_id, school_year_id, requested_email_hash, created_at)
    where purpose = 'adult_otp';

create index access_tokens_session_user_idx
    on access_tokens (purpose, user_id)
    where purpose = 'administrative_session' and revoked_at is null;
