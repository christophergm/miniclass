# 12. Remove the Household entity

- **Status:** Accepted
- **Date:** 2026-08-28
- **Supersedes:** [0006](./0006-household-and-volunteer-access.md)
- **Relates to:** SPEC §8.2, §11.4, §18.6, §19.5
- **Related:** [0002](./0002-authentication-and-access-mechanisms.md),
  [0010](./0010-schema-generated-code-and-migration-conventions.md),
  [0013](./0013-guardian-and-volunteer-access.md)

## Context

SPEC §8.2 models two family constructs and is careful to keep them apart: **Household**, "a grouping
of adults and students used for preference submission scope, magic-link addressing, and sibling
reasoning", and **Guardian relationship**, an adult-to-student edge with a type, "distinct from
household membership, because the two do not always coincide".

They are not equally sourced, and that is the whole of the matter.

**The guardian edge has a source.** The wide import format of §11.4 — one row per adult with that
adult's children named inline — is literally a record set of adult-to-student guardian
relationships. It is what the reference program already produces, every year, without being asked
to.

**The household has none.** No source system emits a household; the survey does not ask for one; the
predecessor never held one. The grouping of *adults with each other* was, in every version of this
design, an inference the project cannot make from any data it has. It would be produced by an
organiser deciding which adults belong together and re-deciding it annually, which is the kind of
manual reconciliation §5.6 exists to refuse.

Household is therefore a modelled entity with a populated schema
(`backend/migrations/20260825100000_households.sql`: `households`, `household_students`,
`household_adults`) and no faithful way to fill it. That is a defect discovered late rather than a
feature not yet built.

Household was nevertheless carrying four jobs. Removing it requires discharging all four:

| Job Household held (§8.2) | Discharge |
|---|---|
| Preference submission scope | Derived from the guardian edge: the students that adult guards |
| Magic-link addressing | Adult-addressed links — now forced by the data model, not merely preferred |
| Sibling reasoning | Already redundant: §10.6 states pairings "replace … sibling co-placement" |
| §19.5 follow-up grouping | Grouped by guardian |

Only the fourth changes observable behaviour, and it changes it in a way worth stating plainly; see
Consequences.

## Decision

**Household is removed from the domain model.** The decision is the operator's; this record exists
to fix its shape and its costs.

**1. The three tables are dropped** — `households`, `household_students`, `household_adults` — with
their policies, indexes, triggers, registry entries and isolation tests. The drop is a new
migration, never an edit to the merged one
([ADR 0010](./0010-schema-generated-code-and-migration-conventions.md)).

**2. `guardian_relationships` survives unchanged** and becomes the sole family construct: an adult,
a student, and a `guardian_relationship_type` of `parent`, `guardian`, `grandparent` or `other`. Its
composite foreign keys are already as [ADR 0007](./0007-tenancy-enforcement-and-data-access.md)
requires.

**3. No replacement entity is introduced.** Family scope is derived at read time as *the students
this adult is a guardian of*. It is never stored, never named, and never given an identifier. This
is the load-bearing half of the decision: the entity is not being renamed, it is being deleted,
because what was missing was the source data and not the label.

**4. Vocabulary follows the model.** The persona "Household guardian" becomes **Guardian**; the
§6.6 role scope "Own household" becomes **Own students**; the §18.6 "Household placement view"
becomes the **Guardian placement view**, showing the students that adult guards.

**5. The `access_token_purpose` enum value `household_submission` is renamed to
`guardian_submission`**, by `alter type … rename value` in a new migration.

**6. A wide import has adult-scoped authority.** A wide-format row sets *exactly* the guardian edges
of the adult on that row, and never touches an edge owned by another adult. This is what keeps §11.4
honest once households are gone; see Consequences.

## Alternatives considered

**Keep households and populate them by hand.** Rejected. That is roughly 90 households re-entered
every year, with no source to reconcile against and no way to tell a stale grouping from a correct
one. §5.6 already loads each year fresh, so the hand-entry does not amortise — it recurs. A
hand-maintained grouping is precisely the manual step this system exists to remove, and building the
system in order to reintroduce it is not a trade, it is a loss.

