# 16. Ranked choices distinguish order, acceptability, objection and absence

- **Status:** Accepted
- **Date:** 2026-09-02
- **Relates to:** SPEC §5.2, §13.1, §13.3–§13.5, §13.8, §17.4, §17.5
- **Related:** [0003](./0003-assignment-solver-technology.md)

## Context

A ranked-choice survey must do two different jobs at once. It must be simple enough for students to
complete accurately, and it must give the assignment engine enough information to distinguish a
highly desired placement from an acceptable fallback and from a placement the student actively does
not want.

Three simpler response models were considered:

1. require every eligible offering to be ranked in a complete order;
2. let students rank some offerings and mark the rest `Not interested`;
3. rate every offering `Very interested`, `Interested` or `Not interested` without ordering any of
   them.

Each removes apparent complexity, but each also collapses information that matters during allocation.
A complete order asks students to make distinctions they may not hold, especially near the bottom of
a long catalog. A partial order with only `Not interested` outside it cannot express "acceptable, but
not a favorite." Three rating buckets express acceptability well but cannot distinguish a first
choice among several highly desired offerings.

The central fairness concern is strategic withholding. A student might identify only one acceptable
offering in the belief that appearing to have no alternatives will increase the chance of receiving
it, while a student who honestly identifies several good alternatives appears easier to move. If the
solver gives priority to the student with fewer stated alternatives, it rewards strategic
inflexibility and penalizes useful, truthful preference information.

This concern is separate from the shape of the response. The survey records **preference**; it does
not confer **priority**. Priority comes from the objective and fairness history in SPEC §17.4 and
§17.5, together with constraints and explicit organizer judgement. It must not arise accidentally
from how many responses a student supplies.

## Decision

### Algorithm

The ranked-choice domain model remains the model in SPEC §13.3. For every offering in a session, a
student's response is exactly one of:

- a unique rank from `1..N`, where `N` is configurable per session;
- `Interested`, meaning acceptable but outside the ranked choices;
- `Not interested`, meaning explicitly unwanted;
- no response, meaning no opinion was expressed.

These states remain distinct throughout storage, solving and reporting. In particular:

**1. Ranking is deliberately partial.** Students identify the choices for which order carries useful
information. They are not required to manufacture a complete ordering over the catalog. The session
may limit the number of ranks so that the amount of ordering requested remains proportionate.

**2. Unranked acceptability is explicit.** `Interested` preserves the distinction between an
acceptable fallback and an objection. A flexible student can disclose additional acceptable choices
without pretending they are tied with the student's favorites or placing them in an arbitrary tail
order.

**3. Objection is explicit but is not absence.** `Not interested` means the student expressed that a
placement is unwanted. No response remains `Neutral`, not `Unwanted`, under SPEC §13.5 and §17.4.1.
Missing information is neither consent nor an objection.

**4. The number of alternatives does not create priority.** A first choice has the same preference
quality whether the student ranked one offering or several. Supplying fewer acceptable alternatives
must not be used as a tie-breaker in that student's favor. A narrower response gives the solver fewer
ways to produce a known-good placement; that loss of information is its consequence, not increased
claim on the one class named.

**5. Bad outcomes are protected before top-choice totals are improved.** Allocation follows the
worst-outcome-first objective in SPEC §17.4.2: first minimize `Unwanted`, then `Neutral`, then
`Acceptable`, then `High`. The solver does not maximize first choices by repeatedly trading one
student's bad placement for marginal improvements to several others. Cross-session fairness deficit
under §17.5 may legitimately distinguish otherwise equivalent students because it records how poorly
they have previously been served.

**6. `Not interested` remains a preference quality, not a hard eligibility constraint.** The solver
strongly avoids an unwanted placement and the system surfaces one when constraints make it
unavoidable. Treating every objection as a prohibition could make a solvable session infeasible and
would conflict with the warn-not-block principle of SPEC §5.2. Genuine prohibitions belong in the
constraint model, not in preference responses.

### User Interface

**1. The student interaction uses four buckets.** Offerings begin in `Not answered`, which maps to no
response rather than `Not interested`. Students move offerings into `Very interested`, `Interested`
or `Not interested`. `Very interested` is ordered from top to bottom and displays the resulting
ordinal rank prominently. Its capacity is the session's configured `N`; adding an offering to a full
bucket returns the lowest-ranked offering to `Not answered`. The other buckets are unordered and
shown alphabetically so their presentation does not imply a rank.

Students may submit with unanswered offerings or with no ranked favorites. The interface explains
that unanswered means no opinion and asks for confirmation when either condition applies, but it does
not block submission. This follows SPEC §5.2 and preserves the four states in §13.3. Changes remain
local until the student submits; partially arranged drafts are not server submissions and kiosk
drafts are not retained in browser storage.

