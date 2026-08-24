-- name: CreateOrganization :one
insert into organizations (name, homeroom_label)
values ($1, $2)
returning id, name, homeroom_label, created_at, updated_at;

-- name: CreateAccessToken :one
insert into access_tokens (token_hash, purpose, expires_at, generation)
values ($1, $2, $3, $4)
returning id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation, created_at, updated_at;

-- name: CreateOrganizationMember :one
insert into organization_members (
    organization_id,
    user_id,
    role,
    invited_email,
    invitation_token_id
)
values ($1, $2, $3, $4, $5)
returning id, organization_id, user_id, role, invited_email, invitation_token_id, created_at, updated_at;

-- name: CreateUser :one
insert into users (provider_subject, email)
values ($1, $2)
returning id, provider_subject, email, created_at, updated_at;

-- name: GetUserByProviderSubject :one
select id, provider_subject, email, created_at, updated_at
from users
where provider_subject = $1;

-- name: GetAccountMembershipsByProviderSubject :many
select
    u.id as user_id,
    u.provider_subject,
    u.email,
    u.created_at as user_created_at,
    u.updated_at as user_updated_at,
    om.id as membership_id,
    om.organization_id,
    o.name as organization_name,
    om.role,
    om.created_at as membership_created_at,
    om.updated_at as membership_updated_at
from users u
join organization_members om on om.user_id = u.id
join organizations o on o.id = om.organization_id
where u.provider_subject = $1
order by om.organization_id;

-- name: GetAccessTokenByHash :one
select id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation, created_at, updated_at
from access_tokens
where token_hash = $1;

-- name: GetAccessTokenByID :one
select id, token_hash, purpose, expires_at, revoked_at, consumed_at, generation, created_at, updated_at
from access_tokens
where id = $1;

-- name: RevokeAccessToken :exec
update access_tokens
set revoked_at = coalesce(revoked_at, now())
where id = $1;

-- name: ConsumeAccessToken :execrows
update access_tokens
set consumed_at = coalesce(consumed_at, now())
where id = $1
  and revoked_at is null
  and consumed_at is null
  and expires_at > now();

-- name: ReplaceOrganizationMemberInvitation :execrows
update organization_members
set invitation_token_id = $2
where invitation_token_id = $1;

-- name: GetOrganizationMemberByInvitationToken :one
select id, organization_id, user_id, role, invited_email, invitation_token_id, created_at, updated_at
from organization_members
where invitation_token_id = $1;

-- name: ClaimOrganizationMember :execrows
update organization_members
set user_id = $2,
    invited_email = null,
    invitation_token_id = null
where id = $1
  and user_id is null
  and invited_email is not null;
