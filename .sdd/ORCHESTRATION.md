# SDD Orchestration Notes

This directory stores project-local guidance for the Gentle AI SDD orchestrator
beyond what `agents/` and the global skill registry already encode.

## Worktree / SDD ledger consistency

`gentle-ai sdd-attempt` binds every running attempt to a specific worktree. The
ledger only tracks work that happens inside the bound `begin_worktree`. If the
implementer (or a delegated sub-agent) executes commits in a *different* worktree
or repository path, the runtime will record `0 changed_lines`, and any
subsequent `status`/`reset`/`rescope`/`acquire` on the same change will return
`objective rescope candidate does not match the terminal zero-drift finish`,
making the change unsafe to reuse until a maintainer repairs the ledger.

Practical consequences observed during `structured-logging-642` (issue #642):

* The replacement build was performed in a pre-created sibling worktree
  (`nrcc-worktrees/structured-logging-642-v3`) while the bound `begin_worktree`
  was the primary checkout. The attempt was recorded as `failed` with full
  provenance preserved; the external result (PR #663) was merged after explicit
  user authorization.
* Always verify the `begin_worktree` value returned by `gentle-ai sdd-attempt status`
  matches the directory in which the sub-agent will actually run before launching
  `gentle-ai sdd-attempt acquire`.

## Strict TDDRED evidence

Strict TDD requires genuine RED on the production-failing code path. Tests that
already pass on untouched `origin/main` cannot be replayed as RED; the apply
agent must re-introduce the production failure surface first. The native
runtime will refuse to settle a corrective attempt whose only artifact is the
recovered implementation.

## Native ledger repair tooling

The `gentle-ai` CLI does NOT currently expose a maintenance command for clearing
stuck terminal zero-drift objectives. When a change is irrecoverable through
`status`/`reset`/`rescope`/`acquire`, stop attempting to relaunch and instead:

1. Settle the current attempt honestly as `failed` or `interrupted`.
2. Record the incident in Engram with the `sdd-.../ledger-drift` topic key.
3. Request maintainer authorization to proceed without further runtime
   verification if external CI signals (GitHub checks, Security workflow) are
   green and the user accepts the residual ledger debt.

