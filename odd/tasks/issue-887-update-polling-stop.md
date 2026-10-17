# Issue #887 — Join update polling before fixture cleanup
Baseline: main a1daeb6. Scope: polling Stop completion and deterministic offline lifecycle tests; no automatic PR merge.
Track: separate focused PR under approved #887; #872 remains blocked. Engram unavailable.

## Specs
- S1: "Genial, veo que la #872 aun ha fallado el CI, como lo podemos resolver?"
- S2: "Si autorizo" — approval of deterministic reproduction, narrow correction in a separate PR under #887, without automatic merge.
- S3 Accepted proposal: "Reproducción determinista y aislada: bloquear una comprobación mediante canales, llamar a Stop() y comprobar si retorna antes de que termine. Sin red, npm ni esperas arbitrarias."
- S4 Accepted proposal: "Corrección separada bajo #887: si se confirma, hacer que el apagado espere al worker de polling. Revisar también cómo cancelar sus operaciones para no introducir un bloqueo indefinido."
- S5 Accepted proposal: "Verificación: RED/GREEN, pruebas del ciclo de vida bajo -race y suite Go completa."
- S6 Accepted proposal: "Retomar #872: una vez integrada esa corrección, actualizar su rama con main, ejecutar CI nueva y mergearla solo si pasa."

## Tasks
- T1 [S1-S4] bounded worker, done: deterministic offline RED/GREEN proves join gap and correction; preserves Stop-before-Start and caller cancellation. Commit: this focused work-unit (see feature branch history).
- T2 [S4-S5] independent verifier after writer checks, done: independent baseline RED/candidate GREEN and concurrency inspection passed; full Go1.26.9 vet/race suite passed. Independent race10 and provisioning timeouts retained, not retried. Commit: this focused work-unit.
- T3 [S2,S5] coordinator, in progress: commit focused work-unit and publish nonclosing Refs #887 PR, verify exact-head CI; do not merge automatically. Commit: this focused work-unit.
- T4 [S6] coordinator, blocked: #872 continuation waits for human integration of lifecycle correction; no blind rerun or dependency merge yet. Commit: none.

## Log
- L1 User: "Genial, veo que la #872 aun ha fallado el CI, como lo podemos resolver?"
- L2 User: "Si autorizo" to proposed reproduction, correction and separate #887 PR without automatic merge.
- L3 CI #872 head64ff70c, run38963380877/job117183741204: TestStartStop TempDir RemoveAll directory not empty, exit1; build skipped. Relevant source identical to base. Poller cache-write overlap is plausible, not yet a reproduction of exact cleanup failure. Existing issue evidence: https://github.com/fgjcarlos/nrcc/issues/887#issuecomment-6224378784.
- L4 Read-only map mvc22fme-1-uem1: Start launches untracked polling; Stop only closes channel despite wait guarantee. Server cancels ctx before Stop. Constructor supports preseeded cache and existing version-function injection. Separate apply Shutdown, repeated Stop/restart semantics out of scope. Preserve Stop-before-Start return; no lock held across join. #887 status:approved freshly verified.
- L5 Writer [S1-S4]: replaced sleep/network-dependent TestStartStop with preseeded JSONStore cache and injected version functions. testing/synctest plus channels proves Stop cannot return during an active check, then verifies persisted cache immediately on return. Both normal and caller-canceled active checks are covered; Stop-before-Start and already-canceled idle Start are preserved. Initial harness run of the focused command below exited 1 after 20s (`panic: test timed out after 20s`): stopChan was created outside the synctest bubble, preventing fake-time advancement. Corrected by constructing the service inside the bubble before observing behavioral RED.
- L6 RED, before production changes: `go test ./internal/service -run '^(TestStartStop|TestUpdateStopBeforeStart|TestUpdateStartCanceledContext)$' -count=1 -timeout=20s` exited 1. Exact output:

```text
--- FAIL: TestStartStop (0.01s)
    --- FAIL: TestStartStop/cancelCaller=false (0.01s)
        update_test.go:393: Stop() returned before the active polling check completed
        update_test.go:403: Stop() returned before polling cache persistence: {CurrentVersion:4.0.0 LatestVersion:4.0.0 UpdateAvailable:false Strategy:npm-global CanInplaceApply:true CheckedAt:0001-01-01 00:00:00 +0000 UTC Error:}
    --- FAIL: TestStartStop/cancelCaller=true (0.00s)
        update_test.go:393: Stop() returned before the active polling check completed
        update_test.go:403: Stop() returned before polling cache persistence: {CurrentVersion:4.0.0 LatestVersion:4.0.0 UpdateAvailable:false Strategy:npm-global CanInplaceApply:true CheckedAt:0001-01-01 00:00:00 +0000 UTC Error:}
FAIL
FAIL	github.com/fgjcarlos/nrcc/internal/service	0.022s
FAIL
```

- L7 GREEN: Start registers a WaitGroup before launching polling; the worker defers Done until pollingLoop (including performCheck/store.Write) returns. Stop closes the existing signal then joins without holding a poller-needed lock. Same focused command as L6 exited 0. Exact output:

```text
ok  	github.com/fgjcarlos/nrcc/internal/service	0.021s
```

- L8 Focused race preservation: `go test -race ./internal/service -run '^(TestStartStop|TestUpdateStopBeforeStart|TestUpdateStartCanceledContext)$' -count=10 -timeout=20s` exited 0. Exact output:

```text
ok  	github.com/fgjcarlos/nrcc/internal/service	1.161s
```

- L9 Lifecycle/risk item 4: Stop waits rather than independently canceling the active check. Existing performCheck npmTimeout and server Shutdown's cancel-before-Stop order remain unchanged; caller cancellation reaches the injected active check in the test. Filesystem writes still finish before worker completion. Non-cooperative injected functions or stalled filesystem I/O can delay Stop; no new cancellation policy, restart/repeated Stop semantics, or apply Shutdown/ForceCheck/constructor changes were introduced. Concurrent Start/Stop remains unsupported and is documented. Exact CI TempDir cleanup failure was not reproduced; only its plausible lifecycle prerequisite was deterministically confirmed. No CLI surface is touched, so command differential checks do not apply. Full Go suite and independent verification remain parent-owned T2/S5; S6 is untouched.
- L10 Independent verifier mvc2a1fx-3-qn1k found no implementation blocker: Add-before-launch, Done after persistence, no mutex held while joining, existing10s timeout and cancel-before-Stop production order preserved; apply Shutdown separate. Offline candidate focused tests exit0/0.021s; baseline a1daeb6 with candidate tests exit1/0.022s for premature return and persistence. Baseline additionally reproduced TempDir RemoveAll directory-not-empty error class, not the exact historical CI execution. Independent timeout/apply-separation probe passed normal0.013s and race1.037s. Scratch/cache provisioning and separate race10 command each exceeded outer30s bounds; no retries, race10 supplied no independent result (writer race10 remains passed). Full suite job-1 pending. Scratch reported at /tmp/issue887-verify.EIg1ks; no cleanup outside project performed by parent.
- L11 Parent full verification job-1 exited0 after7m44s with Go1.26.9: go vet ./..., go test -race -count=1 -timeout20m ./... and git diff --check all passed. Service451.436s, handler274.205s, server53.386s. No production source change after verification. Exact published-head CI remains required before merge; no automatic lifecycle PR merge authorized.
