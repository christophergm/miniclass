-- +goose Up

-- Shared registration links are public record identifiers, not bearer secrets.
-- Existing hash-based entries cannot be safely translated, so retain their
-- history but make them unavailable.
alter table access_tokens
    alter column token_hash drop not null,
    drop constraint access_tokens_hash_length_check,
    add constraint access_tokens_token_hash_check check (
        (purpose = 'guardian_registration_entry' and token_hash is null)
        or (purpose <> 'guardian_registration_entry' and octet_length(token_hash) = 32)
    ),
    add column guardian_registration_revocation_kind text,
    add column guardian_registration_revoked_by_user_id public.xid20 references users (id) on delete set null,
    add constraint access_tokens_guardian_registration_revocation_check check (
        (purpose <> 'guardian_registration_entry' and guardian_registration_revocation_kind is null and guardian_registration_revoked_by_user_id is null)
        or (purpose = 'guardian_registration_entry' and (
            (revoked_at is null and guardian_registration_revocation_kind is null and guardian_registration_revoked_by_user_id is null)
            or (revoked_at is not null and guardian_registration_revocation_kind in ('manual', 'replaced', 'legacy_cutover'))
        ))
    );

drop index access_tokens_token_hash_idx;
create unique index access_tokens_token_hash_idx on access_tokens (token_hash) where token_hash is not null;

update access_tokens
set revoked_at = now(),
    guardian_registration_revocation_kind = 'legacy_cutover'
where purpose = 'guardian_registration_entry'
  and token_hash is not null
  and revoked_at is null;



-- +goose Down

delete from access_tokens where purpose = 'guardian_registration_entry' and token_hash is null;
drop index if exists access_tokens_token_hash_idx;
create unique index access_tokens_token_hash_idx on access_tokens (token_hash);

alter table access_tokens
    drop constraint if exists access_tokens_guardian_registration_revocation_check,
    drop constraint if exists access_tokens_token_hash_check,
    drop column if exists guardian_registration_revoked_by_user_id,
    drop column if exists guardian_registration_revocation_kind,
    alter column token_hash set not null,
    add constraint access_tokens_hash_length_check check (octet_length(token_hash) = 32);
