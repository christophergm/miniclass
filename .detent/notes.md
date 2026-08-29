# Detent handoff notes

## Issue #120 roster ingest scope and source authority

- Documentation-only change in `docs/adr/0014-roster-ingest-scope-and-source-authority.md` and
  `PLAN.md`; no code, schema, or generated artifacts changed.
- ADR 0014 is Accepted and cites SPEC §§5.2/10.1/11/15.2/20.1/21.1. It fixes Phase 2 to
  `roster_json` (community-platform wide export) and `grades_csv` (student name + grade), external
  identifier-only matching, reported enrolment/adult filters, nullable grade/intent, literal adult
  guardian authority, stateless content-hash preview/commit, and deferred individual Conflict
  resolution until the Phase 4 preference-import trigger.
- `PLAN.md` replaces the contradictory real-historical-export database exit criterion with synthetic
  CI fixtures, an opt-in no-database parser conformance check, and an operator demonstration against
  their own instance. It removes the generic/name-matching claims and stale household implication.
- Validation: `git diff --check` passes. `make check` stops before project checks at the pre-existing
  fixed-name `/miniclass-postgres` Docker collision. Commit `277f0da` is pushed and PR #129 is open;
  current-head CI/review checks remain to be verified.
- Skill draft: no — this is a one-off documentation decision, not a reusable procedure.

## Issue #110 query cache identity boundary

- Current worktree is cleanly based on `origin/main` at `494cfb2`; the stale prior entry for #101 was already merged and is unrelated.
- `AuthProvider` now tracks the last signed-in Supabase user ID across sign-out events and clears the shared React Query cache only when a different non-null identity arrives. Same-user `TOKEN_REFRESHED` events and `SIGNED_OUT` leave the cache intact, preventing anonymous refetches while the shell unmounts.
- Added `AuthProvider.test.tsx` covering the exact sign-out → different sign-in transition for `['account']` and a school-year key, plus same-user refresh preservation. Bare `AuthProvider` tests now render under `QueryClientProvider`.
- Focused Vitest/build could not start because this checkout has no installed frontend dependencies (`openapi-typescript`, Vitest, TypeScript, and ESLint binaries absent); frontend lint also fails before ESLint because Bun cannot write its tempdir. Migration round-trip cannot start without `MIGRATION_ROUNDTRIP_DATABASE_URL`, and smoke cannot start without `.env`.
- Backend lint/format, direct `go test -race -v ./... -count=1` (database cases skipped without test URLs), pinned generated-code drift, and `git diff --check` pass. `make test-backend` stops at the pre-existing fixed-name `miniclass-postgres` collision.
- Open items: commit and push, open PR with `Fixes #110` and SPEC §§9.3/11.1, inspect current-head CI/reviews, then complete the Workpad.

## Issue #101 soft-deleted people in relationship and membership listings

- Worktree rebased cleanly onto `origin/main` at `14ea2e0`; no PR existed at start and the issue has no native dependencies or `Depends on:` references.
- Current live year-wide household membership queries already exclude deleted households and members. The missing predicates are in `ListHouseholdStudents`, `ListHouseholdAdults` (deleted household), and `ListGuardianRelationships` (deleted adult/student); registry queries must remain unfiltered.
- Workpad comment `5438876572` tracks the change. It cites SPEC §21.3, preserves link rows, and adds integration regression coverage.
- Restore API is tracked separately in #103. Until it lands the reversibility criterion is proved by clearing `deleted_at` in a migrator transaction that sets `app.organization_id` (RLS is forced on the person tables), which is the stricter check anyway: nothing but the person's own flag changes.
- Both new assertions were confirmed to fail against the pre-fix generated queries before the fix was reinstated.
- Regenerated `internal/db/gen` with the pinned sqlc v1.27.0 through the local `sqlc/sqlc:1.27.0` image; the worker's PATH sqlc is v1.31.1 and the Makefile refuses it.
- Gates run locally: Backend tests, Backend lint, Backend format, Generated code drift, Migration round-trip, Frontend tests, Frontend build, Frontend lint, Repository formatting — all green. Developer tooling (`make smoke`) was not run: it requires ports 8080 and 5173 free, and a developer API process already owns 8080 on this machine. The change touches no tooling; CI covers that gate.
- Open items: commit/push, open PR with `Fixes #101` and the SPEC citation, inspect CI/reviews, then update the Workpad.

## Issue #94 local fake-auth password reset

- Added local-dev reset blocking in `frontend/src/features/auth/ResetPasswordPage.tsx`; it reuses
  `LocalDevAuthBanner` with a no-email-path message and `make login` guidance. The real Supabase path
  remains the existing reset form and success behavior.
- Added parameterized coverage in `ResetPasswordPage.test.tsx` for valid/missing local tokens and a
  real-Supabase submission.
- `git diff --check` passes. Frontend test/build gates cannot start locally because the checkout lacks
  the installed `openapi-typescript` dependency; frontend lint also hits Bun tempdir PermissionDenied.
- Open items: commit/push, open PR with `Fixes #94` and ADR 0011 citation, inspect CI/reviews, and
  update the Workpad after the required checks pass. Skill draft: no — routine auth-surface reuse.

## Issue #95 mid-session token expiry

- The frontend now re-checks local fake-auth JWT expiry on a timer and on token reads. At `exp`, it
  emits a session-ended event, signs out the fake client, and redirects the auth shell to the existing
  local banner naming `make login` and the required Vite restart.
- The API boundary emits the same event only for HTTP 401 responses with RFC 9457 type
  `invalid-token`; the auth provider clears the session, calls Supabase `signOut`, and the sign-in page
  explains that the session expired or is no longer valid. ADR 0009 records this decision.
- Added API event, provider/surface, and fake-clock boundary coverage. Focused frontend auth/API tests
  pass (21), and the full frontend test gate passes (83); frontend build and lint also pass.
- The preserved commit was rebased onto current `origin/main` at `14ea2e0`; the only conflict was this
  handoff-notes file, with both the #94 and #95 entries retained. Backend race tests, backend lint,
  backend format/vet, and `git diff --check` pass.
- Local `make test-backend` is blocked by the pre-existing fixed-name `miniclass-postgres` Docker
  container conflict; direct `go test -race -v ./... -count=1` passes with DB integration cases skipped.
  `make generate` is blocked by local sqlc v1.31.1 versus pinned v1.27.0; migration round-trip lacks
  `MIGRATION_ROUNDTRIP_DATABASE_URL`; `make smoke` lacks the checkout `.env`. CI must verify the
  required current-head checks after push.
- PR #114 is open, non-draft, mergeable, references `Fixes #95`, and cites SPEC §9.3 plus ADR 0009.
  Current head `4280936b` passed all ten required checks in run `33126534875`; no reviews or inline
  comments remain. CI duration was about 87s; slowest checks were Backend tests (83s), Backend lint
  (80s), and Generated code drift (76s). Quiet-window wait was 0s; local final merge gate was under
  1s; latest main CI is successful and no post-merge run is active.
- Skill draft: no — this is a scoped auth lifecycle fix using existing frontend and ADR patterns.

## Issue #92 developer tooling CI check

