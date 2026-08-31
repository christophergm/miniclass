-- +goose Up

create table program_objective_weights (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    rank_high_max integer not null default 3,
    deficit_unwanted_increment double precision not null default 2.0,
    deficit_neutral_increment double precision not null default 1.0,
    deficit_acceptable_increment double precision not null default 0.5,
    deficit_influence double precision not null default 0.5,
    repeat_offering_penalty double precision not null default 10.0,
    repeat_interest_area_penalty double precision not null default 5.0,
    tag_prefers_weight double precision not null default 3.0,
    tag_discourages_weight double precision not null default 3.0,
    pairing_prefers_weight double precision not null default 5.0,
    pairing_discourages_weight double precision not null default 5.0,
    below_minimum_enrollment_penalty double precision not null default 1.0,
    tag_balance_penalty double precision not null default 1.0,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint program_objective_weights_program_fk foreign key (program_id, organization_id, school_year_id)
        references programs (id, organization_id, school_year_id) on delete cascade,
    constraint program_objective_weights_id_organization_key unique (id, organization_id),
    constraint program_objective_weights_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint program_objective_weights_program_unique unique (organization_id, school_year_id, program_id),
    constraint program_objective_weights_rank_check check (rank_high_max >= 2),
    constraint program_objective_weights_nonnegative_check check (
        deficit_unwanted_increment >= 0 and deficit_neutral_increment >= 0 and
        deficit_acceptable_increment >= 0 and deficit_influence >= 0 and
        repeat_offering_penalty >= 0 and repeat_interest_area_penalty >= 0 and
        tag_prefers_weight >= 0 and tag_discourages_weight >= 0 and
        pairing_prefers_weight >= 0 and pairing_discourages_weight >= 0 and
        below_minimum_enrollment_penalty >= 0 and tag_balance_penalty >= 0
    )
);

create table session_objective_weight_overrides (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    session_id public.xid20 not null,
    rank_high_max integer,
    deficit_unwanted_increment double precision,
    deficit_neutral_increment double precision,
    deficit_acceptable_increment double precision,
    deficit_influence double precision,
    repeat_offering_penalty double precision,
    repeat_interest_area_penalty double precision,
    tag_prefers_weight double precision,
    tag_discourages_weight double precision,
    pairing_prefers_weight double precision,
    pairing_discourages_weight double precision,
    below_minimum_enrollment_penalty double precision,
    tag_balance_penalty double precision,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint session_objective_weight_overrides_session_fk foreign key (session_id, organization_id, school_year_id, program_id)
        references sessions (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint session_objective_weight_overrides_id_organization_key unique (id, organization_id),
    constraint session_objective_weight_overrides_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint session_objective_weight_overrides_session_unique unique (organization_id, school_year_id, program_id, session_id),
    constraint session_objective_weight_overrides_rank_check check (rank_high_max is null or rank_high_max >= 2),
    constraint session_objective_weight_overrides_nonnegative_check check (
        (deficit_unwanted_increment is null or deficit_unwanted_increment >= 0) and
        (deficit_neutral_increment is null or deficit_neutral_increment >= 0) and
        (deficit_acceptable_increment is null or deficit_acceptable_increment >= 0) and
        (deficit_influence is null or deficit_influence >= 0) and
        (repeat_offering_penalty is null or repeat_offering_penalty >= 0) and
        (repeat_interest_area_penalty is null or repeat_interest_area_penalty >= 0) and
        (tag_prefers_weight is null or tag_prefers_weight >= 0) and
        (tag_discourages_weight is null or tag_discourages_weight >= 0) and
        (pairing_prefers_weight is null or pairing_prefers_weight >= 0) and
        (pairing_discourages_weight is null or pairing_discourages_weight >= 0) and
        (below_minimum_enrollment_penalty is null or below_minimum_enrollment_penalty >= 0) and
        (tag_balance_penalty is null or tag_balance_penalty >= 0)
    )
);

create index session_objective_weight_overrides_program_idx on session_objective_weight_overrides (organization_id, school_year_id, program_id, session_id);

-- Backfill before RLS and the closed-year trigger are installed so an upgrade
-- remains valid for existing programmes, including closed school years.
-- The referenced programs table is FORCE RLS. Temporarily remove the owner
-- exception while the migrator validates the composite foreign key; the
-- migration session has no tenant setting and must see every existing row.
alter table programs no force row level security;
insert into program_objective_weights (organization_id, school_year_id, program_id)
select organization_id, school_year_id, id from programs;
alter table programs force row level security;

alter table program_objective_weights enable row level security;
alter table program_objective_weights force row level security;
create policy program_objective_weights_tenant_isolation on program_objective_weights
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);
alter table session_objective_weight_overrides enable row level security;
alter table session_objective_weight_overrides force row level security;
create policy session_objective_weight_overrides_tenant_isolation on session_objective_weight_overrides
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger program_objective_weights_set_updated_at before update on program_objective_weights for each row execute function public.set_updated_at();
create trigger session_objective_weight_overrides_set_updated_at before update on session_objective_weight_overrides for each row execute function public.set_updated_at();
create trigger program_objective_weights_closed_year_guard before insert or update or delete on program_objective_weights for each row execute function public.prevent_closed_school_year_mutation();
create trigger session_objective_weight_overrides_closed_year_guard before insert or update or delete on session_objective_weight_overrides for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update, delete on program_objective_weights, session_objective_weight_overrides to miniclass_app;

-- +goose Down

revoke all privileges on program_objective_weights, session_objective_weight_overrides from miniclass_app;
drop trigger if exists session_objective_weight_overrides_closed_year_guard on session_objective_weight_overrides;
drop trigger if exists program_objective_weights_closed_year_guard on program_objective_weights;
drop trigger if exists session_objective_weight_overrides_set_updated_at on session_objective_weight_overrides;
drop trigger if exists program_objective_weights_set_updated_at on program_objective_weights;
drop policy if exists session_objective_weight_overrides_tenant_isolation on session_objective_weight_overrides;
drop policy if exists program_objective_weights_tenant_isolation on program_objective_weights;
drop index if exists session_objective_weight_overrides_program_idx;
drop table session_objective_weight_overrides;
drop table program_objective_weights;
