-- name: CreateAdult :one
insert into adults (
    organization_id,
    school_year_id,
    legal_given_name,
    legal_family_name,
    preferred_given_name,
    email,
    phone,
    external_identifier,
    participation_intent
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
returning id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, email, phone, external_identifier, participation_intent,
    deleted_at, created_at, updated_at;

-- name: ListAdults :many
select id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, email, phone, external_identifier, participation_intent,
    deleted_at, created_at, updated_at
from adults
where organization_id = $1
  and school_year_id = $2
  and deleted_at is null
order by legal_family_name, coalesce(preferred_given_name, legal_given_name), legal_given_name, id;

-- name: GetAdultByID :one
select id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, email, phone, external_identifier, participation_intent,
    deleted_at, created_at, updated_at
from adults
where id = $1
  and organization_id = $2
  and school_year_id = $3
  and deleted_at is null;

-- name: UpdateAdult :one
update adults
set legal_given_name = $4,
    legal_family_name = $5,
    preferred_given_name = $6,
    email = $7,
    phone = $8,
    external_identifier = $9,
    participation_intent = $10
where id = $1
  and organization_id = $2
  and school_year_id = $3
  and deleted_at is null
returning id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, email, phone, external_identifier, participation_intent,
    deleted_at, created_at, updated_at;

-- name: SoftDeleteAdult :execrows
update adults
set deleted_at = coalesce(deleted_at, now())
where id = $1
  and organization_id = $2
  and school_year_id = $3
  and deleted_at is null;

-- name: ListAllActiveAdultsForRegistry :many
select id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, email, phone, external_identifier, participation_intent,
    deleted_at, created_at, updated_at
from adults
where organization_id = $1
  and deleted_at is null
order by id;

-- name: FindAdultForRegistry :one
select id, organization_id, school_year_id, legal_given_name, legal_family_name,
    preferred_given_name, email, phone, external_identifier, participation_intent,
    deleted_at, created_at, updated_at
from adults
where id = $1
  and organization_id = $2
  and deleted_at is null;
