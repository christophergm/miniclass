-- Guardian onboarding tokens are resolved through the identity accessor before
-- a tenant transaction is opened. Every write that changes tenant data still
-- runs through internal/data and records an audit entry.

-- name: CreateGuardianRegistrationEntry :one
insert into access_tokens (token_hash, purpose, expires_at, generation, organization_id, school_year_id)
values ($1, 'guardian_registration_entry', $2,
    (select coalesce(max(generation), 0) + 1 from access_tokens
     where purpose = 'guardian_registration_entry' and organization_id = $3 and school_year_id = $4),
    $3, $4)
returning id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation, parent_token_id, mailbox_verified_at;

-- name: GetGuardianRegistrationEntryByHash :one
select id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation, parent_token_id, mailbox_verified_at
from access_tokens
where token_hash = $1
  and purpose = 'guardian_registration_entry'
  and revoked_at is null
  and consumed_at is null
  and expires_at > $2;

-- name: GetCurrentGuardianRegistrationEntry :one
select id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation, parent_token_id, mailbox_verified_at
from access_tokens
where purpose = 'guardian_registration_entry'
  and organization_id = $1
  and school_year_id = $2
  and revoked_at is null
  and consumed_at is null
order by generation desc, created_at desc, id desc
limit 1;

-- name: RevokeGuardianRegistrationEntries :execrows
update access_tokens
set revoked_at = coalesce(revoked_at, $3)
where purpose = 'guardian_registration_entry'
  and organization_id = $1
  and school_year_id = $2
  and revoked_at is null;

-- name: CreateGuardianInvitationToken :one
insert into access_tokens (token_hash, purpose, expires_at, generation, organization_id, school_year_id)
values ($1, 'guardian_invitation', $2,
    (select coalesce(max(generation), 0) + 1 from access_tokens
     where purpose = 'guardian_invitation' and organization_id = $3 and school_year_id = $4),
    $3, $4)
returning id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation, parent_token_id, mailbox_verified_at;

-- name: GetGuardianInvitationTokenByHash :one
select id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation, parent_token_id, mailbox_verified_at
from access_tokens
where token_hash = $1
  and purpose = 'guardian_invitation'
  and revoked_at is null
  and consumed_at is null
  and expires_at > $2;

-- name: RevokeGuardianInvitationToken :execrows
update access_tokens
set revoked_at = coalesce(revoked_at, $2)
where id = $1
  and purpose = 'guardian_invitation'
  and revoked_at is null;

-- name: ConsumeGuardianInvitationToken :one
update access_tokens
set consumed_at = $2
where id = $1
  and purpose = 'guardian_invitation'
  and revoked_at is null
  and consumed_at is null
  and expires_at > $2
returning id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation, parent_token_id, mailbox_verified_at;

-- name: CreateGuardianOnboardingSession :one
insert into access_tokens (token_hash, purpose, expires_at, generation, organization_id, school_year_id, parent_token_id, last_seen_at, idle_expires_at)
values ($1, 'guardian_onboarding_session', $2, 1, $3, $4, $5, $6, $7)
returning id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation, parent_token_id, mailbox_verified_at;

-- name: GetGuardianOnboardingSessionByHash :one
select id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation, parent_token_id, mailbox_verified_at
from access_tokens
where token_hash = $1
  and purpose = 'guardian_onboarding_session'
  and revoked_at is null
  and consumed_at is null
  and expires_at > $2
  and (idle_expires_at is null or idle_expires_at > $2);

-- name: TouchGuardianOnboardingSession :execrows
update access_tokens
set last_seen_at = $2,
    idle_expires_at = $3
where id = $1
  and purpose = 'guardian_onboarding_session'
  and revoked_at is null
  and consumed_at is null;

-- name: VerifyGuardianOnboardingSession :one
update access_tokens
set requested_email_hash = $2,
    mailbox_verified_at = $3
where id = $1
  and purpose = 'guardian_onboarding_session'
  and revoked_at is null
  and consumed_at is null
returning id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation, parent_token_id, mailbox_verified_at;

-- name: CompleteGuardianOnboardingSession :execrows
update access_tokens
set consumed_at = $2
where id = $1
  and purpose = 'guardian_onboarding_session'
  and revoked_at is null
  and consumed_at is null;

-- name: RevokeGuardianOnboardingSession :execrows
update access_tokens
set revoked_at = coalesce(revoked_at, $2)
where id = $1
  and organization_id = $3
  and school_year_id = $4
  and purpose = 'guardian_onboarding_session'
  and revoked_at is null;

-- name: CreateGuardianOnboardingOTP :one
insert into access_tokens (token_hash, purpose, expires_at, generation, organization_id, school_year_id, parent_token_id, verifier_hash, requested_email_hash)
values ($1, 'guardian_onboarding_otp', $2, 1, $3, $4, $5, $6, $7)
returning id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation, parent_token_id, mailbox_verified_at;

-- name: GetGuardianOnboardingOTPByHash :one
select id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation, parent_token_id, mailbox_verified_at
from access_tokens
where token_hash = $1
  and purpose = 'guardian_onboarding_otp';

-- name: CountRecentGuardianOnboardingOTPRequests :one
select count(*)
from access_tokens
where purpose = 'guardian_onboarding_otp'
  and organization_id = $1
  and school_year_id = $2
  and requested_email_hash = $3
  and created_at >= $4;

