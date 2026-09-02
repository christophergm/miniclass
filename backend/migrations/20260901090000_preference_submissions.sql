-- +goose Up

create type preference_submission_channel as enum (
    'guardian',
    'student_code',
    'administrator_on_behalf'
);

create type interest_profile_rating as enum (
    'very_interested',
    'interested',
    'not_interested',
    'unrated'
);

create type ranked_choice_answer as enum (
    'ranked',
    'interested',
    'not_interested',
    'no_response'
);

-- An offering is already identified by (id, organization_id, school_year_id,
-- program_id). This additional key lets ranked responses prove that an
-- offering belongs to the same session as the submission.
alter table offerings
    add constraint offerings_id_organization_year_program_session_key
    unique (id, organization_id, school_year_id, program_id, session_id);

create table interest_profile_submissions (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    student_id public.xid20 not null,
    channel preference_submission_channel not null,
    actor_type audit_actor_type not null,
    actor_user_id public.xid20 references users (id) on delete set null,
    actor_adult_id public.xid20,
    actor_label text not null,
    submitted_at timestamptz not null default now(),
    created_at timestamptz not null default now(),
    constraint interest_profile_submissions_program_fk foreign key (program_id, organization_id, school_year_id)
        references programs (id, organization_id, school_year_id) on delete restrict,
    constraint interest_profile_submissions_membership_fk foreign key (organization_id, school_year_id, program_id, student_id)
        references program_memberships (organization_id, school_year_id, program_id, student_id) on delete restrict,
    constraint interest_profile_submissions_actor_adult_fk foreign key (actor_adult_id, organization_id, school_year_id)
        references adults (id, organization_id, school_year_id) on delete set null (actor_adult_id),
    constraint interest_profile_submissions_id_organization_key unique (id, organization_id),
    constraint interest_profile_submissions_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint interest_profile_submissions_id_organization_year_program_key unique (id, organization_id, school_year_id, program_id),
    constraint interest_profile_submissions_actor_label_check check (btrim(actor_label) <> '')
);

create table interest_profile_responses (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    submission_id public.xid20 not null,
    interest_area_id public.xid20 not null,
    response interest_profile_rating not null,
    created_at timestamptz not null default now(),
    constraint interest_profile_responses_submission_fk foreign key (submission_id, organization_id, school_year_id, program_id)
        references interest_profile_submissions (id, organization_id, school_year_id, program_id) on delete restrict,
    constraint interest_profile_responses_interest_area_fk foreign key (interest_area_id, organization_id, school_year_id, program_id)
        references interest_areas (id, organization_id, school_year_id, program_id) on delete restrict,
    constraint interest_profile_responses_id_organization_key unique (id, organization_id),
    constraint interest_profile_responses_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint interest_profile_responses_submission_area_unique unique (organization_id, school_year_id, program_id, submission_id, interest_area_id)
);

create table ranked_choice_submissions (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    session_id public.xid20 not null,
    student_id public.xid20 not null,
    channel preference_submission_channel not null,
    actor_type audit_actor_type not null,
    actor_user_id public.xid20 references users (id) on delete set null,
    actor_adult_id public.xid20,
    actor_label text not null,
    submitted_at timestamptz not null default now(),
    created_at timestamptz not null default now(),
    constraint ranked_choice_submissions_session_fk foreign key (session_id, organization_id, school_year_id, program_id)
        references sessions (id, organization_id, school_year_id, program_id) on delete restrict,
    constraint ranked_choice_submissions_membership_fk foreign key (organization_id, school_year_id, program_id, student_id)
        references program_memberships (organization_id, school_year_id, program_id, student_id) on delete restrict,
    constraint ranked_choice_submissions_actor_adult_fk foreign key (actor_adult_id, organization_id, school_year_id)
        references adults (id, organization_id, school_year_id) on delete set null (actor_adult_id),
    constraint ranked_choice_submissions_id_organization_key unique (id, organization_id),
    constraint ranked_choice_submissions_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint ranked_choice_submissions_id_organization_year_program_session_key unique (id, organization_id, school_year_id, program_id, session_id),
    constraint ranked_choice_submissions_actor_label_check check (btrim(actor_label) <> '')
);

