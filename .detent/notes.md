# Issue #303 — named-first quality review and comments

- Frontend review panel is in progress: named unwanted/no-signal/override lists,
  warning acknowledgement indicators, occupancy/distribution, placement details,
  and administrator comment create/edit/delete wrappers use the existing APIs.
- Rework on PR #319 fixes the current-head CI findings: test assertions use
  unique controls or scope duplicated detail text to its modal, and the panel
  avoids ES2021-only `replaceAll` and the unsupported `link` button variant.
- Passed: changed-file `bunx --bun biome check` and `git diff --check`.
  `make test-frontend` cannot start because `openapi-typescript` is absent from
  the uninstalled frontend dependency tree; CI must run frontend test/build.
- Final amended-head CI passed all 13 checks, including frontend test/build,
  backend test/lint/format, generated-code drift, migration round-trip, and
  developer tooling. Generated-code drift (2m) and backend tests (1m41) were
  the slowest checks; no review or inline comments are present.

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

## Issue #295 — authoritative session solve snapshots

- `backend/internal/solver/snapshot.go` builds the v1 request from one
  tenant-scoped persisted snapshot: participating membership/grades, offerings,
  current session choices, effective interest profiles, pins, exclusions,
  placement-bound exceptions, effective weights, and the current draft as the
  stability baseline. The snapshot records `draft_revision` for write freshness.
- The solve HTTP endpoints no longer accept a browser-supplied solver request.
  Starts accept only an optional seed; reruns reconstruct current state and use
  the recorded seed and fingerprint guard. A draft changed during sidecar work
  returns a conflict rather than replacing newer placements.
- Passed: `go test ./...` in `backend/`, `make generate`, `make format`,
  `make lint-backend` (depguard passed), and `git diff --check`. `make
  test-backend` cannot start because external `/miniclass-mailpit` owns the
  fixed Compose container name; this worktree's empty network and volume were
  removed without touching it.

## Issue #296 — stable re-solves below preference quality

- `solver/app/server.py` now fixes every SPEC §17.4.2 quality optimum before
  maximizing retained, unpinned placements from the authoritative
  `prior_placements` baseline (SPEC §17.9). Missing or now-ineligible baseline
  rows are inert; pins and their authorised exceptions are fixed constraints,
  not stability choices.
- Focused sidecar tests cover unchanged/incomplete baselines, a small exclusion
  edit, strict quality non-interference, seeded replay, and a grade-exception
  pin. Passed in `miniclass-solver:latest`: `python -m pytest` (55).
- Passed `go test ./internal/solvercontract ./internal/solverclient
  ./internal/solver`, `make format`, `make lint-backend` (including depguard),
  and `make generate` without generated drift. `make check` reached Backend
  tests but cannot start locally because external `/miniclass-mailpit` owns the
  fixed Compose name; removed only this worktree's empty network and volume.
- PR #312 is open and ready. All 13 current-head CI checks passed; generated
  code drift was slowest (2m4s), followed by backend tests (1m41s). No review
  or inline comments were present at handoff.

## 2026-10-06T13:06:27Z - Failed run output tail

- final_state: failed
- error: stream turn: context canceled
close codex app-server transport: context canceled
worker github REST budget monitor failed: Get "https://api.github.com/rate_limit": context deadline exceeded

Output tail:

```text
I’ll resume issue #298 from its tracker and repository state, update the persistent Workpad, then implement and validate the narrowly scoped stale-draft protection.The current service detects a changed draft revision, but rolls back the entire transaction—so the stale solve run is lost. I’ll change that boundary so the run persists with a distinct non-applied outcome while draft mutation remains guarded and atomic.This change needs a tenant-scoped schema migration and its isolation proof, so I’m applying the repository’s tenant-isolation-harness guidance before editing the data layer.The referenced isolation-harness skill file is unavailable at its declared path, so I’ll follow the repository’s existing tenant-entity registry and integration-test patterns directly instead.
```

## Issue #300 — attributable placement workspace notes

- Adds tenant-scoped `placement_comments` for assignment, offering, and session hosts. The host
  identifier is intentionally retained after a replaced assignment is deleted, so a comment stays
  attributable to its prior placement and can never transfer to a replacement (SPEC §20.3).
