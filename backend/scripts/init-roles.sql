-- Local and CI bootstrap roles. The migration repeats this provision defensively
-- for databases whose cluster initialization scripts were not run.
do $$
begin
    if not exists (select 1 from pg_roles where rolname = 'miniclass_migrator') then
        begin
            create role miniclass_migrator login password 'miniclass_migrator_dev_password';
        exception
            when duplicate_object then null;
        end;
    end if;
end
$$;

do $$
begin
    if not exists (select 1 from pg_roles where rolname = 'miniclass_app') then
        begin
            create role miniclass_app login password 'miniclass_app_dev_password';
        exception
            when duplicate_object then null;
        end;
    end if;
end
$$;

alter role miniclass_app
    nosuperuser nocreatedb nocreaterole noinherit noreplication nobypassrls;
alter role miniclass_app set statement_timeout = '10s';
