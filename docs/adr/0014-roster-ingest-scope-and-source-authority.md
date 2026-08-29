# 14. Roster ingest scope and source authority

- **Status:** Accepted
- **Date:** 2026-08-29
- **Relates to:** SPEC §5.2, §10.1, §11, §15.2, §20.1, §21.1
- **Related:** [0007](./0007-tenancy-enforcement-and-data-access.md),
  [0008](./0008-authorization-capabilities-and-audit.md),
  [0010](./0010-schema-generated-code-and-migration-conventions.md),
  [0012](./0012-remove-the-household-entity.md)

## Context

Phase 2 needs to turn the reference program's recurring roster exports into the canonical records
described by SPEC §11.4. The specification requires both delimited text and a structured document
format (§11.3), but it does not require a universal import framework. The observed source is a
community-platform export in the wide format: one row per adult, with that adult's students inline.
The source contains external identifiers for every observed person, while its classroom labels are
third-party display labels rather than this system's grade vocabulary.

The observed export contains 247 children, of whom 62 have no classroom reference. Only 16 of those
62 have a parent holding any classroom role; the other 185 children form an even eight-classroom
roster (25/25/25/25/24/21/21/19). It contains 324 adults, of whom 226 have both names and guard at
least one enrolled student. Those retained adults produce 303 guardian edges, and every one of the
185 enrolled students retains at least one guardian. These figures establish the observed source
boundary; they are not a general rule for inventing missing data.

The project must also remain faithful to the privacy and audit rules. A person with no role in the
programme should not be retained merely because they appeared in a source (§21.1), exclusions must
be visible rather than silently discarded (§5.2), and an import must be attributable with outcome
counts (§20.1). The plan's former requirement to load a real historical export into a test database
contradicts the repository rule to keep real roster data out of development and test databases.

## Decision

Phase 2 is deliberately scoped to two source kinds:

1. **`roster_json`** consumes the community-platform export in the wide format of §11.4.
2. **`grades_csv`** consumes a two-column CSV containing student name and grade.

Together these satisfy both the CSV and JSON minimums in §11.3. Parsers resolve fields by name or
explicit mapping, never by position, and translate only into the canonical student, adult and
guardian-relationship shape. A fully generic multi-entity pipeline is not part of this phase.

### Source authority and filters

- Matching is by external identifier only (§11.6 rule 1). Every observed source record carries one,
  so the rule's “no further comparison is performed” applies. Name matching (§11.6 rule 2) is not a
  fallback in Phase 2. A future source without external identifiers is out of scope.
- A source child without a classroom reference is not enrolled and is excluded before preview. The
  exclusion is reported with its reason and is never silent. This is the observed source's
  enrolment boundary, not a validation that blocks an organiser (§5.2).
- An adult is imported only when they have both names and guard at least one enrolled student. The
  98 observed exclusions are reported in reason buckets. This applies §21.1: the system does not
  hold personal data for an adult with no role in the programme.
- A wide row has literal adult authority (§11.4): it sets exactly that row's adult's guardian edges.
  It never changes another adult's edges. A hand-added edge for an adult present in the export is
  therefore removed by the next import if that row no longer names the student. The mandatory
  removal listing in the §11.5 preview is the mitigation. An `origin` column is not introduced.
- Grade is nullable. The ordered grade vocabulary and exactly-one homeroom requirement in §10.1 do
  not require a grade to be present, and `roster_json` carries no grade. Missing grade is a
  `Setup`-state condition (§11.1), quarantined at programme membership in Phase 3 (§12.1), not an
  import error.
- Participation intent is nullable. It is a declared survey answer (§15.2), not a roster fact;
  defaulting the 226 retained adults to `unavailable` would fabricate declarations that staffing
  later reads as data.
- Classroom band labels are displayed, never parsed. Deriving grades from a label such as
  `3rd-4th Grade` would import a third party's vocabulary into validation and fail silently when a
  room is renamed.
- Households are not imported. No source provides them, and ADR 0012 removes the entity; guardian
  relationships are the sole sourced family construct.

### Import lifecycle and deferred conflict resolution

Import remains stateless preview → commit, guarded by a content hash. There is no `import_batches`
table. The preview is the durable audit boundary: §20.1 requires per-outcome counts, while ADR 0007
would make a new tenant-scoped batch table carry registry and isolation obligations. Commit is
atomic and idempotent under §11.7.

`Conflict` rows are reported and skipped. The §11.5 requirement that conflicts be resolvable
individually is knowingly deferred: organisers correct the affected records through manual CRUD
(§11.2) and re-import, which is safe because an unchanged source produces `Unchanged` outcomes
(§11.7). This is revisited when preferences import arrives in Phase 4 (§13.8), which is the trigger
for deciding whether individual conflict resolution is worth the added workflow.

## Alternatives considered

- **A fully generic multi-entity pipeline.** Rejected for Phase 2. It would obscure the two observed
  source contracts and multiply validation and authority rules before another source justifies them.
  The canonical shape remains the extension point (§11.3).
- **Persisted import batches.** Rejected. Stateless, content-hash-guarded preview and an atomic
  commit provide the needed safety; a batch table would add tenant-scoped state and isolation cost
  without a requirement in §20.1.
- **Edge provenance on `guardian_relationships`.** Rejected. Literal adult authority already makes
  ownership unambiguous, and an `origin` column would complicate deletion and manual CRUD without
  changing the required preview disclosure (§11.4–§11.5).
- **Band-derived grade validation.** Rejected. Display labels are not the ordered vocabulary of
  §10.1 and parsing them would turn a renamed classroom into a silent semantic change.
- **Importing the 12 named childless adults.** Rejected. They have no role in the programme under
  the observed source and retaining them conflicts with §21.1. Their exclusions remain reported.
- **A one-time additive load with no idempotency.** Rejected. Repeated partial imports are the
  normal workflow; §11.7 requires unchanged re-imports to produce no changes, and the wide format
  must also be able to report and apply relationship removals (§11.4–§11.5).

## Consequences

- Phase 2 has two concrete conformance targets and a narrow, testable authority model. Adding a
  source without external identifiers, or a source that needs name matching, requires a new decision.
- Conflicts require an organiser's manual correction and a re-import. The system does not silently
  choose, and it does not block all other valid rows merely because one row is unresolved; errors
  remain subject to the §11.5 commit rule.
- The import can remove a guardian edge, so preview output must make removals prominent. The next
  import is authoritative for the represented adult, while hand-entered corrections for that adult
  are not durable against a later source re-import.
- Missing grades and participation intent remain honest unknowns. Downstream phases must handle
  those nulls and show the appropriate setup or survey state rather than infer values.
- Real historical data may be used only in an opt-in parser conformance check that touches no
  database, or in an operator demonstration against the operator's own instance. CI uses synthetic
  fixtures and must never load the historical roster into a development or test database.

