# 3. Assignment solver technology

- **Status:** Accepted (validation deferred to Phase 5)
- **Date:** 2026-08-23
- **Implements:** SPEC §17.3, which marks the solver `Implementation-defined`
- **Related:** [0001](./0001-application-stack-and-topology.md)

## Context

SPEC §17.3 keeps the solver technology-agnostic but sets a demanding capability list:

- integer and boolean decision variables;
- linear **and logical** constraints over multiple variables;
- **lexicographic / hierarchical objectives**;
- **variable fixing before solve**, for pins and incremental re-solve;
- **deterministic search given a seed**;
- a **time limit that returns the best solution found so far**;
- **identification of a conflicting constraint subset** when the model is infeasible.

Two of these are unusual. §17.10 makes infeasibility diagnosis a MUST — the system must never report
a bare "infeasible" and must name the responsible constraints in the organiser's vocabulary. And
§17.8 calls determinism *load-bearing*: it gates incremental re-solve (§17.9), draft comparison
(§19.3) and every reproducibility claim in §20.2. §17.8 goes further and requires that time limits be
expressed in **search nodes or iterations** where no deterministic wall-clock limit exists.

The specification also explicitly forbids assuming the problem is a bipartite matching or min-cost
flow, because pairings (§10.6) couple two students' decisions and cannot be expressed as independent
per-student costs.

The problem is small: ~140 students × 12 offerings ≈ 1,700 decision variables before reduction.
§17.1 notes that optimality, determinism and explainability are affordable *precisely because* of
this. Performance targets (§22.2) are full solve under 10 s, incremental re-solve under 2 s,
single-placement explanation under 2 s, infeasibility diagnosis under 30 s.

## Decision

**Google OR-Tools CP-SAT, running in a Python sidecar service, called by the Go API over a versioned
request/response contract.**

CP-SAT satisfies every item on the §17.3 list natively, including the two hard ones: `random_seed`
plus `num_workers=1` (or its deterministic parallel mode) for reproducible search,
`max_deterministic_time` for limits in deterministic units, and
`sufficient_assumptions_for_infeasibility` for the conflicting-constraint subset §17.10 requires.

The contract is treated as a first-class artifact:

- the solve request is a **self-contained document** — participants, offerings, preferences, rules,
  pins, weights, seed — carrying no database handles and no hidden state;
- the response carries assignments with realized quality, metrics, solver status, and either the
  solution or a conflicting-constraint subset;
- both are versioned and validated at the boundary;
- the sidecar is **stateless** and holds no domain knowledge beyond the contract.

The consequence worth stating plainly: the solver is independently testable with golden files, and
replaceable, because nothing but the contract crosses the boundary.

**Validation is deferred to Phase 5** rather than spiked in Phase 0. The risk is judged low — CP-SAT
is a well-understood fit for a 1,700-variable problem — and Phase 5 opens with the historical replay
harness, which is a far better validation than a spike would have been.

## Alternatives considered

**CGO bindings to CP-SAT from Go.** Keeps one deployment unit. Rejected: the Go bindings are thin,
CGO complicates cross-compilation and CI, and the C++ toolchain dependency would be felt on every
build for a component that is called seconds at a time, a few times a day.

**HiGHS or another MIP solver via Go.** Rejected: no native irreducible-infeasible-subset support, so
§17.10 would have to be implemented by hand; lexicographic objectives available only by sequential
optimisation; determinism guarantees weaker.

**A hand-rolled solver in Go.** Feasible at this scale, and appealing under §5.7. Rejected because
the specification does not ask for a heuristic that produces good answers — it asks for provable
optimality (§17.4), reproducible search (§17.8), minimal conflict sets (§17.10) and counterfactual
explanation (§17.11). Writing and maintaining those is a project in itself, and it is the one part of
this system where the predecessor's failure is best documented (A.5 defects 1, 2, 3, 7, 8).

**Bipartite matching or min-cost flow.** Rejected by the specification itself: pairings couple two
students' decisions, and the objective is lexicographic rather than a single additive cost.

## Consequences

- A second runtime enters the project at Phase 5. Docker Compose, CI and Render configuration must
  all accommodate it; the Python toolchain needs pinning in `.prototools` alongside Go, Node and Bun.
- A network hop is added to a 2-second interactive budget. At this model size the hop, not the solve,
  is the likely cost — so payload size and serialisation are worth measuring early. The CI
  performance budget in Phase 5 exists for this reason.
- The sidecar must be unavailable-tolerant: a solver outage degrades drafting, and must not affect
  published artifacts (§22.3) or any read path.
- Because the contract is self-contained, solve reproducibility (§20.2) reduces to storing the
  request document and its fingerprint. This is a significant simplification and should be exploited
  rather than reinvented.
- If Phase 5 measurement contradicts this record, the replacement is a new ADR, and the contract
  boundary is what makes that affordable.
