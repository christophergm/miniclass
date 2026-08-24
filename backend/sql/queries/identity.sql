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
