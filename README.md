# Mini Class Planner

A class planning and assignment system for a school's Friday mini-class programme: roster ingest,
preference collection, constraint-based placement, and published class and dismissal lists.

## Documentation map

| Document | Answers |
|---|---|
| [`SPEC.md`](./SPEC.md) | **What the system does.** Normative, technology-agnostic. The source of truth for behaviour. |
| [`PLAN.md`](./PLAN.md) | **When it gets built, and in what order.** Phases, milestones, exit criteria. |
| [`docs/adr/`](./docs/adr/) | **Why it is built this way.** Architecture decisions, including the rejected alternatives. |
| [`WORKFLOW.md`](./WORKFLOW.md) | How agents pick up, validate and hand off work. |
| [`AGENTS.md`](./AGENTS.md) | Repository rules that apply to every change. |
| This file | How to run it locally. |

If `SPEC.md` and any other document disagree, `SPEC.md` is right and the other document is a bug.

## Architecture

See [ADR 0001](./docs/adr/0001-application-stack-and-topology.md) for the full rationale.

**Stack:**
- Frontend: React + TypeScript + Vite, TanStack Query
- Backend: Go + chi + pgx + sqlc, Goose migrations
- Database: PostgreSQL 18 (Supabase in production, Docker locally)
- Auth: Supabase Auth for administrators; application-owned tokens for household, class-leader and
  public share links ([ADR 0002](./docs/adr/0002-authentication-and-access-mechanisms.md),
  [ADR 0009](./docs/adr/0009-administrator-sessions-and-identity-provider.md))
- Tenancy: PostgreSQL row-level security, enabled and forced, behind a closure-based data layer
  ([ADR 0007](./docs/adr/0007-tenancy-enforcement-and-data-access.md))
- Solver: Python OR-Tools CP-SAT sidecar, from Phase 5
  ([ADR 0003](./docs/adr/0003-assignment-solver-technology.md))

Go is the authoritative application layer. Every data path goes through Go, so that the tenancy
guard, authorization and audit log have exactly one implementation. The browser talks to Supabase for
one thing only — acquiring and refreshing an authentication token — and never for data
([ADR 0009](./docs/adr/0009-administrator-sessions-and-identity-provider.md)).

## Project Structure

```
miniclass/
├── backend/              # Go API server
│   ├── cmd/             # Entry points (api, migrate, seed)
│   ├── internal/        # Application code
│   ├── migrations/      # Database migrations (Goose)
│   ├── sql/             # SQL queries (sqlc)
│   ├── scripts/         # Utility scripts
│   └── tests/           # Integration tests
├── frontend/            # React application
│   ├── src/
│   │   ├── components/  # Shared UI components
│   │   ├── features/    # Feature modules
│   │   ├── lib/         # Utilities and API client
│   │   └── hooks/       # Custom React hooks
│   └── public/          # Static assets
├── docs/adr/            # Architecture decision records
└── compose.yaml         # Local development services (Docker Compose)
```

## Getting Started

### Prerequisites

