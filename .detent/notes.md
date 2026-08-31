## Current handoff — issue #139

- Scope: Re-scope the existing grade and homeroom vocabularies to school years, including migration/backfill/down path, explicit year predicates, data/service/audit/API/ingest/seed/factory/registry/frontend updates, generated artifacts, and isolation/closed-year tests. Governing contract: SPEC §§8.1, 8.2, 10.1, 11.1, 20.1; ADRs 0007, 0010, 0014, 0015.
- Repository state: fast-forwarded to merged dependency commit `f391ece` from `origin/main`; no implementation or PR exists for #139.
- Dependency: #138 is closed as completed and PR #141 merged at `f391ece`; the native GitHub `blocked_by` relation is terminal.
- Skills read: `.detent/skills/add-tenant-scoped-entity.md` and `.detent/skills/postgres-tenant-isolation-harness.md`.
- Implementation: migration, generated sqlc/OpenAPI, year-aware data/service/audit/API paths, registry/factories, seed, ingest, frontend resources/hooks/settings/roster callers, and scoped regressions are complete.
- Validation: `go test -race -v ./... -count=1`, `make lint-backend`, `make format`, `make generated-code-drift`, and `git diff --check` pass; database-backed integration tests are skipped without TEST_DATABASE_URL/TEST_APP_DATABASE_URL. Top-level backend tests are blocked by the existing `/miniclass-postgres` Docker name conflict; migration round-trip lacks MIGRATION_ROUNDTRIP_DATABASE_URL; frontend tools/config are unavailable.
- Open items: commit/push/open a non-draft PR with `Fixes #139` and SPEC/ADR citations, then verify current-head CI/review state and update the Workpad.
- Skill draft: no — existing tenant-entity and isolation-harness skills cover the reusable procedure; no new broadly reusable method was discovered.