**Rename Household to `Family` or `GuardianGroup`.** Rejected. Same entity, same missing source
data, plus the cost of the rename. A name change does not create a record of who lives with whom;
it only makes the absence harder to notice, because the new term arrives without the accumulated
knowledge that the old one could not be filled.

**Infer households from shared surname, address or email domain.** Rejected. §11.6 forbids merging
records on similarity alone, and says why: character-for-character and look-alike joins are the
predecessor's defining failure. Inferring a *grouping* from similarity is the same error one level
up, and it fails on exactly the population §8.2 states is not an edge case. Separated families
routinely differ in surname, address and email domain, and two unrelated families routinely share a
surname. The inference would be most confident where it is most wrong.

**Keep the tables, unused, in case a source appears.** Rejected. An empty modelled entity is
indistinguishable from a broken one: the next reader cannot tell "no households were ever created"
from "household creation is failing". It also is not free. The schema meta-test requires a registry
entry and isolation tests for every tenant-scoped table
([ADR 0007](./0007-tenancy-enforcement-and-data-access.md)), so three tables nothing writes still
carry three factories, three fetches and a generated suite asserting the isolation of rows that do
not exist. If a real source of household data ever appears, it arrives with a migration, and that
migration is cheaper than the standing cost of pretending.

## Consequences

- **§19.5 response tracking no longer partitions the student set.** Grouping by guardian means a
  student with two guardians appears under both. For the report's stated purpose — a single
  follow-up that covers a family, and targeting non-responders — this is correct and arguably
  better, since both adults get chased. But **any count taken over that report double-counts**, and
  a distinct-student figure must be computed separately from the grouped listing. This is written
  down so that nobody later reads the duplicate as a bug and "fixes" the grouping.
- **A student with no guardian relationship is invisible to every guardian-scoped surface.** They
  cannot be reached for preference submission and do not appear on any guardian placement view. Per
  §5.2 this surfaces as a **roster warning** attached to the student, never as a validation that
  refuses the roster or the import. The organiser is told; the organiser is not blocked. Households
  hid this failure mode behind a second layer of indirection rather than preventing it.
- **§11.4's known limitation becomes moot as stated, and the hazard relocates.** "The wide format
  cannot express a student with guardians in two households" is not a sentence that can be written
  once there are no households — but the underlying risk, that importing one adult's row silently
  destroys another adult's relationship to the same child, survives the entity that described it.
  Adult-scoped authority (Decision 6) is the resolution: two adults' rows compose into a
  two-guardian student, because neither row is authoritative over the other's edges, while
  re-importing one adult's row with a child removed still removes that edge. The cost lands on
  §11.5: the `Update` preview MUST list guardian edges being **removed**, not only those added.
  Removal is now the destructive operation an import can perform, and §11.5 already requires that a
  preview show what will change field by field.
- **The enum rename is not cosmetic.** Leaving `household_submission` live would put a term in the
  schema that §23.2 records as abandoned, and a live identifier is a standing invitation to
  reintroduce the concept it names. That contradiction — glossary says gone, schema says here — is
  exactly how a removed concept comes back.
- **This closes ADR 0006's open questions 1 and 2 by elimination, not by answering them.** Question
  1, whether a link addresses a household or an individual adult, has one remaining option: there is
  no household to address. Question 2, adults belonging to two households, dissolves — an adult with
  children in two families is an adult with several guardian edges, which the model already holds
  without ambiguity. 0006's provisional lean toward adult-addressed links is now a structural
  consequence rather than a judgement call.
- **0006's questions 3, 4 and 5 are not resolved here.** Delivery and renewal cadence, access for a
  non-guardian volunteer, and an adult with no email on file are untouched by this decision and are
  carried to [ADR 0013](./0013-guardian-and-volunteer-access.md). Removing households made none of
  them easier.
- **The domain loses a concept people use in conversation.** Organisers say "household", and the
  system will not. The glossary carries the term as abandoned (§23.2) so that the mismatch is
  documented rather than rediscovered.