**Recommended:**
- [proto](https://moonrepo.dev/proto) - Automatically manages Go, Node, Bun, and other tools

**Or install manually:**
- Docker & Docker Compose
- Go 1.26+
- Node.js 24+
- Bun 1.3+
- Make

### First Time Setup

1. **Clone and setup environment:**
   ```bash
   git clone <repo-url>
   cd miniclass
   cp .env.example .env
   ```

2. **Install development tools:**
   
   **Option A - Using proto (Recommended):**
   ```bash
   proto install
   ```
   
   **Option B - Manual installation:**
   - Install Go 1.26+, Node 24+, Bun 1.3+, Docker, Make

3. **Start database:**
   ```bash
   docker compose up -d postgres
   ```

4. **Install backend tools:**
   ```bash
   cd backend
   make install-tools
   ```

5. **Run migrations:**
   ```bash
   make migrate-up
   ```

6. **Seed database (optional):**
   ```bash
   make seed
   ```

7. **Install frontend dependencies:**
   ```bash
   cd ../frontend
   bun install
   ```

### Running Locally

**Terminal 1 - Database:**
```bash
docker compose up postgres
```

**Terminal 2 - Backend:**
```bash
cd backend
make dev
```

**Terminal 3 - Frontend:**
```bash
cd frontend
bun run dev
```

**Access:**
- Frontend: http://localhost:5173
- Backend API: http://localhost:8080
- Adminer (DB GUI): http://localhost:8081

### Frontend shell

The frontend starts at the responsive MiniClass workspace overview. Its initial
navigation is available at:

- `/` — classroom overview
- `/classes` — classes workspace placeholder
- `/assignments` — assignments workspace placeholder
- `/students` — students workspace placeholder
- `/settings` — settings workspace placeholder

Run `bun run dev` from `frontend/` for Vite development with hot module
replacement, or `bun run build` to verify the production bundle.

### Development Commands

#### Backend integration test

The PostgreSQL health integration test requires an isolated test database. Start
the local PostgreSQL service, then set `TEST_DATABASE_URL` to the
`miniclass_test` database created by Docker Compose:

```bash
docker compose up -d postgres
export TEST_DATABASE_URL='postgres://miniclass_migrator:miniclass_migrator_dev_password@localhost:5432/miniclass_test?sslmode=disable'
export TEST_APP_DATABASE_URL='postgres://miniclass_app:miniclass_app_dev_password@localhost:5432/miniclass_test?sslmode=disable'
cd backend
make test
```

Each test run creates a unique schema, applies the migrations, and drops that
schema during cleanup. Do not point `TEST_DATABASE_URL` at a development or
production database.

**Backend:**
```bash
cd backend
make help              # Show all available commands
make dev               # Run with hot reload
make test              # Run integration tests
make migrate-create NAME=my_migration  # Create new migration
make seed              # Load repeatable development fixtures
make reset-db RESET_DB_CONFIRM=1  # Drop, migrate, and seed the local database
make sqlc              # Regenerate DB code
```

**Frontend:**
```bash
cd frontend
bun run dev            # Start dev server
bun run build          # Production build
bun run test           # Run tests
bun run lint           # Lint code
```

## Ports

All ports are configurable via `.env` so that parallel worktrees can run simultaneously.

| Service | Default | Environment variable |
|---|---|---|
| Frontend (Vite) | 5173 | `VITE_PORT` |
| Backend (Go API) | 8080 | `PORT` |
| PostgreSQL | 5432 | `POSTGRES_PORT` |
| Adminer | 8081 | `ADMINER_PORT` |

## Parallel Development (Multiple Worktrees)

To run multiple instances in parallel:

1. Create a new worktree
2. Copy `.env.example` to `.env` in the worktree
3. Edit `.env` to use different ports from the table above
4. Start services with the custom `.env`

`.prototools` is read from the current directory, so each worktree may pin its own tool versions.

## Testing

The reproducible quality gates are:

```bash
cd backend && make test
cd backend && make lint
cd backend && make format
cd backend && make generate && git diff --exit-code
cd backend && ./scripts/migration-round-trip.sh
cd frontend && bun install --frozen-lockfile && bun run test -- --run
cd frontend && bun install --frozen-lockfile && bun run build
cd frontend && bun install --frozen-lockfile && bun run lint
git diff --check
```

CI runs the backend integration test against PostgreSQL and publishes exactly
nine checks: Backend tests, Backend lint, Backend format, Generated code drift,
Migration round-trip, Frontend tests, Frontend build, Frontend lint, and
Repository formatting.

**Backend Integration Tests:**
```bash
cd backend
make test
```

Tests use the `miniclass_test` database automatically created by Docker.

**Frontend Tests:**
```bash
cd frontend
bun run test
```

## Database Management

**Create migration:**
```bash
cd backend
make migrate-create NAME=add_users_table
```

**Apply migrations:**
```bash
make migrate-up
```

**Rollback last migration:**
```bash
make migrate-down
```

**Reset database:**
```bash
RESET_DB_CONFIRM=1 make reset-db
```

Reset drops and recreates the `public` schema, reapplies every migration, and
loads the development seed. It is intentionally guarded by
`RESET_DB_CONFIRM=1` because it destroys all data in `DATABASE_URL`; use only
with the local database from `.env`, never with a shared, staging, or
production database. To inspect the migration state without changing data:

```bash
make migrate-up       # Apply pending migrations
make migrate-down     # Roll back the latest migration
goose -dir migrations postgres "$DATABASE_URL" status
```

The Compose PostgreSQL volume persists across restarts. A database reset does
not remove that volume; stop the services with `docker compose down`. If the
database itself needs to be recreated after a broken local volume, use
`docker compose down -v` and then start PostgreSQL again. This removes the
local Compose volume and all data in it.

## Troubleshooting

- **`connection refused` or `database does not exist`:** confirm Docker is
  running, start PostgreSQL with `docker compose up -d postgres`, and check
  that `DATABASE_URL` uses the same credentials and port as `.env`.
- **`DATABASE_URL is required`:** run commands from `backend/` so the Makefile
  can load the repository `.env`, or export `DATABASE_URL` explicitly.
- **`make reset-db` refuses to run:** include the deliberate confirmation,
  `RESET_DB_CONFIRM=1 make reset-db`, after checking the database URL.
- **Port already in use:** set `POSTGRES_PORT`, `PORT`, `VITE_PORT`, or
  `ADMINER_PORT` in `.env`, then use matching URLs when connecting.
- **Migrations fail after changing the schema:** inspect the migration status,
  fix the migration, and use the confirmed reset command for a clean local
  database. Do not edit the Goose version table manually.
- **Frontend cannot reach the API:** set `VITE_API_URL` to the backend origin
  (for example `http://localhost:8080`), restart Vite, and check
  `http://localhost:8080/api/health` directly.
- **Integration tests touch the wrong database:** set
  `TEST_DATABASE_URL` to `miniclass_test`; tests create and remove an isolated
  schema and must never use the development or production URL.
- **TypeScript errors about modern syntax:** an ancient global `tsc` is shadowing the project's
  TypeScript. All `frontend/package.json` scripts use `bunx --bun` specifically to force resolution
  from `node_modules`; do not remove it.
- **`air` is missing:** it is not a proto plugin and is therefore not in `.prototools`. Install it
  with `go install github.com/air-verse/air@latest`.
- **`command not found: docker-compose`:** expected. Use `docker compose` (v2, with a space).

## Health Check

Verify the stack is running:
```bash
curl http://localhost:8080/api/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2026-08-22T10:30:00Z",
  "database": "connected",
  "version": "0.1.0"
}
```

## Full-stack smoke test

The smoke test starts the local PostgreSQL and Adminer services, applies
migrations, starts the API and frontend, and checks the API and frontend HTTP
surfaces:

```bash
cp .env.example .env        # first run only
cd frontend && bun install  # first run only
cd ..
./scripts/smoke-test.sh
```

The script keeps backend, frontend, migration, and Compose output in a
temporary directory under `TMPDIR` and prints the directory on success. On
failure it prints the last lines of each log and the database service logs.
After the automated checks pass, open
[http://localhost:5173/health](http://localhost:5173/health) and confirm that
the page says **All systems operational**, shows **Connected**, and displays
the API version. The API response is also available at
[http://localhost:8080/api/health](http://localhost:8080/api/health), and
Adminer is at [http://localhost:8081](http://localhost:8081) with `postgres` as
the server name.

For diagnosis, inspect the printed log directory first. API startup or health
failures are in `backend.log` and `migrations.log`; frontend startup or browser
failures are in `frontend.log`; PostgreSQL readiness and container failures are
in `postgres-ready.log` and `compose-up.log`. Common causes are Docker not
running, a port already in use, a stale database volume, or `DATABASE_URL` not
matching the Compose credentials. Stop local containers after the check with:

```bash
docker compose down
```

## Production Deployment

- Backend: Render
- Frontend: Render Static Site
- Database: Supabase Postgres
- Auth: Supabase Auth (administrators only)

Published class and dismissal lists are served by the main API. SPEC §22.3 suggests they *should* be
servable independently of the administrative application; for v1 that is knowingly relaxed, with a
named revisit trigger — see
[ADR 0005](./docs/adr/0005-published-artifact-availability.md). Deployment is built in Phase 10.

## Contributing

See [`AGENTS.md`](./AGENTS.md) for the standing rules and [`WORKFLOW.md`](./WORKFLOW.md) for the
issue lifecycle.

1. Create a feature branch.
2. Make changes, citing the `SPEC.md` section they implement in the pull request.
3. Run the quality gates above.
4. Submit a PR referencing its issue.

## License

TBD
