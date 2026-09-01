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

SPEC §9.3 requires **four deliberately unequal** authentication mechanisms:

| Principal | Mechanism | Scope |
|---|---|---|
| Owner, Administrator, Coordinator | Account with credential, renewable session | Organisation |
| Household | Emailed magic link | Household, submission window |
| Class leader, Homeroom teacher | Tokenised link | Named objects, session |
| Public reader | Unauthenticated share link | One artifact, one session, expiring |

Only administrators have passwords. The specification is explicit that this inequality is
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

2. **The three link mechanisms are owned by this application**, in PostgreSQL, as a single
   `access_token` concept with a discriminated purpose:

   - opaque high-entropy random token, stored hashed, never derived from any identifier;
   - a purpose (`household_submission`, `class_leader`, `homeroom_teacher`, `published_artifact`);
   - the exact object set the token authorizes;
   - an expiry, a revoked-at, and a generation counter so regeneration invalidates the prior URL;
   - single-purpose and non-reusable outside its window, per §9.3.

3. **Authorization is always the application's own**, evaluated server-side on every request, after
   the tenancy guard has run. No principal — including an administrator — reaches data through a
   path that bypasses the guard.

4. **Persona separation is enforced**, per §6.3: the household view and the class-leader view never
   merge, even when the same person holds both. A household link yields the household view and
   nothing else.

## Alternatives considered

**Supabase Auth for everything, including households.** Rejected. Supabase magic links authenticate
an *email address into an account*; SPEC requires a link scoped to a *household's submission window*
for a *specific session*, single-purpose, non-reusable afterwards, and independently revocable. It
would also create accounts for ~90 households that the specification deliberately avoids, and it
cannot express the class-leader or public-share cases at all.

**Drop Supabase Auth and own all four mechanisms.** Genuinely attractive: there are three
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
- Whether a household link addresses a *household* or an *individual adult* is not settled by this
  record. See [ADR 0006](./0006-household-and-volunteer-access.md).
