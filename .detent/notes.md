## Current handoff — issue #138

- Scope: Make the Layer 1 schema meta-test require year-scoped composite uniqueness and composite foreign keys; governed by SPEC §9.2 and ADR 0007 §5 (with ADR 0015 context).
- Key file: `backend/tests/integration/isolation_test.go`.
- Implementation: live `school_year_id` detection now gates `unique(id, organization_id, school_year_id)`; FK inspection checks both source and target year columns when the target has a live year column. Named exceptions are `audit_log` and `students.students_prior_year_fk`.
- Validation: focused integration invocation, `go test -race -v ./...`, `make lint-backend`, `make format`, `make generate && git diff --exit-code`, and `git diff --check` pass. Local database-backed gates are limited by the existing `/miniclass-postgres` name conflict and missing environment prerequisites. PR #141 current head passed all ten required CI checks.
- Open items: verify final PR/review state and leave the issue for Detent's completion-lane transition.
- Skill draft: no — the reusable tenant-isolation harness guidance already covers this method; no new broadly reusable procedure was discovered.
