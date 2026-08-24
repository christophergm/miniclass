# 8. Authorization, capabilities and the audit log

- **Status:** Accepted
- **Date:** 2026-08-23
- **Implements:** SPEC §6.6, §9.4, §20.1
- **Related:** [0002](./0002-authentication-and-access-mechanisms.md),
  [0007](./0007-tenancy-enforcement-and-data-access.md),
  [0009](./0009-administrator-sessions-and-identity-provider.md)

## Context

SPEC §6.6 defines eight capabilities across three account roles, with two separations that matter
from the first phase: only `Owner` manages administrators, and `Coordinator` may not read the audit
log. §6.6 explains why `Coordinator` exists — "a second organizer who does substantive work on one
program but should not be the person who publishes it or removes a family's data" — so the role is
load-bearing rather than decorative.

§9.4 requires authorization to be decided server-side on every request, and requires the tenant check
to precede the permission check.

§20.1 requires an append-only log of significant actions across eight categories, immutable once
written, recording actor, timestamp, action, affected object, a change summary and any reason
supplied. `PLAN.md` states the Phase 1 exit criterion as *"every mutation appears in the audit log"* —
a coverage claim, and coverage claims backed by discipline decay.

## Decision

### Authorization

**1. Handlers ask for capabilities, never roles.** The capability set is closed and transcribed
directly from §6.6 into one Go table, with a table-driven test asserting every cell. A call site
reads `CapabilityReadAuditLog`, not `role == Owner`, so §6.6's "granularity beyond that minimum is
`Implementation-defined`" stays reachable: future per-program scoping composes into capability
resolution instead of rewriting call sites.

**2. The required capability is declared as Huma operation metadata and enforced by middleware,
default-deny.** An operation registered without a capability declaration fails a test that enumerates
every operation in the API.

This is the same move as Layer 1 of the isolation harness in
[ADR 0007](./0007-tenancy-enforcement-and-data-access.md), and it works for the same reason: because
[ADR 0004](./0004-api-contract-and-type-generation.md) makes Go the contract's source of truth, the
complete operation set is introspectable. A handler that forgot its permission check is therefore a
red build rather than a silent hole.

**3. `Principal` is an interface exposing `HasCapability`.** Account principals consult the role
matrix; the link principals arriving in Phase 4 return small fixed sets. This is what keeps ADR
0002's "one authorization implementation" reachable without designing for it now.

**4. Middleware order is: authenticate → resolve principal and organization → check capability →
handler.**

This satisfies §9.4's "the tenant check MUST precede the permission check", though it looks wrong at
a glance and the reasoning is worth recording. The tenant is established at authentication, before
any permission is evaluated. Resource-level tenancy is enforced by row-level security inside the
handler and yields not-found (§9.4's second sentence). Capability checks are a function of endpoint
and role only — they never inspect a resource identifier — so they cannot leak whether another
tenant's record exists: a `Coordinator` hitting an audit endpoint receives the same 403 whether the
identifier is real, foreign, or invented.

### The audit log

**5. Entries are written by the application, inside the same transaction as the change**, and the
requirement is structural rather than conventional. `InTenant` takes the actor and hands the closure
a unit exposing both the queries and `Record(entry)`. **A read-write transaction that recorded no
audit entry fails to commit.** Reads use `InTenantRead`, which is `BEGIN READ ONLY` and cannot mutate
at all.

So the invariant is not "remember to audit" but "an unaudited write cannot commit". The escape hatch
is `NoAuditRequired(reason string)` — explicit, greppable, and visible in review. Phase 2's import
staging will need it; nothing in Phase 1 should.

**6. Coverage assertions reuse the Layer 2 entity registry.** The registry already knows how to
create, update and delete each entity, so asserting that each produces an entry with the right
action, object and actor costs nothing per entity.

**7. `audit_log` schema.** Tenant-scoped and RLS-protected like everything else, plus:

| Aspect | Decision |
|---|---|
| Immutability | Enforced by **privilege**: the app role is granted `INSERT` and `SELECT` only. No `UPDATE`, no `DELETE`, ever. |
| Actor | `actor_type` (`user` / `link` / `system`), nullable `actor_user_id` → `users`, and a denormalised `actor_label` snapshot |
| Action | `text`, with the closed set declared in Go and generated into the contract |
| Change summary | `jsonb`, before/after for changed fields only |
| Other | `reason`, nullable `school_year_id` (§20.1 retains the log with its year), `request_id` for log correlation |

Privilege-level immutability is stronger than a constraint and cannot be forgotten by a future query.
The denormalised actor label exists for §21.3: hard deletion redacts *content*, "never the fact that
an action occurred", so labels and summaries must be redactable without destroying the row or
breaking a foreign key.

`audit_log` is exempt from the closed-year trigger, because it must be able to record events *about*
a closed year — including the closing itself.

**8. Reading is `Owner` and `Administrator` only** (§6.6), with keyset pagination on
`(occurred_at, id)`. It is the one Phase 1 collection that grows without bound.

**9. No hash chaining or tamper-evidence.** Privilege-level append-only is proportionate for three
administrators; anything further is theatre.

## Alternatives considered

**Inline role checks in handlers, or a central matrix with an explicit `Must(capability)` call.**
Rejected. Both fail identically: a handler with no check is a silent hole and nothing can detect it.

**A database-driven permission system.** Rejected as over-engineering. §6.6 explicitly makes finer
granularity `Implementation-defined`, and three roles do not need a permission table.

**Database triggers capturing row changes as the audit log.** Rejected. §20.1 wants *actions* —
"session non-participation recorded", "override accepted, with a reason" — not row diffs. A trigger
cannot know why.

**Triggers as a safety net alongside application entries.** Rejected. It produces two logs where the
specification describes one, and the two will disagree.

**Enumerating mutating operations from the generated OpenAPI document and asserting each produces an
entry.** Considered and found redundant once the unit of work refuses to commit unaudited writes. The
structural guarantee is strictly stronger, and the registry covers the semantic content.

## Consequences

- The `Coordinator` exclusion from the audit log is the first real permission distinction the system
  enforces, and therefore the first place the capability model is genuinely tested rather than
  merely exercised.
- Audit coverage is a property of the transaction boundary, not of author diligence. The cost is that
  every genuinely non-auditable write must say so out loud.
- `NoAuditRequired` is a hole by construction. It is narrow, greppable and reviewable, and its growth
  is visible in diffs.
- Redaction under §21.3 is possible without deleting rows, because nothing in an entry is a live
  reference to deletable content except `actor_user_id`, which is nullable.
- The action vocabulary grows every phase and lives in Go rather than in a database enum, so adding
  one costs no migration. See
  [ADR 0010](./0010-schema-generated-code-and-migration-conventions.md).
- §20.1's category table does not currently list vocabulary definition changes, which the grade and
  homeroom vocabularies need. That is a small specification amendment under **Rules**, not an
  invention here.
