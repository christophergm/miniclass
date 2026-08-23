# Detent handoff notes

- Frontend shell: `frontend/src/main.tsx`, `frontend/src/App.tsx`, and `frontend/src/index.css`; Vite entry is `frontend/index.html`.
- Backend config: `backend/internal/config` loads dotenv values, applies defaults, and requires `DATABASE_URL`.
- Backend database: `backend/internal/db` creates and pings the pgx pool and closes it idempotently.
- Backend API: `backend/internal/api` provides the Chi router, middleware, `NewServerWithConfig`, `/api`, and `/api/health`.
- API entry point: `backend/cmd/api/main.go` loads config, starts the verified database and HTTP server, logs address/environment/version, handles SIGINT/SIGTERM with a 10-second graceful-shutdown timeout, and defers database cleanup.
- Current base: `origin/main` at `bd867a1`, with migrations, health endpoint, HTTP server, and API entry point merged.
- Integration test target: `backend/tests/integration`; `backend/Makefile` target `test` runs `go test -v ./tests/integration/... -count=1`.
- Project CI quality gate: `Validate` runs `git diff --check`; repository validation gate is `true`.
- Go 1.27 may be unavailable in this worker because its toolchain checksum cache is restricted; prior disposable Go 1.26 copies passed backend tests with `GOTOOLCHAIN=local GOSUMDB=off`.
- Issue #8 Workpad: https://github.com/christophergm/miniclass/issues/8#issuecomment-5386372050
