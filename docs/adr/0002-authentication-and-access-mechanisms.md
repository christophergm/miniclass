# 2. Authentication and access mechanisms

- **Status:** Accepted
- **Date:** 2026-08-23
- **Implements:** SPEC §9.3, §9.4, §9.5, §6.6
- **Amended in part by:** [0009](./0009-administrator-sessions-and-identity-provider.md) — token
  verification, browser session mechanics, invitation-based provisioning, and the evaluation of Clerk
  as an alternative provider; [0013](./0013-guardian-and-volunteer-access.md) — adult OTP sessions,
  student survey codes, and step-up MFA
- **Related:** [0001](./0001-application-stack-and-topology.md),
  [0006](./0006-household-and-volunteer-access.md),
  [0008](./0008-authorization-capabilities-and-audit.md)

## Context

SPEC §9.3 requires deliberately unequal authentication mechanisms. The original household terminology in
this record was superseded when ADR 0012 removed the household entity; ADR 0013 resolves the remaining
Phase 4 adult and student access questions:

| Principal | Mechanism | Scope |
|---|---|---|
| Owner, Administrator, Coordinator | Account with credential, renewable session | Organisation |
| Guardian | Application-owned email OTP followed by a bounded session | Current guardian relationships |
| Student | Application-owned high-entropy survey/session code | One student and one instrument |
| Class leader, Homeroom teacher | Tokenised link | Named objects, session |
| Public reader | Unauthenticated share link | One artifact, one session, expiring |

Only administrative users have accounts. The specification is explicit that this inequality is
intentional: "obscurity is the only access control here" for share links, and the design accepts that
in exchange for a workflow that parents and volunteers will actually complete.

§9.5 imposes requirements the link mechanisms must meet: tokens are high-entropy and unguessable;
they **must not encode tenant, session or student identifiers**; every link has an expiry; every link
is independently regenerable — invalidating the prior URL — and revocable.

§9.4 adds two rules that are easy to get wrong: the **tenant check precedes the permission check**,
and a cross-tenant request fails as **not-found, not forbidden**.

The prior architecture note said "Supabase Auth, adult users only — parents / teachers / admins",
which maps three unequal mechanisms onto one and understates the problem.

## Decision

**Split the mechanisms by principal.**

1. **Administrator accounts use Supabase Auth.** The Go API verifies the JWT and maps the verified
   subject onto an application user carrying an organisation and one of `Owner`, `Administrator`,
   `Coordinator`. Supabase is an identity provider only; it is not consulted for authorization.

2. **Non-administrative scoped access is owned by this application**, in PostgreSQL, as application
   verifiers and a single `access_token` concept with discriminated purposes:

   - opaque high-entropy random token, stored hashed, never derived from any identifier;
   - a purpose (`guardian_submission`, `student_code`, `class_leader`, `homeroom_teacher`,
     `published_artifact`);
   - the exact object set the token authorizes;
   - an expiry, a revoked-at, and a generation counter so regeneration invalidates the prior URL;
   - single-purpose and non-reusable outside its window, per §9.3. OTP verifiers follow the same
     single-use and expiry requirements but are not an identity or a bearer grant to administration.

3. **Authorization is always the application's own**, evaluated server-side on every request, after
   the tenancy guard has run. No principal — including an administrator — reaches data through a
   path that bypasses the guard.

4. **Persona separation is enforced**, per §6.3 and ADR 0013: the guardian, student, class-leader,
   homeroom-teacher, and public-reader views never merge, even when the same adult holds more than one
   access path. A guardian session yields guardian data only, and a student code yields one student's
   instrument only.

## Alternatives considered

**Supabase Auth for everything, including guardians.** Rejected. Supabase magic links authenticate
an *email address into an account*; SPEC requires guardian OTP access scoped to an adult's current
relationships, a bounded session, and no account creation. It would also make email possession enough
to enter a surface containing child data, and it cannot express the student-code, class-leader or
public-share cases with their independent scopes.

**Drop Supabase Auth and own all administrator authentication.** Genuinely attractive: there are three
administrator accounts, the token machinery is being built regardless, and §5.7 prefers the direct
approach. Rejected for now on the operator's judgement that other Supabase capabilities may be
adopted later. Should be revisited at R3 if Supabase is still doing nothing but authenticating three
users — the cost of removal grows slowly, and the simplification is real.

**Encode scope in a signed token (JWT) rather than a database row.** Rejected. §9.5 requires
individual revocation and regeneration, which a stateless token cannot provide without a revocation
table — at which point the database row is the simpler design. It also risks encoding identifiers
into the token, which §9.5 forbids.

## Consequences

- There are two authentication paths in the codebase. Both must terminate in the same principal
  abstraction so that authorization and the tenancy guard have exactly one implementation.
- The token table is security-critical and is built in Phase 1 alongside the tenancy guard, even
  though only the administrator path is exercised until Phase 4.
- Tokens are stored hashed, so a database disclosure does not yield working links.
- Supabase is a hard external dependency of the administrative surface. Published artifacts must not
  inherit that dependency; see [ADR 0005](./0005-published-artifact-availability.md).
- Guardian scope is addressed to an individual adult and derived from guardian relationships, not a
  household entity; see [ADR 0012](./0012-remove-the-household-entity.md) and [ADR 0013](./0013-guardian-and-volunteer-access.md).
