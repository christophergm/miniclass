## Current handoff — issue #139

- Scope: Re-scope the existing grade and homeroom vocabularies to school years, including migration/backfill/down path, explicit year predicates, data/service/audit/API/ingest/seed/factory/registry/frontend updates, generated artifacts, and isolation/closed-year tests. Governing contract: SPEC §§8.1, 8.2, 10.1, 11.1, 20.1; ADRs 0007, 0010, 0014, 0015.
- Repository state: implementation is committed through `39b764f` and pushed to PR [#150](https://github.com/christophergm/miniclass/pull/150), based on merged dependency commit `f391ece`.
- Dependency: #138 is closed as completed and PR #141 merged at `f391ece`; the native GitHub `blocked_by` relation is terminal.
- Skills read: `.detent/skills/add-tenant-scoped-entity.md` and `.detent/skills/postgres-tenant-isolation-harness.md`.
- Implementation: migration, generated sqlc/OpenAPI, year-aware data/service/audit/API paths, registry/factories, seed, ingest, frontend resources/hooks/settings/roster callers, and scoped regressions are complete.
- Validation: all ten required CI checks pass on PR #150 head `39b764f`, including backend tests, migration round-trip, frontend tests/build/lint, generated drift, formatting, lint, and developer tooling. Focused local backend tests, `make lint-backend`, `make format`, and `git diff --check` pass; local DB-backed gates remain unavailable without the configured database/container environment.
- Open items: human review of PR #150; the PR is open, non-draft, references `Fixes #139`, and has no actionable review comments.
- Skill draft: no — existing tenant-entity and isolation-harness skills cover the reusable procedure; no new broadly reusable method was discovered.

## Current handoff — issue #142

- Scope: Programme-scoped, year-scoped `interest_areas` vocabulary with stable xid identity, mutable labels, ordinal ordering, soft retirement/reactivation, audited service/API mutations, closed-year trigger protection, Layer 2 registry coverage, and frontend management wiring. Governing contract: SPEC §§8.7, 9.1–9.2, 12.1, 12.3–12.4, 20.1; ADRs 0007, 0008, 0010.
- Key files: `backend/migrations/20260831110000_interest_areas.sql`, `backend/sql/queries/program.sql`, `backend/internal/data/program.go`, `backend/internal/program/service.go`, `backend/internal/api/handlers/program.go`, `backend/internal/testing/registry/interest_area.go`, `backend/tests/integration/program_test.go`, and frontend program resource/page files.
- Repository state: PR #152 is open from commit `a6dcedf`, rebased onto `origin/main` at `df361d3`.
- Validation: `go test ./internal/...`, focused integration compile/registry test, `make lint-backend`, `make format`, and `make generate` pass. Database-backed integration cases skip locally because `TEST_DATABASE_URL` and `TEST_APP_DATABASE_URL` are unset. Frontend tests/build fail because `openapi-typescript` is unavailable; frontend lint fails because Bun cannot write its temp cache.
- Validation: all ten required CI checks pass on PR #152 head `a6dcedf`; no actionable review comments remain. Local root backend/smoke gates remain environment-blocked by the existing `/miniclass-postgres` name conflict or missing `.env`, while migration round-trip lacks its configured URL.
- Open items: Detent owns the completion-lane transition after this handoff; no dependency blocker or human action is declared.
- Skill draft: no — the existing tenant-entity and isolation-harness skills cover the reusable procedure; no new broadly reusable method was discovered.

## Current handoff — issue #140

- Scope: Split the year-scoped grade/homeroom vocabulary into `/y/:schoolYearId/vocabulary` and keep the organisation label plus owner-only administrators at top-level `/settings`, including closed-year read-only treatment and setup guidance.
- Dependency: issue #139 is now closed and PR #150 is merged; this worktree was three commits behind `origin/main` before the dependency update.
- Key files after dependency lands: `frontend/src/App.tsx`, `frontend/src/features/settings/SettingsPage.tsx`, `frontend/src/features/settings/useSettings.ts`, `frontend/src/features/school-years/SchoolYearPages.tsx`, `frontend/src/features/people/PeoplePages.tsx`, and frontend tests.
- Implementation: added the guarded `/y/:schoolYearId/vocabulary` page with configured-label headings, empty setup guidance, closed-year read-only controls, and shared vocabulary sections; moved organization label and administrators to top-level `/settings`; added workspace/app-shell navigation; preserved year-keyed roster vocabulary caching and invalidation.
- Validation: `git diff --check` and backend `make format` pass; backend generation completed without drift. Frontend tests/build/lint cannot run locally because `openapi-typescript` is missing and `bunx` cannot write its temp cache. Full `make check` stops at the existing `/miniclass-postgres` container-name conflict before CI gates run.
- Validation: PR #151 head `4061ab0` is green on all ten required CI checks: Backend tests, Backend lint, Backend format, Generated code drift, Migration round-trip, Frontend tests, Frontend build, Frontend lint, Repository formatting, and Developer tooling. `git diff --check` passes locally; local `make check` remains blocked at the existing `/miniclass-postgres` name conflict before the gates.
- Open items: Detent owns the completion-lane transition; no actionable PR comments or reviews remain.
- Skill draft: no — this was a focused frontend split and exposed no broadly reusable procedure beyond existing skills.

## Current work — issue #143

- Scope: Add year-scoped programme sessions and meeting dates per SPEC §§8.5, 14.1, 11.1, 20.1; dependency #142 is closed.
- Workpad: issue comment `5476900037` contains the persistent plan and `detent-status: in_progress`.
- Skills read: `.detent/skills/add-tenant-scoped-entity.md` and `.detent/skills/postgres-tenant-isolation-harness.md`.
- Implementation: session/meeting-date migration, sqlc/OpenAPI generation, tenant data/service/audit/API paths, registry/factories, frontend resources/hooks, and integration coverage are complete. The final test correction makes meeting-date results explicitly chronological and supplies two dates to exercise closed-year deletion.
- Repository state: commits `47bc421`, `7764c4a`, `2584a3a`, and `0002860` are pushed to PR [#153](https://github.com/christophergm/miniclass/pull/153).
- Validation: backend tests, lint, format, generated-code drift, and repository diff checks pass locally. Root `make test-backend` remains unavailable because the existing `/miniclass-postgres` container name is in use; `make test-migrations` lacks its configured database URL; `make smoke` lacks `.env`; frontend local commands lack `openapi-typescript`/Bun temp-cache access. All ten required CI checks pass on PR #153 head `0002860`.
- Open items: none; PR #153 is open and non-draft with no actionable review comments. Detent owns the completion-lane transition.

## Current work — issue #144

- Scope: Add year-scoped class offerings beneath sessions with Phase 3 catalog fields, composite grade-window and optional interest-area references, audited CRUD/API, closed-year protection, frontend client wiring, and Layer 2 isolation coverage. Governing contract: SPEC §§8.4, 10.1, 12.4, 14.2, 20.1; ADRs 0007, 0008, 0010, 0015.
- Dependency: #143 is merged on current origin/main; worktree is clean and based on b33a8a8.
- Workpad: issue comment 5477388223 contains the persistent plan and in_progress status.
- Skills read: `.detent/skills/add-tenant-scoped-entity.md` and `.detent/skills/postgres-tenant-isolation-harness.md`.
- Implementation in progress: dedicated offerings migration, data/service/API/frontend paths, generated artifacts, registry/factory, and isolation/closed-year tests.
- Open items: focused tests, all ten quality gates, PR/CI/review handoff, and explicit skill-draft decision.
- Skill draft: no — existing tenant-entity and isolation-harness skills cover the reusable procedure; no new broadly reusable method was discovered.
