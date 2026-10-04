# 19. Retire direct student preference access

- **Status:** Accepted
- **Date:** 2026-10-04
- **Implements:** SPEC §6.3–6.4, §9.3–9.4, §13.3, §13.6–13.8, §24.3
- **Supersedes in part:** [0002](./0002-authentication-and-access-mechanisms.md),
  [0013](./0013-guardian-and-volunteer-access.md) — student authentication and preference codes;
  [0016](./0016-ranked-choice-response-model.md) — direct-code revision access only
- **Related:** [0005](./0005-published-artifact-availability.md),
  [0010](./0010-schema-generated-code-and-migration-conventions.md),
  [0017](./0017-guardian-self-registration-as-production-roster-authority.md)

## Context

The operator has decided that guardian login is the only family-facing preference access path.
Guardian forms already support both interest-profile surveys and ranked choices. Independent student
codes duplicate access logic and couple opening instruments to credential generation and distribution.
The system has not been deployed to production, so a compatibility or deprecation period is unnecessary.

Removing credentials must not erase collected preferences or invent guardian attribution for historical
student-code submissions. Students without usable guardian self-service must also retain an audited
administrative fallback.

## Decision

1. **Remove direct student access completely.** Drop `interest_profile_survey_access_codes` and
   `ranked_choice_access_codes` in a new timestamped migration. Remove issuance, hashing, lookup,
   regeneration, revocation, anonymous respondent routes, code-management operations, distribution and
   printing controls, and code-bearing API fields. Do not retain legacy redirects or compatibility fields.

2. **Retain guardian and administrative access.** Guardians use the existing email-OTP login and
   "My students" dashboard. Scope is derived from current relationships, and instrument eligibility and
   response windows remain enforced. Administrator-on-behalf entry and the administrator-authenticated
   ranked-choice kiosk remain supported with truthful actor, target, channel, and time attribution.
   The kiosk is not an independently authenticated student surface.

3. **Preserve historical answers and shared response semantics.** Existing `student_code` submissions
   and audit events remain unchanged, readable, and effective. Keep the historical channel value, but
   reject new student-code submissions in both the application data layer and database insert boundary.
   Do not relabel them as guardian submissions. Current guardians continue to share a student's response
   history; the latest valid rating per area or complete ranked-choice response wins across authorised
   actors. Remove the unused non-survey writer rather than retaining a second submission path.

4. **Preserve instruments, not credentials.** Keep audience snapshots, question/scale snapshots,
   audited late-member additions, deadlines, reopening reasons and warnings, submission history, and
   response tracking. Opening and reopening no longer issue or reactivate credentials. Missing guardian
   relationships or email do not remove students from response denominators or participation eligibility.

5. **Do not change unrelated access mechanisms.** Guardian registration entries, invitation links,
   email OTP, guardian sessions, and read-only published-artifact share links retain their separate
   security models. No replacement survey links, login-return routing, or automated notifications are
   introduced.

## Consequences

- Survey and voting lifecycle transactions no longer depend on student credential work. Both credential
  tables and their deletion/reconciliation dependencies disappear; purge and placeholder database
  functions must be replaced in the same migration.
- Removing API operations and required code-related response fields is an intentional breaking change.
  Generated sqlc, OpenAPI, and frontend types are regenerated from their sources.
- The new migration must be applied with the updated backend: the old backend expects tables that no
  longer exist. Rollback restores empty credential tables, not discarded codes; historical answers are
  unaffected by credential removal.
- Security and integration tests preserve tenant isolation, live guardian scope, instrument eligibility,
  response precedence, audit attribution, kiosk behavior, and deletion/purge semantics. Upgrade tests
  preserve historical responses and verify both write guards. Mobile and accessibility coverage moves
  from anonymous student pages to supported guardian and administrative surfaces.
- Previous ADR descriptions of student-code access are historical, not an additional supported path.
  Other decisions in those records remain in force.
