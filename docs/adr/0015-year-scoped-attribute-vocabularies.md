# 15. Grade and homeroom vocabularies are scoped to the school year

- **Status:** Accepted
- **Date:** 2026-08-30
- **Relates to:** SPEC §5.6, §8.1, §8.2, §10.1, §11.1, §20.1, §24.5
- **Related:** [0007](./0007-tenancy-enforcement-and-data-access.md),
  [0010](./0010-schema-generated-code-and-migration-conventions.md),
  [0012](./0012-remove-the-household-entity.md),
  [0014](./0014-roster-ingest-scope-and-source-authority.md)

## Context

SPEC §8.1 places people under the school year, and says why: "A student record describes a child *in
a given year* — in grade 4, in Serena's homeroom, with these tags. Next year is a new record. This
makes historical data permanently correct."

§10.1 then placed the two vocabularies those facts are drawn from under the **organization**. Grade
and homeroom are concrete fields rather than tags because each carries semantics a tag cannot — grade
is ordinal, because every offering's eligibility rule is a range (§8.4); homeroom is
categorical and single-valued, because the dismissal list pivots on it (§18.4). Both therefore need
managed, identified value sets rather than text columns. Those value sets were organization-scoped
and permanent.

The mismatch was known and written down. §24.5 question 13 states it exactly: "Homeroom vocabularies
are organization-scoped (§10.1) while people are year-scoped (§8.1), so a rename retroactively
changes the homeroom displayed for students in closed years — which §11.1 otherwise guarantees is
immutable." Its recorded resolution was a convention — retire and replace rather than rename — with
renaming "permitted but a known limitation".

Three properties of the as-built schema show the mismatch is wider than the rename it was noticed
through.

**Values are unique per organization for all time.** `grade_levels_code_idx`,
`grade_levels_ordinal_idx` and `homerooms_name_idx` are unique on `(organization_id, …)`. A school
that dissolves the Gold homeroom and later opens a new one cannot reuse the name, and a school whose
grade ladder changes cannot renumber it without renumbering every year that has already closed.

**The ordinal is a fact about an organization, not a year.** `ShiftGradeLevelOrdinals` is scoped to
the organization. Reordering the grade ladder renumbers every year in the tenant, including closed
ones, which is a silent rewrite of the ordering that a closed year's offerings were validated
against.

**The vocabulary escapes closed-year immutability.** The shared trigger
`prevent_closed_school_year_mutation` is attached to every table carrying a `school_year_id`. These
two tables carry none, so they have no guard, and a closed year's homeroom names are editable today.
§11.1 exists because the predecessor's history was mutable files edited by hand after the fact
(§3.3); this is the one place in the current schema where that is still possible.

The mismatch is also a functional limit on ingest. `homerooms_external_identifier_idx` is unique on
`(organization_id, external_identifier)`, so a classroom identifier from the source system can be
registered once per organization, ever. A source that reuses its classroom identifiers annually —
which is the normal behaviour of a school platform — cannot be imported into a second year.

Nothing yet depends on either table but `students.grade_level_id` and `students.homeroom_id`. Phase 3
introduces offerings with grade windows and program membership by grade range, at which point it
would.

## Decision

**Grade and homeroom vocabularies are scoped to the school year, not the organization.** The
decision is the operator's, on the ground that homerooms change from year to year and so may the
grades a school runs. This record fixes its shape and its costs.

**1. Both vocabularies move, not only homeroom.** `grade_levels` and `homerooms` each gain
`school_year_id`. Grade is the rarer case but the same case: a changed ladder retroactively alters
what a closed year's ordering was, and the ordinal is meaningless except as a fact about a
particular year's set of grades. Splitting the two scopes would also mean two accessor conventions,
two cache keys, two foreign-key shapes and two settings surfaces, for tables whose entire downstream
footprint is two columns on one table.

**2. The homeroom label stays on the organization.** §10.1 lets an organization choose the word for
the axis — `homeroom`, `class`, `form`, `advisory`. That is an institutional fact, not an annual one:
a school changes which homerooms exist every year and does not change what it calls them. The label
also sits in the identity layer, where organizations are created and validated
([0007](./0007-tenancy-enforcement-and-data-access.md) §4), a layer that by construction cannot
reach the domain. §10.1 therefore reads as two clauses: the organization configures the label, each
school year defines the value set.

The year-scoped read model serves the label alongside the values, because every surface rendering a
homeroom picker needs both. Serving it is not authority over it: the label is written through the
organization, and only there.

