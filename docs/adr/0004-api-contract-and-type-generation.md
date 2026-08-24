# 4. API contract and type generation

- **Status:** Proposed — must be resolved in Phase 0
- **Date:** 2026-08-23
- **Related:** [0001](./0001-application-stack-and-topology.md)

## Context

[ADR 0001](./0001-application-stack-and-topology.md) makes Go the authoritative application layer and
routes every browser interaction through it. The consequence is that essentially every feature in
[`PLAN.md`](../../PLAN.md) costs a Go handler plus a TypeScript caller, and the system will
accumulate a large API surface: roster CRUD, import preview and commit, catalog authoring, session
lifecycle transitions, survey definition and submission, ranked choices, tags, pairings, staffing,
solve and re-solve, manual assignment operations, ten reporting endpoints, publishing, and share-link
management.

Two properties of this project raise the value of a machine-checked contract above the usual:

- **Most of the code is written by agents in parallel worktrees.** Two agents implementing adjacent
  endpoints cannot see each other's work. Hand-written Go structs and hand-written TypeScript
  interfaces drifting apart is the most likely silent defect class in the entire build, and it is
  invisible to both the Go compiler and `tsc`.
- **Several domain distinctions are semantically load-bearing and structurally invisible.** SPEC
  §13.5 requires *Rated*, *Unrated* and *No response* to remain distinct; §16.5 defines eighteen
  stable warning identifiers; §17.4.1 defines a five-point ordered quality scale. Each of these is a
  closed set that both sides must agree on exactly. A generated contract makes disagreement a build
  failure; hand-written types make it a bug report from an organiser in week three.

Today the frontend hand-writes its types (`frontend/src/lib/api.ts` validates the health response by
hand). That is proportionate for one endpoint and will not survive fifty.

## Decision

**Not yet made.** The candidates:

**A. OpenAPI as the source of truth.** A hand-written OpenAPI document generates Go server
interfaces via `oapi-codegen` and TypeScript types via `openapi-typescript`. Contract-first, so the
schema is reviewable independently of either implementation, and drift is a CI failure. Cost: the
document is a third artifact to maintain, and Go code generation from OpenAPI is more intrusive than
the alternatives.

**B. Go as the source of truth, TypeScript generated.** Annotated Go handlers or types generate an
OpenAPI document or TypeScript declarations directly. Lower ceremony and keeps Go primary, matching
ADR 0001. Cost: the contract is only as reviewable as the Go code, and generators in this direction
vary in quality.

**C. tRPC-style shared schema.** Rejected in advance — it assumes a TypeScript server.

**D. Hand-written both sides, with contract tests.** Cheapest to start, and the current de facto
state. Cost: the failure mode is silent and the mitigation is discipline, which does not scale to
parallel agents.

Whichever is chosen must satisfy:

1. Drift between Go and TypeScript fails CI, not review.
2. The closed sets above are generated, not restated.
3. An agent adding an endpoint has one obvious place to declare it.

## Consequences of deferring

Every endpoint written before this is resolved is a candidate for rework. This is the reason the
decision is scheduled for Phase 0, before Phase 1 introduces roster CRUD — the first substantial
surface.
