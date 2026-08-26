# 11. Local development orchestration and environment contract

- **Status:** Accepted
- **Date:** 2026-08-25
- **Implements:** developer tooling. `SPEC.md` describes the product and has no tooling section, so
  this record is the citation for DX work, per standing rule 1 in [`AGENTS.md`](../../AGENTS.md).
- **Supersedes:** `scripts/dev.sh` (removed)
- **Related:** [0007](./0007-tenancy-enforcement-and-data-access.md) — the migrator/app role
  separation that a local `DATABASE_URL` must respect,
  [0002](./0002-authentication-and-access-mechanisms.md),
  [0009](./0009-administrator-sessions-and-identity-provider.md)

## Context

Running MiniClass locally needs PostgreSQL with three roles, applied migrations, an ES256 keypair for
the local token verifier, a Go API and a Vite dev server. Roughly thirty environment variables tie
those together, and the file that holds them is read by **three parsers that disagree about
quoting**:

| Consumer | Unquoted value containing spaces | Quoted value |
| --- | --- | --- |
| GNU Make `-include ../.env` (`backend/Makefile`) | Fine | Quotes become **part of the value** |
| POSIX `. .env` (`scripts/*.sh`, and any shell) | **Breaks** | Fine |
| `godotenv` (`internal/config.loadDotEnv`) | Fine | Fine (strips) |

No quoting style satisfies Make and a POSIX shell simultaneously. Nothing in the repository stated
that constraint, so it was violated the first time a value needed a space. `scripts/dev.sh`
synthesised a `.env` with the signing keys written inline and unquoted, and
`-----BEGIN EC PRIVATE KEY-----` contains spaces:

```
$ ( set -a; . ./.env; set +a )
sh: EC: command not found
sh: PUBLIC: command not found
```

Under `scripts/smoke-test.sh`'s `set -Eeuo pipefail` that is fatal, so the smoke test was broken on
every machine whose `.env` came from `scripts/dev.sh` — and the error names neither the file nor the
variable, so the cause is not discoverable from the symptom.

Four further defects sat in the same script, each of the same kind: a local default that silently
disagreed with the committed contract.

- It **synthesised** `.env` from a hardcoded list instead of copying `.env.example`, producing 8 keys
  where the example has about 30. `TEST_DATABASE_URL`, `TEST_APP_DATABASE_URL`,
  `POSTGRES_ADMIN_DATABASE_URL`, `VITE_PORT`, `APP_ENV` and `ADMINER_PORT` were all absent, so
  `cd backend && make test` could not work from it.
- It pointed `DATABASE_URL` at the **superuser** `miniclass` role rather than `miniclass_migrator`,
  quietly bypassing the role separation that [ADR 0007](./0007-tenancy-enforcement-and-data-access.md)
  exists to enforce. Row-level security is `nobypassrls` on `miniclass_app`; a superuser connection
  makes the whole tenancy guard advisory.
- It wrote the keypair into `backend/scripts/`, a tracked source directory holding committed SQL,
  saved from commit only by a blanket `*.pem` rule in `.gitignore`.
- It set the browser's API base URL to `http://localhost:8080`, so the client bypassed the dev proxy
  already configured in `vite.config.ts` and called the API cross-origin. That is the only reason
  `frontend/index.html` needed `connect-src http://localhost:8080`. The proximate cause is that
  `VITE_API_URL` had **two different consumers under one name**: `process.env.VITE_API_URL` was the
  node-side proxy target in `vite.config.ts`, while `import.meta.env.VITE_API_URL` was the client
  bundle's API base.

The common thread is that the environment had no stated contract, so every consumer invented one.

## Decision

### One root `.env`, created by copying `.env.example`

There is exactly one environment file, at the repository root. It is created by
`cp .env.example .env` and **never synthesised**.

`frontend/.env` is not used, and that is structural rather than conventional: `vite.config.ts` sets
`envDir` to the repository root, so a stray `frontend/.env` is never read at all. Relying on Vite's
rule that an existing `process.env` value beats a `.env` file would work only for keys the root file
also defines, which makes "there is one environment file" true by luck.

Copying rather than generating is the whole point: a key cannot be omitted from a byte copy. It also
makes `.env.example` and `.env` comparable, which is what lets `scripts/setup.sh` report keys that
exist in the example and are missing from a developer's older `.env` — the failure mode where a new
variable reads as empty rather than as an error.

`.env.example` is the single source of local defaults, including `DATABASE_URL` on
`miniclass_migrator`. No script may restate a default that belongs in the example.

### Invariant: no `.env` value may contain whitespace or `#`

This is the constraint that makes one file safe for all three parsers, and it is stated as a header
comment in `.env.example` rather than left to be rediscovered. `#` is included because Make treats it
as starting a comment and a shell, mid-value, does not.

