# Detent handoff notes

- Frontend shell: `frontend/src/main.tsx`, `frontend/src/App.tsx`, and `frontend/src/index.css`; Vite entry is `frontend/index.html`.
- Backend config: `backend/internal/config` loads dotenv values, applies defaults, and requires `DATABASE_URL`.
- Backend database: `backend/internal/db` creates and pings the pgx pool and closes it idempotently.
- Backend API: `backend/internal/api` provides the Chi router, middleware, `NewServerWithConfig`, `/api`, and `/api/health`.
- Issue #6 entry point: `backend/cmd/api/main.go` loads config, starts the verified database and HTTP server, logs address/environment/version, handles SIGINT/SIGTERM with a 10-second graceful-shutdown timeout, and defers database cleanup. Lifecycle tests are in `backend/cmd/api/main_test.go`.
- Issue #5/PR #23 and all native dependencies for #6 are terminal; current base is `origin/main` at `1b4f608`.
- Go 1.27 is unavailable in this worker because its toolchain checksum cache is restricted. Disposable copies under `$TMPDIR` with a Go 1.26 directive passed focused tests, `go test ./...`, `go build ./cmd/api`, and the missing-`DATABASE_URL` startup smoke test.
- Final local checks: `gofmt -d`, `git diff --check`, and repository gate `true` pass. Live PostgreSQL startup/shutdown was not available.
- Issue #10 adds `frontend/src/lib/api.ts`: `ApiClient.getHealth()` reads `VITE_API_URL`, validates the health contract, and normalizes HTTP/network/decode failures as `ApiError`; tests use the injectable `fetch` option.
- Issue #10 validation: `cd frontend && npm test -- --run`, `npm run build`, and `npm run lint` pass. `npm ci` required an isolated cache under `$TMPDIR` because the shared npm cache was root-owned.
