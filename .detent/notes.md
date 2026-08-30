## Current handoff — issue #126

- Scope: P2-7 import page for `roster_json` and `grades_csv`; governing contract SPEC §§5.2, 6.6, 11.2, 11.4–11.5 and ADRs 0004, 0014.
- Key files: `frontend/src/features/imports/ImportPage.tsx`, `ImportPage.test.tsx`, `useImports.ts`, `frontend/src/lib/apiResources.ts`, `frontend/src/App.tsx`.
- Implementation: generated-client preview/commit wrappers preserve the exact `File` and preview hash; review groups outcomes, expands field changes, calls out guardian removals, groups exclusions, keeps warnings/conflicts committable, blocks Error commits, maps hash conflicts to re-preview guidance, shows committed counts, and links the filtered import audit log.
- Validation: backend tests, backend lint, backend format, `make generate` plus generated-path drift, and `git diff --check` pass. Frontend test/build stop before execution because `openapi-typescript` is not installed; frontend lint is environment-limited by Bun temp/cache permissions. `make check` stops at `db-up` because the shared `/miniclass-postgres` container name is already in use. Migration round-trip lacks `MIGRATION_ROUNDTRIP_DATABASE_URL`; smoke lacks `.env`.
- Open items: install dependencies through the repository's `make setup`, then rerun frontend tests/build/lint and the complete `make check`; verify CI on the pushed PR head.
- Skill draft: no — this was a scoped page implementation using existing generated-client and React Query patterns.
