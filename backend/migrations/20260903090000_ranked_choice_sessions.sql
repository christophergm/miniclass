-- +goose Up

alter table sessions
    add column ranked_choice_enabled boolean not null default false,
    add column ranked_choice_rank_depth integer,
    add column ranked_choice_deadline timestamptz,
    add constraint sessions_ranked_choice_config_check check (
        (not ranked_choice_enabled and ranked_choice_rank_depth is null and ranked_choice_deadline is null)
        or (ranked_choice_enabled and ranked_choice_rank_depth is not null and ranked_choice_rank_depth > 0 and ranked_choice_deadline is not null)
    );

create table ranked_choice_access_codes (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    session_id public.xid20 not null,
    student_id public.xid20 not null,
    code_hash text not null,
    issued_at timestamptz not null default now(),
    revoked_at timestamptz,
    constraint ranked_choice_access_codes_session_fk foreign key (session_id, organization_id, school_year_id, program_id)
        references sessions (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint ranked_choice_access_codes_membership_fk foreign key (organization_id, school_year_id, program_id, student_id)
        references program_memberships (organization_id, school_year_id, program_id, student_id) on delete restrict,
    constraint ranked_choice_access_codes_id_organization_key unique (id, organization_id),
    constraint ranked_choice_access_codes_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint ranked_choice_access_codes_hash_check check (btrim(code_hash) <> '')
);

create unique index ranked_choice_access_codes_active_student_unique
    on ranked_choice_access_codes (organization_id, school_year_id, program_id, session_id, student_id)
    where revoked_at is null;
create unique index ranked_choice_access_codes_hash_unique
    on ranked_choice_access_codes (organization_id, school_year_id, program_id, session_id, code_hash);
create index ranked_choice_access_codes_lookup_idx
    on ranked_choice_access_codes (organization_id, school_year_id, program_id, session_id, code_hash)
    where revoked_at is null;

alter table ranked_choice_access_codes enable row level security;
alter table ranked_choice_access_codes force row level security;
create policy ranked_choice_access_codes_tenant_isolation on ranked_choice_access_codes
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger ranked_choice_access_codes_closed_year_guard
before insert or update or delete on ranked_choice_access_codes
for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update on ranked_choice_access_codes to miniclass_app;

-- +goose Down

revoke all privileges on ranked_choice_access_codes from miniclass_app;
drop trigger if exists ranked_choice_access_codes_closed_year_guard on ranked_choice_access_codes;
drop policy if exists ranked_choice_access_codes_tenant_isolation on ranked_choice_access_codes;
drop index if exists ranked_choice_access_codes_lookup_idx;
drop index if exists ranked_choice_access_codes_hash_unique;
drop index if exists ranked_choice_access_codes_active_student_unique;
drop table ranked_choice_access_codes;
alter table sessions
    drop constraint if exists sessions_ranked_choice_config_check,
    drop column if exists ranked_choice_deadline,
    drop column if exists ranked_choice_rank_depth,
    drop column if exists ranked_choice_enabled;
