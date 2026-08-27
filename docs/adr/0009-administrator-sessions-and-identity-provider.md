# 9. Administrator sessions and identity-provider choice

- **Status:** Accepted
- **Date:** 2026-08-23
- **Implements:** SPEC §9.3, §9.4, §9.5, §6.6
- **Amends:** [0001](./0001-application-stack-and-topology.md) — one sentence, noted below;
  [0002](./0002-authentication-and-access-mechanisms.md) — in part
- **Related:** [0007](./0007-tenancy-enforcement-and-data-access.md),
  [0008](./0008-authorization-capabilities-and-audit.md)

## Context

[ADR 0002](./0002-authentication-and-access-mechanisms.md) chose Supabase Auth for administrator
accounts and left three things unstated: how a token is verified, how the browser holds a session,
and how a person becomes an administrator in the first place. Two further questions were never
asked — whether Supabase is the right identity provider at all, and how any of this is tested
without it.

The last question is the sharpest. Detent agents run sandboxed with no outbound network. After Phase
1 every endpoint is authenticated, so if any test requires a live Supabase project, the entire API
surface becomes untestable in CI and unrunnable by agents.

Facts established at the time of writing: Supabase's asymmetric JWT signing keys are generally
available, publishing a JWKS at `<iss>/.well-known/jwks.json` with ES256 or RS256 and a `kid` header;
access tokens default to a five-minute lifetime with refresh handled client-side; and the legacy
shared HS256 secret still exists but is documented as "no longer recommended".

## Decision

**1. Verification is local, asymmetric and cached.** The JWKS is fetched from the configured issuer
and cached, refreshed on an unknown `kid` at a rate limit so a forged `kid` cannot induce a lookup
storm. `iss`, `aud`, `exp` and `nbf` are validated with 30 seconds of clock skew. **The algorithm is
pinned to ES256/RS256**; `none` and any HS256 token are rejected outright, because accepting a
symmetric algorithm from an asymmetric issuer is the classic key-confusion attack. No
`getUser()` round trip, and no service-role secret anywhere in the API.

**2. The verifier is an interface with two implementations, chosen by configuration.** `supabase` is
the JWKS verifier above; `local` uses a static keypair, with `cmd/devtoken` minting tokens for a
given subject and email. The local verifier **refuses to initialise when the environment is
production**, and that refusal is itself a test.

The consequence is deliberate and is the point of the design: **no developer, agent or CI job ever
needs Supabase credentials.** Supabase is exercised only in a deployed environment. Given the
sandbox, this is a requirement rather than a convenience.

Testing happens at two levels on purpose: unit tests inject a principal directly, and integration
tests mint a locally-signed JWT and traverse the real middleware. Only the second proves the
middleware works; only the first is cheap enough to use everywhere.

**3. The browser holds the session via `supabase-js` and sends `Authorization: Bearer` to Go.**

**This amends [ADR 0001](./0001-application-stack-and-topology.md)**, which states "the browser talks
to Go; it does not talk to Supabase directly". The corrected statement is: *the browser talks to
Supabase for authentication only; every data path goes through Go.* ADR 0001's actual rationale — one
tenancy guard, one authorization implementation, one audit log — is untouched.

**3a. An API rejection ends the browser session.** The frontend API boundary treats only a `401`
response whose RFC 9457 type is `invalid-token` as a terminal session signal. It clears the auth
shell session, asks Supabase to sign out so a rejected persisted session is not reused after reload,
and sends the person to the sign-in surface with an explanation. Other `401` problems, including a
missing bearer, remain ordinary API errors; a request-level response is not enough evidence that a
renewable Supabase session has ended. The local fake client applies the same session-ended path at
the JWT `exp` boundary and tells the developer to run `make login` and restart Vite.

**4. Administrators are provisioned by invitation, never just-in-time.** A verified subject with no
application record gets nothing; otherwise anyone able to sign up to the project becomes a user.

- `Owner` invites by email, creating an `organization_members` row with `invited_email` set and
  `user_id` null, plus an `access_tokens` row of purpose `admin_invitation`.
- Claiming requires **both** the token *and* a verified email matching `invited_email`. The token
  alone would make whoever holds the link an administrator, which is a far higher-stakes bearer
  credential than a household link; the email claim alone is only as strong as `email_verified`.
- 48-hour expiry, consumed on binding. "Resend" **regenerates**, invalidating the prior URL — §9.5's
  exact semantics, exercised in Phase 1 rather than waiting for Phase 6.
- A verified subject with no membership and no matching invitation receives **403** with a distinct
  problem type. The request names no other tenant; it simply has nowhere to go, so 404 would mislead
  and 401 would be wrong.