- Added the tenth `Developer tooling` CI check, with `.env.example` sourcing validation, Compose/PostgreSQL role provisioning, `make setup`, unclaimed `make db-seed`, and `make smoke`; no path filters are used. Added it to `detent.yaml` and root `make check`, and updated `WORKFLOW.md`, README, and QUICKSTART documentation.
- Extended `scripts/smoke-test.sh` to mint a temporary local JWT, follow the generated `/claim?token=...` URL through Vite, assert the frontend route/query contract, complete `POST /api/auth/claim` through the Vite proxy, and verify `/api/me` plus `/api/school-years`. It restores `.env` on exit and handles the normal already-bound Owner quickstart by inviting a synthetic administrator.
- Focused validation passed: shell syntax, YAML parse, `.env.example` sourcing, Vite route-source runtime assertion, backend race tests (integration skipped without test DB URLs), backend lint/format, OpenAPI determinism, frontend tests (69), build, lint, and `git diff --check`.
- Local full `make check` stopped at the pre-existing parent-checkout `miniclass-postgres` fixed-name conflict; migration round-trip lacks `MIGRATION_ROUNDTRIP_*`; local generation has sqlc v1.31.1 while the repository pins v1.27.0. CI must verify the isolated Developer tooling job and all ten checks.
- PR #109 is open, non-draft, mergeable, references `Fixes #92`, and has no reviews or inline comments. Current-head CI run `33067515985` passed all ten checks, including Developer tooling; slowest were Generated code drift (1m30s), Backend tests (1m23s), and Backend lint (1m08s). Quiet-window wait was 0s; local merge-gate duration was about 0s after the already-running local checks; PR CI duration was about 1m34s; no post-merge main run is active.
- This notes-only update must be pushed and its current-head checks rechecked before the final Workpad completion declaration.


## Issue #93 API database role separation

- `DATABASE_URL` remains migrator-only for `cmd/migrate` and migration/reset operations. Added
  required `APP_DATABASE_URL` for `cmd/api`, `cmd/seed`, and `cmd/bootstrap`.
- `internal/data.NewApplicationFromURL` verifies `current_user = miniclass_app`, `nobypassrls`, and
  no `CREATE` on `public`; `New` uses it. Generic `NewFromURL` remains for migrator/test paths.
- Added config, startup, and integration coverage; updated `.env.example`, smoke script, and ADR 0011.
- Full race backend suite, lint/depguard, format/vet, migration round-trip, generated OpenAPI drift,
  shell syntax, and `git diff --check` pass. Root `make test-backend` was blocked by the shared fixed
  `/miniclass-postgres` container name, while backend tests passed against that healthy PostgreSQL 18
  service. Frontend tests/build/lint could not run because dependencies are absent and Bun temp writes
  are denied; generated sqlc drift could not run because local sqlc is v1.31.1 vs pinned v1.27.0.
- Open items: commit/push, open PR with `Fixes #93`, inspect current-head CI/reviews, then update the
  Workpad and hand off. No skill draft planned; this is a scoped role/configuration fix.


## Issue #64 merge fallback

- Rebased PR #82 onto current `origin/main` (`a723ef8`); resolved only the two expected conflicts.
  `App.tsx` retains the authenticated school-year shell and places `/audit-log` inside the protected
  `AppShell`; `api.ts` retains authenticated fetch behavior and adds the audit-log types/query method.
- Backend `make test` passed; database integration cases skipped because `TEST_DATABASE_URL` and
  `TEST_APP_DATABASE_URL` are unset. Backend lint, format/vet, and `git diff --check` passed.
- `make generate && git diff --exit-code` changed only sqlc headers because local sqlc is v1.31.1
  versus pinned v1.27.0; generated changes were discarded. Migration round-trip could not start because
  `POSTGRES_ADMIN_DATABASE_URL` is unset. Frozen Bun install is blocked by Bun tempdir permissions;
  without install, frontend lint passed, while tests/build could not resolve missing `@supabase/supabase-js`.
- Rebase conflict resolution is complete. Final PR-head CI run `32807894013` passed all nine required
  checks; slow checks were Generated code drift (1m29s), Backend lint (1m21s), and Backend tests (1m10s).

## Issue #64 frontend audit log view

- PR #82 is open, non-draft, mergeable, references `Fixes #64`, and cites SPEC §20.1/§6.6.
- Added `frontend/src/features/audit/AuditLog.tsx` and focused tests, typed `/api/me` and `/api/audit-log` client methods, hooks, and `/audit-log` routing. Owner/Administrator access is gated from `/api/me`; Coordinator renders no view. Object-type filtering and opaque-cursor pagination use the existing endpoint.
- Fresh CI confirms frozen Bun tests/build/lint and migration round-trip pass on the rebased head. Local
  `git diff --check` and frontend lint pass; local frozen Bun install is blocked by tempdir permissions,
  and the pre-existing checkout lacks `@supabase/supabase-js` for tests/build. PR has no reviews or inline
  comments, is open/non-draft/mergeable, and remains scoped to the existing implementation. No skill
  draft: this was a routine frontend feature using existing API/query patterns.
## Issue #65 seed corpus and test factories

- Dependencies #55, #56, and #57 are complete through merged PRs #85, #79, and #86. Current `origin/main`
  is `5fd1b2c`; the branch was rebased onto it after resolving only the handoff-note conflict.
- Added deterministic synthetic corpus generation in `internal/seed`: 139 students with the exact
  Appendix B.1 grade split, six homerooms, 102 adults with 13/45/44 participation, 90 households,
  opaque-index relationships, and all required awkward cases. `Load` bootstraps a fresh organization
  and Owner invitation, then composes audited roster services through `internal/testing/factories`.
- Replaced the empty SQL seed with flag-driven `cmd/seed` and `make seed` output for organization,
  school year, roster counts, and claim URL. The isolation registry now uses the same minimal factories;
  depguard restricts seed/factory imports to seed command and test support. Added pure and integration
  corpus distribution/edge-case coverage.
- Validation: focused and full race backend tests, lint/depguard, format/vet, frontend frozen tests
  (25), build, lint, and `git diff --check` pass. Integration and migration round-trip database checks
  skip/fail locally because `TEST_DATABASE_URL`, `TEST_APP_DATABASE_URL`, and
  `POSTGRES_ADMIN_DATABASE_URL` are unset. Local sqlc is v1.31.1; pinned v1.27.0 installation also
  failed on the worker's restricted checksum/C parser environment, so generated artifacts were restored
  unchanged for CI's pinned drift gate.
- PR #87 is open, non-draft, references `Fixes #65`, and all nine current-head checks pass. No reviews
  or inline comments remain. Workpad comment `5404543711` is complete; Detent owns the lane transition.

## Issue #63 frontend households and guardian relationships

