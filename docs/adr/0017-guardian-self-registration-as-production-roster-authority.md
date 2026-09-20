# 17. Guardian self-registration as production roster authority

- **Status:** Accepted
- **Date:** 2026-09-12
- **Supersedes in production:** [0014](./0014-roster-ingest-scope-and-source-authority.md)
- **Amends:** [0013](./0013-guardian-and-volunteer-access.md)
- **Implements:** SPEC §6.2, §8.2, §9.3–§9.4, §11, §20.1, §21
- **Related:** [0002](./0002-authentication-and-access-mechanisms.md),
  [0007](./0007-tenancy-enforcement-and-data-access.md),
  [0008](./0008-authorization-capabilities-and-audit.md),
  [0012](./0012-remove-the-household-entity.md)

## Context

The accepted design treated a third-party community-platform export as the production authority for
students, adults and guardian relationships. That is no longer acceptable under the program's data
governance requirements. Guardians must provide their own data and assert their own relationships to
children directly in this system; production must not copy those records from a directory, spreadsheet
or export collected for another purpose.

The experience must remain closer to a Google Form than to account registration. Adults should not
need passwords or full accounts. They provide their given and family names, may provide an email
address, and register one or more children by given name, family name and grade. More than one adult
must be able to register the same child without either adult learning about the other.

Self-registration creates a security trade-off that import did not. A person who possesses the
organization/year link and knows a child's name and grade can assert a guardian relationship. Requiring
an account, school-issued child identifier, or organizer approval for every submission would provide
more assurance, but would defeat the deliberately low-friction operating model. Conversely, returning
search results or child details would turn matching into a roster-discovery surface.

The existing model also requires homeroom, while the guardian form deliberately does not ask for it.
Registration must therefore be allowed to create an incomplete student record that an organizer
finishes before operational program membership or dismissal publication.

## Decision

### 1. Direct registration is the production authority

Production roster creation begins with guardian self-registration. Bulk roster import is not a
production capability. Existing CSV and JSON import code may remain as development and test tooling,
but production configuration and deployment must fail closed if an import is attempted. Development
and CI use synthetic people only; real roster exports are never loaded into development or test
databases.

Administrative person and relationship tools remain for review, correction, duplicate reconciliation
and deletion. They are not an alternative initial bulk-load workflow. Corrections preserve the
original submission and add audited organizer judgement rather than rewriting provenance.

### 2. Registration uses a narrow organization/year link

An administrator opens a registration window for one organization and school year and distributes one
high-entropy link through an existing trusted channel. The token is stored hashed, contains no tenant
or year identifier, expires, and is revocable and regenerable.

Possession creates a registration-only principal. It can submit one adult's data, add children, and
receive the minimal matching responses below. It cannot list or retrieve the roster, enter guardian or
preference surfaces, inspect placements, or use opaque person identifiers. Registration endpoints are
rate-limited and monitored for unusual volume. A low-friction anti-automation challenge may be used,
but an account may not be required.

### 3. Adult identity is not inferred from names

Given and family names are required; email is optional. In the common-link flow an email remains
unverified until a later email OTP proves mailbox control. Collecting an email proves neither control
of the mailbox nor a relationship to a child.

A unique verified-email match may identify an existing adult. Otherwise, name or unverified-email
similarity never silently merges adult records. Repeated or likely-duplicate submissions become review
items. An adult without email may register but cannot later use email-OTP guardian access unless an
email is added and verified.

### 4. Child matching requires explicit confirmation and reveals no unsubmitted attributes

Matching is limited to the registration link's organization and school year. It compares normalized
given name, family name and grade; fuzzy similarity may suggest that the registrant reconsider an
entry, but may not attach an adult to a child.

- One exact match produces a yes/no confirmation using only the values the adult supplied. No
  homeroom, preferred name, guardian, contact, preference, tag, placement or identifier is returned.
- No exact match creates a new student.
- A rejected exact match creates a new student and an administrator review flag without changing the
  existing student.
- Several exact matches create a provisional student and a review item. The application does not ask
  the adult to distinguish records using undisclosed data and does not choose silently.

Confirming a match reuses the student's opaque identifier and creates only the registering adult's
relationship. It never modifies another adult's edge. This preserves ADR 0012's model: adults register
separately and compose into a student with several guardians without introducing a household entity.

### 5. Self-assertion is accepted as a low-assurance relationship source

The baseline flow intentionally accepts that link possession plus knowledge of name and grade can
create a guardian edge. The compensating controls are:

