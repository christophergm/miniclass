# Issue #237 — v0 solver feasibility model

- Rework reason: PR #255 conflicted solely because `main` advanced its shared
  worktree handoff notes after the solver branch was opened.
- `solver/app/server.py` implements exactly-one assignment, capacity,
  grade-window eligibility, seeded deterministic search, and no partial
  result for infeasible requests. References: SPEC §§17.1, 17.3, 17.8.
- `solver/app/contract.py`, `backend/internal/solvercontract`, and
  `backend/openapi.json` carry ordinal grade eligibility and empty
  `conflict_diagnostics` for later diagnostics work.
- Rebased cleanly onto `origin/main` at `f5f371d`; the only conflict was this
  worktree-local handoff file. The new solver commit is `b52c9f6`.
- Passed after the rebase: solver container tests (5),
  `go test ./internal/solvercontract ./internal/solverclient`, backend lint,
  backend format, generation/drift, and `git diff --check`.
- `make check` cannot start because existing host containers own
  `miniclass-mailpit` and `miniclass-postgres`; do not remove them. Frontend
  test/build lack `openapi-typescript`; frontend lint has the pre-existing
  `PeoplePages.test.tsx` formatting failure tracked in #254.
- Current-head CI on `014ee9c` passed every 10 project gate plus solver
  contract/image checks; backend tests was slowest at about 1m49. No review or
  inline comments are present. Recheck CI after this handoff-note commit.
