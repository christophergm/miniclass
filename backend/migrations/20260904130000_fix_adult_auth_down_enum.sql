-- +goose NO TRANSACTION

-- +goose Up

-- The adult-authentication migration is already merged and its Down path
-- cannot be edited. This compatibility migration owns the corrected rollback
-- for that migration when a database has reached this version.

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

alter table access_tokens rename to access_tokens_adult_auth_down;
create type access_token_purpose_original as enum (
    'admin_invitation',
    'guardian_submission',
    'class_leader',
    'homeroom_teacher',
    'published_artifact'
);
alter table access_tokens_adult_auth_down
    alter column purpose type access_token_purpose_original using purpose::text::access_token_purpose_original;
alter type access_token_purpose rename to access_token_purpose_adult_auth_down;
alter type access_token_purpose_original rename to access_token_purpose;
alter type access_token_purpose owner to miniclass_migrator;
alter table access_tokens_adult_auth_down rename to access_tokens;
drop type access_token_purpose_adult_auth_down;

-- Goose must not attempt the merged migration's defective Down after this
-- compatibility rollback has completed.
delete from goose_db_version
where version_id = 20260904120000
  and is_applied;
