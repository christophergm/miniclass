# Quickstart

From a fresh clone to a running stack with a logged-in local administrator: three commands, then two
terminals. Every command runs from the repository root; `make help` lists the rest.

For *why* any of this is shaped the way it is, read
[ADR 0011](./docs/adr/0011-local-development-orchestration-and-environment-contract.md). This file is
the ordered path; it deliberately explains nothing that the ADR already argues.

## Prerequisites

| Tool | Used for |
| --- | --- |
| Docker with Compose, daemon running | PostgreSQL and Adminer |
| Go 1.26+ | the API, migrations, and the `cmd/*` tools |
| Bun 1.3+ | the frontend |
| `make`, `openssl`, `awk`, `curl`, `psql` | the setup, login and reset paths |

`make setup` checks for most of these and names whatever is missing, so you do not need to audit
this list by hand.

## The three commands

```sh
git clone https://github.com/christophergm/miniclass.git
cd miniclass

make setup         # .env, signing keys, bun install, PostgreSQL, migrations
make db-seed       # a synthetic organisation, its roster, and a bound Owner login
make tools-install # air, sqlc, goose and golangci-lint
```

Then run the two long-lived processes, one per terminal:

```sh
make dev-backend    # API on http://localhost:8080
make dev-frontend   # app on http://localhost:5173
```

`make dev-frontend` mints the development bearer token first, so there is no separate login step;
`make token-mint` is that step on its own when you want it.

Open <http://localhost:5173>. You are signed in as `owner@example.test` in **Synthetic Academy**,
with the Owner role and a roster of 139 synthetic students.

Verify the whole stack, including the invitation claim and authenticated routes, with:

```sh
make smoke
```

It starts its own API and Vite processes, so stop yours first. The check invites a synthetic
administrator when the Owner created by `make db-seed` is already bound.

## What each command actually does

**`make setup`** copies `.env.example` to `.env` (never overwriting an existing one), generates an
ES256 keypair into a gitignored `.secrets/`, installs frontend dependencies, starts PostgreSQL, and
applies migrations. It is idempotent, and it reports keys that `.env.example` has gained since your
`.env` was copied rather than letting them read as empty.

**`make db-seed`** creates a fresh organisation with the deterministic synthetic corpus, issues its
Owner invitation, and immediately claims that invitation for the local provider subject — so a
usable login exists without anyone clicking a link. It refuses to run twice for the same subject,
*before* writing anything, because a second organisation would give that subject two memberships and
break the login it already had.

**`make token-mint`** mints a bearer token for the same subject and writes it into `.env` as
`VITE_DEV_TOKEN`. It re-mints only when the existing token is missing, unreadable, or expiring
within 24 hours, and it prints which of those applied. If the API is running it then calls
`GET /api/me` and tells you whether the token actually resolves to a principal.

### One identity, derived not configured

`.env` defines a single address:

```
DEV_ADMIN_EMAIL=owner@example.test
```

Everything else is derived from it — the seed's invited email, the token's `email` claim, and the
provider subject `local:owner@example.test`. Claiming an invitation compares the invited address
against the verified token address, so if those were configured separately anywhere they would
eventually disagree, the claim would be refused, and you would be left with a token that
authenticates and an account that does not exist. Change the address in `.env` and re-run
`make db-reset CONFIRM=1`; do not change it in only one place.

`.test` is a reserved TLD, and the roster is generated, so nothing here can reach a real person.
Never load real roster data into a local database.

## Resetting

```sh
make db-reset CONFIRM=1
```

Drops and rebuilds the schema, re-applies migrations, re-seeds, and refreshes the token. This is the
remedy `make db-seed` names when it refuses a second run. Stop the API first: its connection pool
outlives the schema the reset replaces.

## When something is wrong

| Symptom | Cause | Fix |
| --- | --- | --- |
| The app shows a red "Local development authentication has no token" banner | `VITE_DEV_TOKEN` is absent or expired | `make token-mint`, then **restart `make dev-frontend`** |
| You minted a token and the app still shows the banner | `VITE_*` values are inlined when Vite starts | restart `make dev-frontend` |
| `403 no-organization` from the API | the token's subject is not bound to a membership | `make db-seed` |
| `409 multiple-organizations` | the subject was bound twice | `make db-reset CONFIRM=1` |
| `401 invalid-token` | the token is expired, or `.secrets/` was regenerated after it was minted | `make token-mint FORCE=1` |
| `make db-seed` refuses immediately | the subject already has a membership — `make db-reset CONFIRM=1` leaves one, because it seeds | nothing after a reset; otherwise `make db-reset CONFIRM=1`, or seed an unclaimed invitation with `make db-seed SEED_OWNER_SUBJECT=` |
| `permission denied for table organizations` | the database predates the migrator-owned schema | `make db-reset CONFIRM=1` |
| `sh: EC: command not found` from a script | a `.env` value contains whitespace | see the invariant at the top of `.env.example` |
| `address already in use` | a previous API or Vite process is still running | stop it, or change `PORT` / `VITE_PORT` in `.env` |

`GET /api/health` is unauthenticated by design, so it answers even when your login is broken. If it
reports `healthy` and `connected`, the problem is identity, not the stack — start at the table above.

Adminer, for looking at the data directly, is at <http://localhost:8081> (server `postgres`, the
credentials in `.env`); start it with `docker compose up -d adminer`.

For everything else — the full command surface, the quality gates, and a longer troubleshooting
table — see [`README.md`](./README.md).
