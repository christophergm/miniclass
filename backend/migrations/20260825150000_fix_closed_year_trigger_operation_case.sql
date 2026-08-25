-- +goose Up

-- PostgreSQL exposes TG_OP as INSERT, UPDATE, or DELETE. The original shared
-- trigger compared those values using lowercase literals, which caused every
-- DELETE to return NEW (NULL) and be silently skipped.
-- +goose StatementBegin
create or replace function public.prevent_closed_school_year_mutation() returns trigger language plpgsql as
$$
declare
    old_school_year_id public.xid20;
    new_school_year_id public.xid20;
    parent_state school_year_state;
    reopen_id text := nullif(btrim(coalesce(current_setting('app.school_year_reopen_id', true), '')), '');
    reopen_prepared boolean := coalesce(current_setting('app.school_year_reopen', true), '') = 'true'
        and nullif(btrim(coalesce(current_setting('app.school_year_reopen_reason', true), '')), '') is not null;
begin
    if tg_table_name = 'school_years' then
        if tg_op = 'INSERT' then
            return new;
        end if;
        if tg_op = 'DELETE' then
            if old.state = 'closed' then
                raise exception 'school year is closed'
                    using errcode = 'P0001',
                          detail = 'records in a closed school year are read-only',
                          hint = 'reopen the school year through the Owner-only lifecycle operation';
            end if;
            return old;
        end if;

        if old.state = 'closed' then
            if new.state = 'active' and reopen_prepared and reopen_id = old.id::text then
                return new;
            end if;
            raise exception 'school year is closed'
                using errcode = 'P0001',
                      detail = 'records in a closed school year are read-only',
                      hint = 'reopen the school year through the Owner-only lifecycle operation';
        end if;
        return new;
    end if;

    if tg_op <> 'INSERT' then
        old_school_year_id := nullif(to_jsonb(old)->>'school_year_id', '')::public.xid20;
        if old_school_year_id is not null then
            select state into parent_state from school_years where id = old_school_year_id;
            if parent_state = 'closed' and not (reopen_prepared and reopen_id = old_school_year_id::text) then
                raise exception 'school year is closed'
                    using errcode = 'P0001',
                          detail = 'records in a closed school year are read-only',
                          hint = 'reopen the school year through the Owner-only lifecycle operation';
            end if;
        end if;
    end if;

    if tg_op <> 'DELETE' then
        new_school_year_id := nullif(to_jsonb(new)->>'school_year_id', '')::public.xid20;
        if new_school_year_id is not null then
            select state into parent_state from school_years where id = new_school_year_id;
            if parent_state = 'closed' and not (reopen_prepared and reopen_id = new_school_year_id::text) then
                raise exception 'school year is closed'
                    using errcode = 'P0001',
                          detail = 'records in a closed school year are read-only',
                          hint = 'reopen the school year through the Owner-only lifecycle operation';
            end if;
        end if;
    end if;

    if tg_op = 'DELETE' then
        return old;
    end if;
    return new;
end;
$$;
-- +goose StatementEnd

-- +goose Down

-- The corrected trigger is required by all year-scoped tables; rollback is
-- intentionally a no-op rather than restoring the silently-skipped DELETE bug.
