# 10. Schema, generated code and migration conventions

- **Status:** Accepted
- **Date:** 2026-08-23
- **Implements:** SPEC §8.7, §20.1, §21.3
- **Related:** [0004](./0004-api-contract-and-type-generation.md),
  [0007](./0007-tenancy-enforcement-and-data-access.md)

## Context

The specification implies roughly thirty tables across ten phases. Almost all of them will be written
by agents in parallel worktrees that cannot see each other. Conventions applied thirty times are
cheap to establish and expensive to change, and conventions that differ between tables are how a
codebase stops being predictable enough for an agent to extend safely.

Two collision modes are specific to concurrent authorship and do not exist in a single-author
project: migration version collisions, and merge conflicts in generated files.

## Decision

### Schema

**Identifiers.** `uuid PRIMARY KEY DEFAULT uuidv7()`, using PostgreSQL 18's native function.
Time-ordered so it clusters well, opaque per §8.7, exposed raw in the API. No slugs and no secondary
public identifiers. §8.7's rule that "nothing in the system MAY use a person's name as a key" is
absolute; every join is on an opaque identifier.

**Closed sets live wherever they are most naturally single-sourced.**

- **A PostgreSQL enum type when the specification fixes the set** — `school_year_state`,
  `organization_role`, `guardian_relationship_type`, `participation_intent`, `audit_actor_type`,
  `access_token_purpose`. The migration declares it, sqlc generates a Go type, Huma generates the
  OpenAPI enum, `openapi-typescript` generates the union: one declaration, four consumers, drift
  impossible. That is exactly [ADR 0004](./0004-api-contract-and-type-generation.md)'s requirement
  that closed sets be generated rather than restated.
- **A Go constant set when the vocabulary grows with the codebase** — notably `audit_log.action`,
  which gains entries every phase and would otherwise cost a migration per action string.

**Never depend on PostgreSQL enum ordering.** Ordinality that matters comes from data:
`grade_levels.ordinal`, not a type's declaration order.

**Timestamps.** `created_at` and `updated_at`, both `timestamptz NOT NULL DEFAULT now()`, with
`updated_at` maintained by one shared trigger function rather than by application code, so it cannot
be forgotten. **No `created_by` columns**: the audit log already answers who, and a second answer
will eventually disagree with the first.

**Soft delete** on `students`, `adults` and `households` only, via `deleted_at`. Hard deletion is
Phase 10 (§21.3, Owner-only). Link rows — household memberships, guardian relationships — are hard
deleted with an audit entry, because removing a relationship is not erasing a person.

- Filtering is **explicit** (`WHERE deleted_at IS NULL`), never a global default. Row-level security
  already occupies the "invisible automatic predicate" slot, and unlike a cross-tenant leak, showing
  a deleted person to their own organization's administrator is low-stakes and sometimes wanted.
- **Unique constraints involving soft-deletable rows must be partial**, e.g.
  `UNIQUE (organization_id, school_year_id, external_identifier) WHERE deleted_at IS NULL`. Otherwise
  a soft-deleted student permanently blocks re-adding that external identifier — a defect that
  surfaces months later, mid-import.

### Migrations

**Versions are timestamped, not sequential.** Two agents each creating `00002_…` produces *no git
conflict* — the filenames differ — and then Goose sees duplicate versions at apply time. Invisible at
merge, broken at apply, is the worst available failure mode.

`--allow-missing` is enabled so a developer's out-of-date database tolerates an earlier-timestamped
migration merging later. Correctness is guaranteed by the CI **fresh-migrate** check, since production
only ever migrates forward. The residual risk — migration A creating what migration B alters, merged
out of order — is prevented by decomposing tasks so concurrent work never touches the same table.

**A merged migration is never edited.** Corrections are new migrations.

### Generated code

**Committed: sqlc Go output and `openapi.json`.** The Go output must compile, and the committed
OpenAPI document keeps the contract reviewable in a diff.

**Not committed: TypeScript types**, which are generated during the frontend build. Staleness in the
TS layer becomes structurally impossible rather than merely checked, and one whole class of merge
conflict disappears. It also keeps the frontend build free of a Go toolchain, which generating the
OpenAPI document there would require.

**Generated files are never hand-merged.** A conflict in `internal/db/gen` or `openapi.json` is
resolved by regenerating. This is the single most likely thing to be got subtly wrong, because a
hand-merged generated file usually *looks* correct.

### Files that every feature touches

**The Layer 2 entity registry is a directory of per-entity files**, each registering itself in
`init()`. A single registry file would make every entity task conflict with every other, which would
serialise the phase.

**Route registration is per feature.** Each feature package exposes `Register(api huma.API, deps)`;
the central file holds one line per feature. One-line conflicts are trivial; a monolithic router is
not.

### API conventions

**No pagination, except for unbounded collections.** Students, adults, households and vocabularies in
a school year are naturally bounded — 139 students does not justify a pagination envelope on twenty
endpoints (§5.7). `audit_log` grows without bound and gets keyset pagination on `(occurred_at, id)`.
Deciding once prevents twenty endpoints from each inventing an answer.

**A registry of RFC 9457 problem types** with stable slugs, declared in Go and generated into the
contract — `tenant-not-found`, `school-year-closed`, `capability-required`, `no-organization`. This is
what lets the frontend switch on a machine-readable cause instead of string-matching error prose, and
it is what makes the 404-not-403 and 409 distinctions elsewhere actionable in the UI.

## Alternatives considered

**`text` + `CHECK` for closed sets.** This was the initial preference and is wrong here: it declares
the same set twice, in a migration and in Go, with nothing comparing them. The PostgreSQL enum is
single-sourced. The cost accepted in exchange is real but small — `ALTER TYPE ... ADD VALUE` cannot
*use* the new value in the same transaction, and Goose wraps migrations in transactions, so adding a
value plus backfilling takes two migrations; values cannot be removed or reordered.

**A global soft-delete filter** in the data layer. Rejected: a second invisible automatic predicate
alongside row-level security, for a far lower-stakes concern, and it makes "show me deleted people"
awkward.

**Sequential migration numbering with conflict resolution by rename.** Rejected: git does not
conflict, so nothing forces the rename.

**Committing generated TypeScript types.** Rejected: pure merge-conflict surface with no reviewability
benefit, since the reviewable artifact is the OpenAPI document.

**Generating the OpenAPI document during the frontend build** rather than committing it. Rejected: it
would require a Go toolchain in the frontend build and remove the contract from code review.

## Consequences

- Adding an entity is a fixed recipe, which is what makes the "add a tenant-scoped entity" Detent
  skill worth writing after the second one.
- Two committed generated artifacts remain a conflict surface. The mitigation is procedural
  ("regenerate, never merge") and lives in `AGENTS.md`.
- Timestamped migration filenames read less clearly in a directory listing than `00001`, `00002`.
  Accepted.
- The `audit_log.action` vocabulary being Go-owned means it is *not* enforced by the database. A typo
  produces a valid row with a wrong action. The mitigation is that actions are constants, not
  literals, at every call site.
- Choosing not to paginate is a decision that becomes a breaking API change if reversed. Acceptable,
  because the only client is in this repository.
