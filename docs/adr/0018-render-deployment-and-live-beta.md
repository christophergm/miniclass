# 18. Render deployment and live-beta boundary

- **Status:** Accepted
- **Date:** 2026-09-30
- **Amends:** [0001](./0001-application-stack-and-topology.md),
  [0003](./0003-assignment-solver-technology.md)
- **Related:** [0005](./0005-published-artifact-availability.md),
  [0007](./0007-tenancy-enforcement-and-data-access.md),
  [0009](./0009-administrator-sessions-and-identity-provider.md)

## Context

ADR 0001 selects Render for the Go API and static frontend, Supabase for managed PostgreSQL, and a
separate Python solver. ADR 0003 requires that solver to remain stateless and reachable only through
its versioned HTTP contract. Neither record defines the deployed topology, promotion path, or the
operational boundary between an early live beta and R3 production readiness.

The application will handle real children's data during a live beta before all Phase 10 safeguards
exist. That makes the residual risk explicit operational work, not an implied weakening of SPEC §21
or §22.3–§22.5. In particular, an untested restore, incomplete full-surface privacy hardening, or
insufficient observability cannot be described as R3 readiness.

## Decision

### 1. Topology and origins

The committed [`render.yaml`](../../render.yaml) is the Render Blueprint for this topology, in
Render's Oregon region:

| Component | Render service | Exposure | Responsibility |
|---|---|---|---|
| Frontend | Static Site | Public | Serves the Vite bundle. |
| API | Go Web Service | Public | The only browser data path and the only application caller of Supabase. |
| Solver | Python Private Service | Private | Stateless, versioned solve contract; it is not reachable from the Internet. |

Supabase has separate staging and production projects. They are separate credentials, databases,
Auth tenants, backup/restore boundaries, and migration targets; staging is never a schema or data
namespace inside production. Development and automated testing continue to use local PostgreSQL and
synthetic data only.

The canonical origins are:

| Environment | Frontend | API |
|---|---|---|
| Production | `https://www.miniclass.org` | `https://api.miniclass.org` |
| Staging | `https://staging.miniclass.org` | `https://api.staging.miniclass.org` |

`https://miniclass.org` redirects permanently to `https://www.miniclass.org`; it serves no
application content. The API config permits only the frontend origin for its environment. The
frontend's `VITE_API_URL`, API `API_BASE_URL`, invitation-claim base URL, Supabase site/redirect
configuration, and CORS configuration use the same environment's pair of origins. Secrets and
provider-managed values are configured in Render and Supabase, not committed to the Blueprint.

### 2. Database roles, migration, and rollback

The API has only the Supabase `miniclass_app` role credentials. It continues to reach data only
through the Go application and `internal/data`, preserving ADR 0001's single application path and
ADR 0007's RLS tenancy guard.

A release migration job uses separate `miniclass_migrator` credentials. It applies timestamped Goose
migrations before the corresponding API revision is promoted; neither the frontend nor the API
receives migrator credentials. Migration changes must preserve the project's expand/contract
compatibility rules so the prior API revision can be restored while a rollback is assessed. A failed
release is rolled back by redeploying the previously promoted immutable revision after checking its
database compatibility. A destructive data correction is a new forward migration or a separately
approved restore procedure, never an edit to an applied migration.

### 3. Immutable promotion

CI builds and tests a commit identified by its full Git SHA. Deployable artifacts and the Render
release are tagged with that exact SHA. Staging is promoted from that exact artifact, and production
is promoted from the same exact SHA after staging approval; a branch name, a moving tag, or a
independently rebuilt production artifact is not a promotion mechanism. The release workflow and
Render integration that enforce this are follow-on implementation work.

### 4. Live beta is not R3

A live beta may run before R3 to validate the complete operational flow with explicit, time-bounded
residual-risk acceptance by the accountable Owner. The acceptance record identifies the environment,
release SHA, known gaps, mitigating controls, approver, decision time, and expiry/review date. It is
not a substitute for a warning, audit entry, or the normal authorization boundary.

Before enabling a live beta, operators must at least confirm the production/staging separation,
restricted origin configuration, private solver reachability, migration/app-role separation, a
rollback path to the previous compatible SHA, and that no secrets or real roster data have entered
source control, development, CI, or staging.

R3 remains the Phase 10 bar. It still requires the full production-surface privacy and retention
work, an end-of-year purge and hard-delete proof, a documented and tested restore drill, and
observability sufficient to answer SPEC §22.5 without a database console. A live-beta acceptance
expires; it cannot waive, defer indefinitely, or relabel any R3 exit criterion.

## Alternatives considered

**Public solver service.** Rejected. It creates an unnecessary Internet-facing boundary for a
service used only by the API and conflicts with ADR 0003's narrow, stateless sidecar contract.

**One Supabase project with staging schemas.** Rejected. It weakens the operational and data
boundary needed for a live beta, and a configuration error could direct staging traffic at production
identities or data.

**Promote `main` or a release branch directly.** Rejected. A moving reference cannot prove that the
artifact tested in staging is the one serving production, and makes rollback ambiguous.

**Call a live beta “production”.** Rejected. It would obscure the residual risks that Phase 10 is
specifically sequenced to close.

## Consequences

- `render.yaml` can provision the service shape without containing credentials or production data;
  the Blueprint intentionally leaves secret values for Render/Supabase configuration.
- Deployment and release-workflow implementation is a prerequisite for the live beta, and must
  enforce this record's exact-SHA promotion and role separation.
- The solver's private service must be wired into the API deployment at Phase 5; its outage may
  degrade drafting but must not expose it publicly or affect published-artifact reads.
- Operating a live beta requires an auditable residual-risk acceptance and scheduled review until R3
  is achieved.
