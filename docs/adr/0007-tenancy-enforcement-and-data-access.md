# 7. Tenancy enforcement and data access

- **Status:** Accepted
- **Date:** 2026-08-23
- **Implements:** SPEC §9.1, §9.2, §9.4
- **Related:** [0001](./0001-application-stack-and-topology.md),
  [0008](./0008-authorization-capabilities-and-audit.md),
  [0010](./0010-schema-generated-code-and-migration-conventions.md)

## Context

SPEC §9.1 makes the organization the tenant boundary, enforced by row rather than by database or
schema, and requires that no reference cross an organization boundary. §9.2 is stricter than a
convention:

> Tenant scoping MUST be enforced centrally and MUST be default-deny. It MUST NOT depend on each
> query remembering to filter. [...] a query issued without one MUST fail rather than return
> unscoped rows. [...] A tenant-scoped entity added without a corresponding test is a defect.

The failure mode is named explicitly in §9.1: "disclosing one school's children to another".

Two facts decide the shape of the answer.

**PostgreSQL referential-integrity checks deliberately bypass row-level security.** A child row can
therefore reference a parent in another organization: the `INSERT` passes its foreign-key check even
though the same transaction cannot `SELECT` that parent. Single-column foreign keys leave §9.1's
no-crossing rule unenforced, and row-level security does not cover the gap.

**Row-level security is introspectable; query discipline is not.** Whether a table has RLS enabled
and forced, and whether a constraint includes `organization_id`, are facts readable from
`pg_class`, `pg_policies` and `pg_constraint`. Whether a hand-written query remembered its `WHERE`
clause is not a fact any test can read. Since most of this system is written by agents in parallel
worktrees who cannot see each other's work, a guard that can be *proved* by a test is worth more
than a guard that must be *remembered* by an author.

## Decision

**1. Row-level security is the enforcement floor.** Every tenant-scoped table gets
`ENABLE ROW LEVEL SECURITY` **and** `FORCE ROW LEVEL SECURITY`, with a policy predicated on
`organization_id = current_setting('app.organization_id')::uuid`.

The `current_setting` call omits the `missing_ok` argument. This is a one-word decision with
opposite security semantics: without it, an unset setting raises SQLSTATE 42704 and the query
*fails*, which is what §9.2 requires; with it, the setting resolves to NULL, the policy evaluates
false, and the query silently returns an empty result set — the exact failure mode §9.2 forbids.

**2. Two database roles**, created by migration. `miniclass_migrator` owns the schema and is the only
role with DDL. `miniclass_app` is what the API connects as: no `BYPASSRLS`, no DDL, DML grants only,
`statement_timeout = '10s'`. `FORCE` means the migrator is subject to policy too, so no third
bypassing role is needed.

**3. Data access occurs only through closures** in `internal/data`:

```go
func (p *Pool) InTenant(ctx, scope, fn func(context.Context, *gen.Queries) error) error     // BEGIN
func (p *Pool) InTenantRead(ctx, scope, fn func(context.Context, *gen.Queries) error) error // BEGIN READ ONLY
```

Each opens a transaction, issues `SET LOCAL app.organization_id`, and commits or rolls back.
`SET LOCAL` rather than `SET`: a session-level setting on a pooled connection carries one tenant's
scope into the next checkout, which is the standard way to convert row-level security into a
cross-tenant leak.

Generated sqlc code lives in `internal/db/gen`, and a `golangci-lint` `depguard` rule permits exactly
one importer: `internal/data`. sqlc emits an exported constructor, so without that rule any package
could build a tenant-free handle against the raw pool and defeat the design. The rule is what makes
the guard unbypassable rather than merely conventional.

**4. An identity layer sits outside RLS**, containing exactly four tables: `organizations`, `users`,
`organization_members`, `access_tokens`. Authentication must resolve a verified subject — or a link
token — to an organization *before* the tenant is known, so this layer cannot be tenant-scoped.
`access_tokens` belongs here because §9.5 forbids a token from encoding a tenant identifier, which
makes a pre-tenant lookup unavoidable.

It is reached only through `internal/data/identity`, import-restricted by `depguard` to
`internal/identity`. That accessor opens a transaction and never sets the GUC, so **tenant-scoped
tables are unreachable from it**: it is not a bypass, it is a path on which the domain does not
exist. The allowlist is a literal slice in the meta-test, so growing it is a visible diff requiring
a spec citation.

**5. Composite foreign keys.** Every tenant-scoped table declares `UNIQUE (id, organization_id)` as
a foreign-key target; year-scoped entities declare `UNIQUE (id, organization_id, school_year_id)`.
References between them carry every scope column:

```
guardian_relationships
  (school_year_id, organization_id)             → school_years (id, organization_id)
  (adult_id,   organization_id, school_year_id) → adults   (id, organization_id, school_year_id)
  (student_id, organization_id, school_year_id) → students (id, organization_id, school_year_id)
```

This closes the RI-bypass hole above, and the same mechanism at no extra cost closes a second silent
defect class the year-scoping of people creates: a household spanning two school years, or a guardian
relationship linking an adult in one year to a student in another.

