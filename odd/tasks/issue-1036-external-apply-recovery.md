# Issue #1036 — External Node-RED boot recovery
Source: https://github.com/fgjcarlos/nrcc/issues/1036
Worktree branch: `fix/1036-external-apply-recovery`.

## Specs
- S1: “After restoring the snapshot at boot, the runtime that loaded the candidate runs the restored settings too, or the operator is clearly told it still runs the candidate.”
- S2: “When `RecoverInterruptedApply` restored a snapshot and the active runtime is not NRCC-managed, resolve the runtime adapter the same way an apply does and run `Restart` plus `Ready` against the restored settings, emitting the `apply.rollback.*` audit events. If the adapter cannot be resolved at boot, log a clear warning naming the runtime that still runs the candidate.”
- S3: “With a non-managed runtime, a test that stops NRCC after the candidate restart and before readiness shows the runtime restarted on the restored `settings.js` at the next boot.”
- S4: “A boot with no pending marker restarts nothing.”

## Tasks
- T1 [done] S1–S4 — Recover the external runtime at boot; route: worker (context backstop) + independent verifier (risk item 5: startup behavior); commit: `99bf157`.

## Log
- L1: Resuelve la issue #1036
- L2: Isolated from unrelated #1027 commits in a worktree at `origin/main` (699c475), which already includes the interrupted-apply file recovery. No PR or publication authorized.
- L3: Frontend embed build, focused main/service/server tests, and independent startup verification passed. The verifier confirmed S1–S4 with isolated fixtures and startup probes; no live runtime restart was performed. The writer did not observe a behavioral RED for the original change; the guard RED was an undefined seam.
- L4: `go test ./...` initially failed in `TestSaveConfig_WritesExplicitRuntimeSettingsAndRestarts` (Docker detection found Node-RED v5.0.10, expected stub v5.0.6). The same failure reproduced on base `699c475`; with Docker pointed at an absent socket, the focused handler test, full Go suite, and `git diff --check` all passed (job-3 exit 0).
