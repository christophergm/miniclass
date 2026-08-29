# 6. Household and volunteer access mechanics

- **Status:** Superseded by [0012](./0012-remove-the-household-entity.md)
- **Date:** 2026-08-23
- **Relates to:** SPEC §13.9 and §24.2, both of which the specification itself leaves open
- **Related:** [0002](./0002-authentication-and-access-mechanisms.md)

## Context

This is not an implementation gap. The specification records it as an open question in two places
(§13.9, §24.2) and lists what has been settled and what has not.

**Settled by the specification:**

- Households authenticate by emailed link, not by password.
- The household view shows household data only and always has the same shape (§6.3).
- Preference records are **bound to a specific student at creation, never by typed name** (§13.7) —
  the single most important rule here, and the one the predecessor violated at enormous cost.
- A household may submit for all its students in one sitting, producing **per-student** records.
- Submissions record who submitted and when.

**Not settled:**

1. Does a link address a **household** or an **individual adult**? Per-adult identity is needed to
   attribute a submission to a person and to hold per-adult availability (§15.3).
2. How are **adults belonging to two households** handled? §8.2 makes this explicitly not an edge
   case — separated families are normal.
3. Delivery and renewal cadence: on request, on a schedule, or per submission window.
4. **How does a non-guardian volunteer obtain access?** They have no household at all, yet §15.3
   requires that they be able to record per-meeting-date availability.
5. What happens to households with no email on file.

Question 4 is the one that most constrains the answer. A purely household-addressed link cannot serve
a volunteer who is not a guardian, so either a second mechanism is needed or the link is
adult-addressed with household-scoped visibility.

## Decision

**Deferred to the start of Phase 4**, deliberately.

The reasoning: nothing in Phases 1–3 needs the answer, the answer benefits from having the real
domain model in front of us, and the specification's own authors did not settle it. Forcing it now
would be guessing with less information than we will have then.

**What Phase 1 must do to keep the option open.** Phase 1 models Adult as a first-class,
school-year-scoped entity with its own identity, independent of household membership, and models
household membership as a relationship rather than as a property of the adult. This is what §8.2
requires anyway. As long as that holds, both resolutions remain available and neither costs a
migration.

**Provisional lean, recorded so it can be argued with rather than inherited silently.** The evidence
in the specification points at **adult-addressed links with household-scoped visibility**: it is the
only shape that answers questions 1, 2 and 4 with one mechanism. §13.7's requirement that submissions
record *who* submitted needs a person; §15.3's non-guardian volunteer has no household; and an adult
in two households can be granted visibility of both without ambiguity about who they are. The
counter-argument is that it makes the link a weaker credential for a stronger claim — an
adult-addressed link asserts identity, and an emailed bearer token is thin evidence of identity.

## Consequences of deferring

- Phase 1 carries a constraint it would probably have honoured anyway.
- Phase 4 opens with a decision rather than with implementation. That is the intent; it is scheduled
  work, not a surprise.
- If Phase 4 arrives and the question is still genuinely open, it should be escalated to the
  programme organisers, where §24.5 already directs several related questions.
