# Issue 876 — Close the runtime-safe apply transaction

## Objective

Issue #876 (`fix(apply): wire restart, readiness and runtime rollback
into the transaction (P0)`) is still `open` even though its whole
implementation chain is already merged into `main`.

The remaining work is **closure, not implementation**: prove each
acceptance criterion of #876 against the merged bytes, record the
evidence, and close the issue honestly.

No new apply behaviour is introduced by this tracker. If a criterion
turns out to be unproven, that becomes a follow-up issue — it does
not silently become a second implementation inside a closure PR.

## State of the merged chain

| PR | Head branch | Merge commit on `main` | Content |
|---|---|---|---|
| #910 | `fix/876-apply-runtime-rollback` | `93a12b9f4185bdfb9403006e4c9c00d14bb01651` | Chain tracker handbook page |
| #911 | `fix/876-runtime-transaction` | `0037ca94e1f31f7300620841efca8a622839e982` | Guarded transaction core |
| #912 | `fix/876-structured-transaction` | `7e7e5e62930b391d471bec7d633f799b54bbe01d` | Structured state restore |
| #913 | `fix/876-runtime-wiring` | `58363111ac8371bcd074388b4ed5aaa857051844` | Production wiring + audit identities |
| #914 | `fix/876-runtime-acceptance` | `0894604613e4374cc9cb2163479c04d2e5f62ed3` | Real runtime acceptance + operator docs |

Baseline under verification: `origin/main` = `d2c5100`.

## Acceptance criteria → evidence

### 1. Transaction guard survives restart, readiness and recovery; one runtime authority

**MET** — `internal/service/apply_runtime.go:29` resolves the adapter
exactly once (`c.resolve(req)`) and stores it on the request
(`internal/service/apply_runtime.go:39`). The same `adapter` value
drives restart (`:65`), readiness (`:71`) and recovery (`:92`).

Guard: `internal/service/apply_runtime.go:415` test
`TestRuntimeApplyKeepsGuardThroughCandidateAndRecovery`; audit
identity continuity `internal/service/apply_runtime_test.go:166`
`TestRuntimeApplyAuditIdentitySurvivesFailureAndRecovery`; pinned
restart/readiness target `internal/service/apply_runtime_test.go:671`
`TestDockerAdapterPinsRestartAndReadinessToDetectedTarget`.

### 2. Failure restores authoritative state, restarts it and verifies readiness

**MET** — `internal/service/apply_rollback.go:176` restores the
snapshot, `:189` restores additional authoritative structured state,
`:197` restarts the restored configuration, `:204` re-probes
readiness. The restore of the structured configuration is wired at
`internal/service/apply_runtime.go:86-91`.

Tests: `internal/service/apply_runtime_test.go:267`
`TestRuntimeStructuredApplyRestoresConfigBeforeRecoveryRestart`,
`:205` `TestRuntimeApplyFailureRestoresAndSurfacesRecoveryFailure`,
`:388` `TestRuntimeApplyReadinessFailureRestoresAndSurfacesRollbackRestartError`.

### 3. Separate bounded recovery context on cancellation; honest rollback failures

**MET** — `internal/service/apply_runtime.go:83` builds the recovery
context from `context.Background()` with a bounded timeout
(`defaultApplyRecoveryTimeout = 30s`, `:10`; overridable through
`req.RecoveryTimeout`), so a cancelled caller cannot abort recovery.
Rollback failures are surfaced instead of swallowed:
`internal/service/apply_runtime.go:96-98` joins
`recovery failed: %w` into the returned `ApplyError`.

Tests: `internal/service/apply_runtime_test.go:361`
`TestRuntimeApplyRecoveryIgnoresCancelledRequestContext`, `:467`
`TestRuntimeApplyRecoveryTimeoutUnwrapsRollbackFailure`.

### 4. A single terminal success, only after verified readiness

**MET** — the only `apply.success` emission on the production path is
`internal/service/apply_runtime.go:102`, emitted with stage
`ApplyStageReady` after both `Restart` and `Ready` succeeded. The
legacy emission in `internal/service/apply.go:265` is disabled for
this path (`internal/service/apply_runtime.go:52` calls
`c.apply.apply(ctx, req, false)`), and the handler explicitly does
not re-emit (`internal/handler/config.go:167`).

Test: `internal/service/apply_runtime_test.go:190`
`TestRuntimeApplyRestartsAndEmitsSuccessOnceAfterReady`.

### 5. Recovery proven with valid syntax that fails at runtime startup, plus cancellation and concurrent apply

**MET** — `internal/service/apply_runtime_real_linux_test.go:237`
`TestApplyRuntimeRealAcceptance` drives a real Node-RED process and
covers exactly the three required scenarios:

- `failed_TLS_candidate_restores_bytes_and_structured_state` — the
  candidate has valid settings syntax but references missing TLS
  material, so it fails at runtime startup, at the readiness stage.
- `caller_cancellation_recovers_independently` — cancellation of the
  caller while the candidate is starting.
- `same_path_apply_rejects_while_real_apply_is_pending` — a competing
  apply is rejected with `ErrApplyInFlight` and mutates nothing.

Supporting unit-level coverage:
`internal/service/apply_coordinator_test.go:190`
`TestConcurrentAndStaleApplyRejected` and `:290`
`TestApplyCoordinator_CancellationPropagates`.

The real acceptance is **opt-in** (`NRCC_APPLY_REAL_ACCEPT=1`) and is not
wired into any CI workflow, so this criterion is proven by a local
run, not by a hosted check. The run recorded below passed all three
scenarios against Node-RED v5.0.4 / Node.js v24.14.0.

## Residual findings (not part of the criteria, tracked separately)

An independent read-only verification of the merged bytes raised two
adapter-level observations that the transaction criteria do not cover.
They are recorded here instead of being silently dropped:

1. **Native adapters do not honour the recovery deadline.**
   `internal/service/apply_adapter.go:257` runs `systemctl restart`
   through `cmd.Run()` at `:258` without `exec.CommandContext`, and
   `internal/service/apply_adapter.go:218` calls the context-free
   `ProcessManager.Restart`. A hung restart can therefore outlive the
   bounded recovery context created at
   `internal/service/apply_runtime.go:83`.
2. **Adapter identity is not runtime identity for native installs.**
   `internal/service/apply_adapter.go:265` probes the fixed
   `nativeDefaultAdminURL()`, and
   `internal/service/process_linux.go:254` attaches to whatever
   already owns the port instead of starting the managed child.

Both live in the pre-existing adapter layer, not in the transaction
wiring this issue delivered, and neither breaks the criteria above.
They are filed as a separate follow-up issue (#918) so the closure of
#876 stays scoped and honest.

## Tasks

- [x] T1 Map every acceptance criterion to merged code and tests.
- [x] T2 Run the focused Go suites on `origin/main` (`d2c5100`).
- [x] T3 Run the opt-in real Node-RED acceptance
      (`NRCC_APPLY_REAL_ACCEPT=1`) as live evidence.
- [x] T4 Run `golangci-lint` and the repository language scan.
- [x] T5 Publish the closure tracker and close issue #876.

## Evidence log

All commands ran on `origin/main` = `d2c5100` (Go 1.27.3 toolchain).

- `go build ./...` — pass; `go vet ./internal/...` — pass.
- `go test ./... -count=1` — **14/14 packages ok, 0 failures**
  (`internal/service` 275.5 s, `internal/handler` 151.1 s,
  `internal/server` 33.2 s, plus `internal/{audit,config,metrics,
  middleware,model,setup,store}`, `scripts/cite_check` and
  `tools/langscan{,/workflow}`).
- Real runtime acceptance, opt-in:
  `NRCC_APPLY_REAL_ACCEPT=1 go test ./internal/service/ -run
  TestApplyRuntimeRealAcceptance -v -count=1` — **PASS** (22.31 s)
  with Node-RED v5.0.4 / Node.js v24.14.0; all three subtests pass:
  `failed_TLS_candidate_restores_bytes_and_structured_state` (6.91 s),
  `caller_cancellation_recovers_independently` (6.84 s),
  `same_path_apply_rejects_while_real_apply_is_pending` (8.37 s).
- `golangci-lint` v2.14.0 (the version CI pins) — **0 issues**.
- `go run ./tools/langscan . -format markdown` — **[]** (no findings).
- Hosted CI on `main` at `d2c5100` — all runs green: `PR Validation`,
  `Acceptance (Docker stacks)`, `Security`, `i18n checks`,
  `Language policy (advisory)`, `Push on main`.

## Out of scope

- New apply behaviour or refactors of the merged transaction.
- The `#875` raw-settings work that is still open in the primary
  checkout (`fix/875-raw-settings-validation`).
