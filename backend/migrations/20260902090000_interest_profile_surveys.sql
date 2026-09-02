-- +goose Up

create type interest_profile_survey_state as enum ('draft', 'open', 'closed');
create type interest_profile_survey_audience_type as enum ('all_members', 'explicit_students', 'grade_level', 'response_state');
create type interest_profile_survey_response_state as enum ('responded', 'not_responded');

create table interest_profile_surveys (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    name text not null,
    introduction text not null default '',
    state interest_profile_survey_state not null default 'draft',
    opens_at timestamptz,
    closes_at timestamptz,
    audience_type interest_profile_survey_audience_type not null default 'all_members',
    audience_grade_level_id public.xid20,
    audience_prior_survey_id public.xid20,
    audience_response_state interest_profile_survey_response_state,
    scale_version text not null default 'interest-profile-3-point-v1',
    opened_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint interest_profile_surveys_program_fk foreign key (program_id, organization_id, school_year_id)
        references programs (id, organization_id, school_year_id) on delete cascade,
    constraint interest_profile_surveys_grade_fk foreign key (audience_grade_level_id, organization_id, school_year_id)
        references grade_levels (id, organization_id, school_year_id) on delete restrict,
    constraint interest_profile_surveys_prior_survey_fk foreign key (audience_prior_survey_id, organization_id, school_year_id, program_id)
        references interest_profile_surveys (id, organization_id, school_year_id, program_id) on delete restrict,
    constraint interest_profile_surveys_id_organization_key unique (id, organization_id),
    constraint interest_profile_surveys_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint interest_profile_surveys_id_organization_year_program_key unique (id, organization_id, school_year_id, program_id),
    constraint interest_profile_surveys_name_check check (btrim(name) <> ''),
    constraint interest_profile_surveys_scale_version_check check (btrim(scale_version) <> ''),
    constraint interest_profile_surveys_window_check check (closes_at is null or opens_at is null or closes_at > opens_at),
    constraint interest_profile_surveys_audience_check check (
        (audience_type = 'all_members' and audience_grade_level_id is null and audience_prior_survey_id is null and audience_response_state is null)
        or (audience_type = 'explicit_students' and audience_grade_level_id is null and audience_prior_survey_id is null and audience_response_state is null)
        or (audience_type = 'grade_level' and audience_grade_level_id is not null and audience_prior_survey_id is null and audience_response_state is null)
        or (audience_type = 'response_state' and audience_grade_level_id is null and audience_prior_survey_id is not null and audience_response_state is not null)
    ),
    constraint interest_profile_surveys_open_window_check check (state = 'draft' or closes_at is not null)
);