**2. Dragging is an enhancement, not the only control.** On wide displays the unanswered catalog is a
column beside the three destination buckets. On narrow displays the same state is presented as four
selectable bucket views rather than shrinking the columns. Students may drag cards, use large
move-to-bucket controls, or use keyboard controls. Ranked cards also provide move-up and move-down
controls. Focus, announcements, contrast and reduced-motion behavior must make every operation
available without relying on drag precision, color or animation. Offering cards show the name and
description needed to choose; operational details such as location, meeting point and dates are
excluded because they do not inform this preference decision.

**3. Submission ends in a dedicated completion state.** A persistent action area summarizes bucket
counts and exposes the submission action. Moving an offering requires no confirmation and offers an
undo action; confirmation is reserved for a final response with no favorites or unanswered offerings.
After a successful submission, the form is replaced by a prominent, age-neutral `Done!` screen. A
student who followed a direct code may use a subtle link to revise answers while voting remains open.

**4. Administrator kiosk mode reuses the interaction, not the student credential.** Under SPEC
§13.8 an authenticated administrator selects the year, program, ranked-choice session and student,
then starts a dedicated full-screen form through an explicit handoff screen. The form submits through
the administrator channel so the response records the actor, target student, channel and time; it
must not generate, reveal or impersonate the student's private access code. An existing response is
identified before handoff, may be reviewed after an administrator warning, and is preloaded for a
legitimate correction.

The completion screen in kiosk mode offers only a subtle administrator return link and requires no
additional PIN. Returning preserves the selected year, program and session, clears the student,
refreshes response status and prevents browser Back from reopening the completed form. The student
picker defaults to students needing a response, supports search and status filters, and distinguishes
duplicate display names with grade while continuing to key every operation by opaque student ID.
Interest-profile surveys may adopt the same general kiosk approach later, but are outside this
interaction decision.

## Alternatives considered

**Require a complete ranking of every offering.** Rejected. It captures a total order, but much of the
additional information is false precision. Students may know their first few choices and have only
broad acceptability or objection judgments about the rest. Requiring an order among those offerings
increases cognitive and interaction cost, particularly for younger students and on small screens,
without reliably improving the input. It also fails to express the important boundary between "least
preferred but acceptable" and "do not place me here" unless another control is added, at which point
the simplicity has disappeared.

**Rank some offerings and treat every unranked offering as `Not interested`.** Rejected. It collapses
acceptable fallbacks into explicit objections. Students would either overstate dislike, depriving the
solver of useful options, or rank classes they do not genuinely prefer merely to say they would
accept them. The resulting ranks would mix preference order with acceptability and would not have a
stable meaning.

**Use only `Very interested`, `Interested` and `Not interested` buckets.** Rejected as the domain
model, though not necessarily as part of a future interface. Buckets are easy to understand and map
well to placement quality, but they lose the ordering among scarce, highly desired offerings. A
student who marks several classes `Very interested` has supplied no answer to which should be tried
first. Fairness history and deterministic tie-breaking can resolve competition between students;
they cannot recover a preference distinction the student was never allowed to express.

**Give students with fewer acceptable choices priority for those choices.** Rejected. This is the
strategic-withholding rule the model must avoid. It makes honesty costly: adding a class the student
would enjoy can only weaken their claim to a favorite. It also confuses a self-reported preference
shape with independently justified need. A student with a prior fairness deficit may properly receive
more weight under SPEC §17.5; a student who merely reports fewer options may not.

**Treat `Not interested` as a hard exclusion.** Rejected. A preference survey is not a reliable place
to encode eligibility, safety, accessibility or organizer-mandated separation. Making every negative
rating hard can leave a student unassignable and turn an expression of preference into an accidental
veto over the session. Hard exclusions remain explicit constraints with their own provenance and
override behavior.

## Consequences

- The model asks for more than three undifferentiated ratings, but every distinction has allocation
  meaning: ordered favorite, acceptable fallback, explicit objection, or unknown.
- Students can disclose flexibility without reducing the quality assigned to their ranked choices.
  Product copy must say this plainly, and the four-bucket interaction must preserve that distinction.
- The solver and reports must never infer `Not interested` from an omitted response, nor infer greater
  priority from a shorter ranked or acceptable set.
- A session may cap `N`, reducing student effort without changing the model. Offerings beyond that cap
  can still be marked `Interested` or `Not interested`.
- Reports can explain outcomes as top-ranked, lower-ranked, acceptable, neutral or unwanted, rather
  than presenting every unranked placement as equally bad.
- An unavoidable unwanted placement remains possible. It is visible as `Unwanted` and is optimized
  before better categories, rather than being hidden by converting the response into a constraint.
- The ranked-choice form requires responsive, non-drag controls and a distinct completion state in
  both direct-link and administrator kiosk channels.
- Interest-profile surveys are not changed by this decision; adapting their unranked interaction is
  intentionally deferred.
- Any future design that merges absence with objection, removes acceptable-but-unranked, rewards
  withholding or submits administrator kiosk responses through student credentials would reverse this
  decision and requires a superseding ADR.