create table ranked_choice_responses (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    session_id public.xid20 not null,
    submission_id public.xid20 not null,
    offering_id public.xid20 not null,
    response ranked_choice_answer not null,
    rank integer,
    created_at timestamptz not null default now(),
    constraint ranked_choice_responses_submission_fk foreign key (submission_id, organization_id, school_year_id, program_id, session_id)
        references ranked_choice_submissions (id, organization_id, school_year_id, program_id, session_id) on delete restrict,
    constraint ranked_choice_responses_offering_fk foreign key (offering_id, organization_id, school_year_id, program_id, session_id)
        references offerings (id, organization_id, school_year_id, program_id, session_id) on delete restrict,
    constraint ranked_choice_responses_id_organization_key unique (id, organization_id),
    constraint ranked_choice_responses_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint ranked_choice_responses_submission_offering_unique unique (organization_id, school_year_id, program_id, session_id, submission_id, offering_id),
    constraint ranked_choice_responses_rank_check check ((response = 'ranked' and rank > 0) or (response <> 'ranked' and rank is null))
);

create unique index ranked_choice_responses_submission_rank_unique
    on ranked_choice_responses (organization_id, school_year_id, program_id, session_id, submission_id, rank)
    where rank is not null;

create index interest_profile_submissions_student_idx
    on interest_profile_submissions (organization_id, school_year_id, program_id, student_id, submitted_at desc, id desc);
create index interest_profile_responses_submission_idx
    on interest_profile_responses (organization_id, school_year_id, program_id, submission_id, interest_area_id);
create index ranked_choice_submissions_student_idx
    on ranked_choice_submissions (organization_id, school_year_id, program_id, session_id, student_id, submitted_at desc, id desc);
create index ranked_choice_responses_submission_idx
    on ranked_choice_responses (organization_id, school_year_id, program_id, session_id, submission_id, offering_id);

alter table interest_profile_submissions enable row level security;
alter table interest_profile_submissions force row level security;
create policy interest_profile_submissions_tenant_isolation on interest_profile_submissions
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);
alter table interest_profile_responses enable row level security;
alter table interest_profile_responses force row level security;
create policy interest_profile_responses_tenant_isolation on interest_profile_responses
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);
alter table ranked_choice_submissions enable row level security;
alter table ranked_choice_submissions force row level security;
create policy ranked_choice_submissions_tenant_isolation on ranked_choice_submissions
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);
alter table ranked_choice_responses enable row level security;
alter table ranked_choice_responses force row level security;
create policy ranked_choice_responses_tenant_isolation on ranked_choice_responses
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger interest_profile_submissions_closed_year_guard
before insert or update or delete on interest_profile_submissions
for each row execute function public.prevent_closed_school_year_mutation();
create trigger interest_profile_responses_closed_year_guard
before insert or update or delete on interest_profile_responses
for each row execute function public.prevent_closed_school_year_mutation();
create trigger ranked_choice_submissions_closed_year_guard
before insert or update or delete on ranked_choice_submissions
for each row execute function public.prevent_closed_school_year_mutation();
create trigger ranked_choice_responses_closed_year_guard
before insert or update or delete on ranked_choice_responses
for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert on interest_profile_submissions, interest_profile_responses, ranked_choice_submissions, ranked_choice_responses to miniclass_app;

-- +goose Down

revoke all privileges on interest_profile_submissions, interest_profile_responses, ranked_choice_submissions, ranked_choice_responses from miniclass_app;
drop trigger if exists ranked_choice_responses_closed_year_guard on ranked_choice_responses;
drop trigger if exists ranked_choice_submissions_closed_year_guard on ranked_choice_submissions;
drop trigger if exists interest_profile_responses_closed_year_guard on interest_profile_responses;
drop trigger if exists interest_profile_submissions_closed_year_guard on interest_profile_submissions;
drop policy if exists ranked_choice_responses_tenant_isolation on ranked_choice_responses;
drop policy if exists ranked_choice_submissions_tenant_isolation on ranked_choice_submissions;
drop policy if exists interest_profile_responses_tenant_isolation on interest_profile_responses;
drop policy if exists interest_profile_submissions_tenant_isolation on interest_profile_submissions;
drop index if exists ranked_choice_responses_submission_rank_unique;
drop index if exists ranked_choice_responses_submission_idx;
drop index if exists ranked_choice_submissions_student_idx;
drop index if exists interest_profile_responses_submission_idx;
drop index if exists interest_profile_submissions_student_idx;
drop table ranked_choice_responses;
drop table ranked_choice_submissions;
drop table interest_profile_responses;
drop table interest_profile_submissions;
alter table offerings drop constraint if exists offerings_id_organization_year_program_session_key;
drop type ranked_choice_answer;
drop type interest_profile_rating;
drop type preference_submission_channel;
