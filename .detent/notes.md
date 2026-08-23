# Detent handoff notes

- Issue #9 adds the initial React/TypeScript frontend shell under `frontend/src`.
- Entry point: `frontend/src/main.tsx`; router and layout: `frontend/src/App.tsx`; global styling: `frontend/src/index.css`.
- Vite entry document is `frontend/index.html`; `BrowserRouter` serves `/` plus placeholder routes for `/classes`, `/assignments`, `/students`, and `/settings`.
- Validation: `cd frontend && npm run build`, `npm run lint`, and repository gate `true`.
- GitHub CLI is authenticated as `christophergm`; issue Workpad and PR operations are available in this worker.

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

## Initial database migration

- Added `backend/migrations/00001_initial_schema.sql` using Goose Up/Down markers.
- The initial schema creates a reversible `health_checks` table with a UUID v7 primary key, status, timestamp, and non-empty status constraint. PostgreSQL 16 compatibility is provided by the schema-local `miniclass_uuid_v7()` function.
- Rework validation: `git diff --check` passes; Docker PostgreSQL smoke testing is unavailable because the Docker daemon is not running.
- Goose CLI is not installed; the repository gate remains `true` and backend tests should be run with `GOTOOLCHAIN=local GOSUMDB=off` on the available Go 1.26 toolchain.

## Migration and seed commands (#7)

- Added `backend/cmd/migrate/main.go` using Goose and pgx stdlib; supports `up`, `down`, and `status`, defaulting to `up`, with credential-safe errors.
- Added `backend/cmd/seed/main.go` and `backend/scripts/seed.sql`; Make `migrate-up`/`migrate-down` now use the command wrapper and `seed` already invokes its command.
- Focused disposable-copy validation: `GOTOOLCHAIN=local GOSUMDB=off go test -mod=mod ./...`, builds, and missing-`DATABASE_URL` smoke checks pass. Repository gate is `true`; live PostgreSQL execution was not available.
- Issue #3 remains open, but its migration PR is the dependency that supplies the `health_checks` table consumed by the seed SQL.

## Backend HTTP server and middleware

- Added `backend/internal/api` with `NewServer`, `NewServerWithConfig`, `NewRouter`, explicit middleware ordering, CORS, structured request logging, panic recovery, JSON content type, and JSON 404/405 errors.
- API root is available at `/api` and `/api/`; feature routes such as health remain for subsequent issues.
- Focused validation: `cd backend && go test ./internal/api/...` passes.
- Full backend validation: `cd backend && go test ./...` passes.
- Repository validation gate: `true` passes; CI's declared `Validate` check runs `git diff --check`.
- This worker has Go 1.26.4 while the module requires Go 1.27; validate with a disposable `TMPDIR` copy whose `go.mod` directive is lowered to 1.26, using `/usr/local/go/bin/go` and `GOTOOLCHAIN=local GOSUMDB=off`.
- PR #19 is open, non-draft, references issue #4, and has no human, bot, or inline review comments.
- Rework validation: focused API tests, full backend tests, `gofmt -d internal/api/*.go`, `git diff --check`, and repository gate `true` all pass in the disposable Go 1.26 validation copy. Direct repository execution is blocked only by the unavailable Go 1.27 toolchain.
- Issue #1 dependency is closed and its project item is Done; issue #4 is In Progress during this rework pass.
- Rework review: PR #19 remains open and non-draft, references `Fixes #4`, has green `Validate` CI, and has no human, bot, or inline review feedback; no source correction was required.
- Rebased the PR branch onto `origin/main` at `600a705` and added the three missing Goose indirect module content checksums to `backend/go.sum`; focused and full backend tests pass without `-mod=mod` using the disposable Go 1.26 modfile.

## API health-check endpoint (#5)

- Issue #5 depends on #4 (HTTP server and middleware) and #2 (database ping); #2 is closed, but #4 remains open.
- PR #19 for #4 is open, non-draft, conflicting, and has no review comments or CI results; issue #4 remains in its worker-owned review lane, so #5 remains blocked until that dependency reaches a terminal state.
- The current branch is based on `origin/main`; no source changes were made for #5 because the health route must register into the pending router.
- The native dependency listing confirms #4 blocks #5; the issue body's `Depends on: #4` remains as a redundant fallback declaration.
- Recheck on 2026-08-23: issue #4 is still OPEN and PR #19 is still OPEN/non-draft with `mergeStateStatus=DIRTY`/`mergeable=CONFLICTING`; PR head is `7ff5a690524ff3e27386b6897fb3cae7e757b727`, and the native `blocked_by` relation is present. `HEAD` and `origin/main` are both `45318ceb0a52f73dc83692d0a80308ba854c999d`; neither contains `backend/internal/api`. No implementation or validation for #5 can proceed without importing unmerged server code.
- Final recheck on 2026-08-23: #4 remains OPEN and PR #19 remains non-draft/conflicting at the same head; GitHub's dependency endpoint confirms #4 blocks #5. `HEAD` and `origin/main` remain `45318ceb0a52f73dc83692d0a80308ba854c999d`, with no `backend/internal/api`; no source or focused validation was added for #5.
- Current recheck on 2026-08-23: #4 is still OPEN; PR #19 is still OPEN/non-draft with `mergeStateStatus=DIRTY` and `mergeable=CONFLICTING` at `7ff5a690524ff3e27386b6897fb3cae7e757b727`. The native `blocked_by` relation from #5 to #4 remains present. `HEAD` and `origin/main` are both `45318ceb0a52f73dc83692d0a80308ba854c999d`; `backend/internal/api` is still absent. No implementation or focused handler validation can proceed until #4 merges or reaches a terminal state.
- Rework resumed on 2026-08-23 after #4 merged as PR #19. The project item is `In Progress`; the branch was rebased onto `origin/main` at `ac9b09a`. Implement the handler against the merged API package, then run focused and full backend validation before opening the #5 PR.
