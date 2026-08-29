-- +goose Up

alter table students
    alter column grade_level_id drop not null;

alter table adults
    alter column participation_intent drop default,
    alter column participation_intent drop not null;

alter table homerooms
    add column external_identifier text,
    add constraint homerooms_external_identifier_check
        check (external_identifier is null or btrim(external_identifier) <> '');

create unique index homerooms_external_identifier_idx
    on homerooms (organization_id, external_identifier)
    where external_identifier is not null;

-- +goose Down

drop index if exists homerooms_external_identifier_idx;
alter table homerooms
    drop constraint if exists homerooms_external_identifier_check,
    drop column external_identifier;

alter table adults
    alter column participation_intent set not null;

alter table students
    alter column grade_level_id set not null;

