-- name: CreateAuditLog :one
insert into audit_log (
    organization_id,
    actor_type,
    actor_user_id,
    actor_label,
    action,
    object_type,
    object_id,
    change_summary,
    reason,
    school_year_id,
    request_id
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
returning id, organization_id, occurred_at, actor_type, actor_user_id, actor_label,
    action, object_type, object_id, change_summary, reason, school_year_id, request_id;

-- name: CountAuditLog :one
select count(*)::bigint
from audit_log;

-- name: GetAuditLogByID :one
select id, organization_id, occurred_at, actor_type, actor_user_id, actor_label,
    action, object_type, object_id, change_summary, reason, school_year_id, request_id
from audit_log
where id = $1;