create table interest_profile_survey_audience_students (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    survey_id public.xid20 not null,
    student_id public.xid20 not null,
    created_at timestamptz not null default now(),
    constraint interest_profile_survey_audience_students_survey_fk foreign key (survey_id, organization_id, school_year_id, program_id)
        references interest_profile_surveys (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint interest_profile_survey_audience_students_student_fk foreign key (student_id, organization_id, school_year_id)
        references students (id, organization_id, school_year_id) on delete restrict,
    constraint interest_profile_survey_audience_students_id_organization_key unique (id, organization_id),
    constraint interest_profile_survey_audience_students_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint interest_profile_survey_audience_students_unique unique (organization_id, school_year_id, program_id, survey_id, student_id)
);

create table interest_profile_survey_questions (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    survey_id public.xid20 not null,
    interest_area_id public.xid20 not null,
    ordinal integer not null,
    label text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint interest_profile_survey_questions_survey_fk foreign key (survey_id, organization_id, school_year_id, program_id)
        references interest_profile_surveys (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint interest_profile_survey_questions_area_fk foreign key (interest_area_id, organization_id, school_year_id, program_id)
        references interest_areas (id, organization_id, school_year_id, program_id) on delete restrict,
    constraint interest_profile_survey_questions_id_organization_key unique (id, organization_id),
    constraint interest_profile_survey_questions_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint interest_profile_survey_questions_unique_area unique (organization_id, school_year_id, program_id, survey_id, interest_area_id),
    constraint interest_profile_survey_questions_unique_ordinal unique (organization_id, school_year_id, program_id, survey_id, ordinal),
    constraint interest_profile_survey_questions_ordinal_check check (ordinal > 0),
    constraint interest_profile_survey_questions_label_check check (btrim(label) <> '')
);

create table interest_profile_survey_scale_options (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    survey_id public.xid20 not null,
    value text not null,
    label text not null,
    ordinal integer not null,
    created_at timestamptz not null default now(),
    constraint interest_profile_survey_scale_options_survey_fk foreign key (survey_id, organization_id, school_year_id, program_id)
        references interest_profile_surveys (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint interest_profile_survey_scale_options_id_organization_key unique (id, organization_id),
    constraint interest_profile_survey_scale_options_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint interest_profile_survey_scale_options_unique_value unique (organization_id, school_year_id, program_id, survey_id, value),
    constraint interest_profile_survey_scale_options_unique_ordinal unique (organization_id, school_year_id, program_id, survey_id, ordinal),
    constraint interest_profile_survey_scale_options_value_check check (btrim(value) <> ''),
    constraint interest_profile_survey_scale_options_label_check check (btrim(label) <> ''),
    constraint interest_profile_survey_scale_options_ordinal_check check (ordinal > 0)
);

create table interest_profile_survey_audience_snapshots (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    survey_id public.xid20 not null,
    student_id public.xid20 not null,
    created_at timestamptz not null default now(),
    constraint interest_profile_survey_audience_snapshots_survey_fk foreign key (survey_id, organization_id, school_year_id, program_id)
        references interest_profile_surveys (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint interest_profile_survey_audience_snapshots_student_fk foreign key (student_id, organization_id, school_year_id)
        references students (id, organization_id, school_year_id) on delete restrict,
    constraint interest_profile_survey_audience_snapshots_id_organization_key unique (id, organization_id),
    constraint interest_profile_survey_audience_snapshots_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint interest_profile_survey_audience_snapshots_unique unique (organization_id, school_year_id, program_id, survey_id, student_id),
    constraint interest_profile_survey_audience_snapshots_survey_student_unique unique (survey_id, organization_id, school_year_id, program_id, student_id)
);

create table interest_profile_survey_access_codes (
    id public.xid20 primary key default public.xid(),
    organization_id public.xid20 not null references organizations (id) on delete cascade,
    school_year_id public.xid20 not null,
    program_id public.xid20 not null,
    survey_id public.xid20 not null,
    student_id public.xid20 not null,
    code_hash text not null,
    issued_at timestamptz not null default now(),
    revoked_at timestamptz,
    constraint interest_profile_survey_access_codes_survey_fk foreign key (survey_id, organization_id, school_year_id, program_id)
        references interest_profile_surveys (id, organization_id, school_year_id, program_id) on delete cascade,
    constraint interest_profile_survey_access_codes_snapshot_fk foreign key (survey_id, organization_id, school_year_id, program_id, student_id)
        references interest_profile_survey_audience_snapshots (survey_id, organization_id, school_year_id, program_id, student_id) on delete cascade,
    constraint interest_profile_survey_access_codes_id_organization_key unique (id, organization_id),
    constraint interest_profile_survey_access_codes_id_organization_year_key unique (id, organization_id, school_year_id),
    constraint interest_profile_survey_access_codes_hash_unique unique (organization_id, survey_id, code_hash),
    constraint interest_profile_survey_access_codes_hash_check check (btrim(code_hash) <> '')
);

alter table interest_profile_submissions
    add column survey_id public.xid20,
    add constraint interest_profile_submissions_survey_fk foreign key (survey_id, organization_id, school_year_id, program_id)
        references interest_profile_surveys (id, organization_id, school_year_id, program_id) on delete restrict;

create index interest_profile_submissions_survey_idx
    on interest_profile_submissions (organization_id, school_year_id, program_id, survey_id, student_id, submitted_at desc, id desc);
create index interest_profile_survey_audience_students_lookup_idx
    on interest_profile_survey_audience_students (organization_id, school_year_id, program_id, survey_id, student_id);
create index interest_profile_survey_audience_snapshots_lookup_idx
    on interest_profile_survey_audience_snapshots (organization_id, school_year_id, program_id, survey_id, student_id);
create index interest_profile_survey_access_codes_lookup_idx
    on interest_profile_survey_access_codes (organization_id, survey_id, student_id)
    where revoked_at is null;

alter table interest_profile_surveys enable row level security;
alter table interest_profile_surveys force row level security;
create policy interest_profile_surveys_tenant_isolation on interest_profile_surveys
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);
alter table interest_profile_survey_audience_students enable row level security;
alter table interest_profile_survey_audience_students force row level security;
create policy interest_profile_survey_audience_students_tenant_isolation on interest_profile_survey_audience_students
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);
alter table interest_profile_survey_questions enable row level security;
alter table interest_profile_survey_questions force row level security;
create policy interest_profile_survey_questions_tenant_isolation on interest_profile_survey_questions
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);
alter table interest_profile_survey_scale_options enable row level security;
alter table interest_profile_survey_scale_options force row level security;
create policy interest_profile_survey_scale_options_tenant_isolation on interest_profile_survey_scale_options
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);
alter table interest_profile_survey_audience_snapshots enable row level security;
alter table interest_profile_survey_audience_snapshots force row level security;
create policy interest_profile_survey_audience_snapshots_tenant_isolation on interest_profile_survey_audience_snapshots
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);
alter table interest_profile_survey_access_codes enable row level security;
alter table interest_profile_survey_access_codes force row level security;
create policy interest_profile_survey_access_codes_tenant_isolation on interest_profile_survey_access_codes
    using (organization_id = current_setting('app.organization_id')::public.xid20)
    with check (organization_id = current_setting('app.organization_id')::public.xid20);