**3. References carry the year.** `students_grade_level_fk` and `students_homeroom_fk` become
three-column, `(id, organization_id, school_year_id)`, per
[0007](./0007-tenancy-enforcement-and-data-access.md) §5. The vocabulary tables' two-column unique
is replaced by the three-column one rather than supplemented, matching `adults` and
`guardian_relationships`; nothing but `students` references either table, so a two-column reference
has no user and removing its target makes writing one structurally impossible.

This is the load-bearing half of the decision. There is no `app.school_year_id` GUC — row-level
security enforces the organization only — so year isolation rests on the composite foreign key plus
an explicit year predicate in every statement. Ten of the fourteen vocabulary statements are
currently scoped by primary key or by RLS alone.

**4. Layer 1 is tightened to require what §5 of 0007 already states.** The meta-test presently
accepts *either* the two-column or the three-column unique on a table with a `school_year_id`, and
its foreign-key check asserts only that `organization_id` appears. Both become requirements: a table
with a live `school_year_id` column must declare the three-column unique, and a foreign key to a
table with a `school_year_id` column must carry it. `audit_log` remains exempt as it already is —
append-only, nothing's foreign-key target — and `students.prior_year_student_id` remains the one
named exception of 0007 §5. The rule keys on the presence of the column rather than on year-scoping,
because `school_years` is year-scoped and correctly has no `school_year_id` of its own.

**5. Closed-year immutability now covers the vocabulary.** The shared trigger is attached to both
tables, which is what the change buys: a closed year's grades and homerooms become read-only, and
the reopen path of §11.1 — Owner-only, reason required, audited — is the only way to alter them. The
refusal surfaces as the existing 409 `school-year-closed` problem, as it does for students, adults,
guardian relationships and imports.

**6. Retirement survives, and delete is not introduced.** Year-scoping handles the year boundary
directly: a homeroom that ceases to exist next year is simply absent from next year's vocabulary. It
does not handle the within-year case — a homeroom dissolved in November, with that year's students
still referencing it — which is what `retired_at` is for. Neither table gains a delete grant; an
entry mistyped during `Setup` is retired, and a conditional refusal to delete a referenced row would
be a new validation needing its own citation under §5.2.

**7. A new year's vocabulary is entered by hand.** No value is copied, derived or inherited from a
prior year. See Alternatives considered.

**8. §24.5 question 13 is resolved by dissolution rather than by convention.** Its premise is the
scope mismatch, which no longer exists. Within an open year a rename is a correction to that year's
own facts, permitted and audited. Across years there is no rename, because it is a different row. A
closed year cannot be edited at all, and that is now enforced by a trigger rather than by asking
organisers to prefer retirement.

## Alternatives considered

**Retire and replace rather than rename, as §24.5 question 13 originally resolved.** Rejected as
insufficient rather than wrong. It is sound advice that depends on organisers following it, and the
schema does not require it: the rename it discourages remains available, unguarded, and silently
rewrites closed years. It also addresses only the rename. It leaves permanent per-organization
uniqueness of names, codes and external identifiers, leaves the ordinal renumbering every year at
once, and leaves the vocabulary outside closed-year immutability. A convention that covers one of
four symptoms of a scope mismatch is a reason to fix the scope.

**Move homeroom only, leaving grade an organization-level ladder.** Rejected. Grade ladders change
rarely, but the ordinal is unique per organization for all time, so one mid-ladder insertion
renumbers every closed year — the same defect as the rename, on the field where ordering is
load-bearing because offering eligibility is a range (§8.4). The saving would be nil and the cost two
divergent conventions for one concept.

**Move the homeroom label to the school year with the values.** Rejected. The label names the axis
and the correction concerns the values. Moving it would cross the identity/domain boundary of
[0007](./0007-tenancy-enforcement-and-data-access.md) §4 — organizations are created through the
un-scoped identity accessor, which cannot reach an RLS-forced domain table — so organization
creation could no longer set it, and every year creation would carry a value that never varies. The
objection that a relabelled axis alters a closed year's display is weak on inspection: published
artifacts are self-contained point-in-time snapshots (§18.2,
[0005](./0005-published-artifact-availability.md)), so a published dismissal list's heading is
already frozen, and a relabelled axis on a historical screen changes no fact about which homeroom a
child was in.

