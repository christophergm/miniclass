## Current work — issue #185

- Scope: admin-managed interest-profile surveys and lifecycle per SPEC §§13.5–13.6; dependency #184 is closed and merged on origin/main.
- Key files: `backend/migrations/20260902090000_interest_profile_surveys.sql`, `backend/sql/queries/interest_profile_surveys.sql`, `backend/internal/data/survey.go`, `backend/internal/preference/survey.go`, API/program handlers, `frontend/src/lib/apiResources.ts`, and `frontend/src/features/programs/usePrograms.ts`.
- Implementation: year-scoped survey definitions with ordered vocabulary questions, configurable frozen rating options, all/explicit/grade/response-state audiences, open-time audience snapshots, hashed high-entropy student codes, deadline-aware automatic close, reopen/regeneration semantics, survey-bound retained submissions, audit actions, API routes, generated sqlc/OpenAPI, and Layer 2 registry coverage.
- Validation: focused survey lifecycle and empty-audience/deadline integration tests pass; full `GOTOOLCHAIN=local go test -race ./... -count=1`, `make format`, `make lint-backend`, generation/drift, and `git diff --check` pass. PR #194 head `c652604` passes all ten required CI checks, including migration round-trip and frontend lint/build/tests. CI ran for 120s; slow checks were Backend tests (120s), Generated code drift (106s), and Backend lint (81s). Local `make check` remains environment-limited by Docker address-pool exhaustion; no post-merge main CI applies while the PR is open.
- Repository/PR: commits `aca75b6`, `7c01125`, `425e90b`, and `c652604` are pushed to open, non-draft, clean PR [#194](https://github.com/christophergm/miniclass/pull/194), which references `Fixes #185`. No reviews or inline comments require action. Quiet-window wait: 0s; local merge-gate (`git diff --check`): 0.011s.
- Open items: none; Detent owns the completion-lane transition.
- Skill draft: no — the existing tenant-entity and PostgreSQL isolation-harness procedures cover the reusable method.

## Current work — issue #184

- Scope: retained interest-profile and ranked-choice preference submissions per SPEC §§8.3, 13.1–13.5, 13.7; dependency #183 is closed and merged on origin/main.
- Key files: `backend/migrations/20260901090000_preference_submissions.sql`, `backend/sql/queries/preference_submissions.sql`, `backend/internal/data/preference.go`, `backend/internal/preference/service.go`, registry/factory files, and `backend/tests/integration/preference_test.go`.
- Implementation: four append-only year-scoped tables with opaque student/program/session/area/offering composite FKs, actor/channel/timestamp attribution, forced RLS, closed-year guards, insert/select privileges, effective profile overlay, complete ranked-choice validation, deterministic latest response, audit action, generated sqlc, and Layer 2 registry coverage.
- Validation: full `GOTOOLCHAIN=local go test -race ./... -count=1`, `make format`, `make lint-backend`, `make generate`, and `git diff --check` pass. `make test-migrations` is unavailable because `MIGRATION_ROUNDTRIP_DATABASE_URL` is unset; run all ten CI gates after push.
- Repository/PR: commit `07f0e43` is pushed to PR #193, which is open, non-draft, merge-clean, and references `Fixes #184`; no actionable reviews or inline comments.
- Validation: all ten required PR checks pass on head `07f0e43`; PR CI duration 113s; slow checks were Backend tests and Generated code drift (110s each), then Backend lint (82s). Local `make check` stops at Docker address-pool exhaustion, while local migration/frontend/smoke gates lack configured dependencies as recorded in the Workpad. Post-commit `make generate && git diff --exit-code && git diff --check` passes.
- Open items: none; Detent owns the completion-lane transition.
- Skill draft: no — the existing tenant-entity and isolation-harness guidance covered the reusable procedure; no new broadly reusable method was discovered.

## Current work — issue #163

- Scope: Dedicated offering create/edit pages from the session summary, labeled one-property-per-row form, and “Maximum enrollment” UI terminology per SPEC §§8.4, 14.2, 16.1, 16.3, 16.5, 22.4.
- Key files: `frontend/src/features/programs/OfferingPages.tsx`, `frontend/src/features/programs/ProgramPages.tsx`, `frontend/src/features/programs/usePrograms.ts`, `frontend/src/App.tsx`, and `frontend/src/features/programs/ProgramPages.test.tsx`.
- Implementation: nested `/offerings/new` and `/offerings/:offeringId/edit` routes; shared form preserves the existing required `capacity` payload; closed-year controls remain read-only; create/edit save and cancel return to the session.
- Validation: rebased conflict-free onto `origin/main` at `345f2a9`; local `make lint-backend`, `make format`, generation drift, and `git diff --check` pass. Full `make check` stops before its gates because Docker reports all predefined address pools are exhausted; frontend tests/build lack `openapi-typescript`, and frontend lint cannot write Bun's temp cache. Previous PR #168 head `9b14bea` passed all ten required CI checks; the rebased head needs current-head CI after push. No post-merge main CI applies while the PR is open.
- Open items: push the rebased head with lease protection and verify current-head CI/review state. Detent owns the completion-lane transition.
- Skill draft: no — this is a focused frontend presentation change with no broadly reusable procedure discovered.

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
- Rework: rebased the objective-page implementation onto `origin/main` at `c9dc70b`; resolved the `ProgramPages.test.tsx` conflict by retaining both the #161 lifecycle assertions and #162 objective assertions.
- Implementation: `frontend/src/features/programs/ProgramPages.tsx` has dedicated programme and session objective pages with all 13 parameters, explanations, effective/inherited display, explicit session audit reason, and closed-year read-only behavior; `frontend/src/App.tsx` adds both routes; authoring pages link to them and no longer render objective inputs.
- Validation: `make lint-backend`, `make format`, `make generate && git diff --exit-code`, and `git diff --check` pass. `make check` and `make test-backend` stop at Docker address-pool exhaustion; migration round-trip lacks its URL; frontend test/build lack `openapi-typescript`; frontend lint cannot write Bun's temp cache; smoke lacks `.env`.
- Validation after rebase: PR #167 head `09edf36` passes all ten required checks; CI duration was 1m56s, with Generated code drift (1m55s), Backend tests (1m44s), and Developer tooling (1m13s) the slowest. Local `git diff --check` passes; no source changes were made after the green run.
- Open items: none; PR #167 is open, non-draft, references `Fixes #162`, has clean merge state, and has no review or inline comments requiring action. No skill draft; this is a focused frontend conflict resolution with no reusable procedure discovered.
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
## Current work — issue #164

- Scope: Remove session ordinal storage and public exposure; order sessions by earliest meeting date, case-insensitive name, and opaque ID per SPEC §§8.5, 14.1, and 17.8. Keep session create/edit date changes atomic and preserve the one-or-more date invariant.
- Repository state: clean worktree based on origin/main; no PR exists yet. Persistent Workpad comment: https://github.com/christophergm/miniclass/issues/164#issuecomment-5483864210.
- Key files: `backend/migrations/20260831130000_sessions.sql`, `backend/sql/queries/session.sql`, `backend/internal/data/session.go`, `backend/internal/program/session.go`, `backend/internal/api/handlers/session.go`, `frontend/src/features/programs/ProgramPages.tsx`, `frontend/src/features/programs/usePrograms.ts`, and session integration/frontend tests.
- Repository state: implementation is committed through `2709d41` and pushed to PR #169, which is open, non-draft, references `Fixes #164`, and has no actionable comments.
- Validation: local full backend tests, vet, lint, generated-code drift, and repository diff checks pass. All ten required PR checks pass on head `2709d41`; local root check is blocked before gates by exhausted Docker address pools, and local frontend/migration/smoke commands are environment-limited as recorded in the Workpad.
- Open items: none; Detent owns the completion-lane transition.
- Blockers: none.

## Current work — issue #160

- Scope: Move compact Phase 3 programme, interest-area, session/date, and session non-participation authoring into accessible modal forms; offerings and objectives remain dedicated pages. Governing contract: SPEC §§8.3, 8.5, 12.1, 12.3, 14.1, 14.2, 22.4.
- Key files: `frontend/src/components/ui/modal-form.tsx`, `frontend/src/components/ui/modal-form.test.tsx`, `frontend/src/features/programs/ProgramPages.tsx`, `frontend/src/features/programs/ProgramPages.test.tsx`, and `frontend/src/features/programs/usePrograms.ts`.
- Implementation: reusable modal focus management, focus trap, Escape/backdrop/cancel dismissal, discard confirmation for dirty forms, and focus restoration; all scoped create/edit workflows use summaries plus modals. Session create/edit uses the atomic name-plus-date payload and requires at least one date. Meeting-date query invalidation follows atomic session saves.
- Validation: `make format`, `make lint-backend`, `make generate`, and `git diff --check` pass. `make check`/`make test-backend` cannot create Docker networks because predefined address pools are exhausted; `make test-migrations` lacks its configured URL; frontend tests/build lack `openapi-typescript`; frontend lint cannot write Bun's temp cache; smoke lacks `.env`.
- Open items: commit/push, open PR referencing `Fixes #160`, verify current-head CI/review state, then update this Workpad to complete if the review gate is green.
- Skill draft: no — this is a focused UI pattern and no broadly reusable project procedure was discovered.
## Current work — issue #165

- Scope: Fix frontend interest-area up-arrow reordering per SPEC §§12.1, 12.3, 12.4, and 22.4.
- Key files: `frontend/src/features/programs/ProgramPages.tsx` and `frontend/src/features/programs/ProgramPages.test.tsx`.
- Implementation: up-arrow now submits the full list with only the selected area and immediate predecessor swapped; regression covers three areas, up/down payloads, and first/last disabled boundaries.
- Validation: focused Bun test cannot start locally because frontend dependencies are absent (`react/jsx-dev-runtime`); CI should run the provisioned frontend gates. Workpad comment `5484747880` is the persistent tracker comment.
- Open items: run available checks, commit/push, open PR with `Fixes #165`, verify current-head CI/reviews, then complete the Workpad handoff.
- Skill draft: no — this is a small, one-off frontend reorder correction with no broadly reusable procedure.

## Current handoff — issue #172

- Scope: shared year-scoped navigation and breadcrumb for Programs, Adults, Students, and Settings; year Settings actions for vocabulary/import; one-program-per-row list; closed-year read-only navigation per SPEC §§8.7 and 11.1.
- Implementation: `SchoolYearLayout` wraps every resolved `/y/:schoolYearId/*` route; the year root redirects to Settings; Settings retains vocabulary/import destinations in closed years; imports disable mutations for closed years; Programs has a header create action and one program per row.
- Repository: commits `494793a` and `124e797` are pushed to PR [#176](https://github.com/christophergm/miniclass/pull/176), which references `Fixes #172` and is open/non-draft.
- Validation: final PR run `33448480472` passes all ten required checks. Backend lint/format, generation, and `git diff --check` pass locally. Local frontend tests/build are unavailable because `openapi-typescript` is absent; frontend lint cannot write Bun's temp cache; local full check stops at exhausted Docker address pools; migration round-trip lacks its configured URL; smoke lacks `.env`.
- Telemetry: quiet-window wait 0s; local merge-gate duration 0.011s; PR CI duration 1m56s; slow checks were Backend tests (111s), Generated code drift (108s), and Backend lint (85s); no post-merge main CI applies while the PR is open.
- Open items: none. No PR reviews, inline comments, or issue blockers. Detent owns the completion-lane transition.
- Skill draft: no — this was a focused frontend routing/presentation change and exposed no broadly reusable procedure.

## Current work — issue #173

- Scope: Move school-year creation and editing into focused modal flows per SPEC §§5.2, 5.4, and 11.1.
- Key files: `frontend/src/features/school-years/SchoolYearPages.tsx`, `frontend/src/features/school-years/SchoolYearPages.test.tsx`, and `frontend/src/components/ui/button.tsx`.
- Implementation: `/years` now opens a create modal; year Settings has an Edit modal containing label save, read-only timestamps, and state-specific lifecycle actions. Close is destructive-styled; closed-year reopen remains owner-only and requires a reason.
- Validation: backend format/lint/vet and `git diff --check` pass; generation runs without generated-file changes. Local frontend checks cannot run because frontend dependencies, including `openapi-typescript`, are not installed. Full `make check` stops at Docker address-pool exhaustion before gates. PR #177 final head `0e52f46` passes all ten required CI checks; the first run caught and was fixed by scoping an ambiguous modal test query.
- Open items: none; PR #177 is open, non-draft, merge-clean, references `Fixes #173`, and has no actionable review comments. Detent owns the completion-lane transition. No dependency blocker.
- Skill draft: no — this is a focused UI refinement using the existing modal pattern and exposed no broadly reusable procedure.
## Current work — issue #174

- Scope: Make program detail session-first and move membership, interest-area, and assignment planner authoring to dedicated settings pages per SPEC §§12.1–12.4, 14.1, and 17.7.
- Current implementation: `frontend/src/features/programs/ProgramPages.tsx` now has `ProgramDetailPage`, `ProgramSettingsPage`, `ProgramMembershipPage`, and `ProgramInterestAreasPage`; `frontend/src/App.tsx` routes settings subpages and renamed planner entry points. Program detail has no `All programs` control or embedded administrative authoring.
- Validation: local `make format`, `make lint-backend`, `make generate` with generated-artifact drift check, and `git diff --check` pass. Local `make check` stops at Docker address-pool exhaustion; local frontend tests/build/lint are unavailable because dependencies are absent and Bun cannot write its temp cache. PR #178 head `aa9a494` passes all ten required CI checks, is open/non-draft/merge-clean, and has no actionable reviews or comments.
- Open items: none; Detent owns the completion-lane transition. CI duration was approximately 1m46s; slow checks were Backend tests and Generated code drift at approximately 105s each. No post-merge main CI applies while the PR is open.
- Skill draft: no — this is a focused frontend routing/presentation change with no broadly reusable procedure discovered.
## Current handoff — issue #175

- Scope: route `/y/:schoolYearId` to the sole program detail/home when exactly one program exists, otherwise to `/y/:schoolYearId/programs`; do not infer activity from session lifecycle state.
- Key files: `frontend/src/App.tsx`, `frontend/src/features/programs/ProgramPages.tsx`, and `frontend/src/features/programs/ProgramPages.test.tsx`.
- Implementation: `ProgramYearEntryPage` loads the year-scoped program list and uses an explicit count-based `Navigate`; the complete Programs list remains a direct route. No program activity or session lifecycle state is consulted.
- Validation: local backend lint/format, generation drift, and `git diff --check` pass. Local backend tests, migration round-trip, frontend tests/build/lint, and smoke are environment-limited as recorded in the PR. PR #179 head `f6fd5b3` passes all ten required checks; slow checks were Backend tests (104s), Generated code drift (88s), and Backend lint (80s). PR CI duration was 108s; quiet-window wait was 0s; local merge-gate duration was not applicable beyond the clean `git diff --check`; no post-merge main CI applies while the PR is open.
- Repository state: commits `f6fd5b3` and `092c830` are pushed to PR #179, which references `Fixes #175`, is open, non-draft, clean, and has no actionable reviews or comments. The final head is `092c830`.
- Open items: none; Detent owns the completion-lane transition.
- Skill draft: no — this is a focused frontend routing change with no broadly reusable procedure discovered.

## Current work — issue #183

- Scope: define the Phase 4 principal, capability, session, authentication, recovery, attribution, and
  authorization contract per SPEC §§6.2, 6.5–6.6, 9.3–9.4, 13.8 and ADRs 0002/0008/0013.
- Implementation: clarified administrative-account versus scoped non-account principals; explicit
  guardian/student-code boundaries; current relationship-derived scope; OTP/session bounds and
  revocation; MFA recovery/reset invalidation; mode reauthentication; duplicate/no-email behavior;
  administrator-on-behalf attribution; tenant/audit transaction obligations; and security-test coverage.
  Removed stale household wording from ADR 0002 and fixed the malformed guardian sentence in SPEC.
- Key files: `SPEC.md`, `docs/adr/0002-authentication-and-access-mechanisms.md`, and
  `docs/adr/0013-guardian-and-volunteer-access.md`.
- Validation: local `make format`, `make lint-backend`, `make generate`, and `git diff --check` pass;
  generated artifacts are unchanged. Root `make check` stops before gates because Docker reports all
  predefined address pools exhausted. PR #192 head `d6344e4` passes all ten required checks; PR CI took
  111s, with Backend tests 108s, Generated code drift/Developer tooling 91s, and Backend lint 86s.
- Open items: none; PR #192 is open, non-draft, merge-clean, references `Fixes #183`, and has no
  actionable review comments. Quiet-window wait 0s; local merge-gate duration under 1s; no post-merge
  main CI applies while the PR is open. Detent owns the completion-lane transition.
- Skill draft: no — this is a one-off contract clarification using existing ADR/spec conventions; no
  broadly reusable procedure was discovered.

## Current work — issue #186

- Scope: Integrate ranked-choice windows with session lifecycle per SPEC §§13.3, 13.7–13.8, and 14.1, 14.3–14.6; dependency #185 is closed and merged on origin/main.
- Key files: `backend/migrations/20260903090000_ranked_choice_sessions.sql`, `backend/internal/data/session.go`, `backend/internal/data/ranked_choice.go`, `backend/internal/preference/service.go`, `backend/internal/preference/ranked_choice.go`, `backend/internal/program/session.go`, `backend/internal/program/session_lifecycle.go`, session API/generated artifacts, frontend session lifecycle/configuration, and preference/Layer 2 tests.
- Implementation: optional per-session rank depth/deadline, VotingOpen/deadline/participant enforcement, hashed session/student access codes issued on opening, duplicate-rank validation before persistence, append-only latest-valid submissions, reopening warning/reason/new deadline, and registry/RLS/closed-year coverage.
- Validation: focused Go/API/integration compilation, full `GOTOOLCHAIN=local go test -race ./... -count=1`, `make format`, `make lint-backend`, generation/drift (`make generate && git diff --exit-code`), and `git diff --check` pass. PR #195 run `33583087306` passes all ten required checks on head `c23571a`; Backend tests, Generated code drift, and Developer tooling were the slowest checks (121s, 89s, and 84s). `make test-backend`/`make check` remain locally limited by Docker address-pool exhaustion; migration round-trip lacks its configured URL; frontend local commands lack installed dependencies/Bun cache access; smoke lacks `.env`.
- Repository/PR: commits `2fc4593`, `7f40319`, `8ea3cde`, `e864a69`, and `c23571a` are pushed to open non-draft PR #195, which references `Fixes #186` and is merge-clean. No reviews or inline comments require action. Quiet-window wait 0s; local merge-gate `git diff --check` under 1s; PR CI duration 125s; no post-merge main CI applies while the PR is open.
- Open items: none; Detent owns the completion-lane transition. No dependency blocker or human action is declared.
- Skill draft: no — the existing tenant-entity and PostgreSQL isolation-harness procedures cover the reusable method; no new broadly reusable procedure was discovered.

## Current handoff — issue #187

- Scope: adult email OTP, bounded/revocable guardian sessions with live relationship scope, explicit adult/account links, administration/survey/guardian mode boundaries, mandatory step-up MFA, single-use recovery codes, audited Owner reset, neutral duplicate/unknown/no-email behavior, rate limiting, and transactional SMTP delivery per SPEC §§6.2, 6.6, 9.3–9.4, 13.8, 22.5 and ADR 0013.
- Key files: `backend/migrations/20260904120000_adult_authentication.sql`, `backend/internal/identity/adult_auth.go`, `backend/internal/auth/{adult,capabilities,middleware}.go`, `backend/internal/api/handlers/adult_auth.go`, `backend/internal/testing/registry/adult_account_link.go`, `backend/tests/integration/adult_auth_test.go`, `frontend/src/features/auth/{GuardianAccessPage,MfaPage}.tsx`, `frontend/src/lib/{auth,apiResources}.ts`, and `scripts/smoke-test.sh`.
- Implementation: hashed single-use OTP challenges and bearer sessions with absolute/idle expiry, live guardian scope resolution, application-issued administrative sessions tied to MFA generations, encrypted TOTP secrets, hashed recovery codes, audited mode/link/MFA actions, neutral delivery/error behavior, OpenAPI/sqlc artifacts, frontend guardian/MFA flows, and smoke coverage for MFA step-up.
- Validation: local `GOTOOLCHAIN=local go test -race ./... -count=1`, `make lint-backend`, `make format`, `make generate`, `make -C backend generated-code-drift`, `git diff --check`, and `bash -n scripts/smoke-test.sh` pass. Local `make check`/migration/frontend/smoke commands are environment-limited by Docker address-pool exhaustion, missing migration URL, absent frontend dependencies/Bun cache access, and missing `.env`; the provisioned CI run verified those gates successfully.
- Repository/PR: commits `aa40c22`, `9918305`, `e37071a`, and `2cf1783` are pushed to open non-draft PR [#196](https://github.com/christophergm/miniclass/pull/196), which references `Fixes #187`, is merge-clean, and has no reviews or inline comments. Run `33589048076` passes all ten required checks; CI duration was 2m13s. Slow checks were Backend tests (129s), Developer tooling (77s), and Backend lint (79s); Generated code drift took 108s. Quiet-window wait 0s; local merge-gate `git diff --check` under 1s; no post-merge main CI applies while the PR is open.
- Open items: none; Detent owns the completion-lane transition. No dependency blocker or human action is declared.
- Skill draft: no — the existing tenant-entity and PostgreSQL isolation-harness procedures covered the reusable method; no new broadly reusable procedure was discovered.
## Current work — issue #188

- Scope: student access-code distribution for interest-profile surveys and ranked-choice sessions per SPEC §§13.8, 19.5, 22.4 and ADR 0013; dependency #187 is merged.
- Key files: `backend/internal/preference/{access_code,ranked_choice,survey}.go`, API handlers/routes, `frontend/src/features/programs/{AccessCodeDistribution,ProgramAccessCodesPage,ProgramPages}.tsx`, and generated `backend/openapi.json`.
- Implementation: organizer-only regeneration/revocation endpoints with audited reasons, homeroom/name metadata returned only on issuance, session and survey code rotation/revocation, grouped print-friendly frontend distribution, and guessing/replay/cross-student/cross-tenant integration coverage.
- Validation: full `GOTOOLCHAIN=local go test -race ./... -count=1`, `make format`, `make lint-backend`, `make generate` with sqlc drift clean, focused tests, and `git diff --check` pass. Local frontend gates cannot run because `openapi-typescript`/frontend dependencies are absent; migration round-trip lacks its URL; smoke lacks `.env`; Docker-backed `make test-backend` cannot allocate a network because address pools are exhausted. PR #197 head `8c8aba3` passes all ten required checks in run `33597260620`; slow checks were Backend tests (132s), Generated code drift (110s), and Backend lint (88s).
- Repository/PR: commits `9aaf38c`, `afafffe`, and `8c8aba3` are pushed to open, non-draft PR [#197](https://github.com/christophergm/miniclass/pull/197), which references `Fixes #188`, is merge-clean, and has no actionable reviews or inline comments. Quiet-window wait: 0s; local merge-gate `git diff --check`: under 1s; no post-merge main CI applies while the PR is open.
- Open items: none; Detent owns the completion-lane transition.
- Skill draft: no — the existing access/auth and tenant-isolation procedures cover this implementation; no broadly reusable method was discovered.

## Current work — issue #189

- Scope: guardian, student-code, and administrator-on-behalf interest-profile and ranked-choice submission flows per SPEC §§6.2, 6.5, 13.7–13.8, 22.4; dependency #188 is merged.
- Key files: `backend/internal/preference/forms.go`, `backend/internal/api/handlers/preference.go`, `backend/internal/api/routes.go`, `backend/tests/integration/preference_test.go`, `frontend/src/features/preferences/`, `frontend/src/lib/apiResources.ts`, `frontend/src/features/programs/usePrograms.ts`, and `frontend/e2e/preferences.spec.ts`.
- Implementation: instrument-bound form read models, live guardian scope checks, student-code and administrator submission attribution, effective interest-profile prefill, ranked-choice replacement, restricted student surfaces, administrator selection, access-code links, and mobile-first guardian/student editors.
- Validation: focused and full race-enabled backend tests, `make format`, `make lint-backend`, `make generate`, backend generated-code drift, direct Biome format, and `git diff --check` pass. `make check` stops at Docker network creation because all predefined address pools are exhausted; migration round-trip lacks `MIGRATION_ROUNDTRIP_DATABASE_URL`; frontend gates need CI-installed dependencies; smoke lacks `.env`. Playwright mobile coverage is wired into the Frontend tests CI stage with mocked API data.
- Open items: commit/push, open a non-draft PR referencing `Fixes #189`, verify current-head CI and review comments, then complete the Workpad handoff. No dependency blocker or human action is declared.
- Skill draft: no — the existing auth, tenant-data, and isolation guidance covered the reusable method; no new broadly reusable procedure was discovered.