One deliberate exception: the §8.7 prior-year link references a student in a *different* year of the
same organization, so it is two-column, nullable, and `ON DELETE SET NULL`.

**6. The organization is never a request parameter.** It is derived once, by the authentication
middleware, from the principal's membership, and is available only through `tenancy.FromContext`. No
handler accepts an organization identifier. A resource identifier belonging to another organization
therefore meets RLS, matches zero rows, and returns **not-found** — which is §9.4's requirement
obtained structurally rather than by handler discipline.

**7. The isolation harness has two layers.**

*Layer 1* is generic and needs no per-table work. For every table on which `miniclass_app` holds any
privilege it asserts: presence on the four-name allowlist or an `organization_id uuid NOT NULL`
column; RLS enabled *and* forced; a policy predicated on `app.organization_id`; `UNIQUE (id,
organization_id)`; every foreign key to another tenant-scoped table includes `organization_id`; and
behaviourally, that a query with no GUC set fails with 42704. It also asserts that every
tenant-scoped table has a Layer 2 registry entry.

*Layer 2* is one registry line per entity — a factory and a fetch-by-id — from which the harness
generates, per entity: invisibility of another organization's rows, not-found on cross-organization
fetch by id, zero rows affected by cross-organization update and delete, and rejection of an insert
referencing a parent in another organization.

The HTTP-level "not-found, not forbidden" mapping gets *one* hand-written test, not one per entity:
generic where the risk is per-table, hand-written once where the code is shared.

**8. No cross-tenant path exists in the API.** §9.2 permits an `Implementation-defined` audited
mechanism; that is the bootstrap CLI, out of process, writing its own audit entry.

**9. Tests isolate by organization, not by schema.** One migrated database per test package; each
test mints its own organization. This is faster and parallel-safe, and it dogfoods the guard — if
tests cannot contaminate each other, the tenancy works, and every test becomes a weak isolation
test. Schema-per-test survives for exactly one job: the migration up→down→up round-trip check.

## Alternatives considered

**A Go-only guard: a tenant-scoped repository facade, every query taking `organization_id`.**
Rejected. This is precisely "each query remembering to filter", which §9.2 forbids by name. The
facade guarantees a tenant is *known*, not that the SQL *used* it: `SELECT * FROM students WHERE
school_year_id = $1` compiles, passes review, passes tests, and leaks. And no test can prove the
absence of that mistake, which forfeits the introspectability argument above.

**Both: RLS plus explicit `WHERE organization_id = $1` in every query.** Rejected. Once policies are
forced the predicates are redundant, they double the surface an agent can get wrong, and they
reinstate the habit the specification rejects. Explicit predicates are permitted only where the
query planner demonstrably needs one, as a deliberate and commented performance decision.

**Keeping `organization_members` under RLS with a compound policy on a second `app.user_id` GUC.**
Rejected as a bad trade. `users` has no `organization_id` at all — a person may hold membership in
two organizations — and `organizations` is outside by definition, so this shrinks the allowlist from
four to three while adding a second GUC and a compound policy to the most security-sensitive table
in the schema.

**Path-addressed tenancy, `/api/organizations/{orgId}/...`.** Rejected. Making the organization a
request parameter obliges every handler to verify membership before checking permissions, which is
the "every handler must remember" pattern this record exists to eliminate.

**Carrying organization and role in the identity provider's token claims.** Rejected; see
[ADR 0009](./0009-administrator-sessions-and-identity-provider.md). It would remove the pre-tenant
lookup entirely, but delegates authorization to the identity provider, delays revocation to token
refresh, and puts permission changes outside the audit log required by §20.1.

## Consequences

- **Security logic now lives in migrations**, which agents touch constantly, and a new table with a
  forgotten `ENABLE ROW LEVEL SECURITY` is silent. Layer 1 of the harness is the entire mitigation,
  so it is not optional: it must land in the same change as the first tenant-scoped table, never
  after it.
- Compose, CI and the test harness all need two roles and two pools. This is real setup cost, and it
  is the price of the choice.
- Every request that touches the database runs in an explicit transaction. That is also a benefit:
  it is the unit of work in which [ADR 0008](./0008-authorization-capabilities-and-audit.md) writes
  the audit entry atomically with the change it describes.
- Closure-based transactions are noisier than a `defer tx.Rollback()` style and awkward when a use
  case composes two repositories. Accepted, because the alternative hands out a live tenant-scoped
  handle whose lifetime the caller controls — which is how such a handle ends up in a struct field.
- **A test may never assert on a global count or an empty table**, because another test's
  organization shares the database. Counts are scoped to the test's own organization. This has no
  automated enforcement and will bite someone, so it is written into `AGENTS.md`.
- `access_tokens` sits outside the guard. That is a deliberate hole, justified by §9.5 and narrowed
  by the fact that the identity path cannot reach the domain.
- Adding a tenant-scoped entity is now a fixed recipe of about ten steps. That repetition is the
  subject of a Detent skill, written after the second entity rather than imagined before the first.