The invariant is **enforced, not merely documented**: `load_env` in `scripts/lib.sh` validates a file
before sourcing it and reports the offending line and variable by name. That converts
`sh: EC: command not found` into a diagnosis. `scripts/setup.sh` additionally refuses to copy an
`.env.example` that violates the invariant, and warns about an existing `.env` that does.

A fourth parser turns out to be in play — Docker Compose's own `--env-file` reader, which does handle
quotes. It is compatible with the invariant, which is a useful demonstration that the invariant is the
intersection of all reasonable dotenv dialects rather than an artifact of these three.

### Signing keys are files, in a gitignored `.secrets/`

A PEM key cannot satisfy the invariant, so it does not live in `.env`. `scripts/setup.sh` generates
the ES256 keypair into `.secrets/` at the repository root, and `.env` carries paths only.
`internal/config` gains `AUTH_LOCAL_PUBLIC_KEY_FILE` and `AUTH_LOCAL_PRIVATE_KEY_FILE`; the existing
inline `AUTH_LOCAL_PUBLIC_KEY` and `AUTH_LOCAL_PRIVATE_KEY` are retained as a fallback, because a
deployment target that injects secrets as environment variables and has no writable filesystem is a
real shape. `cmd/devtoken` already accepted `AUTH_LOCAL_PRIVATE_KEY_FILE`.

`.secrets/` is listed in `.gitignore` **explicitly**, not left to `*.pem`, so that adding a secret in
some other format cannot quietly become committable.

**A relative key path resolves against the process working directory**, and every process that reads
one runs from `backend/`, so the committed values are `../.secrets/local_auth_private.pem`. This reads
oddly in a root-level file and is the least satisfying part of this record. It is accepted because the
alternatives are worse (below), and mitigated by `internal/config` reporting the **absolute** resolved
path when a key file is missing, so a wrong working directory says so instead of looking like a
missing key.

### `API_PROXY_TARGET` is the proxy target; `VITE_API_URL` is the client's

The node-side dev-proxy target becomes `API_PROXY_TARGET`, leaving `VITE_API_URL` unambiguously the
client bundle's API base. `API_PROXY_TARGET` has no `VITE_` prefix precisely so that Vite cannot
expose it to the browser.

`VITE_API_URL` is **empty in local development**. The browser therefore requests a relative `/api`,
which is same-origin on the Vite port and reaches the API through the proxy that was already
configured. Two things follow: the CSP's `connect-src` returns to `'self'`, and local development
stops differing from production in its request origin. `scripts/smoke-test.sh` asserts the proxied
path (`GET /api/health` on the Vite origin), so the claim behind `connect-src 'self'` is checked
rather than asserted.

`style-src 'unsafe-inline'` is retained, having been re-verified rather than assumed: Vite 5 delivers
**all** dev CSS by creating a `style` element and assigning `textContent` (`updateStyle` in
`vite/dist/client/client.mjs`), and its HMR error overlay injects its own stylesheet. The production
build needs neither — CSS ships as an external stylesheet and no component uses inline styles — but
`index.html` is the shared entry template and the CSP is a static `<meta>`, so removing it would break
development. A nonce is the principled fix and requires server-rendered HTML; it is not taken here.
`frame-ancestors` stays absent, being genuinely ignored in a `<meta>`-delivered policy.

### The two-terminal development model

MiniClass runs as two long-lived processes with independent hot reload: `make -C backend dev` and
`cd frontend && bun run dev`. Nothing supervises them.

`frontend/package.json`'s `dev` script sources `../.env` itself when present. `envDir` already gives
Vite the `VITE_*` values; the sourcing is for the two things `envDir` cannot reach — `VITE_PORT`,
which the script itself interpolates, and `API_PROXY_TARGET`, which is deliberately unprefixed so that
Vite cannot expose it to the browser. The accepted cost is that this sourcing **overrides** an ambient
exported value, where `godotenv`, Make's `export` and Vite's own precedence do not: doing otherwise
means hand-rolling a dotenv parser inside a `package.json` string. Scripts that need to override
therefore pass values on the command line, and the script forwards its own arguments through to Vite
so that `bun run dev -- --host 127.0.0.1` still works.

`scripts/setup.sh` performs everything one-shot and idempotent — verify prerequisites, create `.env`
if absent, generate keys if absent, `bun install`, start PostgreSQL, wait for readiness, apply
migrations — and **never overwrites an existing `.env`**. `scripts/lib.sh` holds the helpers both
scripts need (`require_command`, `load_env`, `wait_for_postgres`, `log_dir`), so the smoke test and
setup cannot drift into two different notions of "PostgreSQL is ready".

Local tokens are minted with a **30-day** lifetime. `cmd/devtoken` defaults to five minutes, which is
right for a test and wrong for a person, who otherwise re-mints a token several times an hour and
learns to distrust every authentication failure. A local token is signed by a key on the developer's
own disk, so its lifetime is not a security boundary. Seed and token behaviour is DX-3, which
implements this as `scripts/login.sh`: the lifetime stays a caller's argument rather than becoming
`cmd/devtoken`'s default, since the five-minute default is the correct one for a test.

