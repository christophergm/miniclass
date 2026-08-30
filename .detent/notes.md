## Current handoff — issue #125

- Scope: P2-6 `grades_csv`: header-name parsing, whole-cell normalized legal/preferred-name matching, update-only grade vocabulary resolution, preview/commit/idempotency, and audit summary. Governing contract: SPEC §§10.1–10.2, 11.3, 11.5–11.7, 20.1, Appendix A.5; ADR 0014.
- Key files: `backend/internal/ingest/grades.go`, `backend/internal/ingest/envelope.go`, `backend/internal/ingest/commit.go`, `backend/internal/ingest/preview_test.go`; existing parser: `backend/internal/ingest/roster/grades.go`.
- Implementation: exact duplicate assignments remain independently reviewable and deduplicated at write time; contradictory normalized-name assignments are `Conflict`; unmatched names are `Conflict` and never create students; unknown grades are `Error`.
- Validation so far: `go test ./...` passes; focused ingest tests pass; `git diff --check` passes. Required make gates and PR handoff remain.
- Open items: run all ten project gate commands as available, commit/push, open PR with `Fixes #125`, inspect CI/review, and update Workpad status only after the PR gate is ready.
- Skill draft: no — implementation uses existing import/data patterns and has not exposed a broadly reusable non-routine method.
