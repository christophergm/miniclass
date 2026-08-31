## Current work — issue #161

- Scope: Clarify session lifecycle control labels and direct transitions per SPEC §§5.2, 5.4, 14.3–14.5, 20.1, 22.4.
- Key files: `frontend/src/features/programs/ProgramPages.tsx` and `frontend/src/features/programs/ProgramPages.test.tsx`.
- Implementation: exact `Choose allowed state` placeholder; forward legal transitions show `Transition` and call the existing direct API path; backward legal transitions show `Preview Transition…` and retain reason/confirmation safeguards; tests cover legal targets, direct payload, and preview confirmation flow.
- Validation: commit `c65c242` is pushed to PR #166. `make lint-backend`, `make format`, `make generate && git diff --exit-code`, and `git diff --check` pass. `make check` stops at backend container setup because Docker reports all predefined address pools are exhausted; migration round-trip lacks its URL; frontend test/build lack `openapi-typescript`/Vitest; frontend lint lacks Bun temp-cache access; smoke lacks `.env`. All ten PR checks pass on the current head; CI duration was 1m49s, with Generated code drift (1m49s), Backend tests (1m41s), and Backend lint (1m21s) the slowest.
- Open items: none; PR #166 is open, non-draft, references `Fixes #161`, and has no actionable review comments. Detent owns the completion-lane transition.
- Skill draft: no — this is a focused presentation/interaction change and exposed no broadly reusable procedure.

## Current handoff — issue #139

