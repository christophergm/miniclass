-- Development seed data. The initial migration currently provides health_checks.
-- Keep this fixture idempotent so `make seed` can be rerun safely.
INSERT INTO health_checks (status)
SELECT 'seeded'
WHERE NOT EXISTS (
    SELECT 1
    FROM health_checks
    WHERE status = 'seeded'
);
