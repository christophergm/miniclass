-- +goose Up

-- A stale marker preserves draft assignment work when an organiser reopens an
-- earlier lifecycle stage. The assignment tables arrive in a later phase; the
-- marker belongs to the session because it describes the current draft as a
-- whole and must survive that later schema addition.
alter table sessions add column draft_assignments_stale boolean not null default false;

-- +goose Down

alter table sessions drop column draft_assignments_stale;