The first `Owner` is bootstrapped out of band by the CLI, which creates the organization and an
invitation and prints the URL. The bootstrap path therefore holds no special runtime privilege.

**5. The identity provider remains Supabase.** Clerk was evaluated in detail and rejected; see
below.

**6. A principal holding more than one organization membership produces an explicit, named error.**
Phase 1 assumes a single membership. Silently choosing an organization is the one outcome that must
not happen.

## Alternatives considered

**Go mints its own httpOnly session cookie, proxying sign-in.** Rejected on cost, not on merit. The
hidden expense is refresh: five-minute access tokens mean Go would have to store refresh tokens,
implement rotation, handle races between concurrent tabs, and deal with reuse detection — a
meaningful volume of security-sensitive code we would own, for three administrators. It also requires
`SameSite=None; Secure` cookies across two Render services, exact-origin CORS with credentials, and a
CSRF strategy. Bearer tokens have no CSRF surface and keep CORS credential-free.

**Exchanging a Supabase token for a Go-minted cookie.** Rejected for the same reason with less of the
benefit.

**Clerk instead of Supabase Auth.** Genuinely attractive and rejected for three specific reasons.

Clerk's differentiator is first-class Organizations with memberships, roles and invitations, plus
prebuilt UI — which would remove the sign-in page, the administrator-management screens, and the
`admin_invitation` token purpose entirely. Verification would be near-identical (JWKS, RS256,
cached), and its session storage has a better XSS posture than `supabase-js`'s `localStorage`.

But: (a) §6.6's `Owner` / `Administrator` / `Coordinator` does not fit Clerk's free-tier Admin /
Member, and custom roles sit behind a paid add-on — and `Coordinator` is load-bearing, not cosmetic.
(b) §20.1 requires the audit log to record "permission changes; administrator addition and removal";
if those happen in Clerk's hosted UI they never touch our database, and closing the gap requires
webhook ingestion with signature verification, retries, idempotency and divergence detection — a new
asynchronous subsystem bought to satisfy a requirement we already satisfy for free by owning the
table. (c) Using Clerk for the switcher and invitations while keeping roles local means two
membership stores that can disagree.

The only Clerk configuration avoiding all three is *Clerk for user identity only, with organizations,
roles and invitations local* — at which point its differentiator is unused and what remains is a
nicer sign-in widget for the price of a second vendor. Supabase already hosts production Postgres,
so Supabase Auth costs no additional vendor.

Two smaller points cut opposite ways and are recorded for the revisit: Clerk's free tier has **no
MFA**, whereas Supabase offers TOTP on its free tier, which matters for three accounts guarding
children's names and placements; against that, Clerk's identity product is unambiguously more
polished.

**Revisit trigger:** if the system ever needs SSO/SAML, enforced MFA beyond TOTP, or genuine
self-serve multi-tenant signup, reopen this record.

**Carrying organization and role in the token's `app_metadata`.** Rejected. It would eliminate the
pre-tenant lookup that forces the identity layer of
[ADR 0007](./0007-tenancy-enforcement-and-data-access.md), but it contradicts ADR 0002's "Supabase is
an identity provider only; it is not consulted for authorization", makes revoking an administrator
take effect only at token refresh, and puts permission changes outside the §20.1 audit log.

## Consequences

- **`supabase-js` keeps the session in `localStorage`, so an XSS can exfiltrate a long-lived refresh
  token** — worse than stealing a session cookie, because it survives the tab. Mitigations are a
  strict CSP on the static site and keeping untrusted HTML out of the React tree. In a SPA an XSS can
  also simply drive the API as the user, so the cookie's advantage is narrower than it first appears,
  but the residual risk is real and is scheduled for review at **R3**, where ADR 0002 already
  schedules a Supabase re-evaluation.
- The design is **identity-provider agnostic by construction** — a verifier interface with a local
  implementation, and organizations, users, memberships, roles and invitations all owned locally. A
  future provider change is a bounded change to one package rather than a migration. That is what
  makes rejecting Clerk today a low-cost decision rather than a bet.
- There are two authentication code paths, and only one exists in Phase 1.
- **ADR 0002's consequence that "both authentication paths must terminate in the same principal
  abstraction" is not discharged by Phase 1.** An invitation token is a one-shot claim, not a
  principal that reads data, so no link *principal* exists until Phase 4. Recording this as a Phase 4
  obligation is more honest than claiming an abstraction designed against a single implementation is
  general.
- Supabase remains a hard external dependency of the administrative surface at runtime, and of
  nothing at development or test time.