- Built on the P1-13a frontend commit `d84f7e6` (PR #81 is still open); household UI is in
  `frontend/src/features/people/HouseholdPages.tsx`, `householdApi.ts`, and related guardian files.
- Routes are year-scoped under `/y/:schoolYearId/households`; person lists/details show all household
  links and a warning-only empty-household state. Guardian relationships are explicitly independent
  membership records and editable from both person directions.
- Focused frontend tests currently cover list/detail two-household rendering, member add/remove,
  relationship type edits, reverse adult view, and the non-blocking missing-household warning.
- PR #83 is open, non-draft, clean/mergeable, references `Fixes #63`, and cites SPEC §§8.2/11.2/5.2.
- Current-head CI run 32804112171 passed all nine required checks. Slowest: Backend lint 1m24s,
  Backend tests 1m14s, Generated code drift 1m13s. No reviews or inline comments remain.
- Local validation passed: 19 frontend tests, frontend build/lint, backend tests/lint/format, generated
  drift, git diff --check, and PostgreSQL 18 disposable migration up/down/up. The exact migration
  wrapper lacks host `psql`; its equivalent container sequence passed and cleaned up the database.
- Telemetry: quiet-window wait 0s; local frontend gate about 5s; backend checks about 19s; disposable
  migration equivalent about 2s; PR CI about 1m24s; latest main CI successful, no post-merge run active.
- Skill draft: no — this was a scoped frontend feature using existing people-page patterns.

## Issue #59 audit log read endpoint

- PR #76 is open, non-draft, references `Fixes #59`, cites SPEC §20.1/§6.6 and ADR 0008/0010.
- Added `GET /api/audit-log`, `internal/data` keyset query on `(occurred_at, id)`, object-type filter,
  opaque cursor, closed action enum in OpenAPI, and Coordinator authorization coverage.
- Local backend tests/lint/format, frontend tests/build/lint, OpenAPI drift, and `git diff --check` pass;
  CI current-head all nine required checks pass. Migration round-trip passed in CI; local wrapper lacked
  `POSTGRES_ADMIN_DATABASE_URL`.
- Workpad comment `5398726896` is complete; no reviews or actionable comments remain. No skill draft.



## Issue #54 grade and homeroom vocabularies

- Added migration `20260824180000_vocabularies.sql` with forced-RLS tenant tables `grade_levels`
  and `homerooms`, opaque XID keys, uniqueness, ordinal ordering, retirement timestamps, and
  editable `organizations.homeroom_label` privilege.
- Added `internal/data/vocabulary.go`, audited `internal/vocabulary` service operations, Huma routes
  under `manage_roster`, and committed sqlc/OpenAPI output. Retired rows are excluded by default
  picker/list queries but remain fetchable by ID; grades reorder by explicit ordinal.
- Added per-entity registry files and a vocabulary integration test. Layer 2 confirms cross-tenant
  read/fetch/update/retire/foreign-parent isolation for both tables.
- Validation passed: PostgreSQL 18 race-enabled `make test` with migrator/app roles, `make lint`,
  `make format`, pinned sqlc v1.27.0 generation plus `make generated-code-drift`, equivalent
  migration up/down/up, frozen Bun tests/build/lint, and `git diff --check`.
- Open items: commit/push, open PR with `Fixes #54` and SPEC §§10.1/20.1, inspect current-head CI
  and reviews, then update the Workpad to complete. No blocker or human action is known.

## Issue #56 adults

- Added `20260824190000_adults.sql` with a first-class, year-scoped `adults` table: XID keys,
  composite `(school_year_id, organization_id)` parent integrity, forced tenant RLS, closed-year
  trigger, soft delete, and a partial active-row external-identifier unique index. The schema has
  no role, household, tag, or `general_availability` fields.
- `internal/data/adult.go` keeps generated sqlc access behind the tenant boundary; `internal/people`
  owns audited CRUD, the `lead`/`help`/`unavailable` intent, and the shared preferred-given-name
  display helper. Adult routes are `/api/school-years/{schoolYearID}/adults[/{adultID}]` and require
  `manage_roster`; OpenAPI and pinned sqlc artifacts are committed.
- Added database-backed CRUD/soft-delete/audit coverage, Layer 2 registry coverage, and updated the
  tenancy meta-test to recognize the required three-column identity key for year-scoped entities.
- Validation passed with PostgreSQL 18 and both roles: `make test`, `make lint`, `make format`, full
  integration isolation/CRUD tests, migration up/down/up equivalent (the wrapper itself requires
  unavailable host `psql`), pinned sqlc generation, OpenAPI generation/drift, and `git diff --check`.
- Open items: frozen Bun frontend tests/build/lint, stage and commit, push, open the PR with `Fixes #56`
  and SPEC §8.2/§15.2/§11.2 citation, then inspect current-head checks and review comments.

## Issue #55 students (rechecked 2026-08-25)

- Dependency #54 is now closed through merged PR #78; `origin/main` contains the vocabulary tables
  and the student implementation can reference `grade_levels` and `homerooms`.
- Rebased the worktree onto current `origin/main` (`d6da8cde`). Added migration
  `20260825090000_students.sql`, generated sqlc queries/model, `internal/data/student.go`, audited
  `internal/people` CRUD, Huma student routes under `manage_roster`, OpenAPI output, and Layer 2
  registry/isolation coverage. The prior-year self-reference is the required two-column exception;
  a supporting `(id, organization_id)` uniqueness constraint is retained alongside the required
  year-scoped `(id, organization_id, school_year_id)` key.
- Database validation passed on PostgreSQL 18: fresh up/down/up, student CRUD/soft-delete/re-add,
  prior-year link, tenant isolation, registry coverage, and closed-year trigger meta-tests.
- Unit/race backend tests, lint, format/vet, pinned sqlc generation in the `sqlc:1.27.0` container,
  OpenAPI drift, frozen Bun frontend tests/build/lint, and `git diff --check` pass. The exact migration
  wrapper was attempted but host `psql` is unavailable; its equivalent container-client sequence
  passed.
- PR #85 is open, non-draft, mergeable, references `Fixes #55`, and cites SPEC §§8.2/8.7/10.1/11.2.
  Current-head CI passed all nine required checks; durations were Backend tests 1m15s, Backend lint
  1m07s, Backend format 34s, Generated code drift 1m26s, Migration round-trip 46s, Frontend tests
  13s, Frontend build 17s, Frontend lint 9s, and Repository formatting 6s. No review or inline
  comments remain. Quiet-window wait is not applicable before review handoff; post-merge main CI is
  not running.
- Skill draft: no — this turn uses the existing tenant-isolation patterns without a new reusable method.

## Issue #62 frontend students and adults

- Added `frontend/src/features/people/` with shared student/adult types, raw API boundary for
  year-scoped list/get/create/update/soft-delete, server field-error extraction, semantic tables,
  search/filter controls, detail forms, delete confirmation, and adult participation intent editing.
- Routes are available at `/y/:schoolYearId/students[/:personId]` and the equivalent adults paths;
  unscoped aliases render a choose-school-year message. The UI renders API `display_name` directly and
  sorts by legal family name then legal given name. `include_deleted=true` is sent only after the
  deliberate “Show deleted” toggle.
- Focused coverage in `PeoplePages.test.tsx`: API display-name usage, legal-name sort, student search/
  grade/homeroom filters, adult intent, and inline server errors.
- Current dependency PRs #77 (auth shell), #79 (adult API), and #55 (student API) remain unmerged;
  the UI is kept isolated and uses their established year-scoped path shape. When auth shell lands,
  its authenticated fetch transport must be wired into the people API boundary before release.
- Validation passed: frozen Bun install, 12 frontend tests, frontend build, frontend lint, and
  `git diff --check`. No skill draft: this used ordinary feature-component patterns.

## Issue #46 trusted proxy Real-IP extraction

- Replaced deprecated unconditional `chi/middleware.RealIP` with `internal/api/realip.go`.
  `TRUSTED_PROXY_CIDRS` is a comma-separated CIDR setting; empty configuration ignores
  X-Forwarded-For and X-Real-IP. The immediate TCP peer must be trusted, and XFF is merged,
  walked right-to-left, and fail-closed on malformed entries. Effective `RemoteAddr` mutation
  remains for downstream request behavior and logging semantics.
- Wired the setting through `config.Config`, `NewServerWithConfig`, `ServerOptions`, and
  `RouterOptions`; documented it in `.env.example` and README. Added focused tests for direct,
  trusted, multi-header, X-Real-IP, untrusted, malformed, and invalid-config cases.
- Validation passed: focused API/config tests; backend `make test` (integration test skipped because
  this worker has no `TEST_DATABASE_URL`), `make lint`, `make format`, `make generate && git diff
  --exit-code`, and `make generated-code-drift`; PostgreSQL 18 disposable migration up/down/up
  sequence; frozen Bun frontend tests (8), build, and lint; `git diff --check`.
- The migration script's host `psql` prerequisite is absent in this worker, so its exact wrapper
  was attempted and then its equivalent database sequence was run with the container's PostgreSQL
  client; the disposable database was removed after verification.
- PR #49 is open, non-draft, references `Fixes #46`, cites SPEC §20.1, and has no review or inline
  comments. The nine required current-head CI checks passed; the slowest were Generated code drift
  (83s), Backend lint (66s), Backend tests (60s), and Migration round-trip (50s). Latest main CI is
  complete and successful; no post-merge main run is active.
- Final handoff: update the persistent Workpad to `complete`; Detent owns the completion-lane transition.

## Issue #57 households, memberships, and guardian relationships

- Retry 2026-08-25: dependency #55 is closed through PR #85; #56 is closed through PR #79. The
  worktree was rebased onto `origin/main` at `b4be02b` before final handoff. Workpad comment
  `5404194535` records the retry diagnosis, plan, evidence, and final status.
- Added migration `20260825100000_households.sql`, sqlc queries/output, data/service layers, Huma
  household/membership/guardian routes, registry factories, Layer 2 registry assertions, and an
  integration acceptance test. Membership is many-to-many; guardian relationships are separate;
  household deletion is soft and link deletion is hard with audit entries.
- Corrected the merged closed-year trigger in `20260825150000_fix_closed_year_trigger_operation_case.sql`:
  PostgreSQL exposes `TG_OP` in uppercase, and the old lowercase comparisons silently cancelled
  every DELETE by returning `NEW` (NULL). Household membership deletion now uses ordinary scoped
  row-count deletes; temporary predicate probes were removed.
- PR #86 is open, non-draft, mergeable, references `Fixes #57`, and cites SPEC §§8.2/9.2/11.2.
  Rebased-head CI run `32855387627` passed all nine required checks. Slowest checks were Generated
  code drift (1m26s), Backend tests (1m19s), and Backend lint (1m04s); the PR CI duration was about
  1m29s from start to final check. No reviews, inline comments, blockers, or human action remain.
- Local `make test`, `make lint`, `make format`, pinned sqlc/OpenAPI generation plus drift check,
  frozen Bun tests (25), build, lint, and `git diff --check` pass. Integration tests skip without
  test DB URLs. The exact migration wrapper was attempted but stops because
  `POSTGRES_ADMIN_DATABASE_URL` is unavailable; CI migration round-trip passed. Quiet-window wait is
  not applicable before review handoff; post-merge main CI run `32855108245` completed successfully.
- Skill draft: no — the work used existing tenant-isolation and migration patterns; no new reusable
  procedure was needed.

## Issue #38 Detent merge gate

- Dependency #37 is closed through merged PR #43, and `origin/main` publishes exactly the nine required check names.
- `detent.yaml` now requires all nine exact CI check names and uses `git diff --check` as the local command gate.
- Detent docs (`docs/concepts.md` and `docs/merge-train.md`) confirm command gates require the configured local command plus green current-head required CI and the quiet period; this is recorded in the PR description.
- PR #44 is open, non-draft, references `Fixes #38`, and has no reviews or actionable comments. Current-head CI passed all nine checks on run `32708716582`; slowest checks were Generated code drift (1m21s), Backend tests (1m09s), and Backend lint (1m02s).
- Final local validation: YAML parse, exact workflow-name comparison, `git diff --check`, and `true` all passed. No skill draft: this was a routine one-file configuration change.

## Issue #35 XID identifiers

- Added reversible backend/migrations/00002_xid.sql with the public.xid20
  domain, the public.xid() generator, encode/decode and component helpers.
- Everything xid-related lives in the public schema; there is no util schema.
- The domain is deliberately named xid20 rather than xid. pg_catalog is
  resolved before the search_path, so a column declared as an unqualified xid
  becomes the built-in 4-byte transaction-id type; verified on PostgreSQL 18
  that this happens even with the domain in util and util first in
  search_path, so moving schemas does not avoid the collision. pg_catalog also
  already has an xid(xid8) function.
- Added frontend/src/lib/xid.ts and focused tests using local RFC 4648
  base32hex logic; bun add could not write temporary files in this worker.
- Updated AGENTS.md with XID and lowercase-SQL standards.
- Fixed two decoding bugs found while validating the consolidated migration:
  public.xid_time() read raw ASCII bytes instead of decoding base32, and
  public.xid_counter() lost its high bits because PostgreSQL gives << and |
  equal precedence (and + higher precedence than <<). The component helpers
  now decode once in a subquery and parenthesize every shift.
- Migration validated on PostgreSQL 18: goose up/down for both migrations,
  health_checks.id converted to public.xid with the public.xid() default and
  restored to uuid/uuidv7() on down. Asserted xid_time within a second of
  current_timestamp, xid_pid = pg_backend_pid(), xid_machine from
  pg_control_system(), xid_counter = currval('public.xid_serial'), and
  xid_encode(xid_decode(id)) = id. SQL decode output matches
  frontend/src/lib/xid.ts byte for byte.
- Backend make test passes; integration test skips without TEST_DATABASE_URL.
- Frontend build/lint pass and Node-launched Vitest passes 12 tests. Bun's
  Vitest wrapper fails locally with port.addListener incompatibility.
- No skill draft: the implementation used existing project patterns.

## Issue #37 quality gates, timestamped migrations, and roles

- Existing in-progress changes expanded CI to exactly nine named checks, added race-enabled backend
  tests, format/vet, pinned golangci-lint v1.64.8, generated-artifact drift, and migration round-trip.
- `backend/sqlc.yaml` now emits to `internal/db/gen`; generation skips the currently absent
  `backend/sql/queries` directory. Goose uses `WithAllowMissing`.
- Added timestamped `20260824090000_database_roles.sql`. Compose bootstrap provisions roles and makes
  the default database owned by `miniclass_migrator`; CI provisions roles and transfers the test DB.
  `miniclass_app` is non-superuser, non-CREATEROLE, non-CREATEDB, `nobypassrls`, and has a 10s timeout.
- Cluster roles are intentionally retained on migration down: rollback revokes per-database grants and
  restores ownership to `miniclass_migrator`, because a database migration cannot safely drop roles used
  by another database.
- Validation: backend `make test` and `make format`/`make lint` pass; PostgreSQL 18 integration and
  scratch `up/down/up` pass; fresh Compose-style bootstrap migration and role attributes pass; frontend
  Bun install/test (12), build, and lint pass after the test script was changed to invoke Vitest under
  Node 24. CI's first run also confirmed the prebuilt golangci-lint binary needed `install-mode: goinstall`
  to build against the repository's Go 1.26 target.
- Open items: update the Workpad, create/push the PR with `Fixes #37`, then inspect current-head CI and
  review comments before final completion.

## Issue #14 quality gates

- `backend/Makefile` target `test` now runs `go test -v ./... -count=1`, including unit and integration packages.
- `.github/workflows/ci.yml` publishes `Backend tests`, `Frontend tests`, `Frontend build`, `Frontend lint`, and `Repository formatting`; backend CI supplies PostgreSQL and `TEST_DATABASE_URL`.
- `WORKFLOW.md` and `README.md` document the local commands and exact CI check names.
- Validation: frontend `npm ci`, 9 tests, build, lint, YAML parse, and `git diff --check` pass. Local backend execution could not start because only Go 1.26 is installed and the restricted host cannot download Go 1.27; CI uses `actions/setup-go` from `backend/go.mod`.
- First CI backend run exposed a Goose parsing failure for the dollar-quoted UUID function; migration `00001_initial_schema.sql` now wraps that function in Goose statement-boundary directives.
- Open item: commit/push, open PR with `Fixes #14`, inspect current-head CI/reviews, then update the Workpad.

- Issue #9 adds the initial React/TypeScript frontend shell under `frontend/src`.
- Entry point: `frontend/src/main.tsx`; router and layout: `frontend/src/App.tsx`; global styling: `frontend/src/index.css`.
- Vite entry document is `frontend/index.html`; `BrowserRouter` serves `/` plus placeholder routes for `/classes`, `/assignments`, `/students`, and `/settings`.
- Validation: `cd frontend && npm run build`, `npm run lint`, and repository gate `true`.
- GitHub CLI is unauthenticated in this worker (`gh auth status` reports missing credentials), so issue Workpad/PR operations require a later authenticated handoff.

## Backend configuration management

- Added `backend/internal/config` with dotenv loading, typed settings, defaults, and validation.
- `DATABASE_URL` is required; optional settings use `.env.example` defaults where defined.
- Focused validation: `go test ./backend/internal/config/...` passes after adding `backend/go.sum` for `godotenv`.
- Full local validation gate is `true` per `WORKFLOW.md`; repository has no CI configuration or declared project check names.
- GitHub CLI is authenticated as `christophergm`; PR #17 is open for the Detent branch and references issue #1.
- PR #17 is non-draft and clean with no configured CI checks or review comments; Workpad comment is complete.

## Backend PostgreSQL connection and health ping

- Added `backend/internal/db/db.go` with config-backed pgx pool creation, startup ping, package/method `PingDB`, pool access, and idempotent close.
- Added focused failure-path tests in `backend/internal/db/db_test.go`; added pgx dependency checksums to `backend/go.sum`.
- Local Go 1.26 cannot honor the repository's Go 1.27 directive because toolchain checksum-cache access is restricted. Disposable Detent temp copy with only `go.mod` set to 1.26 passes `GOSUMDB=off go test ./...`.
- Repository gate `true`, `gofmt -d`, and `git diff --check` pass. CI has one `Validate` job running `git diff --check`; no CI checks are currently configured beyond that.
- Workpad comment on issue #2 is complete. PR #18 is open, non-draft, references `Fixes #2`, and has no actionable review comments; keep the issue in its Detent-managed worker lane for promotion.

## Frontend health-check page

- Added `frontend/src/lib/api.ts` with typed `fetchHealth`, `VITE_API_URL` support, response validation, and normalized `ApiError` failures.
- Added `frontend/src/lib/hooks/useHealth.ts` with TanStack Query and a 30-second refetch interval; `/health` is linked from `frontend/src/App.tsx`.
- `frontend/src/features/health/HealthCheck.tsx` renders loading, healthy/degraded, and failed states with backend version, timestamp, and database status.
- Component tests are in `frontend/src/features/health/HealthCheck.test.tsx`; Vitest uses jsdom via `frontend/vite.config.ts`.
- Validation passed: `cd frontend && npm run test -- --run` (4 tests), `npm run lint`, `npm run build`, repository gate `true`, and `git diff --check`.
- Open item for handoff: commit/push, open or update the PR with `Fixes #11`, recheck GitHub CI/reviews, and update the issue Workpad to the final status.

## Frontend home page

- Root overview in `frontend/src/App.tsx` now includes the existing `HealthCheck` component, with a link to the detailed `/health` route and responsive home-page styling in `frontend/src/index.css`.
- Regression coverage: `frontend/src/App.test.tsx` verifies the root overview and health component are rendered.
- Validation: `cd frontend && npm run test -- --run` (9 tests), `npm run lint`, `npm run build`, repository gate `true`, and `git diff --check` pass.
- No skill draft created; the change used the existing frontend patterns and did not expose a reusable non-obvious procedure.

## Full-stack local smoke test (#13)

- Current `origin/main` contains the merged migration, seed, backend API entrypoint and health route, backend integration test, typed frontend API client, and frontend health page needed to execute the smoke test.
- Dependency #8 is closed through merged PR #26. Dependency #12 remains open administratively, but the required frontend implementation is merged in PR #21 and present on current `main`.
- `scripts/smoke-test.sh` starts PostgreSQL/Adminer, applies migrations, starts backend/frontend processes, checks `/api/health` and `/health`, prints the browser check, and preserves logs under `TMPDIR`.
- README now documents prerequisites, service URLs, the smoke command, the manual browser verification, Adminer, and failure diagnosis. `.env.example` uses the backend origin for `VITE_API_URL` because the frontend client appends `/api/health`.
- Frontend validation passes: `npm run test -- --run` (8 tests), `npm run lint`, and `npm run build`.
- Backend validation passes in a disposable `TMPDIR` copy with Go 1.27 lowered to 1.26: `GOTOOLCHAIN=local GOSUMDB=off go test ./...` (integration test skips without `TEST_DATABASE_URL`).
- The documented smoke command was attempted with `.env.example`; it stopped at `docker compose up` because the local Docker daemon is unavailable. No live database/browser result is claimed.
- Project CI contract: `Validate` runs `git diff --check`; local repository gate is `true`.

## Issue #15 local development/reset documentation

- README now documents first setup, service URLs, repeatable seed behavior, reset safety, volume handling, and troubleshooting.
- `backend/Makefile reset-db` requires `RESET_DB_CONFIRM=1` and uses `psql -v ON_ERROR_STOP=1`; `backend/scripts/seed.sql` avoids duplicate `seeded` health rows.
- Current schema only has `health_checks`; there are no teacher/class/student/assignment tables or fixtures yet.
- Dependencies #7, #13, and #14 are closed. Validate with `true`, `git diff --check`, frontend checks, and backend tests where the installed Go toolchain permits.
- Validation: `true`, `git diff --check`, `sh -n scripts/smoke-test.sh`, reset confirmation guard, frontend `npm ci`, 9 tests, lint, and build pass. Backend `go test ./... -count=1` is unavailable locally because Go 1.26 is installed while `backend/go.mod` requires Go 1.27.

## Issue #31 PostgreSQL 18 upgrade

- Local Compose uses `postgres:18-alpine`; CI's backend service uses `postgres:18`.
- `STRUCTURE.md` now documents PostgreSQL 18. The initial migration uses PostgreSQL 18's built-in `uuidv7()` and removes the PostgreSQL 16 compatibility function.
- Validation: disposable-TMPDIR backend `GOTOOLCHAIN=local GOSUMDB=off go test -v ./... -count=1` passes (integration test skips without `TEST_DATABASE_URL`); frontend `npm ci`, 9 tests, build, lint, Compose config, and `git diff --check` pass. Exact `cd backend && make test` cannot run because Go 1.27 is not installed locally.
- PR #32 is open, non-draft, references `Fixes #31`, and all five current-head CI checks pass with no review comments. The Workpad is ready for the completion declaration.

## Issue #40 frontend API client generation

- Dependency P0-5a (#39) is closed through merged PR #47; `origin/main` now contains the committed
  `backend/openapi.json` contract.
- Added `openapi-typescript` generation to `frontend/package.json`; `build`, `dev`, and `test`
  generate `frontend/src/lib/api.generated.ts` from the backend contract. The generated file is
  ignored and remains uncommitted.
- Replaced hand-written health response validation with `openapi-fetch`, retaining the UI-facing
  `ApiClient` and normalized `ApiError`; deleted `frontend/src/lib/api.test.ts` as required.
- Validation: frozen Bun install, 8 frontend tests, frontend build, frontend lint, and
  `git diff --check` pass. Generated file was checked locally and is ignored.
- PR #48 is open, non-draft, clean, references `Fixes #40`, and cites SPEC §13.5/§16.5/§17.4.1
  plus ADR 0004/0010. All current-head checks pass: Backend tests, Backend lint, Backend format,
  Generated code drift, Migration round-trip, Frontend tests, Frontend build, Frontend lint, and
  Repository formatting. No review or inline comments remain.
- Skill draft: no; this was a routine dependency and client wiring change.

## Issue #33 Node 24 and Bun upgrade

- Frontend now declares Node.js `>=24` and Bun `1.3.14` in `frontend/package.json`, with `frontend/bun.lock` replacing `frontend/package-lock.json`.
- CI uses Node 24 plus `oven-sh/setup-bun@v2` and runs frozen Bun installs for the existing `Frontend tests`, `Frontend build`, and `Frontend lint` checks.
- README, QUICK_START, STRUCTURE, IMPLEMENTATION_PLAN, WORKFLOW, and the smoke script use Bun; no npm/package-lock references remain.
- Validation: Bun install, 9 frontend tests, build, lint, CI YAML parse, shell syntax, and `git diff --check` pass. `cd backend && make test` is blocked locally by Go 1.26 attempting a restricted Go 1.27 toolchain download; CI uses `backend/go.mod`.
- Open items: commit/push, open PR with `Fixes #33`, inspect current-head CI/reviews, and update the Workpad.

## Issue #41 frontend foundation

- Recovered work removes the fabricated dashboard and placeholder routes from `frontend/src/App.tsx`; `/` and unknown routes redirect to `/health`.
- Tailwind v4 is loaded from `frontend/src/styles/globals.css` through `@tailwindcss/vite`; shadcn configuration is in `frontend/components.json`, with exactly `button`, `input`, and `table` under `frontend/src/components/ui`.
- The `@/*` alias is configured in `frontend/tsconfig.json` and `frontend/vite.config.ts`; `frontend/src/index.css` is deleted.
- `bun run test -- --run` passes 12 tests after the test script uses `bunx vitest` (Node-backed Vitest); forcing `bunx --bun vitest` fails in this worker with Bun's `port.addListener` incompatibility. Build, lint, backend `make test`, shell syntax, and `git diff --check` pass; the integration test skips without `TEST_DATABASE_URL`.
- Completed: PR #45 is open and non-draft with `Fixes #41`; rebased onto `origin/main` and all nine current-head CI checks pass. No review comments or actionable findings remain.

## Issue #39 Huma v2 API contract

- Rebased onto current `origin/main`; commit `c9d4faa` adds Huma v2.39.1 over `humachi`/chi, typed registrations for `/api`, `/api/`, and `/api/health`, and committed `backend/openapi.json`.
- RFC 9457 responses use `application/problem+json`; `backend/internal/api/problems` registers route-not-found, method-not-allowed, internal-error, and database-unavailable slugs. The forced JSON middleware is removed.
- `backend/cmd/openapi` plus `make openapi`, `make generate`, and `make generated-code-drift` generate and compare OpenAPI deterministically. The existing CI Generated code drift job runs both generation and the deterministic check.
- Validation passed: `make test` with race detector (integration DB test skipped without `TEST_DATABASE_URL`), `make lint` after a narrow required-RealIP suppression, `make format`, `make generate && git diff --exit-code`, `make generated-code-drift`, frontend Bun install/tests (12), build, lint, repository diff check, and migration round-trip against disposable PostgreSQL 18.
- Follow-up issue #46 tracks replacing deprecated `chi/middleware.RealIP` with trusted-proxy-aware extraction; this issue retains the required middleware behavior.
- PR #47 is open, non-draft, references `Fixes #39`, cites SPEC §13.5/§16.5/§17.4.1 and ADR 0004, and is clean with no review comments.
- Current-head CI is green: Backend tests, Backend lint, Backend format, Generated code drift, Migration round-trip, Frontend tests, Frontend build, Frontend lint, and Repository formatting.
- Merge fallback rebase conflict was limited to this handoff file; Issue #41 and Issue #39 notes were retained, with no source conflict resolution.
- Merge fallback gate: `git diff --check` passed on the clean rebased head; branch was pushed with `--force-with-lease`.
- Final handoff: update the persistent Workpad to `complete`; Detent owns the completion-lane transition.

## Issue #50 identity schema, token primitive, and bootstrap

- Recovered implementation is based on `origin/main` and remains scoped to P1-1. The old pool in
  `internal/db` is replaced by `internal/data`; generated sqlc output is in `internal/db/gen` and
  only the data packages import it. `internal/identity` is the only production importer of the
  unscoped `internal/data/identity` accessor; `cmd/bootstrap` uses `identity.NewStore`.
- Migration `20260824120000_identity_schema.sql` adds the four identity tables and enums with
  `public.xid20`/`public.xid()` keys, invitation-state checks, hashed-token metadata, trigger
  timestamps, and drops `health_checks`. Goose statement-boundary directives are required around
  the trigger function.
- `internal/identity` generates 32-byte URL-safe bearer values, hashes them with SHA-256, bootstraps
  an Owner invitation atomically, and regenerates pending admin invitations with generation+1,
  old-token revocation, and membership relinking. `cmd/bootstrap` prints only the claim URL and
  expiry.
- Focused unit tests pass. PostgreSQL 18 integration passes for health plus bootstrap/regeneration;
  the full backend race suite, vet/format, lint, sqlc/OpenAPI generation, frozen Bun frontend
  tests/build/lint, and `git diff --check` pass. The exact migration wrapper was attempted but host
  `psql` is unavailable; its identical up/down/up sequence passed through `docker exec psql` and
  left no disposable database.
- PR #71 is open, non-draft, mergeable, references `Fixes #50`, cites SPEC §9.1/§9.3/§9.5/§6.6 and
  ADR 0007/0009/0010, and has no review or inline comments. After the generated-artifact correction
  commit `ee21b13`, all nine current-head required checks passed; the Workpad is the remaining final
  handoff and Detent owns the completion-lane transition.

## Issue #51 tenancy guard, Layer 1 harness, and audit log

- `internal/data` now owns `Tx`, `InTenant`, `InTenantRead`, transaction-local `set local
  app.organization_id`, and the audit/no-audit commit invariant. `internal/audit` owns actor/action
  vocabulary; generated audit queries remain behind `internal/data`.
- Migration `20260824140000_tenancy_audit.sql` adds `audit_log` with `audit_actor_type`, RLS enabled
  and forced, an `app.organization_id` policy without `missing_ok`, `(id, organization_id)` uniqueness,
  and app-role `insert`/`select`-only privileges. It grants the four identity tables in schema-isolated
  tests and revokes app access to `goose_db_version`.
- `internal/testing` provides the two-pool schema-isolated harness; Layer 1 assertions live with the
  existing integration package to serialize migrations that replace public XID functions.
- `backend/scripts/verify-depguard.sh` proves both protected imports fail outside their allowlisted
  packages; `make lint` runs it after golangci-lint.
- Focused and full backend race tests pass with PostgreSQL 18 and both roles; backend lint, format/vet,
  pinned sqlc generation, generated-code drift, migration round-trip, frozen Bun tests/build/lint,
  and `git diff --check` pass. PR and current-head CI remain open items.

## Issue #52 authentication, capabilities, and `/api/me`

- Added `internal/auth` with ES256/RS256-pinned JWT verification, 30s `iss`/`aud`/`exp`/`nbf`
  validation, local static-key and Supabase cached-JWKS implementations, rate-limited unknown-kid
  refresh, `Principal`, and the complete SPEC §6.6 capability matrix.
- Huma operations use `x-required-capability` metadata plus bearer security declarations; global
  middleware authenticates, resolves one local membership, then checks capability. `/api/me` and
  `/api/auth/claim` are implemented; claim requires verified email matching the invitation and
  atomically consumes the bearer. `cmd/devtoken` mints local tokens without Supabase credentials.
- `internal/data/identity` remains imported only by `internal/identity`; auth/API use contracts from
  `internal/auth`. Generated sqlc identity queries and `openapi.json` are committed; no migration was
  needed because identity tables already exist.
- Focused unit tests, API HTTP tests, `make test`, `make lint`, `make format`, OpenAPI/generated-code
  drift, frozen Bun frontend tests/build/lint, PostgreSQL 18 migration round-trip, and `git diff --check`
  pass. PostgreSQL integration tests use the two-pool harness and skip here without
  `TEST_DATABASE_URL`/`TEST_APP_DATABASE_URL`; CI supplies both.
- PR #73 is open, non-draft, mergeable, cites SPEC §§6.6/9.3/9.4 and ADRs 0008/0009, and has no review
  or inline comments. The final complete-tree CI run passed all nine required checks; Workpad completion
  declaration remains the final handoff.

## Issue #58 administrator management

- Added Owner-only `/api/administrators` list/invite, role-change, removal, invitation resend, and
  invitation revoke operations. `WithIdentity` wires the Store as the manager; invitation links default
  to `http://localhost:5173/claim` and may be overridden with `WithInvitationClaimBaseURL`.
- Tenant mutations run through `data.InTenant`, and invitation/permission/add/remove actions record
  same-transaction audit entries. The unscoped identity accessor remains restricted to `internal/identity`.
- Added integration coverage for regeneration invalidation, revoke, role changes, removal, audit count,
  tenant scoping, and last-Owner protection; it skips here because both test database URLs are absent.
- Focused backend/API tests and OpenAPI/sqlc generation pass. PR #74 is open, non-draft, mergeable, and
  cites SPEC §§6.6/9.3/9.5/20.1 plus ADRs 0007/0008/0009/0010. Its final current-head run passed all
  nine required checks; slowest were Generated code drift (1m27s), Backend tests/lint (1m04s), and
  Migration round-trip (1m08s). The Workpad is complete; Detent owns the completion-lane transition.

## Issue #53 school years, lifecycle, closed-year trigger, and Layer 2

- Added timestamped migration `20260824160000_school_years.sql`: `school_year_state`, tenant-scoped
  `school_years` with forced RLS and XID keys, and `public.prevent_closed_school_year_mutation()`.
  Reopen preparation is transaction-local, reasoned, and target-ID scoped; `audit_log` remains exempt.
- `internal/data/school_year.go` keeps generated SQL behind the data boundary. `internal/schoolyear`
  enforces setup→active→closed, forbids active→setup, requires Owner+reason for closed→active, and
  audits create, edit, delete, and every transition. Huma CRUD routes use `manage_school_year` and
  map the trigger to `school-year-closed`/409.
- `internal/testing/registry` has a per-entity school-year registration. Layer 1 now checks registry
  coverage and closed-year triggers; Layer 2 tests cross-tenant read/fetch/update/delete and foreign
  parent insert behavior.
- Focused and full backend race tests pass with PostgreSQL 18 and both roles; lint, format/vet, exact
  sqlc v1.27 generation, OpenAPI generation/drift, and migration up/down/up pass. The exact migration
  wrapper was attempted but host `psql` is absent; the same sequence passed through `docker exec psql`.
- Frozen Bun frontend tests/build/lint and `git diff --check` pass. PR #75 is open, non-draft, mergeable,
  references `Fixes #53`, cites SPEC §§11.1/8.1/9.2/20.1 and ADR 0007, and has no review or inline
  comments. All nine current-head checks pass; slowest were Generated code drift (1m28s), Backend lint
  (1m09s), and Backend tests (1m02s). The exact wrapper's host `psql` limitation is the only local
  environment note; no blocker or human action remains.

## Issue #61 frontend school years and settings

- Added `/years` as the landing page, `/y/:schoolYearId` lifecycle/detail view, and `/settings`.
  School-year creation, label editing, setup/active/closed transitions, owner-only reasoned reopen,
  closed-year read-only messaging, and RFC 9457 field errors are covered in the schoolyear feature.
- Added settings UI for ordered grade levels, homerooms, homeroom label, and Owner-only administrator
  management. Retired grade/homeroom entries remain in the response but `activeGradeLevels` and
  `activeHomerooms` exclude them from pickers. API paths match the P1-5 and P1-9 contracts.
- Added the shared raw JSON API request/error parser for settings resources; generated OpenAPI output
  remains untouched because the dependent backend contracts are still on their review PRs.
- Validation passed: frozen Bun install, 14 frontend tests, frontend build, frontend lint, backend
  tests (integration skipped without test database URLs), backend lint, backend format, generated-code
  generation with no backend artifact diff, and `git diff --check`. Migration round-trip wrapper was
  attempted but cannot run locally because `POSTGRES_ADMIN_DATABASE_URL` is unset.
- PR #80 is open, non-draft, mergeable, references `Fixes #61`, cites SPEC §§11.1/10.1/6.6, and has
  no review or inline comments. All nine current-head checks passed on `d2e2532`; slow checks were
  Backend tests, Backend lint, and Generated code drift. CI migration round-trip passed despite the
  local wrapper lacking `POSTGRES_ADMIN_DATABASE_URL`. Skill draft: no — this was standard
  frontend/API integration work.

## 2026-08-27T12:14:40Z - Failed run output tail

- final_state: failed
- error: stream turn: context canceled
close codex app-server transport: signal: terminated: stderr: [2m2026-08-27T12:09:57.290204Z[0m [31mERROR[0m [2mcodex_core::tools::router[0m[2m:[0m [3merror[0m[2m=[0mexec_command failed for `/bin/zsh -lc 'rg -n "ListGuardianRelationships|ListHouseholdStudents|ListHouseholdAdults|SoftDelete(Student|Adult|Household)|deleted_at" backend/tests backend/internal | head -240'`: CreateProcess { message: "Rejected(\"Failed to create unified exec process: No such file or directory (os error 2)\")" }
[2m2026-08-27T12:14:39.285451Z[0m [31mERROR[0m [2mcodex_models_manager::manager[0m[2m:[0m failed to refresh available models: timeout waiting for child process to exit
worker github REST budget monitor failed: Get "https://api.github.com/rate_limit": context deadline exceeded

Output tail:

```text
I’ll inspect the current issue/PR state and the existing household/guardian listing queries, reproduce the soft-delete leak, then make the smallest scoped fix and run the relevant backend plus repository gates. I’ll keep the Workpad and `.detent/notes.md` current throughout.The issue is open and has no existing PR or Workpad comment. A maintainer note confirms the household membership UI now uses the listing endpoints, and restore behavior is tracked separately in #103; this fix should remain a query-time exclusion that preserves reversible link rows. I’m checking dependency state and the current SQL/test architecture before editing.The dependency check is clear: #101 has no native blockers or `Depends on:` references. The branch is clean but behind the freshly fetched `origin/main`, so I’ll rebase it before source changes. The live household member queries already filter deleted members in two paths; the remaining leak is the guardian query plus household-listing predicates that must also exclude deleted households and preserve registry queries.The rebase onto current `origin/main` completed cleanly. The implementation target is now precise: add `exists` checks for both live person endpoints to `ListGuardianRelationships`, and require an active household in the per-household student/adult listings. The year-wide membership queries already enforce all three live-record predicates; registry queries remain intentionally unfiltered. I’m adding regression coverage that also proves retained link rows are still removable/servable through their explicit historical paths.The pinned sqlc container is available, so generated output was regenerated with the repository-required v1.27.0 rather than the worker’s v1.31.1 binary. The regression test now covers both deleted relationship endpoints, both membership kinds, direct row retention, and the deleted-household query path. Next I’m running focused backend tests; integration cases will use the repository’s two-role database harness if its test URLs are available.The focused compile exposed one test-only API mismatch: membership retention is intentionally available through `internal/data`, not the `people.Service`. I’m correcting the assertions to use the tenant-scoped read transaction, which also keeps the test aligned with the repository’s data-access boundary.The focused backend compilation/tests now pass. The integration package reports success in this checkout, but I’m verifying whether its database-backed cases actually ran or were skipped, then I’ll run the full required gates and address any environment-specific failures without weakening coverage.The first required gate is currently blocked by a pre-existing Docker resource collision: `make test-backend` cannot start its fixed-name `miniclass-postgres` container because another container already owns that name. I won’t remove or overwrite that container; I’m inspecting whether it is a compatible repository database so I can run the equivalent backend checks safely, while recording the exact gate result.
```

## Issue #103 restore soft-deleted roster records

- Added opt-in `include_deleted` list queries and `deleted_at` responses for students, adults, and households.
- Added audited, reason-required restore operations with `manage_roster` Huma routes; restore uses tenant-scoped including-deleted lookup and records actor/time/reason.
- Restored frontend deleted filters, styling, and restore actions for people and households, using generated OpenAPI types and query invalidation.
- Pinned sqlc v1.27.0 generated artifacts via `CGO_ENABLED=0 GOSUMDB=off`; normal local Make generation refuses worker sqlc v1.31.1.
- Validation: backend unit/all tests, lint, format/vet, generated OpenAPI drift, frontend 80 tests/build/lint, and `git diff --check` pass. `make test-backend` cannot start because Docker is unavailable; migration round-trip/smoke remain environment-dependent.
- Open items: commit/push, open PR with `Fixes #103` and SPEC §§21.3/9.2/5.4, inspect CI/reviews, then complete Workpad.

### Review follow-up on PR #116

CI was green but four acceptance-criteria gaps survived it, because nothing in the suite covered them:

- **No isolation test for the new query paths (SPEC §9.2, AGENTS.md rule 2).** The Layer 2 registry
  harness probes read/fetch/update/delete only; `include_deleted` and `Restore*` were unprobed, and
  `include_deleted` is the one statement that deliberately drops a filter. `tests/integration/roster_restore_test.go`
  now probes both. The service-level call fails earlier than expected — a foreign organisation cannot
  resolve the owner's school year at all — so the new statements are also probed directly through
  `InTenantRead`/`InTenant`, below that check, which is where §9.2's "must not depend on each query
  remembering to filter" actually bites.
- **Households had no restore coverage**, though the acceptance criteria name them alongside students
  and adults. Added, including the not-deleted and blank-reason rejections.
- **No test read a restore audit entry back.** The existing tests assert `auditCount >= n`, which
  cannot show that actor, time and reason survived. The household test now asserts the entry's
  action, object, year, actor label, occurrence time, reason and change summary (SPEC §5.4).
- **Two restore errors carried misleading problem slugs**: a 400 "reason required" was typed
  `resource-not-found` and a 409 "not deleted" was typed `school-year-transition-invalid`. Added
  `restore-reason-required` and `roster-record-not-deleted`, following the existing
  `school-year-reason-required` precedent, and regenerated `openapi.json`.
- **Household restore bypassed `useRosterMutation`**, so it refetched one query by hand, left the
  rest of the year's cache stale, and dropped the rejected promise on the floor — a failed restore
  showed the user nothing. Now on the shared mutation, with the deleted-row treatment the people
  table already had.
- Added frontend tests for both roster surfaces asserting the request the filter issues and the call
  the restore action makes. #96 removed these controls precisely because they were inert; nothing
  would have caught them going inert again.
- Validation: full backend suite with `-race` against the local Postgres (all packages pass,
  integration included), `gofmt`/`go vet`, generated-code drift, frontend 82 tests, build and lint,
  `git diff --check`. **`make lint` cannot run locally**: golangci-lint v1.64.8 is built with go1.26.4
  against a go1.27.0 toolchain, so it reports 15 spurious `undefined: <package> (typecheck)` errors in
  files nobody touched. CI pins its own toolchain and is the authoritative gate for that check.

## 2026-08-27T23:47:09Z - Failed run output tail

- final_state: failed
- error: stream turn: context canceled
close codex app-server transport: signal: terminated: stderr: [2m2026-08-27T23:39:27.837061Z[0m [31mERROR[0m [2mcodex_core::tools::router[0m[2m:[0m [3merror[0m[2m=[0mexec_command failed for `/bin/zsh -lc 'rg -n "Restore|restore|include_deleted|deleted_at|reason" backend/internal frontend/src backend/sql | head -400'`: CreateProcess { message: "Rejected(\"Failed to create unified exec process: No such file or directory (os error 2)\")" }
[2m2026-08-27T23:44:17.614222Z[0m [31mERROR[0m [2mcodex_core::tools::router[0m[2m:[0m [3merror[0m[2m=[0mapply_patch verification failed: Failed to find expected lines in /Users/chrismott/dev/personal/miniclass-worktrees/miniclass-christophergm_miniclass_103-c15424b0c426/backend/internal/api/handlers/household.go:
	UpdatedAt     time.Time `json:"updated_at"`
worker github REST budget monitor failed: Get "https://api.github.com/rate_limit": context deadline exceeded

Output tail:

```text
I’m resuming the in-progress #103 implementation from the preserved worktree state. I’ll verify the current source, Workpad, and test results first, then finish validation and the PR handoff without changing scope.The branch is now confirmed based on the current `origin/main`; the earlier soft-delete listing fix is already merged there, so this worktree is clean and correctly positioned for #103. The issue has no PR or Workpad yet. I’m creating the required tracking comment and mapping the existing roster, audit, and generated-contract patterns before implementing the restore/include-deleted surface.The implementation shape is clear: list queries will take an explicit boolean and retain the default active-only predicate; restore will be a separate audited `POST .../{id}/restore` operation with a required reason, using “including deleted” lookup only inside the restore transaction. I’ll add a distinct `restore` audit action so the audit log records the operation precisely, then wire the three list pages to an opt-in filter and inline restore action.The backend changes are in place: all three list paths now have an explicit opt-in flag, each entity has a restore data operation and service audit entry, and the API has capability-protected restore routes. I’m regenerating sqlc/OpenAPI next; compilation will catch any generated-parameter or Huma contract details before I add the frontend and regression tests.
```

## 2026-08-28T01:16:21Z - Failed run output tail

- final_state: failed
- error: stream turn: context canceled
close codex app-server transport: signal: terminated
worker github REST budget monitor failed: Get "https://api.github.com/rate_limit": dial tcp: lookup api.github.com: no such host

Output tail:

```text
I’m continuing the preserved #103 implementation from the current worktree. I’ll inspect the existing diff and tracker state, finish any incomplete restore/list wiring, then run the full project gates and complete the PR handoff with the required Workpad evidence. I’m using the backend-development skill because this change crosses the repository’s API, data-access, and test layers.
```