create trigger interest_profile_surveys_set_updated_at before update on interest_profile_surveys for each row execute function public.set_updated_at();
create trigger interest_profile_survey_questions_set_updated_at before update on interest_profile_survey_questions for each row execute function public.set_updated_at();
create trigger interest_profile_surveys_closed_year_guard before insert or update or delete on interest_profile_surveys for each row execute function public.prevent_closed_school_year_mutation();
create trigger interest_profile_survey_audience_students_closed_year_guard before insert or update or delete on interest_profile_survey_audience_students for each row execute function public.prevent_closed_school_year_mutation();
create trigger interest_profile_survey_questions_closed_year_guard before insert or update or delete on interest_profile_survey_questions for each row execute function public.prevent_closed_school_year_mutation();
create trigger interest_profile_survey_scale_options_closed_year_guard before insert or update or delete on interest_profile_survey_scale_options for each row execute function public.prevent_closed_school_year_mutation();
create trigger interest_profile_survey_audience_snapshots_closed_year_guard before insert or update or delete on interest_profile_survey_audience_snapshots for each row execute function public.prevent_closed_school_year_mutation();
create trigger interest_profile_survey_access_codes_closed_year_guard before insert or update or delete on interest_profile_survey_access_codes for each row execute function public.prevent_closed_school_year_mutation();

grant select, insert, update, delete on interest_profile_surveys, interest_profile_survey_audience_students, interest_profile_survey_questions, interest_profile_survey_scale_options to miniclass_app;
grant select, insert on interest_profile_survey_audience_snapshots to miniclass_app;
grant select, insert, update on interest_profile_survey_access_codes to miniclass_app;

-- +goose Down

revoke all privileges on interest_profile_survey_access_codes, interest_profile_survey_audience_snapshots, interest_profile_survey_scale_options, interest_profile_survey_questions, interest_profile_survey_audience_students, interest_profile_surveys from miniclass_app;
drop trigger if exists interest_profile_survey_access_codes_closed_year_guard on interest_profile_survey_access_codes;
drop trigger if exists interest_profile_survey_audience_snapshots_closed_year_guard on interest_profile_survey_audience_snapshots;
drop trigger if exists interest_profile_survey_scale_options_closed_year_guard on interest_profile_survey_scale_options;
drop trigger if exists interest_profile_survey_questions_closed_year_guard on interest_profile_survey_questions;
drop trigger if exists interest_profile_survey_audience_students_closed_year_guard on interest_profile_survey_audience_students;
drop trigger if exists interest_profile_surveys_closed_year_guard on interest_profile_surveys;
drop trigger if exists interest_profile_survey_questions_set_updated_at on interest_profile_survey_questions;
drop trigger if exists interest_profile_surveys_set_updated_at on interest_profile_surveys;
drop policy if exists interest_profile_survey_access_codes_tenant_isolation on interest_profile_survey_access_codes;
drop policy if exists interest_profile_survey_audience_snapshots_tenant_isolation on interest_profile_survey_audience_snapshots;
drop policy if exists interest_profile_survey_scale_options_tenant_isolation on interest_profile_survey_scale_options;
drop policy if exists interest_profile_survey_questions_tenant_isolation on interest_profile_survey_questions;
drop policy if exists interest_profile_survey_audience_students_tenant_isolation on interest_profile_survey_audience_students;
drop policy if exists interest_profile_surveys_tenant_isolation on interest_profile_surveys;
drop index if exists interest_profile_survey_access_codes_lookup_idx;
drop index if exists interest_profile_survey_audience_snapshots_lookup_idx;
drop index if exists interest_profile_survey_audience_students_lookup_idx;
drop index if exists interest_profile_submissions_survey_idx;
alter table interest_profile_submissions drop constraint if exists interest_profile_submissions_survey_fk;
alter table interest_profile_submissions drop column if exists survey_id;
drop table interest_profile_survey_access_codes;
drop table interest_profile_survey_audience_snapshots;
drop table interest_profile_survey_scale_options;
drop table interest_profile_survey_questions;
drop table interest_profile_survey_audience_students;
drop table interest_profile_surveys;
drop type interest_profile_survey_response_state;
drop type interest_profile_survey_audience_type;
drop type interest_profile_survey_state;
