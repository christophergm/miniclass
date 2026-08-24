# Detent handoff notes

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
  sqlc v1.27 generation, OpenAPI generation/drift, and focused migration up/down/up pass. The exact
  migration wrapper was attempted but host `psql` is absent; the same sequence passed through
  `docker exec psql`. Frontend gates and final CI remain to run.