- Comments are administrator-only `manage_assignments` API resources. Writes use audited tenant
  transactions; only the author can edit or soft-delete, and edit audit summaries retain prior
  text. Workspace responses include active comments only and are protected at the response route
  by the administrator capability (SPEC §§10.5, 16.6, 20.3–20.4).
- Passed `go test ./...` in `backend/`, `make format`, `make lint-backend` including depguard,
  `make generate`, and `git diff --check`. Migration round-trip and compose-backed backend test
  gates still need their configured database/container environment. Current-head PR #316 CI later
  passed all 13 checks, including backend tests and migration round-trip.

## 2026-10-06T13:10:41Z - Failed run output tail

- final_state: failed
- error: stream turn: context canceled
close codex app-server transport: context canceled: stderr: [2m2026-10-06T13:10:41.619424Z[0m [31mERROR[0m [2mcodex_core::tools::router[0m[2m:[0m [3merror[0m[2m=[0mexec_command failed: UnknownProcessId { process_id: 98719 }
worker github REST budget monitor failed: Get "https://api.github.com/rate_limit": context deadline exceeded

Output tail:

```text
I’m resuming #298 from the existing worktree. I’ll first inspect the issue, Workpad, and the saved partial change; because this alters tenant-scoped persistence, I’m using the repository’s tenant-entity and isolation-harness guidance before touching the implementation.
```

## Issue #298 — stale draft solve protection

- Adds immutable `application_status`: `applied`, `superseded`, or
  `not_applicable`. A stale successful run is retained and audited without
  replacing assignments or advancing the revision. SPEC §§17.9–17.10, 20.1–20.2.
- Passed `go test ./...` in `backend/`, format, lint/depguard, generation, and
  `git diff --check`. Migration round-trip needs CI because its database URL is unset.

## Issue #299 — draft warnings and placement quality

- Adds the administrator-only `GET .../assignment-quality` read model. It computes
  non-blocking assignment/offering/session warnings and named draft review lists
  from one tenant-scoped snapshot; stored realized quality stays historical while
  current preference context is shown separately. Implements SPEC §§16.5–16.6,
  17.13, 19.1–19.2.
- Focused Go tests, `make format`, `make lint-backend` (including depguard),
  `make generate`, and `git diff --check` pass when Go and golangci caches are
  redirected to the Detent temporary directory.

## 2026-10-06T14:23:19Z - Failed run output tail

- final_state: failed
- error: stream turn: context canceled
close codex app-server transport: context canceled: stderr: [2m2026-10-06T14:12:01.759585Z[0m [31mERROR[0m [2mcodex_core::tools::router[0m[2m:[0m [3merror[0m[2m=[0mapply_patch verification failed: Failed to find expected lines in /Users/chrismott/dev/personal/miniclass-worktrees/miniclass-christophergm_miniclass_299-395621e32924/backend/internal/program/assignment_quality_test.go:
		Offerings: []data.Offering{{ID: "offering", Capacity: 1, MinimumViableEnrollment: &minimum, MinGradeLevelID: "grade-1", MaxGradeLevelID: "grade-1"}},
[2m2026-10-06T14:14:40.651555Z[0m [31mERROR[0m [2mcodex_core::tools::router[0m[2m:[0m [3merror[0m[2m=[0mapply_patch verification failed: Failed to find expected lines in /Users/chrismott/dev/personal/miniclass-worktrees/miniclass-christophergm_miniclass_299-395621e32924/backend/internal/program/assignment_quality.go:
	Answers     []data.SessionResultAnswer
[2m2026-10-06T14:23:12.712733Z[0m [31mERROR[0m [2mcodex_core::tools::router[0m[2m:[0m [3merror[0m[2m=[0mexec_command failed: UnknownProcessId { process_id: 11417 }
worker github REST budget monitor failed: Get "https://api.github.com/rate_limit": dial tcp: lookup api.github.com: no such host

Output tail:

```text
I’ll pick up #299 from the live tracker and worktree state, update its persistent Workpad, then implement and validate the issue’s scoped behavior. Since this changes tenant-scoped persistence, I’ll use the tenant-isolation guidance before editing it.The workspace is clean and already includes #298’s merged predecessor on `main`; #299 has no pull request yet. I found the repository-local tenant-isolation and entity guidance, so I’ll apply those patterns to the required migration and tests.I’ve confirmed the scope is a read-only assignment-quality dashboard—not a new persistence model. I’m adding it as a single tenant-scoped snapshot projection, so warnings are recomputed from current draft context while each assignment’s stored realized quality remains historical.The implementation is in place with a focused pure evaluator test, but local Go validation is presently blocked before compilation: the configured Go 1.26.4 toolchain download cannot update the host module checksum cache under this sandbox. I’ll still inspect and tighten the generated/API wiring, then run all checks that are available.The focused backend tests now pass. The endpoint and OpenAPI are generated; the report includes named unplaced/unwanted/no-signal/overridden placements, stored quality alongside current preference context, occupancy, and the required assignment, offering, and catalog/staleness warning IDs.Formatting completed. Backend lint reached the linter but its default host-local cache is read-only in this Detent sandbox; I’ll rerun it with its cache redirected into the Detent temporary directory.The backend test gate, formatting, lint/depguard, generation, and whitespace checks are green; migration round-trip is the only local gate unavailable because its required database URL is unset. I’ve committed the scoped implementation and will now push it and open the ready-for-review PR.PR [#315](https://github.com/christophergm/miniclass/pull/315) is open and ready (not draft), references #299, and currently has no reviews, inline comments, or CI runs yet. I’m recording the evidence and the remaining CI/migration wait in the Workpad; the issue stays in progress until current-head required checks complete.CI is now running on the PR’s current head. The first completed checks are green (`Solver contract tests`, `Frontend lint`, and `Repository formatting`); the remaining project gates are in progress, including backend tests, migration round-trip, generation, and frontend build.All 13 current-head CI checks are now successful, including backend tests, migration round-trip, generated-code drift, frontend test/build, and developer tooling. The PR is clean, ready, and has no actionable review feedback.
```

## Issue #301 — assignment board and solve workflow

- Frontend-only route: `programs/:programId/sessions/:sessionId/assignments` reads the persisted
  workspace, quality projection, and immutable solve-run endpoint.
- It makes unplaced students prominent, shows offering occupancy, placements and pins, filters by
  student, supports an empty/manual-first draft, and preserves the board through infeasible,
  superseded, unavailable, and access-error outcomes. Placement mutations remain deferred.
- Added component coverage for persisted placements, the unplaced area, and re-solve initiation.
- Changed-file Biome formatting and `git diff --check` pass. `make test-frontend` cannot start:
  `openapi-typescript` is missing from the uninstalled dependency tree. `make lint-frontend` reaches
  formatting but reports pre-existing drift in six unrelated frontend test files.

## 2026-10-06T16:38:37Z - Failed run output tail

- final_state: failed
- error: stream turn: context canceled
close codex app-server transport: context canceled: stderr: [2m2026-10-06T15:50:03.203641Z[0m [31mERROR[0m [2mcodex_core::tools::router[0m[2m:[0m [3merror[0m[2m=[0mapply_patch verification failed: Failed to find expected lines in /Users/chrismott/dev/personal/miniclass-worktrees/miniclass-christophergm_miniclass_301-d396a4f47529/frontend/src/lib/apiResources.ts:
export type ResponseTracking = Schemas["ResponseTrackingResponse"];
export type InterestProfileResults = Schemas["InterestProfileResultsResponse"];
export type RankedChoiceResults = Schemas["RankedChoiceResultsResponse"];
[2m2026-10-06T15:55:50.977092Z[0m [31mERROR[0m [2mcodex_core::tools::router[0m[2m:[0m [3merror[0m[2m=[0mapply_patch verification failed: invalid patch: multiple operations target /Users/chrismott/dev/personal/miniclass-worktrees/miniclass-christophergm_miniclass_301-d396a4f47529/frontend/src/features/programs/AssignmentBoardPage.tsx
[2m2026-10-06T15:59:21.665559Z[0m [31mERROR[0m [2mcodex_core::tools::router[0m[2m:[0m [3merror[0m[2m=[0mapply_patch verification failed: Failed to find expected lines in /Users/chrismott/dev/personal/miniclass-worktrees/miniclass-christophergm_miniclass_301-d396a4f47529/.detent/notes.md:
# Issue #300 — attributable placement workspace notes
worker github REST budget monitor failed: Get "https://api.github.com/rate_limit": dial tcp: lookup api.github.com: no such host

Output tail:

```text
I’ll resume #301 from its live issue, pull request, Workpad, and repository state, then carry the assignment-board workflow through the configured review gate.#300 is merged and #301 has no existing Workpad or pull request. I’m recording the scoped plan now, then I’ll map the existing workspace and solve APIs into the frontend rather than introducing new backend state.The backend already exposes all required persisted projections and solve-run endpoints, so this remains a frontend-only change. I’ll add an `/assignments` session route, typed resource wrappers, and a focused board that refreshes the persisted draft after each solve outcome.The board is now wired to persisted workspace/quality reads and the existing immutable solve-run API. It deliberately leaves placement controls out of scope, while surfacing no-draft, infeasible, superseded, and unavailable outcomes without clearing the draft.Local frontend gates are constrained by an unprepared dependency tree: the generated-client command is absent, and the repository-wide lint additionally reports six pre-existing formatting files. I found cached tool binaries, so I’m using those read-only caches to generate and type-check this change without installing anything.The focused UI test and changed-file formatting are in place. Full frontend execution still cannot bootstrap because this worktree has no installed frontend package tree; I’ve recorded that separately from the unrelated repository-wide formatter drift.PR #317 is open, ready for review, and references #301. All required checks have started; I’ve captured the initial state in the Workpad and will monitor the current head rather than treating local dependency setup as a delivery blocker.CI has identified one failure: `Frontend lint`. I’m checking its log now to distinguish our changed files from the known repository-wide formatter drift before deciding whether a scoped correction is needed.
```

## Issue #301 — lint follow-up

- PR #317's only failed current-head check was Frontend lint: three
  `react-hooks/exhaustive-deps` warnings in `AssignmentBoardPage.tsx` from unstable empty fallback
  arrays. The collections are now memoized before dependent maps and sets.
- Changed-file Biome formatting and `git diff --check` pass. The worktree has no frontend dependency
  tree, so the full frontend test/lint/build gate remains CI-backed.

- Final amended-head CI passed all 13 checks: Frontend lint, tests, and build; Backend tests, lint,
  and format; generated-code drift; migration round-trip; repository formatting; developer tooling;
  and solver/image smoke checks. The rerun took about two minutes; backend tests and generated-code
  drift were the slowest checks. PR #317 is ready, with no review or inline comments.

## Issue #302 — accessible placement editing controls

- The frontend board now uses persisted revision-CAS operations for moves, swaps, pin/unpin, and
  assignment exclusions. Offering drops open the same explicit move action as keyboard controls;
  same-offering drops are no-ops and student drops never imply a swap.
- Hard-rule conflicts prompt for deliberate confirmation with an optional reason. Draft revision
  conflicts refresh the persisted board and explain that no retry occurred.
- Component coverage includes keyboard move, swap, pin, exclusion, and override cancellation.
  Changed-file Biome lint/format and `git diff --check` pass. `make test-frontend` cannot run here
  because `openapi-typescript` is absent from the uninstalled dependency tree; CI must run generated
  types and the full frontend gates.

## 2026-10-06T18:08:27Z - Failed run output tail

- final_state: failed
- error: stream turn: context canceled
close codex app-server transport: context canceled
worker github REST budget monitor failed: Get "https://api.github.com/rate_limit": dial tcp: lookup api.github.com: no such host

Output tail:

```text
I’m resuming the #303 rework by checking the live issue, pull request feedback, current diff, and the existing Workpad. Then I’ll make only the review-requested corrections, re-run the complete gate, and update the handoff evidence.
```

## 2026-10-06T19:24:11Z - Failed run output tail

- final_state: failed
- error: stream turn: context canceled
close codex app-server transport: signal: killed
worker github REST budget monitor failed: Get "https://api.github.com/rate_limit": context deadline exceeded

Output tail:

```text
I’m resuming the #303 rework from the live issue, pull request, feedback, and existing local change. I’ll preserve the partial work, address only actionable review findings, then re-run the required gate and update the persistent Workpad.The rework is limited to two CI findings: the new test made uniqueness assertions against text that now appears in both the board and review panel, and the review panel uses an unsupported string API/button variant for the project’s TypeScript target. I’m correcting those narrow points and will run the affected component test and frontend build if the installed dependency tree permits it.The source corrections are applied and the whitespace gate is clean. This workspace still lacks `frontend/node_modules`, so I’ll document the local limitation without installing dependencies; CI will execute the actual test and build gates on the amended head.
```
