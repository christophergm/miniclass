# 14. Roster ingest scope and source authority

- **Status:** Accepted
- **Date:** 2026-08-29
- **Relates to:** SPEC §11 (all), §10.1, §11.1, §15.2, §5.2, §20.1, §21.1, Appendix A.5
- **Related:** [0007](./0007-tenancy-enforcement-and-data-access.md),
  [0008](./0008-authorization-capabilities-and-audit.md),
  [0010](./0010-schema-generated-code-and-migration-conventions.md),
  [0012](./0012-remove-the-household-entity.md),
  [0013](./0013-guardian-and-volunteer-access.md)

## Context

Phase 2 implements SPEC §11. The specification describes ingest in the abstract — pluggable parsers,
a canonical shape, two-phase preview and commit, matching rules, idempotency — and is deliberately
silent about what any particular source document contains. This record exists because the actual
source was measured before the phase was decomposed, and what it contains changes several decisions
that would otherwise have been made on general principle.

There are two sources, and they are not alike.

**The community-platform export.** One JSON document, a flat array of adult accounts with each
adult's children nested inline. It is the wide format of §11.4, produced annually without being
asked for. Measured:

| Property | Observed |
|---|---|
| Adult records | 324, **each with a unique opaque identifier and a unique email** |
| Distinct children | 247, **each with a unique opaque identifier** |
| Guardian edges | 381 |
| Children nested under more than one adult | 129, **none of which disagree** on name or groups |
| Children with a classroom reference | **185** |
| Children with no classroom reference | **62** |
| Classrooms | 8, in 5 grade bands, one school |
| Children per classroom | 25, 25, 25, 25, 24, 21, 21, 19 |
| Adults with no name | 42, **all of them invitations never accepted** |
| Adults with no children | 49 |
| Adults with a phone or postal address | **1 of 324** |
| **Grade** | **absent — the source carries no grade field at all** |
| Households | **absent — no household object, no household key** ([ADR 0012](./0012-remove-the-household-entity.md)) |

Three of those rows are load-bearing.

**The export is external-identifier-complete.** Every adult and every child carries a stable opaque
identifier, and repeated child records are byte-consistent. §11.6's first matching rule — "If the
row carries one and it matches, that is the match. No further comparison is performed" — is
therefore sufficient for the entire document, and rule 2's normalised-name comparison is never
reached. This matters because it makes §11.7 idempotency a dictionary lookup against an index Phase
1 already built, rather than the expensive, ambiguity-generating machinery it would be for a
name-keyed source. Idempotency was going to be deferred; the data made deferring it more expensive
than doing it.

**The export carries homeroom but not grade.** A child's classroom reference yields a homeroom. Its
band label — a string of the form `3rd-4th Grade` — *bounds* the grade to at most two values but
does not state it. Grade arrives separately, from a two-column CSV produced by the school office,
matched by name. So the roster is loaded from one source and completed by another, on different
timelines, and there is necessarily an interval in which a student has a homeroom and no grade.

**A quarter of the children are not students.** 62 of 247 have no classroom. None of them has a
recoverable classroom, and only 16 have a parent holding any classroom role at all. The remaining
185 form an even eight-classroom roster. The absence is not dirty data: these are alumni, younger
siblings, departed families and administrative accounts — people with a school account and no seat.

Phase 2's scope was also deliberately reduced by the operator. The full §11 engine — CSV roster
parsing, name-matched roster rows, interactive conflict resolution — is not built here. This record
fixes which parts are built, which are deferred, and what each narrowing costs.

## Decision

**1. Two import kinds.** `roster_json` consumes the community-platform export as the §11.4 wide
format. `grades_csv` consumes a two-column CSV of student name and grade. §11.3's requirement of "at
minimum delimited text (CSV)" and "at least one structured document format (JSON)" is met by the
pair; neither format is deferred. A CSV *roster* parser is not built, and no roster source without
external identifiers is supported.

**2. Roster matching is by external identifier only** (§11.6 rule 1). Every record in the source
carries one, so no further comparison is performed. `grades_csv` has no identifier available and is
the one place rule 2 is reached; see Decision 9.

**3. Enrolment filter.** A source child with no classroom reference is excluded before preview, as
not enrolled. Exclusions are reported with counts, never silent.

**4. Adult filter.** An adult is imported when both names are present **and** they are a guardian of
at least one enrolled student. This keeps 226 of 324 adults and 303 guardian edges, and **every one
of the 185 enrolled students retains at least one guardian**. The 98 exclusions are reported in
three buckets: no name, no children, children but none enrolled.

**5. A wide row's authority is literal** (§11.4). Committing an adult's row sets exactly that
adult's guardian edges, adding and removing as required, and never touches an edge belonging to
another adult. Removal is scoped by nothing further: an edge the document does not name is removed
regardless of how it came to exist.

