-- +goose Up

alter table assignment_overrides drop constraint assignment_overrides_reason_check;
alter table assignment_overrides add constraint assignment_overrides_reason_check check (reason = btrim(reason));

-- +goose Down

alter table assignment_overrides drop constraint assignment_overrides_reason_check;
alter table assignment_overrides add constraint assignment_overrides_reason_check check (btrim(reason) <> '');