**Copy the previous year's vocabulary forward when a year is created.** Rejected for now, and this is
the deliberate scope cut rather than a judgement about the mechanism. It is the one convenience the
change plausibly needs, because a new year starts empty and §5.6 loads each year fresh: twelve
entries are re-entered annually, and an ordinal ladder retyped is an ordinal ladder eventually
mistyped. Against that, automatic copying is rollover in all but name — §5.6's deeper claim is that a
year starts empty and honest, and a year that arrives pre-populated hides the one question an
organiser must answer annually, which is whether the homerooms changed. An explicit, audited,
repeatable copy action avoids that objection, but it is new product behaviour with its own API
surface, audit shape and empty-state UI, and twelve hand-typed entries a year is a small cost to
carry in the meantime. **Revisit when an organiser reports the annual re-entry as friction, or when a
second concurrent program makes the vocabulary materially larger.** Should it arrive, it creates
fresh rows with fresh identifiers in the target year, copies only entries not retired, records who
copied from which year, and is additive so that repeating it is safe.

**An organization-level template vocabulary that each year instantiates from.** Rejected. It
reintroduces the organization-scoped list this decision removes, needs its own editing surface, and
drifts silently from every year that legitimately diverged — for no capability a copy action would
not already provide.

**A prior-year link between corresponding vocabulary rows,** analogous to the §8.7 prior-year student
link. Rejected. Nothing asks what a homeroom was last year, and each cross-year reference is another
exception to 0007 §5, whose value is that it has exactly one. Decisively: such a link could only be
set where a copy action was used, so with copy-forward cut it would be null everywhere, and even with
it the link would be absent for every hand-entered year — a correspondence recorded sometimes, with
no way to distinguish "no predecessor" from "predecessor not captured". That is worse than absent,
because a half-populated column reads like data.

**Assigning existing vocabulary rows to a single designated year during migration.** Rejected. It
orphans students in every other year, and `students.homeroom_id` is `NOT NULL`, so the migration
either fails at apply time or takes the roster with it. Discarding the vocabulary outright has the
same consequence for the same reason. [0014](./0014-roster-ingest-scope-and-source-authority.md)
permits an operator demonstration against the operator's own instance, so a real roster may exist in
a local database; losing synthetic seed data is free, and losing that is not.

## Consequences

- **A new school year cannot receive a roster until its vocabulary is populated.** `homeroom_id` is
  `NOT NULL` (§24.5 question 6), and ingest resolves homerooms without ever creating them
  ([0014](./0014-roster-ingest-scope-and-source-authority.md)), so both hand entry and import are
  blocked until the year has homerooms. This is a `Setup`-state condition in the sense of §11.1 and
  not a defect, but it must be discoverable: the year's vocabulary surface needs an empty state that
  says what to do, and the import resolution failure must name the year's own vocabulary rather than
  a global settings page that no longer exists.
- **Grade and homeroom identity is not comparable across years.** Two years' "Grade 3" are two rows
  with two identifiers and no relationship. Nothing in this specification compares them: the
  prior-year link is a nullable annotation with nothing depending on it (§8.7), fairness deficit is
  per program per year (§17.5), and grade windows live on offerings inside a session inside a year
  (§8.4). A future cross-year report earns an opaque link as its own decision. It must not join on
  grade code or homeroom name, because names are never keys (§8.7).
- **Reordering the grade ladder now affects one year.** This is the intended behaviour and the
  reverse of today's. The regression it replaces is silent, so the reorder test asserts that a
  sibling year's ordinals are unchanged.
- **Editing a closed year's vocabulary now fails with 409.** Previously it succeeded. Organisers who
  relied on correcting a homeroom name after a year closed must use the §11.1 reopen, which records
  who reopened the year and why. The refusal is permitted by §11.1's own words; it is not a new
  policy but an existing one reaching a table that was escaping it.
- **The same source classroom identifier may now recur in each year,** which is what allows a school
  platform export to be imported into a second year. The corresponding hazard is that the ingest
  homeroom index and the ingest current-state load must be re-scoped in the same change: making the
  identifier index per-year while still loading homerooms org-wide turns every recurring classroom
  identifier into an ambiguous-match error.
- **The migration's interesting branch is not covered by CI.** The seed corpus creates one school
  year, so the backfill's fan-out degenerates to one-to-one in development and test, and the
  round-trip check runs against an empty database. The multi-year path executes only on an instance
  holding two years. The mitigation is that the backfill is short enough to read rather than a
  harness invented for one migration; the limitation is recorded here so that nobody mistakes a green
  pipeline for evidence.
- **§20.1 needs no amendment.** Its **Rules** category already covers "changes to the
  concrete-attribute vocabularies of §10.1, including retirement". What changes is that the audit
  entry now carries the school year it belongs to, which the entry shape already supports and the
  vocabulary writer never set.
