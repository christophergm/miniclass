# Detent handoff notes

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
