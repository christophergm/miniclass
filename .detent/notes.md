## Current handoff — issue #125

- Scope: P2-6 `grades_csv`: header-name parsing, whole-cell normalized legal/preferred-name matching, update-only grade vocabulary resolution, preview/commit/idempotency, and audit summary. Governing contract: SPEC §§10.1–10.2, 11.3, 11.5–11.7, 20.1, Appendix A.5; ADR 0014.
- Key files: `backend/internal/ingest/grades.go`, `backend/internal/ingest/envelope.go`, `backend/internal/ingest/commit.go`, `backend/internal/ingest/preview_test.go`; existing parser: `backend/internal/ingest/roster/grades.go`.
- Implementation: exact duplicate assignments remain independently reviewable and deduplicated at write time; contradictory normalized-name assignments are `Conflict`; unmatched names are `Conflict` and never create students; unknown grades are `Error`.
- Validation: focused ingest tests, `go test ./...`, `go test -race -v ./... -count=1`, `make lint-backend`, `make format`, `make generate` plus generated-path drift, and `git diff --check` pass. All ten required PR CI checks pass on PR #134 head `ae05578`; local test-backend, migration, frontend, and smoke commands are environment-limited as recorded in the Workpad.
- Handoff: PR #134 is open, non-draft, references #125 with `Fixes #125`, and has no review or inline comments. Detent should advance the issue through its configured review lane.
- Skill draft: no — implementation uses existing import/data patterns and has not exposed a broadly reusable non-routine method.
