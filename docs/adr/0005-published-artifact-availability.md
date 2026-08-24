# 5. Published-artifact availability and topology

- **Status:** Proposed — shape decided in Phase 0, built in Phase 6
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

**Not yet made.** The requirement is accepted; the mechanism is open. Candidates:

**A. Static snapshot rendered at publish time, served by a separate Render static site or CDN.**
Strongest availability story — no runtime dependency on anything. Requires the token check to move
to the edge, or the URL's unguessability to be the entire control. Note that §9.5 already concedes
obscurity is the only access control for share links, so this may be less of a compromise than it
first appears. Revocation becomes a delete-and-redeploy rather than a database update, which is
slower and needs thought against §18.8.

**B. A separate lightweight Go read-only service** sharing the database but with no dependency on
Supabase Auth or the solver. Keeps token revocation immediate and share-link semantics identical to
everywhere else. Survives administrative-application failure but not database failure.

**C. Serve from the main API with aggressive caching.** Simplest; satisfies the letter of a SHOULD
but not its intent, since a Supabase outage or a bad deploy still takes the dismissal list down at
12:45.

Whichever is chosen must preserve the §18.8 and §9.5 link lifecycle: expiring, regenerable —
invalidating the prior URL — revocable, and failing cleanly with an explanation rather than an error
page.

## Open questions to settle in Phase 0

- Is link revocation required to take effect immediately, or is a publish-cycle delay acceptable?
  This single question largely decides between A and B.
- Does the print stylesheet requirement (§22.4) constrain the rendering location?
- What is the acceptable staleness window between publish and availability?

## Consequences of deferring

Low. Phase 6 is the first phase that publishes anything, and the snapshot semantics required by
§18.2 are the same under all three options. The risk of deferring past Phase 0 is that Phases 1–5
quietly assume artifacts render from live domain queries, which would foreclose option A — so the
decision is scheduled early even though the build is late.
