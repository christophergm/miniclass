## Issue #225 — organiser placeholder and reconciliation UI

- Verified 2026-09-30: branch is based on `origin/main` at `d16a61f`; native
  dependency #224 is closed.
- Issue has no PR yet. Workpad comment:
  https://github.com/christophergm/miniclass/issues/225#issuecomment-5910700256
- Scope: organiser placeholder creation, recognisable organiser surfaces,
  audited reconciliation with confirmation, and actionable review signals.
- Preserve Phase 4B API authorization and keep guardian matching unable to
  discover placeholder records. References: SPEC §§11.7, 18.2, 20.1.
- Implemented: dedicated creation and reconciliation routes, placeholder badges
  in organiser roster/review, audit-reason inputs, confirmation, and
  signal-to-remediation links. No backend contract changes were needed.
- Validation passed: backend lint and format, generated-artifact drift check,
  frontend lint and build, repository formatting, and the frontend unit phase
  (205 tests).
- Local limitations: the aggregate gate cannot start Docker because an external
  container owns /miniclass-mailpit; frontend E2E Chromium launch is denied
  Mach-port access by the sandbox. CI must run the database, E2E, migration,
  backend-test, and smoke stages.
- Pending: commit, push, open PR, and verify current-head CI/comments.
