-- name: ProofHealthChecks :many
SELECT id, status, checked_at
FROM health_checks
ORDER BY checked_at DESC;

-- name: ProofHealthCheckCount :one
SELECT count(*) FROM health_checks;
