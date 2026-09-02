-- name: CreateAdultOTP :one
insert into access_tokens (
    token_hash, purpose, expires_at, generation, organization_id, school_year_id,
    adult_id, verifier_hash, requested_email_hash
)
values ($1, 'adult_otp', $2, 1, $3, $4, $5, $6, $7)
returning id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation;

-- name: CountRecentAdultOTPRequests :one
select count(*)
from access_tokens
where purpose = 'adult_otp'
  and organization_id = $1
  and school_year_id = $2
  and requested_email_hash = $3
  and created_at >= $4;

-- name: GetAdultOTP :one
select id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation
from access_tokens
where id = $1
  and purpose = 'adult_otp';

-- name: GetAdultOTPByHash :one
select id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation
from access_tokens
where token_hash = $1
  and purpose = 'adult_otp';

-- name: ConsumeAdultOTP :one
update access_tokens
set consumed_at = now()
where id = $1
  and purpose = 'adult_otp'
  and verifier_hash = $2
  and revoked_at is null
  and consumed_at is null
  and expires_at > $3
  and attempts < $4
returning id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation;

-- name: IncrementAdultOTPAttempts :execrows
update access_tokens
set attempts = attempts + 1
where id = $1
  and purpose = 'adult_otp'
  and revoked_at is null
  and consumed_at is null
  and expires_at > $2
  and attempts < $3;

-- name: CreateGuardianSession :one
insert into access_tokens (
    token_hash, purpose, expires_at, generation, organization_id, school_year_id,
    adult_id, idle_expires_at, last_seen_at
)
values ($1, 'guardian_session', $2, 1, $3, $4, $5, $6, $7)
returning id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation;

-- name: CreateAdministrativeSession :one
insert into access_tokens (
    token_hash, purpose, expires_at, generation, user_id, idle_expires_at,
    last_seen_at, mfa_generation
)
values ($1, 'administrative_session', $2, 1, $3, $4, $5, $6)
returning id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation;

-- name: GetActiveSessionByHash :one
select id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation,
    created_at, updated_at, organization_id, school_year_id, adult_id, user_id,
    verifier_hash, requested_email_hash, attempts, idle_expires_at, last_seen_at,
    mfa_generation
from access_tokens
where token_hash = $1
  and purpose in ('guardian_session', 'administrative_session')
  and revoked_at is null
  and consumed_at is null
  and expires_at > $2
  and (idle_expires_at is null or idle_expires_at > $2);

-- name: TouchSession :execrows
update access_tokens
set last_seen_at = $2,
    idle_expires_at = $3
where id = $1
  and revoked_at is null
  and consumed_at is null;

-- name: RevokeSession :execrows
update access_tokens
set revoked_at = coalesce(revoked_at, $2)
where id = $1
  and purpose in ('guardian_session', 'administrative_session');

-- name: RevokeAdministrativeSessions :execrows
update access_tokens
set revoked_at = coalesce(revoked_at, $2)
where purpose = 'administrative_session'
  and user_id = $1
  and revoked_at is null;

-- name: GetMFAState :one
select id, mfa_secret_ciphertext, mfa_enrolled_at, mfa_generation
from users
where id = $1;

-- name: UserBelongsToOrganization :one
select exists (
    select 1
    from organization_members
    where user_id = $1 and organization_id = $2 and user_id is not null
);

-- name: SetMFASecret :one
update users
set mfa_secret_ciphertext = $2,
    mfa_enrolled_at = $3,
    mfa_generation = mfa_generation + 1
where id = $1
returning id, mfa_secret_ciphertext, mfa_enrolled_at, mfa_generation;

-- name: ResetMFASecret :one
update users
set mfa_secret_ciphertext = null,
    mfa_enrolled_at = null,
    mfa_generation = mfa_generation + 1
where id = $1
returning id, mfa_secret_ciphertext, mfa_enrolled_at, mfa_generation;

-- name: DeleteMFARecoveryCodes :exec
delete from mfa_recovery_codes where user_id = $1;

-- name: CreateMFARecoveryCode :exec
insert into mfa_recovery_codes (user_id, code_hash) values ($1, $2);

-- name: ConsumeMFARecoveryCode :execrows
update mfa_recovery_codes
set used_at = coalesce(used_at, $3)
where user_id = $1
  and code_hash = $2
  and used_at is null;

-- name: CreateAdultAccountLink :one
insert into adult_account_links (organization_id, school_year_id, adult_id, user_id)
values ($1, $2, $3, $4)
returning id, organization_id, school_year_id, adult_id, user_id, created_at, updated_at;

-- name: GetAdultAccountLink :one
select id, organization_id, school_year_id, adult_id, user_id, created_at, updated_at
from adult_account_links
where organization_id = $1 and school_year_id = $2 and user_id = $3;

-- name: GetAdultAccountLinkByAdult :one
select id, organization_id, school_year_id, adult_id, user_id, created_at, updated_at
from adult_account_links
where organization_id = $1 and school_year_id = $2 and adult_id = $3;

-- name: DeleteAdultAccountLink :execrows
delete from adult_account_links
where id = $1 and organization_id = $2 and school_year_id = $3;

-- name: ListAdultAccountLinks :many
select id, organization_id, school_year_id, adult_id, user_id, created_at, updated_at
from adult_account_links
where organization_id = $1 and school_year_id = $2
order by adult_id, user_id, id;

-- name: ListAllAdultAccountLinksForRegistry :many
select id, organization_id, school_year_id, adult_id, user_id, created_at, updated_at
from adult_account_links
where organization_id = $1
order by id;

-- name: FindAdultAccountLinkForRegistry :one
select id, organization_id, school_year_id, adult_id, user_id, created_at, updated_at
from adult_account_links
where id = $1 and organization_id = $2;

-- name: TouchAdultAccountLinkForRegistry :execrows
update adult_account_links
set updated_at = now()
where id = $1 and organization_id = $2;

-- name: ListGuardianStudentIDs :many
select gr.student_id
from guardian_relationships gr
join adults a on a.id = gr.adult_id
    and a.organization_id = gr.organization_id
    and a.school_year_id = gr.school_year_id
    and a.deleted_at is null
join students s on s.id = gr.student_id
    and s.organization_id = gr.organization_id
    and s.school_year_id = gr.school_year_id
    and s.deleted_at is null
where gr.organization_id = $1
  and gr.school_year_id = $2
  and gr.adult_id = $3
order by gr.student_id;

-- name: FindActiveAdultsByEmail :many
select id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, email, phone, external_identifier, participation_intent,
    deleted_at, created_at, updated_at
from adults
where organization_id = $1
  and school_year_id = $2
  and lower(email) = lower($3)
  and deleted_at is null
order by id;
