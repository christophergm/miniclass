# 13. Guardian and volunteer access mechanics

- **Status:** Open — deliberately carried; resolved at the start of Phase 4
- **Date:** 2026-08-28
- **Relates to:** SPEC §13.9 and §24.2, both of which the specification itself leaves open
- **Related:** [0002](./0002-authentication-and-access-mechanisms.md),
  [0012](./0012-remove-the-household-entity.md)

## Context

This record carries the residue of [ADR 0006](./0006-household-and-volunteer-access.md), restated in
a household-free world. 0006 listed five open questions;
[ADR 0012](./0012-remove-the-household-entity.md) closed two of them by removing the entity they
were about. The remaining three are unchanged in substance and are renumbered here.

This is still not an implementation gap. The specification records the question as open in two
places (§13.9, §24.2) and lists what it has settled.

**Settled.** Partly by the specification, partly by 0012.

- **Links are adult-addressed.** 0006 leaned this way on the evidence; 0012 made it the only
  available shape, because there is no household to address. An emailed link authenticates a person.
- **The authenticated guardian view shows only the students that adult is a guardian of**, and has
  **exactly one shape** regardless of the viewer's volunteer role (§6.3). A guardian who also leads
  a class reaches class information through the class link, exactly as a leader with no children in
  the program does. Merging the two would make an authenticated view's contents vary by an unrelated
  role.
- **Preference records are bound to a specific student at the moment of creation, never by typed
  name** (§13.7). This is the single most important rule in the area and the one the predecessor
  violated at enormous cost (§3.2, §3.3). Nothing in the access design may weaken it.
- **A submission records who submitted it and when** (§13.7). Adult-addressed links are what make
  that attribution meaningful rather than nominal.
- **An adult may submit for all the students they guard in one sitting**, producing **per-student**
  records (§13.7).

**Not settled.**

1. **Delivery and renewal cadence.** On request, on a schedule, or per submission window. §9.5
   requires every link to expire and to be independently regenerable and revocable, which constrains
   the mechanism but does not choose the cadence.
2. **How does a non-guardian volunteer obtain access?** An external instructor guards no students at
   all, yet §15.3 requires that they be able to record per-meeting-date availability. This was the
   most constraining question in 0006 and it remains so: removing households did not touch it, since
   the volunteer's problem was never that they lacked a household — it was that they have nothing to
   be scoped *to*. It is now also the **only structural question left**, because the adult-addressed
   link resolved the others.
3. **What happens to an adult with no email on file.** §8.2 makes email required only "if the adult
   is to receive a magic link", so the roster legitimately contains adults who cannot be reached
   this way, and their students are then unreachable for preference submission.

## Decision

**Deferred to the start of Phase 4**, deliberately, and for the same reasons 0006 gave: nothing in
Phases 1–3 needs the answer, the answer is better made with the real domain model in front of us,
and the specification's own authors did not settle it. Question 2 in particular is a question about
what the program actually does with external instructors, not a question about software.

**What the model must preserve.** 0006 asked Phase 1 to model Adult as a first-class, year-scoped
entity with its own identity, independent of household membership. That constraint is now
structural rather than a discipline to be maintained: after 0012 there is no grouping an adult could
be subordinated to, and the guardian edge is a relationship by construction. Any resolution of the
three questions above is still available, and none of them costs a migration of the people tables.

**The counter-argument that survives from 0006.** An adult-addressed link asserts **identity** — it
says *you are this person* — and an emailed bearer token is thin evidence of identity. When
adult-addressing was a choice, that was its price and could be weighed against a household-addressed
alternative that claimed less. It is no longer a choice, so it is a cost the project simply carries.
The mitigations available are the ordinary ones §9.5 already requires: short expiry, regeneration
that invalidates the prior URL, and revocation. None of them turns a bearer token into proof of
identity, and the resolution of question 1 should be made in full knowledge of that.

## Consequences of deferring

- Phase 4 opens with a decision rather than with implementation. That is the intent; it is scheduled
  work, not a surprise.
- Question 2 is the schedule risk. A second access mechanism for non-guardian volunteers is more
  work than a cadence choice, and it is discovered at the point where it blocks §15.3.
- Questions 1 and 3 are operational as much as technical, and their answers may differ per
  organisation. Resolving them may produce configuration rather than a fixed mechanism.
- If Phase 4 arrives and question 2 is still genuinely open, it should be escalated to the programme
  organisers, where §24.5 already directs several related questions.
