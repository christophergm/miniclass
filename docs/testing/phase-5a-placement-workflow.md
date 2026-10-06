# Phase 5A placement-workflow demonstration

Use an administrator account to open a session's **Assignments** board.  An empty
draft is intentional: place each synthetic participating student with **Place** or
by dragging them onto an offering, then use the keyboard-accessible **Move** and
**Swap** controls as needed.  Confirm a prompted hard-rule exception deliberately
and record the optional reason; pin, exclude, and re-solve as appropriate.

The board's **Unplaced students** count reaches zero when the persisted draft is
complete.  Review named unwanted, no-signal, overridden, and warning entries
before aggregate metrics.  Warnings remain visible and do not prevent completion.
Reloading the board demonstrates that the saved draft, pins, constraints, and
comments persist.  There is deliberately no publish, finalise, or session-state
control on this surface.

The synthetic browser proof is `frontend/e2e/assignment-workspace.spec.ts`.
Production-boundary coverage is retained by the solver performance test and the
assignment-operation, workspace-authorisation, placement-comment, and stale-run
integration tests (SPEC §§9.2, 16.2–16.7, 17.8–17.13, 18.1, 22.2).
