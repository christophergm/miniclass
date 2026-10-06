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

# Issue #242 — CI solver deterministic performance budgets

- `backend/internal/solverclient/performance_integration_test.go` reads the
  synthetic expected-scale CSV fixture and exercises the production Go HTTP
  client against `SOLVER_BASE_URL`. It checks canonical byte determinism, a
  full solve below 10 s, and a pinned re-solve below 2 s (SPEC §§17.8, 22.2).
- `make test-solver` delegates to the opt-in backend sidecar test; the existing
  `Solver image` CI job starts and health-checks the image before setting
  `SOLVER_BASE_URL=http://localhost:8090` and running it.
- Local validation: the CSV-to-contract test passes. The live sidecar test is
  blocked locally because localhost:8090 resets connections and Compose cannot
  start a healthy solver in this sandbox; CI must provide live-boundary timing.
- Filed Backlog #261 after CI-contract review: `Backend lint` runs only
  golangci-lint and omits the mapped depguard proof.

## Issue #289 — placement-workspace specification alignment

- `SPEC.md` clarifies session-hosted administrator comments; public and guardian visibility remains
  prohibited.
- Pinned hard-rule exceptions are limited to the named placement, consume their authorised capacity,
  and are discarded on unpin; changed placements do not inherit overrides or assignment comments.
- Validated: `git diff --check` passes; reviewed the cited spec sections against the approved
  Phase 5A plan. No automated test applies to this documentation-only update.

## Issue #290 — persisted assignment editing state

- Adds timestamped migration `20261006100000_assignment_editing_state.sql`: nullable solve-run
  provenance for manual assignments; session `draft_revision`; tenant-scoped assignment exclusions
  and assignment overrides; RLS, closed-year guards, entity factories, and §16.2 bidirectional
  participant/assignment guards.
- Re-solves preserve an assignment row (and its future placement-bound context) when the student
  remains in the same offering. Changed or removed placements are deleted before replacement, so
  overrides cannot migrate. Successful draft replacement advances the session revision.
- Passed `go test ./...` from `backend/`, `make generate`, `make format`, `make lint-backend`
  (including depguard), and `git diff --check`. `make test-migrations` is blocked locally because
  `MIGRATION_ROUNDTRIP_DATABASE_URL` is unset; `make test-backend` cannot start because an external
  `/miniclass-postgres` container holds the fixed Compose name. Its temporary network/volume were
  removed without touching that container.

## Issue #291 — administrator placement workspace

- Adds `GET /api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/assignment-workspace`, protected by `manage_assignments`. It returns the persisted draft revision, participating memberships, offerings, assignments, exclusions, overrides, and current ranked-choice answers; empty drafts serialize each collection as `[]`.
- `internal/program/assignment_workspace.go` assembles the projection in one `InTenantRead` transaction and verifies the full year/program/session scope via `GetSession` before reading related rows. The endpoint implements SPEC §§8.6, 16.2, 17.13, 18.1.
- Passed: `go test ./...` in `backend/`, `make generate`, `make format`, `make lint-backend` (including depguard), and `git diff --check`. `make test-backend`/migration tests cannot start because an externally owned `/miniclass-mailpit` holds the fixed Compose name; created worktree network/volume were removed. Frontend test/build lack `openapi-typescript`; frontend lint reports pre-existing formatting drift in six test files.

## Issue #292 — atomic moves, swaps, and pin controls

- Adds assignment mutation endpoints with optimistic `draft_revision` checking. The revision CAS is the first write in the audited tenant transaction, serializing final-state capacity/grade/exclusion evaluation and rolling back stale or unconfirmed mutations.
- Moves and swaps create manual pinned placements; pin/unpin is persisted. Changed or unpinned placements discard their placement-bound overrides. A new migration permits the SPEC §16.7 optional override reason.
- Focused integration coverage exercises empty-draft placement, swap, pin/unpin, prompted exclusion confirmation, and stale revision rejection. Passed `go test ./...`, `make generate`, `make format`, `make lint-backend`, and `git diff --check`.
- `make check` cannot start because an externally owned `/miniclass-mailpit` holds the fixed Compose container name. The attempted run's worktree-specific Docker volume and network were removed without touching that container.

## Issue #293 — student offering exclusions

- Adds audited, revision-CAS `POST`/`DELETE` assignment-exclusion operations under the assignment-management capability. Add/remove are idempotent no-ops when already in the requested state.
- Adding an exclusion retains any conflicting current assignment and returns it as an explicit conflict; it never moves or unplaces a student. Removing an exclusion clears only that placement's now-obsolete `exclusion` override, retaining independent approvals.
- Focused tests cover cross-program references, rollback of a failed CAS operation, repeats, removal, and conflict visibility. Passed focused Go tests, `make format`, `make lint-backend`, and `git diff --check`; full gate remains to run.
- `make check` reaches Backend tests but cannot start because external container `/miniclass-mailpit` owns the fixed Compose name. Removed only the failed attempt's empty worktree volume/network; do not remove that container.
- CI initially caught missing `NoAuditRequired` declarations on idempotent no-op mutations; amended `a75ad23` adds them. Recheck the replacement PR head's Backend tests before handoff.

## Issue #294 — solver exclusions and authorised pinned exceptions

- In progress. The merged #293 implementation provides the persisted exclusion and
  placement-override concepts; this issue is limited to their canonical solver
  contract, feasibility semantics, and synthetic scenarios (SPEC §§16.3, 16.7,
  17.3, 17.8–17.10).
- Predecessor #293 is closed and `origin/main` is the current worktree base.
- Added canonical exclusions, placement-bound `capacity`, `grade`, and
  `exclusion` exceptions, and inactive prior-placement baselines to both
  contracts. The Python feasibility model permits an exception only on its
  matching pin, reserves capacity excess without spare capacity, and reports
  stale/invalid constraints.
- Passed local solver pytest (51), `go test ./...` in `backend/`, `make
  format`, `make lint-backend`, `make generate`, and `git diff --check`.
  `make check` remains unable to start because the externally owned healthy
  `/miniclass-mailpit` container holds the fixed Compose name; this run removed
  only its own empty network and volume afterward.
- PR #310 is open, ready, and green on all 13 current-head CI checks. The
  slowest were generated-code drift (1m51) and backend tests (1m50); no review
  or inline comments were present at handoff.
