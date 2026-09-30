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

# Issue #238 — v0 solver placement quality

- `solver/app/contract.py` accepts the canonical quality snapshot: ranked
  choices take precedence over interest profiles, and an offering may carry an
  optional interest-area ID. `high_rank_max` is the configurable ranked-choice
  boundary from SPEC §17.4.1.
- `solver/app/server.py` maps each feasible assignment to `top`, `high`,
  `acceptable`, `neutral`, or `unwanted`, then sequentially minimizes
  `unwanted`, `neutral`, `acceptable`, and `high`, fixing each optimum before
  the next. Returned assignments record `realized_quality`.
- The mirrored Go contract in `backend/internal/solvercontract` validates and
  canonically orders the new inputs and realized output. `backend/openapi.json`
  is generated from it.
- Passed: `PYTHONDONTWRITEBYTECODE=1 python3 -m pytest` in `solver/` (20),
  `go test ./internal/solvercontract ./internal/solverclient`, `make format`,
  `make lint-backend`, and `git diff --check`.
- Docker image validation is unavailable in this sandbox because Docker Buildx
  cannot update its host-local activity file. `make test-backend` stops before
  tests at the pre-existing named mailpit container conflict; do not remove the
  existing containers. Frontend test/build lack `openapi-typescript`, and
  frontend lint reports the existing `PeoplePages.test.tsx` format drift.

# Issue #239 — pinned placement re-solve

- `pins` is now a canonical v1 solve-request field in the Python sidecar and
  mirrored Go contract. Valid pins bind the assignment decision before capacity
  and lexicographic optimization (SPEC §§17.3, 17.8, 17.9).
- Invalid pins return an infeasible, assignment-free response with stable
  diagnostics naming the participant and offering: not participating, missing
  offering, grade out of range, conflicting placement, or capacity exceeded.
- Tests cover fixed-seat capacity consumption, unrelated preference edits,
  invalid-pin diagnostics, and Go canonical snapshot retention. Passed Go
  solver/API package tests, `make format`, `make lint-backend`, generation, and
  `git diff --check`. Python test/image validation cannot run locally: pytest
  is absent and Docker Buildx lacks permission to update its activity file.
- `make check` stops at the known `miniclass-mailpit` named-container conflict;
  the local empty compose volume/network it created were removed afterward.

# Issue #240 — immutable solve runs and drafts

- Added `assignments` as the tenant- and school-year-scoped current draft;
  `solve_runs` retains canonical request/response plus effective v0 quality
  settings and the realized-quality distribution. See SPEC §§8.6, 20.1–20.2.
- `internal/solver.Service.Start` records every completed sidecar response,
  replacing the draft only for feasible/optimal results in the same audited
  tenant transaction. Infeasible and unavailable attempts cannot replace it.
- Reproduction now accepts the current canonical snapshot, forces the recorded
  seed, and returns an explicit mismatch error before running if its fingerprint
  differs. The API maps this to `solve-run-input-mismatch` (409).
- Focused Go package tests pass, including pure response validation. The new
  PostgreSQL integration test is present but skipped locally because
  `TEST_DATABASE_URL` and `TEST_APP_DATABASE_URL` are unset; run `make
  test-backend` in CI or a configured environment for the isolation harness.

# Issue #241 — synthetic CSV scenario harness

- `solver/tests/scenario_harness.py` compiles normalized CSV directories into
  the v1 sidecar request; multiple directories compose without a database.
- `solver/tests/scenarios/` uses opaque synthetic IDs only and covers capacity,
  grade eligibility, ranked and interest preference models, lexicographic
  trade-offs, pins, infeasibility, determinism, and a 140-student / 8-offering
  expected-scale case. `expected.csv` records status and focused assignments.
- `solver/tests/test_scenarios.py` runs fixtures directly through `solve`,
  asserts complete eligible capacity-respecting results, byte determinism, and
  the SPEC §22.2 full/re-solve timing budgets. `PLAN.md` Phase 5 now names this
  synthetic harness instead of historical real-session replay.
- Passed in the pinned `miniclass-solver:latest` image with the workspace
  mounted: `python -m pytest` (38). Local host Python has no pytest; Compose
  image rebuild is blocked by Docker Buildx activity-file permissions.
- Also passed: `make lint-backend`, `make format`, `make generate` with no
  generated drift, and `git diff --check`. `make check` cannot start its first
  gate because an externally owned `miniclass-mailpit` already uses the fixed
  Compose container name; the failed attempt's worktree-local volume and
  network were removed without touching that container.
- PR #260 is open and ready for review. All 12 current-head CI checks passed;
  frontend tests was slowest at about 7m31 while installing Chromium. No PR or
  inline review comments are present.
