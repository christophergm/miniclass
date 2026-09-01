# 13. Adult and student access mechanics

- **Status:** Accepted
- **Date:** 2026-09-01
- **Implements:** SPEC §6.2, §6.5–6.6, §9.3–9.4, §13.8, §19.5
- **Related:** [0002](./0002-authentication-and-access-mechanisms.md),
  [0012](./0012-remove-the-household-entity.md)

## Context

The household entity was removed by ADR 0012. The remaining access design needed to resolve three
questions: how adults authenticate and renew access, how non-guardian volunteers reach the system, and
what happens when an adult has no email address. Phase 4 also introduced a deliberate decision to allow
students to submit preferences directly without creating student accounts.

The system must distinguish limited guardian access from administrative access. Email OTP is convenient
for the small population of guardians, but control of an email mailbox is not sufficient assurance for
an administrative surface containing substantial personal information about children.

Volunteer sign-up and availability are handled in Konstella, so the system does not need a volunteer
self-service access path. Organizers enter staffing assignments and any confirmations in the application.

## Decision

### One adult identity with separate capabilities

An adult may have both a year-scoped adult record with guardian relationships and an administrative
account. The account-to-adult link is explicit and uses opaque identifiers. Matching email addresses may
suggest a link but must never create one silently. Distinct adult records must not share an email for
OTP access; duplicates require resolution. Changing an email does not change the identity link.

The application presents separate modes:

- **Guardian mode** shows only the adult's current guardian-scoped students and their open preference
  forms. It does not show program-wide data, class rosters, administrative data, or another adult's
  students.
- **Administration mode** is available only to an adult with administrative capabilities and requires
  step-up MFA. It can include administrator-on-behalf preference entry for a selected student.
- **Survey mode** is a restricted guardian experience. Returning to administration requires
  reauthentication/step-up. A mode boundary reduces accidental exposure but cannot establish who is at
  the keyboard of an already-unlocked browser.

### Adult authentication

Guardian access begins with a short-lived, single-use email OTP and creates a bounded, revocable session.
The session scope is derived from current guardian relationships on every request, not stored in the
session as a permanent student list. OTP alone never authorizes administration.

Administrative access uses the existing account identity provider and requires mandatory MFA. Recovery
uses single-use recovery codes or an explicit, audited Owner-assisted reset; email OTP alone is not an
administrative MFA fallback. Administrative sessions are invalidated after an MFA reset.

Transactional email for OTP delivery is in scope. Bulk and workflow notifications remain out of scope,
including emailing student codes, survey invitations, reminders, or follow-ups.

An adult without an email is unreachable for self-service preference submission. The student remains
eligible and appears in response tracking; an administrator can submit on the student's behalf.

### Student survey access

Students are not account users. A student may use a high-entropy, survey-scoped access code to submit
and revise their own response:

- one code is bound to one student and one interest-profile survey or ranked-choice session;
- the code is stored hashed, regenerable, revocable, and valid only while its instrument is open;
- codes are generated when the instrument opens from its frozen audience;
- an organizer-only list presents codes grouped by homeroom in a print-friendly view;
- code-list view/print is not audited, but generation and regeneration are audited;
- no automated student-code email is sent in Phase 4.

Interest-profile surveys and ranked-choice responses use the same narrow respondent pattern but remain
separate domain concepts. Interest surveys have administrator-configured windows. Ranked-choice access
follows the session voting window and stops at its configured deadline before assignment.

### Volunteer and artifact access

Volunteer sign-up, participation intent, topic interests, availability, and class proposals are
collected outside the system, currently through Konstella. There is no non-guardian volunteer
self-service access path in v1.

Read-only class-leader and homeroom-teacher artifact links remain separate, narrow, session-scoped
published-artifact access. They do not provide preference, staffing, or administrative access.

## Consequences

- Guardian convenience and administrative assurance are separated without requiring a second adult
  identity or duplicating guardian data.
- The application owns OTP/session and student-code security boundaries; tokens are opaque, hashed,
  scoped, expiring, regenerable, and revocable.
- Transactional email becomes a Phase 4 dependency, while bulk communication remains an organizer task.
- Students can improve response rates through direct access without introducing accounts, passwords, or
  general student privacy surfaces.
- A student with no guardian or an adult with no email does not block participation; organizer entry is
  the fallback.
- Volunteer data may be incomplete because Konstella is the external source; staffing remains advisory.
- Class-leader and homeroom-teacher links continue to be separate from authenticated guardian sessions,
  preserving persona and data-surface separation.
