# 5. Published-artifact availability and topology

- **Status:** Accepted — decided in Phase 0, built in Phase 6
- **Date:** 2026-08-23
- **Implements:** SPEC §22.3, §18.2, §18.8
- **Related:** [0001](./0001-application-stack-and-topology.md),
  [0002](./0002-authentication-and-access-mechanisms.md)

## Context

SPEC §22.3 sets modest availability requirements for the administrative application — brief weekday
outages are acceptable — and then carves out one hard exception:

> Published pages SHOULD be servable independently of the administrative application and remain
> available when solving, importing or reporting is degraded.

The reason is concrete rather than architectural. The homeroom dismissal list is consulted at a fixed
weekly moment — 12:45 on a Friday — with children waiting to be sent to their classes. There is no
graceful degradation available to a teacher holding a phone in a corridor. Meanwhile the
administrative application depends on Supabase Auth ([ADR 0002](./0002-authentication-and-access-mechanisms.md)),
on the solver sidecar ([ADR 0003](./0003-assignment-solver-technology.md)), and on whatever else
accumulates by R3 — a dependency set the dismissal list has no business inheriting.

Two properties of published artifacts make this tractable:

- §18.2 makes published content a **point-in-time snapshot, not a live view**. There is no
  requirement to query the domain model at read time.
- §18.5 restricts published content severely: no tags above `Public`, no tag notes, no comments, no
  adult email addresses. The snapshot is small and contains no sensitive data.

So the artifact is genuinely static between publish events. Nothing forces it to be served by the
application that produced it.

## Decision

**Published artifacts are served by the main Go API. §22.3's independence clause is knowingly
relaxed.**

That clause is a **SHOULD**, not a MUST, so this is spec-compliant — but it is a deliberate
relaxation and is recorded as one rather than glossed: **the dismissal list inherits the
administrative application's availability, including its dependency on a good deploy and on Supabase
Auth being reachable.** The judgement is that v1 simplicity is worth more than the resilience,
given a single operator, a single deployment and no production history yet.

**Revisit trigger, so this is not inherited silently: if a Friday is ever disrupted by the
administrative application being unavailable at 12:45, this record is reopened.**

**One property is retained regardless of topology.** §18.2 makes published content a point-in-time
snapshot, not a live view, so **publishing materialises a stored, self-contained snapshot and nothing
renders a published artifact from live domain queries.** This is required by §18.2 independently of
who serves the bytes, costs nothing now, and has the side benefit of keeping options A and B below
reachable as a pure deployment change if the trigger ever fires.

The §9.5 and §18.8 link lifecycle is unaffected: expiring, regenerable — invalidating the prior URL —
revocable, and failing cleanly with an explanation rather than an error page.

## Alternatives considered

**A. Static snapshot rendered at publish time, served by a separate static site, CDN or object
store.** The strongest availability story, and it survives even a database outage. Rejected on two
grounds beyond simplicity. First, **revocation must take effect immediately**: an organizer who
revokes a dismissal list because it reached the wrong parent, and is told it stops working after the
next publish cycle, has not revoked anything. Second, **§18.8 requires links to "fail cleanly with an
explanation rather than an error page"** — a deleted object returns a bare 404, not "this link expired
at the end of session 6", so a fallback service would be needed anyway, leaving two link lifecycles
to keep consistent. It would also make publishing — the most routine operation in the system — into a
deploy.

**B. A separate lightweight Go read-only service** sharing the database, with no dependency on
Supabase Auth or the solver. This was the recommended option before the relaxation above: it keeps
revocation immediate and link semantics identical, and it survives the likeliest Friday failures — a
bad deploy of the admin API, an auth outage, a wedged solver — though not a database outage.
Rejected for v1 as a second deployment unit, two Render services, and Compose and CI configuration
that the operator judged not yet earned.

## Consequences

- **The dismissal list can be taken down by a bad deploy at 12:45 on a Friday.** This is the accepted
  risk, stated plainly so that nobody is surprised by it, and it is what the revisit trigger watches
  for.
- One deployment unit rather than two: no extra Render service, no third database role, and Phase 6
  shrinks.
- Publishing still writes a snapshot row, so Phases 1–5 must not assume live rendering of published
  artifacts. This is the one constraint this record propagates backwards.
- Print styling (§22.4) is unconstrained by topology; the API serves its own stylesheet.
- The staleness window between publish and availability is zero — publishing writes the snapshot and
  it is immediately live.
