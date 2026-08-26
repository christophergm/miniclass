# Quickstart

From a fresh clone to a running stack with a logged-in local administrator, in four commands.

For *why* any of this is shaped the way it is, read
[ADR 0011](./docs/adr/0011-local-development-orchestration-and-environment-contract.md). This file is
the ordered path; it deliberately explains nothing that the ADR already argues.

## Prerequisites

| Tool | Used for |
| --- | --- |
| Docker with Compose, daemon running | PostgreSQL and Adminer |
| Go 1.26+ | the API, migrations, and the `cmd/*` tools |
| Bun 1.3+ | the frontend |
| `openssl`, `awk`, `curl`, `psql`, `make` | the setup, login and reset scripts |

`make setup` checks for most of these and names whatever is missing, so you do not need to audit
this list by hand.

## The four commands

```sh
git clone https://github.com/christophergm/miniclass.git
cd miniclass

make setup   # .env, signing keys, bun install, PostgreSQL, migrations
make seed    # a synthetic organisation, its roster, and a bound Owner login
make login   # a 30-day bearer token for that Owner, written into .env
```

Then run the two long-lived processes, one per terminal:

```sh
make -C backend dev          # API on http://localhost:8080
cd frontend && bun run dev   # app on http://localhost:5173
```

Open <http://localhost:5173>. You are signed in as `owner@example.test` in **Synthetic Academy**,
with the Owner role and a roster of 139 synthetic students.

Verify the whole stack, including the authenticated route, with:

```sh
./scripts/smoke-test.sh
```

## What each command actually does

**`make setup`** copies `.env.example` to `.env` (never overwriting an existing one), generates an
ES256 keypair into a gitignored `.secrets/`, installs frontend dependencies, starts PostgreSQL, and
applies migrations. It is idempotent, and it reports keys that `.env.example` has gained since your
`.env` was copied rather than letting them read as empty.

**`make seed`** creates a fresh organisation with the deterministic synthetic corpus, issues its
Owner invitation, and immediately claims that invitation for the local provider subject — so a
usable login exists without anyone clicking a link. It refuses to run twice for the same subject,
*before* writing anything, because a second organisation would give that subject two memberships and
break the login it already had.

**`make login`** mints a bearer token for the same subject and writes it into `.env` as
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
`make reset CONFIRM=1`; do not change it in only one place.

`.test` is a reserved TLD, and the roster is generated, so nothing here can reach a real person.
Never load real roster data into a local database.

## Resetting

```sh
make reset CONFIRM=1
```

Drops and rebuilds the schema, re-applies migrations, re-seeds, and mints a fresh token. This is the
remedy `make seed` names when it refuses a second run.

## When something is wrong

| Symptom | Cause | Fix |
| --- | --- | --- |
| The app shows a red "Local development authentication has no token" banner | `VITE_DEV_TOKEN` is absent or expired | `make login`, then **restart the Vite dev server** |
| You ran `make login` and the app still shows the banner | `VITE_*` values are inlined when Vite starts | restart `bun run dev` |
| `403 no-organization` from the API | the token's subject is not bound to a membership | `make seed` |
| `409 multiple-organizations` | the subject was bound twice | `make reset CONFIRM=1` |
| `401 invalid-token` | the token is expired, or `.secrets/` was regenerated after it was minted | `./scripts/login.sh --force` |
| `make seed` refuses immediately | the subject already has a membership | `make reset CONFIRM=1`, or seed an unclaimed invitation with `make -C backend seed SEED_OWNER_SUBJECT=` |
| `permission denied for table organizations` | the database predates the migrator-owned schema | `make reset CONFIRM=1` |
| `sh: EC: command not found` from a script | a `.env` value contains whitespace | see the invariant at the top of `.env.example` |
| `address already in use` | a previous API or Vite process is still running | stop it, or change `PORT` / `VITE_PORT` in `.env` |

`GET /api/health` is unauthenticated by design, so it answers even when your login is broken. If it
reports `healthy` and `connected`, the problem is identity, not the stack — start at the table above.

Adminer, for looking at the data directly, is at <http://localhost:8081> (server `postgres`, the
credentials in `.env`); start it with `docker compose up -d adminer`.