**6. Grade and participation intent become nullable.** §10.1 requires the grade vocabulary to be
*ordered* and requires every student to have exactly one *homeroom*; it does not require grade to be
present. §15.2 makes participation intent a declared survey answer. Neither has a source in the
export, and null means *not yet known*, which is a different fact from any value the vocabulary can
hold. `students.homeroom_id` stays `not null`.

**7. Classroom band labels are displayed, never parsed.** The band is carried through to the preview
as context for the organiser mapping classrooms to homerooms, and nothing derives from it.

**8. Preview and commit are stateless.** Preview returns a classification and a content hash of the
submitted document; commit takes the document and the hash and refuses on a mismatch. No import
batch or import row is persisted. §20.1's requirement is an audit entry with per-outcome counts,
which is written inside the mutating transaction.

**9. `grades_csv` matches on the whole name string and never splits it.** The normalised cell is
compared against the student's normalised legal full name, then against preferred given name plus
legal family name. Normalisation is §11.6's — case-insensitive, surrounding whitespace removed,
internal whitespace collapsed. More than one match is a `Conflict` and is refused. The kind is
**update-only**: a row matching no student is reported and never created, because a created student
would have no homeroom.

**10. §11.5's individual conflict resolution is deferred.** `Conflict` records are reported and
skipped. The organiser corrects through the manual CRUD §11.2 requires and re-imports, which is safe
because import is idempotent. **Revisit trigger:** the preferences import of §13.8, in Phase 4.

**11. A match against a soft-deleted person is a `Conflict`** — reported with the deletion date,
skipped, and never automatically restored.

**12. A field the source does not assert is never written.** Preferred name, phone, grade and
participation intent survive re-import untouched. For the fields a source does assert, the source
wins.

**13. Homerooms are keyed by external identifier.** A nullable, per-organisation-unique
`external_identifier` is added to `homerooms`, and classroom resolution joins on it. Grade levels
resolve by normalised code or label, because a two-column CSV offers no opaque key. The rule is:
join on the source's opaque identifier where it provides one; resolve by label only where it does
not.

**14. Import never creates a vocabulary entry** (§10.2). An unresolved classroom or an unrecognised
grade label is a row `Error` naming what is missing, so the organiser creates it and previews again.

**15. Households are not imported**, there being no source for them
([ADR 0012](./0012-remove-the-household-entity.md)).

**16. §11.3's SHOULD to materialise a parsed source to CSV before import is not implemented.** The
preview is mandatory and operates on normalised records, so human review is guaranteed without it.

## Alternatives considered

**A one-time additive load with no matching and no idempotency.** Rejected, and this was the
operator's initial preference, reversed on evidence. It removes less than it appears to: intra-file
duplicate detection is still required, because a household survey export contains resubmissions. It
contradicts §11.7, which is a MUST driven by an observed workflow — the organiser re-exports as
families respond. And its escape hatch closes at Phase 3: additive-only is survivable while nothing
references a person, so a bad import can be recovered by deleting the year and reloading, but from
programme membership onward that option is gone. Set against an estimated 40–45 households of manual
re-entry per re-export, and against the fact that this source's external identifiers make matching a
dictionary lookup, the saving was not real.

**A fully generic multi-entity ingest pipeline up front.** Rejected as premature in its strong form
and adopted in its weak one. The generalisation that pays is the *envelope* — kind registry, the
two-phase protocol, the preview shape, the audit entry — which has two customers in this phase and a
third in §13.8. The generalisation that does not is graph resolution: intra-batch reference
resolution, local-key symbol tables and dependency ordering are roster-specific, and preferences
have one record type and no edges. What must be generic from the start is the preview *shape*: a
record tree with a roll-up, because one wide row is simultaneously `Update`, `Unchanged` and
`Create` for different records, and a flat per-row status cannot represent it. That is the expensive
thing to retrofit; the rest is not.

**Persisted import batches and rows.** Rejected. The feature that needed durable rows was
interactive per-row conflict resolution, which Decision 10 defers. §20.1 requires an audit entry,
not a batch table. Two new tenant-scoped tables are not free: ADR 0007's schema meta-test requires a
registry factory and generated isolation tests for each, so the cost is three artifacts per table
for a workflow that is not being built. The content hash closes the only real gap statelessness
leaves.

**Provenance on the guardian edge, to protect hand-made corrections from Decision 5.** Considered at
length and rejected in favour of simplicity, on the operator's call. An `origin` column of `import`
or `manual` would let an import remove only the edges it created, satisfying both halves of §11.4's
intent. It is one column. It was rejected because it is a concept the specification does not name,
and because §11.4's mandatory removal listing in preview already surfaces the loss. The cost is real
and is recorded under Consequences.

