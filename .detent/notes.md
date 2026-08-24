# Detent handoff notes

## Issue #38 Detent merge gate

- Dependency #37 is closed through merged PR #43, and `origin/main` publishes exactly the nine required check names.
- `detent.yaml` now requires all nine exact CI check names and uses `git diff --check` as the local command gate.
- Detent docs (`docs/concepts.md` and `docs/merge-train.md`) confirm command gates require the configured local command plus green current-head required CI and the quiet period; this is recorded in the PR description.
- Open items: validate config and repository gate, commit/push, open the PR with `Fixes #38`, inspect current-head CI/reviews, and update the Workpad.

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

## Issue #33 Node 24 and Bun upgrade

- Frontend now declares Node.js `>=24` and Bun `1.3.14` in `frontend/package.json`, with `frontend/bun.lock` replacing `frontend/package-lock.json`.
- CI uses Node 24 plus `oven-sh/setup-bun@v2` and runs frozen Bun installs for the existing `Frontend tests`, `Frontend build`, and `Frontend lint` checks.
- README, QUICK_START, STRUCTURE, IMPLEMENTATION_PLAN, WORKFLOW, and the smoke script use Bun; no npm/package-lock references remain.
- Validation: Bun install, 9 frontend tests, build, lint, CI YAML parse, shell syntax, and `git diff --check` pass. `cd backend && make test` is blocked locally by Go 1.26 attempting a restricted Go 1.27 toolchain download; CI uses `backend/go.mod`.
- Open items: commit/push, open PR with `Fixes #33`, inspect current-head CI/reviews, and update the Workpad.
