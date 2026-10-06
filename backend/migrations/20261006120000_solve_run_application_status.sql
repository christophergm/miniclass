-- +goose Up

alter table solve_runs
    add column application_status text not null default 'not_applicable',
    add constraint solve_runs_application_status_check
        check (application_status in ('applied', 'superseded', 'not_applicable'));

-- +goose Down

alter table solve_runs drop constraint if exists solve_runs_application_status_check;
alter table solve_runs drop column application_status;