**Validating the CSV grade against the classroom's band label.** Rejected. Every band spans at most
two grades, so the band is arithmetically a checksum on the name match, and the name match is the
one place in Phase 2 where the predecessor's defining failure (§3.3, Appendix A.5 defects 4–5) can
recur. It was still rejected, because deriving `{3, 4}` from the string `3rd-4th Grade` encodes a
third party's naming convention in our validation logic and fails silently the year a room is
renamed. Modelling permitted grades on `homerooms` was the alternative form and was rejected for the
same reason plus a second: §10.1 describes no such relationship, so it would be invention.

**Nullable homeroom, or a sentinel homeroom for the 62.** Rejected. §10.1 requires exactly one
homeroom per student because the dismissal list pivots on it, so nullable contradicts the
specification and would push a null into every artifact §18 publishes. A sentinel puts 62
non-participants into the dismissal list as a section, which is worse than omitting them.

**Treating the 62 classroom-less children as row `Error`s.** Rejected. `Error` blocks the entire
commit (§11.5), so a quarter of the document would prevent the other three quarters from loading,
over records that are correctly formed and simply describe people who are not students here. That
inverts §5.2.

**Resolving homerooms by name.** Rejected. The source provides an opaque identifier, so joining on
the label would discard it in favour of a name — standing rule 7 and §8.7 — and would break for
every child in a renamed room.

**Importing the 12 named adults with no children.** Rejected, marginally. They are plausibly staff
or committee members who might later lead a class, but §21.1 enumerates the personal data the system
holds and there is currently no role for them to occupy; ADR 0013 carries non-guardian volunteer
access as an open question. Their email addresses appear in the preview's exclusion report, so
adding back the two or three who matter is a minute's manual work.

**Defaulting participation intent to `unavailable` rather than making it nullable.** Rejected. It
would fabricate 226 declarations nobody made, which §15 and §18 staffing would later read as data —
"no volunteers available" when the truth is "nobody has been asked". `help` errs the other way.

**Automatically restoring a soft-deleted person who reappears in the export.** Rejected. It would
make deletion ineffective for as long as the person remains in the source, and standing rule 5 holds
that an automated process does not silently overturn a recorded human decision. The partial unique
index makes the naive alternative worse than either: it excludes soft-deleted rows, so an unguarded
re-import creates a second record for the same child with no constraint violation and no warning.

## Consequences

- **A guardian edge added by hand to an adult who appears in the export is removed by the next
  re-import.** This is the direct cost of Decision 5 and it is not mitigated, only surfaced: §11.4
  requires the preview to list removals, so the organiser can see it before committing. It is
  written down here, and asserted by a test, so that the behaviour is deliberate rather than
  discovered. If it bites in practice, edge provenance is the remedy and is already specified above.
- **The preview is the only control on the `grades_csv` name match.** Band validation was rejected,
  the CSV carries no identifier, and §11.6 forbids resolving similarity automatically. Legibility of
  the preview is therefore a functional requirement, not presentation: it is the mechanism by
  which a mis-join is caught.
- **A student may exist with no grade, and the roster is incomplete by design between two imports.**
  §11.1's `Setup` state exists for exactly this. Nulls are quarantined at programme membership
  (§12.1), which is the chokepoint that keeps them out of the catalog, the engine and every
  published artifact; that gate is Phase 3 work and is tracked separately.
- **`PLAN.md`'s Phase 2 exit criterion contradicted `AGENTS.md` and is amended.** The criterion
  required a *real* historical roster export to import cleanly; the standing rule forbids loading
  real roster data into a development or test database. The criterion is replaced by three parts:
  automated tests run against a committed synthetic corpus; an opt-in parser conformance check reads
  a real export, touches no database, and asserts aggregate counts only; and "a real export imports
  cleanly" becomes an operator demonstration against their own instance, evidenced by the audit
  entry.
- **Several fixture cases cannot be derived from the real export, because it is too clean.** It
  contains no contradictory repeated child, no two enrolled children sharing a normalised name, and
  no unresolvable classroom. Those paths exist only under synthetic fixtures, which means the
  fixtures are not a convenience — they are the sole coverage for three refusal behaviours.
- **The enrolment and adult filters are inferences about a source, encoded in code.** If a future
  export changes such that an enrolled child can lack a classroom, the importer silently drops real
  students. The mitigation is that exclusions are reported with counts: 62 of 247 is a number an
  organiser will recognise, and 185 of 247 would stop them. This is a genuine judgement about a
  third-party document and is the most likely part of this record to need revisiting.
- **Re-import is safe to repeat, which is what makes deferring conflict resolution tolerable.** The
  §11.5 gap in Decision 10 is only survivable because §11.7 holds. Those two decisions must be
  revisited together.
- **`preferred_given_name` is never populated by import.** The source has no such field. Preferred
  names are entered by hand and, per Decision 12, survive re-import.
- **Adult phone and postal address are effectively never populated.** One record of 324 carries
  either. Any later feature that assumes a reachable phone number should assume email instead.
