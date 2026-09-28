-- +goose Up

-- The function is security definer so it can iterate tenant rows protected by
-- forced RLS. Receive the caller's schema as a value and qualify every domain
-- relation dynamically; this preserves a fixed safe search_path and lets the
-- schema-isolated integration harness exercise the same function.
-- +goose StatementBegin
create or replace function public.find_guardian_login_contexts(target_email text, target_schema name)
returns table (
    organization_id public.xid20,
    school_year_id public.xid20,
    adult_id public.xid20,
    organization_name text,
    school_year_label text
)
language plpgsql
volatile
security definer
set search_path = pg_catalog
as $$
declare
    candidate_organization record;
begin
    for candidate_organization in execute format(
        'select id, name from %1$I.organizations order by name, id',
        target_schema
    ) loop
        perform set_config('app.organization_id', candidate_organization.id::text, true);
        return query execute format(
            'select distinct
                adult.organization_id,
                adult.school_year_id,
                adult.id,
                $2::text,
                school_year.label
            from %1$I.adults adult
            join %1$I.guardian_relationships relationship
              on relationship.organization_id = adult.organization_id
             and relationship.school_year_id = adult.school_year_id
             and relationship.adult_id = adult.id
            join %1$I.students student
              on student.organization_id = relationship.organization_id
             and student.school_year_id = relationship.school_year_id
             and student.id = relationship.student_id
             and student.deleted_at is null
            join %1$I.school_years school_year
              on school_year.organization_id = adult.organization_id
             and school_year.id = adult.school_year_id
            where adult.organization_id = $3
              and adult.deleted_at is null
              and lower(adult.email) = lower(btrim($1))
              and school_year.state in (''setup'', ''active'', ''closed'')',
            target_schema
        ) using target_email, candidate_organization.name, candidate_organization.id;
    end loop;
end
$$;
-- +goose StatementEnd

grant execute on function public.find_guardian_login_contexts(text, name) to miniclass_app;
revoke execute on function public.find_guardian_login_contexts(text) from miniclass_app;
drop function public.find_guardian_login_contexts(text);

-- +goose Down

-- +goose StatementBegin
create function public.find_guardian_login_contexts(target_email text)
returns table (
    organization_id public.xid20,
    school_year_id public.xid20,
    adult_id public.xid20,
    organization_name text,
    school_year_label text
)
language plpgsql
volatile
security definer
set search_path = public
as $$
declare
    candidate_organization record;
begin
    for candidate_organization in select id, name from organizations order by name, id loop
        perform set_config('app.organization_id', candidate_organization.id::text, true);
        return query
        select distinct
            adult.organization_id,
            adult.school_year_id,
            adult.id,
            candidate_organization.name,
            school_year.label
        from adults adult
        join guardian_relationships relationship
          on relationship.organization_id = adult.organization_id
         and relationship.school_year_id = adult.school_year_id
         and relationship.adult_id = adult.id
        join students student
          on student.organization_id = relationship.organization_id
         and student.school_year_id = relationship.school_year_id
         and student.id = relationship.student_id
         and student.deleted_at is null
        join school_years school_year
          on school_year.organization_id = adult.organization_id
         and school_year.id = adult.school_year_id
        where adult.organization_id = candidate_organization.id
          and adult.deleted_at is null
          and lower(adult.email) = lower(btrim(target_email))
          and school_year.state in ('setup', 'active', 'closed');
    end loop;
end
$$;
-- +goose StatementEnd

grant execute on function public.find_guardian_login_contexts(text) to miniclass_app;
revoke execute on function public.find_guardian_login_contexts(text, name) from miniclass_app;
drop function public.find_guardian_login_contexts(text, name);
