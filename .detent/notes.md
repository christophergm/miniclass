# Issue #237 — v0 solver feasibility model

- Rework reason: PR #255 conflicted solely because `main` advanced its shared
  worktree handoff notes after the solver branch was opened.
- `solver/app/server.py` implements exactly-one assignment, capacity,
  grade-window eligibility, seeded deterministic search, and no partial
  result for infeasible requests. References: SPEC §§17.1, 17.3, 17.8.
- `solver/app/contract.py`, `backend/internal/solvercontract`, and
  `backend/openapi.json` carry ordinal grade eligibility and empty
  `conflict_diagnostics` for later diagnostics work.
- Rebase onto `origin/main` is in progress. Resolve this worktree-local note
  conflict by retaining the #237 handoff, then run all project validation
  gates, push, and verify fresh current-head CI before handoff.
