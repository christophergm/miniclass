## Issue effort selection

Every issue created for this repository must include an explicit reasoning effort override:

```detent-agent
schema: 1
effort: high
```

Choose the effort from this project-specific rubric:

- `medium` — Small mechanical work with exact acceptance criteria.
- `high` — Standard features and fixes with some ambiguity or cross-cutting impact.
- `xhigh` — New subsystems or tricky state, concurrency, restart, recovery, or interaction work.
- `max` — Exceptional operator-designated work that must never be selected automatically.

Leave `model` unset so the issue inherits the fleet-standard model.

## Standing rules

1. **Cite the spec.** Every pull request names the `SPEC.md` section it implements. Behaviour with no
   spec citation is either undiscovered scope or invention; both need a human. A change that is
   purely developer tooling — build, scripts, local environment, CI — may cite an ADR instead, because
   `SPEC.md` describes the product and has no tooling section.
2. **No tenant-scoped table without an isolation test.** SPEC §9.2 states that omitting one is a
   defect, not an oversight.
3. **Never weaken a test to make CI green.** If a test is wrong, fix the test in its own change with
   its own justification.
4. **Warn, do not block.** SPEC §5.2 is pervasive. Any new validation that refuses an organiser
   action needs an explicit spec citation permitting it.
5. **Judgement is data.** SPEC §5.4. When a person overrides the system, record who, when, and why —
   never silently accept and never merely permit.
6. **Sensitivity is enforced at render time**, in every surface including exports and print views —
   never at query time only.
7. **Names are never keys.** Every join is on an opaque identifier (SPEC §8.7).
8. **Out-of-scope discoveries become tracker issues**, not scope creep in the current change.

## Data access

Rules in this section exist to make SPEC §9.2's tenancy guard unbypassable rather than merely
conventional. The reasoning is in
[ADR 0007](./docs/adr/0007-tenancy-enforcement-and-data-access.md); do not restate it here.

- **All database access goes through `internal/data`.** The generated sqlc package `internal/db/gen`
  is imported nowhere else. *Enforced by `Backend lint` (`depguard`).*
- **The un-scoped accessor `internal/data/identity` is imported only by `internal/identity`.** It
  reaches the four identity tables and cannot reach the domain. *Enforced by `Backend lint`
  (`depguard`).*
- **Writes go through `InTenant` and must record an audit entry**, or call
  `NoAuditRequired(reason)`. A read-write transaction with neither does not commit. Reads use
  `InTenantRead`. *Enforced by the unit-of-work commit check and the entity registry tests.*
- **A new tenant-scoped table registers a factory** in its own file under the entity registry.
  *Enforced by the schema meta-test, which fails if any tenant-scoped table has no entry.*
- **A new endpoint is declared with `huma.Register` and declares its required capability.**
  *Enforced by the operation-enumeration test and `Generated code drift`.*

## Identifier and SQL standards

- Use the PostgreSQL public.xid20 domain with public.xid() as the default for
  every new application-generated identifier. Do not introduce UUID or
  sequential integer identifiers for new application entities unless the
  issue explicitly requires an external identifier or a non-entity key.
- Keep the xid domain, generator, and helpers in the public schema. Do not name
  the domain xid: pg_catalog is resolved ahead of the search_path, so a column
  declared as an unqualified xid silently becomes the built-in 4-byte
  transaction-id type no matter which schema the domain lives in.
- Schema-qualify the identifier API (public.xid20, public.xid()) in migrations
  and queries.
- Use lowercase SQL keywords, identifiers, function names, and migration
  statements. Keep SQL formatting readable, and use quoted mixed-case names
  only when integrating with an external schema that cannot be changed.
- An xid is opaque but **not unguessable** — it encodes a timestamp, machine id, process id and
  counter. Entity identifiers may appear in URLs, because access control there is authentication plus
  row-level security. Share-link tokens (SPEC §9.5), where obscurity *is* the control, must be
  high-entropy random values stored hashed, and must never be an xid or derived from one.

## Generated code and migrations

- **Committed:** the sqlc output in `internal/db/gen`, and `openapi.json`. **Not committed:** the
  frontend's TypeScript API types, which are generated during the frontend build.
- **Never hand-merge a generated file.** Resolve a conflict in `internal/db/gen` or `openapi.json`
  by regenerating. A hand-merged generated file usually looks correct and is not.
- **Migration versions are timestamped**, never sequential. Two agents both creating `00002_…`
  produces no git conflict and then breaks at apply time.
- **Never edit a merged migration.** Corrections are new migrations.

## Test conventions

These two have no automated enforcement, which is why they are called out rather than assumed.

- **Never assert on a global count or an empty table.** The suite shares one database and isolates
  tests by organisation, so another test's rows are present. Scope every count and every listing
  assertion to the organisation the test created.
- **Never load real roster data into a development or test database.** Test and seed data is
  generated with synthetic names. The historical exports contain 139 real children.
