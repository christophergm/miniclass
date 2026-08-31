# 0. Record architecture decisions

- **Status:** Accepted
- **Date:** 2026-08-23

## Context

The project has a large normative specification ([`SPEC.md`](../../SPEC.md)) that is deliberately
technology-agnostic: it describes *what* the system does and explicitly leaves implementation to the
implementer, marking several choices `Implementation-defined` and requiring that "implementations
MUST document the choice they make".

Up to this point, architecture and tooling decisions have been recorded as narrative documents
written at a moment in time — `achitecture.md`, `STRUCTURE.md`, `TOOLING_AUDIT.md`,
`DOCKER_UPGRADE.md`, `IMPLEMENTATION_PLAN.md`, `QUICK_START.md`. Within weeks several of them were
already wrong: they described the migrations directory as empty, the project as "scaffolding,
implementation needed", and each other as authoritative. They recorded *what was done* but not *why*,
or *what was rejected*, which is the part that matters when the decision is revisited.

Much of the implementation is carried out by agents working in parallel worktrees. Agents need a
short, stable, machine-readable record of settled decisions far more than they need narrative prose,
and they need to be able to tell a settled decision from an open one.

## Decision

Architecture decisions are recorded as numbered, immutable Architecture Decision Records in
`docs/adr/`.

- One decision per file, named `NNNN-short-title.md`, numbered sequentially and never renumbered.
- Each record carries a **Status** of `Proposed`, `Accepted`, `Open`, `Deprecated`, or
  `Superseded by NNNN`.
- Records are **not edited to reflect a change of mind**. A changed decision is a new record that
  supersedes the old one; the old record's status is updated to point at the new one and is
  otherwise left intact.
- A record states the context, the decision, the alternatives considered and why they were rejected,
  and the consequences — including the bad ones.
- Where a decision resolves something the specification marks `Implementation-defined`, the record
  cites the spec section. This is how the project discharges that obligation.

`Open` is a first-class status. Some decisions are deliberately deferred, and the reason for
deferral and the point at which they must be resolved are worth recording as carefully as a
settled decision.

Point-in-time narrative documents are not maintained. Operational instructions live in `README.md`;
sequencing lives in [`PLAN.md`](../../PLAN.md); behaviour lives in `SPEC.md`.

## Alternatives considered

**Continue with narrative architecture documents.** Rejected: demonstrably rots, records outcome
without rationale, and gives agents no way to distinguish settled from provisional.

**Record decisions only in the specification.** Rejected: the specification is deliberately
technology-agnostic and is a description of the system's behaviour, not of the project's engineering
choices. Mixing the two would make it harder to tell which parts are requirements.

**Record decisions in pull request descriptions and commit messages.** Rejected: not discoverable,
and not readable as a set.

## Consequences

- Reading `docs/adr/` in numerical order explains how the system came to be shaped as it is,
  including the paths not taken.
- Reversing a decision costs a new file. This is intentional friction.
- `README.md` and `PLAN.md` link to ADRs rather than restating them, so there is one place to change.

## Index

| ADR | Title | Status |
|---|---|---|
| [0001](./0001-application-stack-and-topology.md) | Application stack and topology | Accepted |
| [0002](./0002-authentication-and-access-mechanisms.md) | Authentication and access mechanisms | Accepted |
| [0003](./0003-assignment-solver-technology.md) | Assignment solver technology | Accepted |
| [0004](./0004-api-contract-and-type-generation.md) | API contract and type generation | Accepted |
| [0005](./0005-published-artifact-availability.md) | Published-artifact availability and topology | Accepted |
| [0006](./0006-household-and-volunteer-access.md) | Household and volunteer access mechanics | Superseded by 0012 |
| [0007](./0007-tenancy-enforcement-and-data-access.md) | Tenancy enforcement and data access | Accepted |
| [0008](./0008-authorization-capabilities-and-audit.md) | Authorization, capabilities and the audit log | Accepted |
| [0009](./0009-administrator-sessions-and-identity-provider.md) | Administrator sessions and identity-provider choice | Accepted |
| [0010](./0010-schema-generated-code-and-migration-conventions.md) | Schema, generated code and migration conventions | Accepted |
| [0011](./0011-local-development-orchestration-and-environment-contract.md) | Local development orchestration and environment contract | Accepted |
| [0012](./0012-remove-the-household-entity.md) | Remove the Household entity | Accepted |
| [0013](./0013-guardian-and-volunteer-access.md) | Guardian and volunteer access mechanics | Open |
| [0014](./0014-roster-ingest-scope-and-source-authority.md) | Roster ingest scope and source authority | Accepted |
| [0015](./0015-year-scoped-attribute-vocabularies.md) | Grade and homeroom vocabularies are scoped to the school year | Accepted |
