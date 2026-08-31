## Current handoff — issue #139

- Scope: Re-scope the existing grade and homeroom vocabularies to school years, including migration/backfill/down path, explicit year predicates, data/service/audit/API/ingest/seed/factory/registry/frontend updates, generated artifacts, and isolation/closed-year tests. Governing contract: SPEC §§8.1, 8.2, 10.1, 11.1, 20.1; ADRs 0007, 0010, 0014, 0015.
- Repository state: implementation is committed through `39b764f` and pushed to PR [#150](https://github.com/christophergm/miniclass/pull/150), based on merged dependency commit `f391ece`.
- Dependency: #138 is closed as completed and PR #141 merged at `f391ece`; the native GitHub `blocked_by` relation is terminal.
- Skills read: `.detent/skills/add-tenant-scoped-entity.md` and `.detent/skills/postgres-tenant-isolation-harness.md`.
- Implementation: migration, generated sqlc/OpenAPI, year-aware data/service/audit/API paths, registry/factories, seed, ingest, frontend resources/hooks/settings/roster callers, and scoped regressions are complete.
- Validation: all ten required CI checks pass on PR #150 head `39b764f`, including backend tests, migration round-trip, frontend tests/build/lint, generated drift, formatting, lint, and developer tooling. Focused local backend tests, `make lint-backend`, `make format`, and `git diff --check` pass; local DB-backed gates remain unavailable without the configured database/container environment.
- Open items: human review of PR #150; the PR is open, non-draft, references `Fixes #139`, and has no actionable review comments.
- Skill draft: no — existing tenant-entity and isolation-harness skills cover the reusable procedure; no new broadly reusable method was discovered.

## Current handoff — issue #140

- Scope: Split the year-scoped grade/homeroom vocabulary into `/y/:schoolYearId/vocabulary` and keep the organisation label plus owner-only administrators at top-level `/settings`, including closed-year read-only treatment and setup guidance.
- Dependency: issue #139 is now closed and PR #150 is merged; this worktree was three commits behind `origin/main` before the dependency update.
- Key files after dependency lands: `frontend/src/App.tsx`, `frontend/src/features/settings/SettingsPage.tsx`, `frontend/src/features/settings/useSettings.ts`, `frontend/src/features/school-years/SchoolYearPages.tsx`, `frontend/src/features/people/PeoplePages.tsx`, and frontend tests.
- Open items: rebase onto the merged #139 base, then implement and validate #140. No code changes or validation run for #140 yet.
- Skill draft: no — this run was blocked before exposing a reusable implementation method.
