-- +goose Up

-- This historical bootstrap table is converted to public.xid20 by 00002_xid.sql.
-- gen_random_uuid() is available on Supabase-supported PostgreSQL versions.
create table health_checks (
    id uuid primary key default gen_random_uuid(),
    status text not null default 'healthy',
    checked_at timestamptz not null default now(),
    constraint health_checks_status_check check (status <> '')
);

-- +goose Down

drop table if exists health_checks;
