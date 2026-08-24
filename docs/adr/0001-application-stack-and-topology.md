# 1. Application stack and topology

- **Status:** Accepted
- **Status:** Accepted
- **Date:** 2026-08-23
- **Supersedes:** `achitecture.md` (removed)
- **Amended in part by:** [0009](./0009-administrator-sessions-and-identity-provider.md) — the browser
  does talk to Supabase, for authentication only; every data path still goes through Go
- **Related:** [0002](./0002-authentication-and-access-mechanisms.md),
  [0003](./0003-assignment-solver-technology.md),
  [0005](./0005-published-artifact-availability.md),
  [0007](./0007-tenancy-enforcement-and-data-access.md)

## Context

The system described in [`SPEC.md`](../../SPEC.md) is small by design (§5.7) and small in fact
(§22.1): roughly 140 students, 90 households, 60 adults and 8–13 offerings per session, with one
administrator working at a time and occasionally two. Total data volume is "tens of thousands of
rows" per tenant-year. Genuine concurrency peaks are read-only — households submitting before a
deadline, and published links opened at the start of a session.

Against that, the system has one genuinely hard part: a constraint solver that must be optimal,
deterministic, explainable and able to diagnose its own infeasibility (§17).

The specification is technology-agnostic and requires implementations to document their choices.

## Decision

The stack scaffolded in the repository is confirmed:

| Layer | Choice |
|---|---|
| Frontend | React 18 + TypeScript + Vite, TanStack Query for server state, React Router |
| UI | shadcn/ui |
| Drag and drop | dnd-kit (Phase 9 drafting workspace) |
| Backend | Go, chi router |
| Database access | pgx + sqlc |
| Migrations | Goose |
| Database | PostgreSQL 18 — Docker Compose locally, Supabase managed in production |
| Authentication | Supabase Auth for administrators only; see [ADR 0002](./0002-authentication-and-access-mechanisms.md) |
| Solver | Python OR-Tools CP-SAT sidecar; see [ADR 0003](./0003-assignment-solver-technology.md) |
| Hosting | Render for the Go API and the static frontend |
| Tool versions | proto (`.prototools`) pins Go, Node and Bun |

**Go is the authoritative application layer.** The browser talks to Go; it does not talk to Supabase
directly. Assignment logic, the tenancy guard, authorization, validation, workflow and the audit log
live in exactly one place.

```
React ──HTTPS/JSON──▶ Go API ──pgx/sqlc──▶ PostgreSQL
                        │
                        └──▶ Solver sidecar (Python, CP-SAT)
```

This is a deliberate constraint rather than a default. SPEC §9.2 requires a single central
default-deny tenancy guard through which every read, write, aggregate and report passes. A second
data path that reaches Postgres without traversing that guard would defeat it, so there is not one.

## Alternatives considered

**Browser talks directly to Supabase with row-level security.** Rejected. It would move the tenancy
guard into database policy, split authorization across two systems, and make §9.2's "queries issued
without tenant context must fail" much harder to assert in tests. It also does nothing for the
solver, the audit log or the import engine, which are the bulk of the work.

**A single Go binary with no solver sidecar.** Not rejected on merit, but see
[ADR 0003](./0003-assignment-solver-technology.md): the capability list in §17.3 is closely matched
by CP-SAT and poorly matched by anything available natively in Go.

**Drop Supabase in favour of plain Postgres on Render.** Genuinely arguable, given three
administrator accounts and §5.7's preference for the direct approach over the scalable one. Rejected
for now — see [ADR 0002](./0002-authentication-and-access-mechanisms.md) — because the managed
Postgres, Storage and Auth surface may earn its place as the system grows, and the cost of keeping it
is low while the application layer stays authoritative.

## Consequences

- One deployment unit becomes three: Go API, static frontend, solver sidecar. CI, Compose and Render
  configuration must all account for the sidecar from Phase 5.
- Every feature costs a Go handler and a TypeScript caller. The contract between them is therefore
  worth generating rather than hand-writing; that is the subject of
  [ADR 0004](./0004-api-contract-and-type-generation.md).
- Published artifacts have an availability requirement the administrative application does not
  (§22.3), which this topology does not by itself satisfy. See
  [ADR 0005](./0005-published-artifact-availability.md).
- Supabase is a dependency the system does not currently exercise heavily. If it is still only
  providing authentication for three accounts by R3, that is a signal to revisit this record.