- Scope: Re-scope the existing grade and homeroom vocabularies to school years, including migration/backfill/down path, explicit year predicates, data/service/audit/API/ingest/seed/factory/registry/frontend updates, generated artifacts, and isolation/closed-year tests. Governing contract: SPEC §§8.1, 8.2, 10.1, 11.1, 20.1; ADRs 0007, 0010, 0014, 0015.
- Repository state: implementation is committed through `39b764f` and pushed to PR [#150](https://github.com/christophergm/miniclass/pull/150), based on merged dependency commit `f391ece`.
- Dependency: #138 is closed as completed and PR #141 merged at `f391ece`; the native GitHub `blocked_by` relation is terminal.
- Skills read: `.detent/skills/add-tenant-scoped-entity.md` and `.detent/skills/postgres-tenant-isolation-harness.md`.
- Implementation: migration, generated sqlc/OpenAPI, year-aware data/service/audit/API paths, registry/factories, seed, ingest, frontend resources/hooks/settings/roster callers, and scoped regressions are complete.
- Validation: all ten required CI checks pass on PR #150 head `39b764f`, including backend tests, migration round-trip, frontend tests/build/lint, generated drift, formatting, lint, and developer tooling. Focused local backend tests, `make lint-backend`, `make format`, and `git diff --check` pass; local DB-backed gates remain unavailable without the configured database/container environment.
- Open items: human review of PR #150; the PR is open, non-draft, references `Fixes #139`, and has no actionable review comments.
- Skill draft: no — existing tenant-entity and isolation-harness skills cover the reusable procedure; no new broadly reusable method was discovered.

## Current work — issue #148

- Scope: programme objective-weight defaults and nullable per-session overrides with effective-weight reads, audited writes, closed-year protection, tenant isolation, generated sqlc/OpenAPI, and frontend resources/hooks. Governing contract: SPEC §§12.1, 14.1, 17.7, 20.2; ADRs 0003 and 0010.
- Key files: `backend/migrations/20260831150000_objective_weights.sql`, `backend/sql/queries/objective_weights.sql`, `backend/internal/data/objective_weights.go`, `backend/internal/program/objective_weights.go`, `backend/internal/api/handlers/objective_weights.go`, registry/objective-weight files, and frontend resource/hooks.
- Implementation: explicit typed defaults and nullable session override tables; migration backfills before closed-year triggers and temporarily removes the parent FORCE-RLS owner exception during the composite-FK backfill; effective merge is deterministic; session override writes require an audit reason; five assignment-capability API operations and frontend wrappers/hooks are wired. Writable program defaults use a distinct OpenAPI input schema, and the two writes use the repository’s established PATCH convention so generated clients retain request bodies.
- Validation: `GOTOOLCHAIN=local go test ./...`, focused integration compile, `make lint-backend`, `make format`, `make generate`, and `git diff --check` pass after the fixes. PR #156 final head `a99f3f0` passes all ten required CI checks, including backend tests, migration round-trip, frontend build/tests/lint, generated drift, repository formatting, and developer tooling. Local root test/migration/frontend/smoke gates remain environment-limited as previously noted.
- Open items: human review of PR #156; it is open, non-draft, references `Fixes #148`, and has no actionable review comments. No dependency blocker or human action declared.
- Skill draft: no — this was a scoped configuration model and the existing tenant-entity/isolation procedures covered the reusable work.

## Current handoff — issue #142

- Scope: Programme-scoped, year-scoped `interest_areas` vocabulary with stable xid identity, mutable labels, ordinal ordering, soft retirement/reactivation, audited service/API mutations, closed-year trigger protection, Layer 2 registry coverage, and frontend management wiring. Governing contract: SPEC §§8.7, 9.1–9.2, 12.1, 12.3–12.4, 20.1; ADRs 0007, 0008, 0010.
- Key files: `backend/migrations/20260831110000_interest_areas.sql`, `backend/sql/queries/program.sql`, `backend/internal/data/program.go`, `backend/internal/program/service.go`, `backend/internal/api/handlers/program.go`, `backend/internal/testing/registry/interest_area.go`, `backend/tests/integration/program_test.go`, and frontend program resource/page files.
- Repository state: PR #152 is open from commit `a6dcedf`, rebased onto `origin/main` at `df361d3`.
- Validation: `go test ./internal/...`, focused integration compile/registry test, `make lint-backend`, `make format`, and `make generate` pass. Database-backed integration cases skip locally because `TEST_DATABASE_URL` and `TEST_APP_DATABASE_URL` are unset. Frontend tests/build fail because `openapi-typescript` is unavailable; frontend lint fails because Bun cannot write its temp cache.
- Validation: all ten required CI checks pass on PR #152 head `a6dcedf`; no actionable review comments remain. Local root backend/smoke gates remain environment-blocked by the existing `/miniclass-postgres` name conflict or missing `.env`, while migration round-trip lacks its configured URL.
- Open items: Detent owns the completion-lane transition after this handoff; no dependency blocker or human action is declared.
- Skill draft: no — the existing tenant-entity and isolation-harness skills cover the reusable procedure; no new broadly reusable method was discovered.

## Current work — issue #162

- Scope: Move assignment objective tuning from programme/session authoring into dedicated pages per SPEC §§5.2, 12.1, 14.1, 17.4, 17.7, 20.2, 22.4.
- Implementation in progress: `frontend/src/features/programs/ProgramPages.tsx` now has dedicated programme and session objective pages with all 13 parameters, explanations, effective/inherited display, explicit session audit reason, and closed-year read-only behavior; `frontend/src/App.tsx` adds both routes; authoring pages link to them and no longer render objective inputs.
- Tests added for discoverability, route presentation, all parameters, edits, inherited/effective values, and closed-year disabling.
- Validation: local `make generate`, backend lint/format, backend generated-code drift, and `git diff --check` pass. Local frontend commands are unavailable because dependencies are not installed. PR #167 workflow-dispatch validation for head `4619aec` in run `33430195023` passes all ten required checks; GitHub's PR-specific check view is not populated by workflow-dispatch, and no new pull-request run appeared for the notes-only push.
- Open items: keep the current-head check evidence in the Workpad, inspect review comments, and update the issue Workpad completion status.
- Skill draft: no — this is a focused frontend relocation using existing objective API/hooks; no broadly reusable procedure was discovered.

## Current work — issue #149

- Scope: consolidate the Phase 3 programme/session/catalog authoring flow per SPEC §§12, 14, 8.3–8.5, and 22.4; dependencies #142–#148 are closed.
- Key files: `frontend/src/features/programs/ProgramPages.tsx`, `frontend/src/features/programs/usePrograms.ts`, `frontend/src/lib/apiResources.ts`, `frontend/src/App.tsx`, `frontend/src/features/programs/ProgramPages.test.tsx`, and `backend/tests/integration/phase3_authoring_test.go`.
- Implementation: nested session authoring route; programme detail now includes membership, interest areas, sessions, meeting-date entry, complete offering fields/editing, lifecycle transition previews, non-blocking feasibility warnings, session non-participation, and objective defaults; added missing generated-client wrappers/hooks.
- Validation: focused Go compilation and `TestPhase3AuthoringAPIRoundTrip` pass (DB-backed test skips when URLs are unset); `make lint-backend`, `make format`, and `git diff --check` pass. `make check` stops before gates because Docker address pools are exhausted. Frontend tests/build lack `openapi-typescript`; frontend lint cannot write Bun temp files; migration round-trip lacks its URL; smoke lacks `.env`. `make generate` completes without generated-file changes.
- Open items: run available CI checks after push, review current-head comments, update Workpad telemetry, and open a non-draft PR referencing `Fixes #149`.
- Skill draft: no — this is a one-off frontend consolidation using existing generated-client and tenant-data conventions; no broadly reusable procedure was discovered.

## Current work — issue #145

- Scope: Implement the seven-state session lifecycle from SPEC §§14.3–14.5, §5.2, and §20.1; dependency issues #143 and #144 are merged on current `origin/main`.
- Key files: `backend/migrations/20260831160000_session_lifecycle.sql`, `backend/internal/program/session_lifecycle.go`, session/offering data and service files, session API handler/routes, generated sqlc/OpenAPI, and `backend/tests/integration/program_test.go`.
- Implementation: legal transition planner and row-locked audited transition service; empty-catalog and stale-draft publication gates; backward preview/confirmation with non-blocking invalidation warnings; retained stale-draft marker; Complete read-only protections; API problem mappings and frontend resource wrapper.
- Validation: focused tests, full `GOTOOLCHAIN=local go test -race ./... -count=1`, `make lint-backend`, `make format`, `make generate` execution, and `git diff --check` pass. Exact local backend/migration/frontend/smoke commands are environment-limited by Docker network exhaustion, missing migration URL, missing `openapi-typescript`/Bun temp permissions, and missing `.env`; PR CI must verify all ten required checks.
- Repository state: commit `4b93c3c` is pushed to PR [#157](https://github.com/christophergm/miniclass/pull/157), based on current `origin/main`.
- Open items: none; PR #157 is open and non-draft, references `Fixes #145`, all ten required CI checks pass, and no actionable review comments remain. Detent owns the completion-lane transition.
- Skill draft: no — this was a focused lifecycle implementation covered by existing backend and tenant-data guidance; no broadly reusable procedure was discovered.

## Current handoff — issue #140

- Scope: Split the year-scoped grade/homeroom vocabulary into `/y/:schoolYearId/vocabulary` and keep the organisation label plus owner-only administrators at top-level `/settings`, including closed-year read-only treatment and setup guidance.
- Dependency: issue #139 is now closed and PR #150 is merged; this worktree was three commits behind `origin/main` before the dependency update.
- Key files after dependency lands: `frontend/src/App.tsx`, `frontend/src/features/settings/SettingsPage.tsx`, `frontend/src/features/settings/useSettings.ts`, `frontend/src/features/school-years/SchoolYearPages.tsx`, `frontend/src/features/people/PeoplePages.tsx`, and frontend tests.
- Implementation: added the guarded `/y/:schoolYearId/vocabulary` page with configured-label headings, empty setup guidance, closed-year read-only controls, and shared vocabulary sections; moved organization label and administrators to top-level `/settings`; added workspace/app-shell navigation; preserved year-keyed roster vocabulary caching and invalidation.
- Validation: `git diff --check` and backend `make format` pass; backend generation completed without drift. Frontend tests/build/lint cannot run locally because `openapi-typescript` is missing and `bunx` cannot write its temp cache. Full `make check` stops at the existing `/miniclass-postgres` container-name conflict before CI gates run.
- Validation: PR #151 head `4061ab0` is green on all ten required CI checks: Backend tests, Backend lint, Backend format, Generated code drift, Migration round-trip, Frontend tests, Frontend build, Frontend lint, Repository formatting, and Developer tooling. `git diff --check` passes locally; local `make check` remains blocked at the existing `/miniclass-postgres` name conflict before the gates.
- Open items: Detent owns the completion-lane transition; no actionable PR comments or reviews remain.
- Skill draft: no — this was a focused frontend split and exposed no broadly reusable procedure beyond existing skills.

## Current work — issue #147

- Scope: deterministic, non-blocking catalog feasibility warnings for aggregate capacity, grade coverage, minimum viability, area coverage, and unmatched offerings. Governing contract: SPEC §§14.2, 5.2, 16.5, 19.4; ADR 0008.
- Implementation: `backend/internal/program/feasibility.go` evaluates a read snapshot; the service reads memberships, session non-participation, offerings, year grades, and active programme areas under `InTenantRead`. API exposes `GET .../catalog-feasibility` and adds `feasibility_warnings` to session responses; frontend has generated-client resource/type/hook wiring and cache invalidation.
- Area-signal note: interest profiles/ranked choices are not persisted in current Phase 3, so live area demand is empty. The pure evaluator accepts explicit aggregate high-rating signals and tests area gaps against the current vocabulary.
- Validation: focused Go tests, race-enabled `GOTOOLCHAIN=local go test -race ./... -count=1`, backend lint/format, generated-code drift, and `git diff --check` pass. The exact ten required PR checks also pass. Local `make test-backend` cannot create a Docker network because address pools are exhausted; `make test-migrations` lacks its configured URL; frontend tests/build lack `openapi-typescript`; frontend lint cannot write Bun temp files; smoke lacks `.env`.
- Repository state: PR #158 is open, non-draft, references `Fixes #147`, and has no actionable review comments. PR CI took 1m57s; slow checks were Backend tests (1m55s), Generated code drift (1m51s), and Developer tooling (1m25s). Quiet-window wait was 0s; local merge-gate execution was approximately 30s. No post-merge main CI applies while the PR is open.
- Open items: none; Detent owns the completion-lane transition.
- Skill draft: no — this is a focused warning evaluator/API wiring change; no new reusable procedure beyond existing guidance was discovered.

## Current work — issue #146

- Scope: Add session-scoped non-participation records with required reasons, audited CRUD/list API, unchanged programme membership, participating-membership projection, closed-year refusal, and tenant isolation. Governing contract: SPEC §§5.2, 8.3, 20.1; ADRs 0007, 0008, 0010.
- Key files: `backend/migrations/20260831150000_session_non_participations.sql`, `backend/sql/queries/session_non_participation.sql`, `backend/internal/data/session_non_participation.go`, `backend/internal/program/non_participation.go`, `backend/internal/api/handlers/non_participation.go`, `backend/internal/testing/registry/session_non_participation.go`, and `backend/tests/integration/session_non_participation_test.go`.
- Dependency: #143 is closed and PR #153 is merged; this worktree is based on `origin/main` at `b33a8a8`.
- Validation: `make generate`, `make lint-backend`, `make format`, `make -C backend generated-code-drift`, focused `GOTOOLCHAIN=local go test ./internal/api/... ./internal/program ./internal/data ./internal/testing/... ./tests/integration -run 'Test(SessionNonParticipation|LayerTwoRegistry|OpenAPI)'`, full `GOTOOLCHAIN=local go test ./internal/... ./tests/integration`, and `git diff --check` pass. Root `make test-backend` is blocked by the existing `/miniclass-postgres` container-name conflict; migration round-trip lacks its configured URL; frontend tests/build lack `openapi-typescript`; frontend lint cannot write Bun's temp cache; smoke lacks `.env`.
- Open items: commit/push, open PR referencing `Fixes #146`, verify current-head CI and review comments, then update the Workpad completion status.
- Skill draft: no — the existing tenant-entity and isolation-harness skills cover the reusable procedure; no new broadly reusable method was discovered.

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
- Workpad: issue comment 5477388223 contains the persistent plan, validation evidence, and complete status.
- Skills read: `.detent/skills/add-tenant-scoped-entity.md` and `.detent/skills/postgres-tenant-isolation-harness.md`.
- Implementation: dedicated offerings migration, data/service/API/frontend paths, generated artifacts, registry/factory, and isolation/closed-year tests. The migration includes the parent-compatible interest-area key required by the composite FK.
- Repository state: commits `0fa59aa` and `959c100` are pushed to PR #154, based on b33a8a8.
- Validation: local backend tests including race, lint, format, generation/drift after commit, focused Offering/Layer 2 tests, and `git diff --check` pass. Root `make check` stops at the existing `/miniclass-postgres` container-name conflict; local frontend gates lack `openapi-typescript`/Bun temp-cache access, migration round-trip lacks its URL, and smoke lacks `.env`.
- CI: all ten required checks pass on PR #154 head `959c100` in run 33386375617. PR CI duration was 1m53s; slow checks were Backend tests 1m51s, Generated code drift 1m48s, and Backend lint 1m15s. No post-merge main CI applies because the PR is not merged.
- Open items: none; PR #154 is open, non-draft, references `Fixes #144`, and has no actionable review comments.
- Skill draft: no — the existing tenant-entity and isolation-harness skills cover the reusable procedure; no new broadly reusable method was discovered.
