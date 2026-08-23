# Detent handoff notes

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

## API entry point blocker

- Rechecked 2026-08-23: `HEAD` is `b875cda` and `origin/main` is `45318ce`; issue #6 has no source changes or PR.
- `backend/internal/api` is absent on this base. PR #19 for required issue #4 remains open, non-draft, and conflicting (`mergeStateStatus=DIRTY`, `mergeable=CONFLICTING`); its implementation exists only on the unmerged PR head.
- Issue #5 remains open and also waits on #4, so the API route and health-handler contracts are not available here.
- GitHub's dependency endpoint previously returned 404 for #6, so the issue body's legacy `Depends on: #4` declaration remains the machine-readable fallback.
- Re-check after #4 merges or reaches a terminal state. Do not copy the dependency commit into this issue.
