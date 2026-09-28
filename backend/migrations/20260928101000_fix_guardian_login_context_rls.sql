-- +goose Up

-- The original lookup function was created before its RLS interaction was
-- exercised. Recreate it so each tenant-domain read uses the same local scope
-- setting as internal/data rather than attempting an unscoped domain read.
-- +goose StatementBegin
create or replace function public.find_guardian_login_contexts(target_email text)
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

-- +goose Down

-- This migration only corrects a function definition. The predecessor
-- migration's down path removes the function.
