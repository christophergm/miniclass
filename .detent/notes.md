## Current handoff — issue #127

- Scope: Gate annual program membership on a known student grade; governing contract SPEC §§5.2, 8.3, 10.1, 12.1, 14.2 and ADR 0014.
- Key files: `backend/migrations/20260830200000_programmes.sql`, `backend/internal/data/program.go`, `backend/internal/program/service.go`, `backend/internal/api/handlers/program.go`, `frontend/src/features/programs/ProgramPages.tsx`.
- Implementation: year-scoped `programs` and `program_memberships` tables with composite FKs, forced RLS and closed-year triggers; audited program/membership services; known-grade refusal names the student; later grade clearing retains and flags membership; missing-grade count links to the roster; generated sqlc/OpenAPI and frontend client/page are included.
- Isolation: Layer 2 registry entries and integration coverage exercise both new tables, cross-tenant invisibility/mutation/foreign-parent rejection, and the grade gate/flag behavior with organization-scoped fixtures.
- Validation: direct `go test -race -v ./...`, `make lint-backend`, `make format`, `make generate`, and `git diff --check` pass. `make test-backend` is blocked before tests by the existing `/miniclass-postgres` container-name conflict. Frontend test/build cannot start because `openapi-typescript` is not installed; frontend lint cannot write Bun temp files. Migration round-trip lacks `MIGRATION_ROUNDTRIP_DATABASE_URL`. Smoke requires `.env`.
- Open items: stage, run cached whitespace/generated checks, commit/push, open/update PR with `Fixes #127` and SPEC/ADR citations, then verify CI/review state.
- Skill draft: no — the reusable tenant-entity and backend procedures already exist and this change did not expose a new broadly reusable method.