- trusted-channel distribution, expiry, revocation and regeneration of the registration link;
- strict tenant/year scope and no roster browsing;
- match responses that reveal no information the registrant did not enter;
- rate limits, unusual-volume monitoring and repeated-claim review;
- immutable submission provenance and audited reconciliation; and
- a separate OTP proof before later authenticated guardian access.

Email OTP proves mailbox control, not guardianship. This decision accepts that distinction rather than
claiming stronger identity assurance than the workflow provides. Registration-link possession alone
never creates a guardian session and never reveals the matched child's existing data.

### 6. Known-email invitations are optional

An organization may later supply known adult email addresses and issue a unique registration code
bound to organization, school year and normalized email. The code is high-entropy, hashed and
single-use or explicitly renewable. It proves control of a mailbox already known to the organization,
but still does not prove the adult's relationship to a child.

The common-link flow remains the baseline. Invitation delivery, reminders, bounce handling and consent
policy are separate deferred work; the self-registration phase must not depend on them. An
organization may eventually configure invitation-only registration if its governance policy requires
it.

### 7. Registration records provenance and permits incomplete setup data

A submission atomically records the adult input, each child match outcome, relationships created,
organization, year, time and access channel. People and relationships retain creation provenance.
Rejected and ambiguous matches, likely duplicates, missing homerooms, missing emails, duplicate emails,
repeated claims and unusual volume appear in organizer review.

Because guardians are not asked for homeroom, a registered student may temporarily have none. That is
a visible setup condition, not a failed registration. The organizer assigns it before operational
program membership or dismissal publication. Reconciliation moves relationships and dependent records
by opaque identifier and must not silently discard either registrant's assertion.

## Alternatives considered

### Continue production import, then let guardians correct it

Rejected for governance reasons. It still copies children's and adults' data from a source collected
for another purpose and makes that source authoritative before a guardian participates. A correction
form does not change the origin of the initial record.

### Require every adult to create an account

Rejected. Password or account creation adds abandonment and support cost disproportionate to a small,
annual form. Administrative accounts remain deliberately separate and higher assurance.

### Require organizer approval before creating any person or relationship

Rejected as the baseline. It turns self-registration into a manual transcription queue and prevents
families from completing the flow in one sitting. Organizer review remains risk-based around duplicate,
ambiguous and unusual submissions.

### Match children only after submission, in an administrator queue

Rejected. Exact, minimally disclosed confirmation prevents many duplicate students at the moment the
adult can correct the entry. Deferring all matching creates avoidable cleanup without improving
privacy if the confirmation reveals no additional data.

### Use fuzzy matching or names as identity keys

Rejected. Similarity is useful as a warning, never as authority. Names are not unique, change over
time, and are explicitly forbidden as joins by SPEC §8.7. Fuzzy attachment would recreate the
predecessor's defining identity failures in a more consequential form.

### Require email and verify it before accepting registration

Rejected for the baseline. Some adults have no usable email, and mailbox control does not prove a
relationship to a child. Email remains optional; invitation-only mode is available as a future
organization policy where the added assurance is worth the friction.

### Show homeroom or other attributes to distinguish exact matches

Rejected. That converts matching into disclosure. If the three collected child fields are insufficient,
the system creates a provisional record and asks an administrator to reconcile it using authorized
surfaces.

## Consequences

- ADR 0014 remains a record of the implemented development importer, but no longer defines production
  roster authority or delivery sequencing.
- The application gains a public write surface for children's data. Registration token scope,
  tenant isolation, rate limiting, minimal disclosure and audit tests are mandatory, not optional end-
  to-end coverage.
- Guardian relationships now carry source/provenance. Self-registered, invited and administrator-
  corrected facts must remain distinguishable.
- `students.homeroom_id` can no longer be unconditionally non-null at creation. Operational membership
  and dismissal publication need explicit completeness checks rather than relying on that schema
  constraint.
- Duplicate reconciliation becomes a first-class organizer operation. It must preserve dependent data,
  relationships and provenance and therefore deserves dedicated tests and audit events.
- Adults without email can register but cannot use later OTP-based guardian access. This is surfaced as
  an outreach condition, not a registration error.
- Invitation mode can improve assurance that a registrant belongs to the intended community, but does
  not establish guardianship and brings delivery, consent and bounce-management obligations.
- The retained importer is useful for synthetic setup and parser/data-layer testing, but must be absent
  from production API enumeration and administrator capabilities.