-- name: ConsumeGuardianOnboardingOTP :one
update access_tokens
set consumed_at = $3
where id = $1
  and purpose = 'guardian_onboarding_otp'
  and verifier_hash = $2
  and revoked_at is null
  and consumed_at is null
  and expires_at > $3
  and attempts < $4
returning id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation, parent_token_id, mailbox_verified_at;

-- name: IncrementGuardianOnboardingOTPAttempts :execrows
update access_tokens
set attempts = attempts + 1
where id = $1
  and purpose = 'guardian_onboarding_otp'
  and revoked_at is null
  and consumed_at is null
  and expires_at > $2
  and attempts < $3;

-- name: CreateGuardianInvitationContact :one
insert into guardian_invitation_contacts (organization_id, school_year_id, invitation_token_id, email)
values ($1, $2, $3, $4)
returning id, organization_id, school_year_id, invitation_token_id, email, created_at, updated_at;

-- name: GetGuardianInvitationContactByEmail :one
select id, organization_id, school_year_id, invitation_token_id, email, created_at, updated_at
from guardian_invitation_contacts
where organization_id = $1
  and school_year_id = $2
  and lower(email) = lower($3);

-- name: GetGuardianInvitationContactByID :one
select id, organization_id, school_year_id, invitation_token_id, email, created_at, updated_at
from guardian_invitation_contacts
where id = $1
  and organization_id = $2
  and school_year_id = $3;

-- name: GetGuardianInvitationContactByTokenID :one
select id, organization_id, school_year_id, invitation_token_id, email, created_at, updated_at
from guardian_invitation_contacts
where invitation_token_id = $1
  and organization_id = $2
  and school_year_id = $3;

-- name: UpdateGuardianInvitationContactToken :one
update guardian_invitation_contacts
set invitation_token_id = $4
where id = $1
  and organization_id = $2
  and school_year_id = $3
returning id, organization_id, school_year_id, invitation_token_id, email, created_at, updated_at;

-- name: ListGuardianInvitationContacts :many
select c.id, c.organization_id, c.school_year_id, c.invitation_token_id, c.email, c.created_at, c.updated_at,
    t.expires_at, t.revoked_at, t.consumed_at, t.generation
from guardian_invitation_contacts c
join access_tokens t on t.id = c.invitation_token_id
where c.organization_id = $1
  and c.school_year_id = $2
order by lower(c.email), c.id;

-- name: TouchGuardianInvitationContactForRegistry :execrows
update guardian_invitation_contacts
set updated_at = updated_at
where id = $1 and organization_id = $2;

-- name: DeleteGuardianInvitationContact :execrows
delete from guardian_invitation_contacts
where id = $1 and organization_id = $2 and school_year_id = $3;

-- name: ListAllGuardianInvitationContactsForRegistry :many
select id, organization_id, school_year_id, invitation_token_id, email, created_at, updated_at
from guardian_invitation_contacts
where organization_id = $1
order by id;

-- name: FindGuardianInvitationContactForRegistry :one
select id, organization_id, school_year_id, invitation_token_id, email, created_at, updated_at
from guardian_invitation_contacts
where id = $1 and organization_id = $2;

-- name: CreateGuardianOnboardingConsent :one
insert into guardian_onboarding_consents (
    organization_id, school_year_id, session_token_id, verified_email,
    terms_version, privacy_version, signup_notice_version, signup_notice_hash,
    accepted_at, source_surface
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
returning id, organization_id, school_year_id, session_token_id, verified_email,
    terms_version, privacy_version, signup_notice_version, signup_notice_hash,
    accepted_at, source_surface;

-- name: GetGuardianOnboardingConsent :one
select id, organization_id, school_year_id, session_token_id, verified_email,
    terms_version, privacy_version, signup_notice_version, signup_notice_hash,
    accepted_at, source_surface
from guardian_onboarding_consents
where session_token_id = $1 and organization_id = $2 and school_year_id = $3;

-- name: ListAllGuardianOnboardingConsentsForRegistry :many
select id, organization_id, school_year_id, session_token_id, verified_email,
    terms_version, privacy_version, signup_notice_version, signup_notice_hash,
    accepted_at, source_surface
from guardian_onboarding_consents
where organization_id = $1
order by id;

-- name: FindGuardianOnboardingConsentForRegistry :one
select id, organization_id, school_year_id, session_token_id, verified_email,
    terms_version, privacy_version, signup_notice_version, signup_notice_hash,
    accepted_at, source_surface
from guardian_onboarding_consents
where id = $1 and organization_id = $2;

-- name: TouchGuardianOnboardingConsentForRegistry :execrows
update guardian_onboarding_consents
set accepted_at = accepted_at
where id = $1 and organization_id = $2;

-- name: DeleteGuardianOnboardingConsentForRegistry :execrows
delete from guardian_onboarding_consents
where id = $1 and organization_id = $2;

-- name: GetGuardianSignupNotice :one
select id, guardian_signup_notice, guardian_signup_notice_version, guardian_signup_notice_hash
from organizations
where id = $1;

-- name: UpdateGuardianSignupNotice :one
update organizations
set guardian_signup_notice = $2,
    guardian_signup_notice_version = $3,
    guardian_signup_notice_hash = $4
where id = $1
returning id, guardian_signup_notice, guardian_signup_notice_version, guardian_signup_notice_hash;
