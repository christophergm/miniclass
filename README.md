# Mini Class Planner

A class planning and assignment system for a school's Friday mini-class programme: roster ingest,
preference collection, constraint-based placement, and published class and dismissal lists.

## Documentation map

| Document | Answers |
|---|---|
| [`SPEC.md`](./SPEC.md) | **What the system does.** Normative, technology-agnostic. The source of truth for behaviour. |
| [`PLAN.md`](./PLAN.md) | **When it gets built, and in what order.** Phases, milestones, exit criteria. |
| [`docs/adr/`](./docs/adr/) | **Why it is built this way.** Architecture decisions, including the rejected alternatives. |
| [`QUICKSTART.md`](./QUICKSTART.md) | The shortest path from a fresh clone to a logged-in local stack. |
| [`WORKFLOW.md`](./WORKFLOW.md) | How agents pick up, validate and hand off work, and the nine quality gates. |
| [`AGENTS.md`](./AGENTS.md) | Repository rules that apply to every change. |
| This file | How to run, drive and troubleshoot it locally. |

If `SPEC.md` and any other document disagree, `SPEC.md` is right and the other document is a bug.

## Architecture

See [ADR 0001](./docs/adr/0001-application-stack-and-topology.md) for the full rationale.

**Stack:**
- Frontend: React + TypeScript + Vite, TanStack Query, Tailwind CSS v4 and shadcn/ui
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
├── Makefile              # The entry point: every command runs from here
├── backend/              # Go API server
│   ├── cmd/              # Entry points (api, migrate, seed, devtoken, openapi, bootstrap)
│   ├── internal/         # Application code
│   ├── migrations/       # Database migrations (Goose)
│   ├── sql/              # SQL queries (sqlc)
│   ├── scripts/          # Backend utility scripts and role provisioning SQL
│   └── tests/            # Integration tests
├── frontend/             # React application
│   ├── src/
│   │   ├── components/   # Shared UI components
│   │   ├── features/     # Feature modules
│   │   ├── lib/          # Utilities and API client
│   │   └── hooks/        # Custom React hooks
│   └── public/           # Static assets
├── scripts/              # setup, login and smoke-test scripts
├── docs/adr/             # Architecture decision records
├── .env.example          # The single source of local configuration defaults
└── compose.yaml          # Local development services (Docker Compose)
```

## Getting Started

Everything runs from the repository root through `make`. `make help` lists every command, grouped;
[`QUICKSTART.md`](./QUICKSTART.md) is the same path with less commentary.

### Prerequisites

| Tool | Used for |
|---|---|
| Docker with Compose v2, daemon running | PostgreSQL and Adminer |
| Go 1.26+ | the API, migrations, and the `cmd/*` tools |
| Bun 1.3+ | the frontend |
| `make`, `openssl`, `awk`, `curl`, `psql` | the setup, login, reset and smoke-test paths |

[proto](https://moonrepo.dev/proto) manages the Go, Node and Bun versions pinned in `.prototools`;
`proto install` is the one-command way to get them. `make setup` checks for the rest and names
whatever is missing, so this list does not need auditing by hand.

### First run

```sh
git clone https://github.com/christophergm/miniclass.git
cd miniclass

make setup             # .env, signing keys, bun install, PostgreSQL, migrations
make db-seed           # a synthetic organisation, its roster, and a bound Owner login
make tools-install     # air, sqlc, goose and golangci-lint, for hot reload and linting
```

`make setup` is idempotent. It copies `.env.example` to `.env` and **never overwrites an existing
one**; when `.env.example` has since gained a key, it reports that key by name rather than letting it
read as empty.

### Configuration

There is exactly one environment file, `.env` at the repository root — no `frontend/.env`; Vite's
`envDir` points here. It is read by GNU Make, by POSIX shell sourcing, by `godotenv` and by Docker
Compose, which is why one invariant holds:

> **No value in `.env` may contain whitespace or `#`.**

Only an unquoted value containing neither is read identically by all four. Anything that cannot
satisfy that — a PEM key, for instance — lives in a file and is referenced by path. `make setup`
generates the local ES256 signing keypair into `.secrets/`, which is gitignored explicitly, and
`.env` carries only `AUTH_LOCAL_PUBLIC_KEY_FILE` and `AUTH_LOCAL_PRIVATE_KEY_FILE`. Those paths are
relative to `backend/`, because every process that reads them runs from there.

`.env` defines one local identity, `DEV_ADMIN_EMAIL`. The seeded invitation, the token's `email`
claim and the provider subject are all derived from it, so they cannot disagree.

The reasoning behind all of this is
[ADR 0011](./docs/adr/0011-local-development-orchestration-and-environment-contract.md).

## Running Locally

MiniClass runs as **two long-lived processes, one per terminal**. Nothing supervises them, so each
hot-reloads independently and logs to its own terminal.

```sh
make dev-backend     # terminal 1: API on http://localhost:8080
make dev-frontend    # terminal 2: app on http://localhost:5173
```

`make dev` prints exactly those two lines and exits; there is no combined runner.

Their prerequisites differ because their needs differ. `dev-backend` starts PostgreSQL first and
needs no token, since it verifies whatever arrives. `dev-frontend` refreshes the development token
first, because Vite inlines `VITE_DEV_TOKEN` when it starts — which is also why a token minted after
Vite started does not take effect until Vite is restarted.

**Access:**

| Surface | URL |
|---|---|
| App | <http://localhost:5173> — signed in as `owner@example.test` in Synthetic Academy |
| API | <http://localhost:8080/api/health> — unauthenticated by design, so it answers when a login does not |
| Adminer | <http://localhost:8081> — server `postgres`, credentials from `.env`; start it with `docker compose up -d adminer` |

To check the whole stack, including the authenticated route, without touching the two terminals:

```sh
make smoke
```

`make smoke` starts its own throwaway API and Vite processes, so stop yours first or it will report
the ports as busy. It keeps logs in a temporary directory under `TMPDIR` and prints the path.

## Development Commands

`make help` is generated from this Makefile, so it cannot go stale. The groups are:

**Setup**

| Command | Does |
|---|---|
| `make help` | List every command, grouped |
| `make setup` | Prepare a checkout: `.env`, signing keys, `bun install`, PostgreSQL, migrations |
| `make tools-install` | Install the pinned Go tools (air, sqlc, goose, golangci-lint) |
| `make generate` | Regenerate the committed backend artifacts (`internal/db/gen`, `openapi.json`) |
| `make smoke` | Run the full-stack smoke test in throwaway processes |

**Database**

| Command | Does |
|---|---|
| `make db-up` | Start PostgreSQL and wait for it to be healthy |
| `make db-down` | Stop the local database services; the data volume survives |
| `make db-migrate` | Apply every pending migration |
| `make db-rollback` | Roll back the most recent migration |
| `make db-status` | Show which migrations are applied |
| `make db-migration-new NAME=add_widgets` | Create a timestamped migration; versions are never sequential |
| `make db-seed` | Create a fresh synthetic organisation and bind the local admin login |
| `make db-reset CONFIRM=1` | Drop the schema, migrate, seed, and refresh the login |

`make db-seed` creates a **fresh** organisation on every successful run — it is not a set of
repeatable fixtures — with a deterministic synthetic corpus of 139 students, and it claims the Owner
invitation for the local subject so that a usable login exists without anyone clicking a link. It
refuses to run a second time for the same subject, *before* writing anything, because two
memberships for one subject resolve to no tenant at all. The remedy it names is
`make db-reset CONFIRM=1`.

`make db-reset` refuses without `CONFIRM=1`, since it destroys every row in `DATABASE_URL`. Stop the
API before running it: a running API pools connections to the schema the reset replaces. Use
`make db-migrate` when an empty migrated database is what you want.

Test and seed data is generated with synthetic names. Never load real roster data into a development
or test database.

**Development**

| Command | Does |
|---|---|
| `make dev` | Print how to run the two development processes |
| `make dev-backend` | Run the API with hot reload; starts PostgreSQL first |
| `make dev-frontend` | Run the Vite dev server with hot reload; refreshes the token first |
| `make token-mint` | Refresh `VITE_DEV_TOKEN` in `.env` when it is stale (`FORCE=1` always mints) |

`make token-mint` re-mints only when the token is absent, unreadable, or expiring within 24 hours,
and says which applied. Local tokens last 30 days: they are signed by a key on your own disk, so
their lifetime is not a security boundary, and a five-minute token teaches you to distrust every
authentication failure. If the API is running, the target then calls `GET /api/me` and tells you
whether the token resolves to a principal.

**Quality gates**

| Command | Does |
|---|---|
| `make test` | Run both test suites |
| `make test-backend` | Run the Go unit and integration tests |
| `make test-frontend` | Run the frontend tests once |
| `make test-migrations` | Apply, roll back, and reapply every migration on a scratch database |
| `make lint` | Lint both components |
| `make lint-backend` | Run golangci-lint and the depguard boundary proof |
| `make lint-frontend` | Run ESLint |
| `make format` | Check Go formatting and run the vet analyzer |
| `make build-frontend` | Type-check and build the production frontend bundle |
| `make check` | Run all nine CI gates in CI order, failing fast |

`make format`, `make lint` and `make test` are the fast loop during work. `make check` is what to run
before pushing: it reproduces the nine checks CI publishes, in CI order, and each failure names the
CI check it maps to, for example:

```
FAILED: Generated code drift (CI check: "Generated code drift") — run 'make generate' and commit the result
```

The gate list, with the single-gate command for each, is in [`WORKFLOW.md`](./WORKFLOW.md). No gate
installs dependencies: `make setup` owns `bun install` and `make tools-install` owns the Go tools, so
a lockfile change is always deliberate. Backend tests use the `miniclass_test` database created by
Docker Compose and create a unique schema per run, which they drop on cleanup;
`make test-migrations` creates and drops its own scratch database. Neither touches your development
data.

Where a root command is not enough — a single Go package, one Vitest file — `backend/Makefile` and
`frontend/package.json` are the implementations, and their names are unchanged.

## Ports

All ports are configurable in `.env` so that parallel worktrees can run simultaneously.

| Service | Default | Environment variable |
|---|---|---|
| Frontend (Vite) | 5173 | `VITE_PORT` |
| Backend (Go API) | 8080 | `PORT` |
| PostgreSQL | 5432 | `POSTGRES_PORT` |
| Adminer | 8081 | `ADMINER_PORT` |

The browser reaches the API through the Vite dev proxy, so `VITE_API_URL` — the client bundle's API
base — is **empty** locally and requests are same-origin relative `/api` calls. `API_PROXY_TARGET` is
the node-side proxy target and deliberately carries no `VITE_` prefix, so Vite cannot expose it to the
browser. Setting `VITE_API_URL` locally breaks the Content-Security-Policy in `frontend/index.html`,
which allows `connect-src 'self'` only.

For deployments behind a reverse proxy, set `TRUSTED_PROXY_CIDRS` to a comma-separated list of the
proxy networks. Forwarded client-IP headers are ignored unless the connecting peer belongs to one of
those networks; leave it empty when the API is directly exposed.

## Parallel Development (Multiple Worktrees)

1. Create a new worktree.
2. Run `make setup` in it, which creates that worktree's own `.env` and signing keys.
3. Edit `.env` to use different ports from the table above, and a different `POSTGRES_DB`.
4. Run `make db-up` and the two development processes as usual.

`.prototools` is read from the current directory, so each worktree may pin its own tool versions.

## Troubleshooting

Start with `GET /api/health`: it is unauthenticated by design, so it answers even when your login is
broken. If it reports `healthy` and `connected`, the problem is identity rather than the stack.

```sh
curl http://localhost:8080/api/health
# {"status":"healthy","timestamp":"2026-08-22T10:30:00Z","database":"connected","version":"0.1.0"}
```

| Symptom | Cause | Fix |
|---|---|---|
| A red "Local development authentication has no token" banner | `VITE_DEV_TOKEN` is absent or expired | `make token-mint`, then restart `make dev-frontend` |
| The banner persists after minting a token | `VITE_*` values are inlined when Vite starts | restart `make dev-frontend` |
| `403 no-organization` | the token's subject is bound to no membership | `make db-seed` |
| `409 multiple-organizations` | the subject was bound twice | `make db-reset CONFIRM=1` |
| `401 invalid-token` | the token expired, or `.secrets/` was regenerated after it was minted | `make token-mint FORCE=1` |
| `make db-seed` refuses immediately | the subject already has a membership | `make db-reset CONFIRM=1` |
| `permission denied for table organizations` | the database predates the migrator-owned schema | `make db-reset CONFIRM=1` |
| `sh: EC: command not found`, or any `command not found` naming part of a value | a `.env` value contains whitespace | move the value into `.secrets/` and reference its path; see the invariant at the top of `.env.example` |
| A variable reads as empty and nothing else explains it | `.env.example` gained a key after your `.env` was copied | `make setup` names every missing key |
| `MIGRATION_ROUNDTRIP_DATABASE_URL is unset` | the same, for the migration round-trip gate | add the `MIGRATION_ROUNDTRIP_*` keys from `.env.example` |
| `connection refused`, or `database does not exist` | PostgreSQL is not running | `make db-up`; confirm the Docker daemon is running |
| `address already in use` | a previous API, Vite or smoke-test process is still running | stop it, or change `PORT` / `VITE_PORT` in `.env` |
| A migration fails after a schema change | the migration itself, or a schema that drifted | `make db-status`, fix the migration, then `make db-reset CONFIRM=1`. Never edit the Goose version table, and never edit a merged migration |
| `air: command not found` | `air` is not a proto plugin, so it is not in `.prototools` | `make tools-install` |
| `golangci-lint … is required` | the pinned version is not installed | `make tools-install` |
| `sqlc … is required` | a different `sqlc` is on `PATH`, and its version is written into every generated file | `make tools-install`; never commit generated files produced by an unpinned `sqlc` |
| TypeScript errors about modern syntax | an ancient global `tsc` is shadowing the project's TypeScript | nothing: every `frontend/package.json` script uses `bunx --bun` to force resolution from `node_modules`. Do not remove it |
| A frontend import or type is missing after pulling | dependencies are behind the lockfile | `make setup` |
| `command not found: docker-compose` | expected | use `docker compose`, v2, with a space |

A local Compose volume survives `make db-down` and `make db-reset CONFIRM=1`. To discard the database
itself, `docker compose down -v` removes the volume and everything in it, after which `make setup`
recreates the roles and schema.

## Production Deployment

- Backend: Render
- Frontend: Render Static Site
- Database: Supabase Postgres
- Auth: Supabase Auth (administrators only)

`AUTH_PROVIDER=local` is refused outright when `APP_ENV=production`, so the local signing keypair
cannot become a production credential.

Published class and dismissal lists are served by the main API. SPEC §22.3 suggests they *should* be
servable independently of the administrative application; for v1 that is knowingly relaxed, with a
named revisit trigger — see
[ADR 0005](./docs/adr/0005-published-artifact-availability.md). Deployment is built in Phase 10.

## Contributing

See [`AGENTS.md`](./AGENTS.md) for the standing rules and [`WORKFLOW.md`](./WORKFLOW.md) for the
issue lifecycle.

1. Create a feature branch.
2. Make changes, citing the `SPEC.md` section they implement in the pull request. Developer tooling
   cites an ADR instead, because `SPEC.md` has no tooling section.
3. Run `make check`.
4. Submit a PR referencing its issue.

## License

TBD
