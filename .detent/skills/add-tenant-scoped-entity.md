---
name: add-tenant-scoped-entity
description: Checklist for adding a tenant-scoped entity while preserving RLS, data-access, audit, generated-code, API, and isolation-test invariants.
when_to_use: Use when a feature adds a PostgreSQL table whose rows belong to an organization, optionally within a school year.
---

# Add a tenant-scoped entity

Use this checklist for one entity. Keep the change scoped to the issue, cite the
applicable SPEC sections in the PR, and do not edit a merged migration.

## Define the shape

- [ ] Read SPEC §§8.7, 9.1–9.2, 13.5, 20.1 and the applicable ADRs (normally ADR 0007 and
      ADR 0010). Decide whether the entity is school-year-scoped.
- [ ] Reserve a fresh timestamped migration filename. Use lowercase SQL keywords,
      identifiers, function names, and statements.
- [ ] Use `id public.xid20 primary key default public.xid()`; never use a name as a key.
- [ ] Add `organization_id public.xid20 not null` and, if year-scoped,
      `school_year_id public.xid20 not null`.
- [ ] Add `unique (id, organization_id)`; for a year-scoped entity also add the
      parent-compatible uniqueness needed by composite references, normally
      `unique (id, organization_id, school_year_id)`.
- [ ] Make every reference to another tenant-scoped table a composite foreign key carrying
      all applicable scope columns. Do not rely on RLS to enforce referential integrity.
- [ ] Add the shared `updated_at` trigger where the entity is mutable. Use soft-delete only
      where the specification calls for it, and make uniqueness involving soft-deleted rows
      partial.

## Enforce the database boundary

- [ ] Enable **and force** row-level security.
- [ ] Add a policy whose `using` and `with check` expressions compare
      `organization_id` with `current_setting('app.organization_id')::public.xid20`.
      Omit `missing_ok`; an unset tenant must fail, not silently return no rows.
- [ ] If year-scoped, add the `public.prevent_closed_school_year_mutation()` trigger to
      insert/update/delete operations. Verify any permitted reopen path remains scoped to the
      target year and records its reason and actor.
- [ ] Grant the app role only the operations the feature needs. Revoke those grants in the
      migration down path, then drop triggers, policies, tables, and types in dependency order.

## Add the application path

- [ ] Add queries under `backend/sql/queries`, with explicit columns and no organization
      request parameter. Regenerate sqlc; commit `internal/db/gen` output and never hand-merge it.
- [ ] Add the typed data facade under `internal/data`. Keep generated sqlc imports behind
      that boundary. Reads use `InTenantRead`; writes use `InTenant` and record an audit entry
      in the same transaction, or call `NoAuditRequired` with a specific reason.
- [ ] Add the feature service with `context.Context` first, typed inputs, and validation that
      warns rather than blocks an organizer action unless the specification permits refusal.
- [ ] Define audit action constants in `internal/audit`; do not use action string literals at
      call sites. Record who, when, and why for judgement or override decisions.
- [ ] Add handlers and declare every endpoint with `huma.Register` (through the feature route
      registration pattern). Declare the required capability for every operation and expose
      only opaque identifiers. Map cross-tenant resources to not-found, not forbidden.
- [ ] Regenerate and commit `backend/openapi.json`; frontend TypeScript API types remain
      generated and uncommitted.

## Prove isolation

- [ ] Add one factory and complete fetch/read/update/delete/foreign-parent operations in a new
      file under `backend/internal/testing/registry`. Use synthetic fixtures and normal audited
      service writes where possible.
- [ ] Add or extend integration coverage so the entity is invisible across organizations,
      cross-organization fetch/update/delete behaves as not-found or zero affected, and a
      foreign-organization parent insert is rejected. Scope every count/list assertion to the
      test organization; never assert on a global count or an empty table.
- [ ] For year-scoped entities, cover the closed-year trigger and any allowed Owner reopen
      path, including the required audit evidence.
- [ ] Confirm the generic Layer 1 metadata test sees the table and the Layer 2 registry test
      exercises it. A tenant-scoped table without both layers is a defect.

## Validation map

| Checklist concern | Check that catches it | Still requires review |
| --- | --- | --- |
| RLS enabled/forced, tenant policy, scope columns, uniqueness, composite foreign keys, and registry coverage | `cd backend && make test` (integration isolation harness) | Policy intent and correct scope columns |
| Cross-tenant behavior and closed-year behavior | `cd backend && make test` (Layer 2 and feature integration tests) | Adequacy of cases and audit assertions |
| Raw generated sqlc imports and identity-accessor boundaries | `cd backend && make lint` (`depguard`) | Whether the data API is the right abstraction |
| Go formatting and vet | `cd backend && make format` | Readability and error semantics |
| sqlc/OpenAPI generated artifacts | `cd backend && make generate && git diff --exit-code` or `make generated-code-drift` | Never resolve generated conflicts by hand |
| Migration applies, rolls back, and reapplies | `cd backend && ./scripts/migration-round-trip.sh` | Migration ordering and compatibility with existing data |
| API operation enumeration and declared capabilities | Backend tests plus generated-code drift | Capability choice and not-found semantics |
| Repository whitespace | `git diff --check` | Lowercase SQL, timestamp choice, SPEC/ADR citations, and audit completeness |

Run focused package tests first, then the full project gate. Use only synthetic test data.