### `make check` is the CI equivalent

A single root target runs what CI runs, so "it passes locally" and "it passes in CI" are the same
claim. The root `Makefile` remains near-empty until DX-4, which owns it along with `README.md` and
`WORKFLOW.md`; CI is DX-5.

## Alternatives considered

**Quoted values in `.env`.** Rejected: quotes become part of the value under Make's `include`, so
`DATABASE_URL` would carry literal quotes into the connection string. Fixing that means dropping
`-include ../.env` from `backend/Makefile`, which is what makes `cd backend && make test` work from a
single file.

**Two files with a generator** — a canonical source plus a shell-safe and a Make-safe rendering.
Rejected: it reintroduces synthesis, which is the defect this record exists to remove, and it makes
"which file do I edit" a question. The invariant is one sentence and costs nothing once stated.

**Keys inline, plus dropping shell sourcing.** Rejected: sourcing `.env` is how `scripts/*.sh` and any
ad-hoc `psql` invocation get their configuration, and forbidding it forever to accommodate one value
inverts the cost.

**Keys inline, plus dropping Make's `include`.** Rejected for the same reason in the other direction:
it would break `cd backend && make test` from a single file, which is an explicit acceptance criterion
for this work.

**Absolute key paths, written into `.env` by `setup.sh` during the copy.** This removes the
working-directory subtlety entirely and was close to being chosen. Rejected because it makes `.env` a
substitution of `.env.example` rather than a copy of it, which weakens the "never synthesised" rule to
"mostly not synthesised", and makes `.env` machine-specific so it can no longer be diffed against the
example. The `setup.sh` refusal to write a value containing whitespace loses its main live case as a
result, and is kept anyway as a guard on `.env.example` itself.

**`internal/config` locating the repository root** by walking up for `.git` or `.env.example`, so key
paths could be root-relative. Rejected: a search that silently succeeds against the wrong directory is
worse than an explicit path that fails loudly, and it stops working outside a checkout.

**A root `Makefile` as the sole entry point**, wrapping both dev processes. Rejected in favour of
per-component commands plus a thin root target: two hot-reloading watchers multiplexed by Make give
interleaved output, one shared exit status and a Ctrl-C that leaves orphans — which is exactly what the
commented-out background-and-tail block in the removed `scripts/dev.sh` had already discovered.

**A task runner** (Just, Task, mise). Rejected: another prerequisite to install and another syntax to
learn, to replace a Makefile that already works. Make is present on every developer and CI machine
this project targets.

**Keeping `scripts/dev.sh`.** Rejected: its `.env` synthesis is the root cause, and once `.env` is a
copy, keys are files and the two processes run in two terminals, nothing is left for it to do.

## Consequences

- The smoke test works on a machine set up by `scripts/setup.sh`, and fails with a named variable
  rather than `sh: EC: command not found` on one that was not.
- A local `DATABASE_URL` now runs as `miniclass_migrator`, so row-level security is actually in force
  locally and an isolation defect can surface in development instead of first in CI.
- Adding a variable means editing `.env.example`, and every developer's existing `.env` is then
  reported as missing it rather than silently reading it as empty.
- **Any value that needs a space must become a file.** That is a real constraint on future
  configuration and the reason it is stated where it will be read.
- `../.secrets/…` in a root-level `.env` looks wrong until you know that backend commands run from
  `backend/`. This is the residual cost of the invariant; the missing-file error names the absolute
  path to make it self-diagnosing.
- The frontend `dev` script's `.env` sourcing overrides the ambient environment, unlike every other
  consumer. A wrapper that needs to override must pass values as arguments.
- `.secrets/` is unencrypted key material on disk. Acceptable for a local development keypair that
  signs nothing outside the developer's machine, and the reason `AUTH_PROVIDER=local` is refused
  outright when `APP_ENV=production`.
- Nothing yet enforces the invariant at review time; it is enforced when a script loads the file.
  A CI check belongs with DX-5.
- Two defects were found by running the smoke test against a correct `.env` for the first time, and
  were left for their owning issue rather than fixed here. `GET /api/health` was registered with
  `CapabilityAuthenticated` and no unresolved-principal exemption, so the smoke test could not reach
  it without seeded identity data; and `cmd/seed` failed with `permission denied for table
  organizations`. Both were resolved in DX-3. Health now declares `CapabilityPublic`, a capability
  that exists so that "no authentication" is something an operation must state rather than something
  it can achieve by omission. The permission failure turned out not to be a property of the
  `miniclass_migrator` role at all: it reproduces only on a database whose objects were created by
  the superuser under the pre-this-ADR `DATABASE_URL`, leaving the migrator with neither ownership
  nor grants. On a database created by `scripts/setup.sh` the tables are migrator-owned and the seed
  succeeds, so the remedy is `make reset CONFIRM=1` rather than a code change.
