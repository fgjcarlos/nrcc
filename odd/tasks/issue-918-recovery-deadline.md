# Issue #918 — native adapters honour the recovery deadline and runtime authority

Fix: make native runtime adapters honour the bounded recovery context and the
runtime authority selected by the apply transaction, so a hung restart cannot
outlive the recovery deadline and a foreign listener cannot satisfy readiness.

## Specs

- S1: "Every adapter restart path is cancellable through the supplied context,
  so the bounded recovery context in `internal/service/apply_runtime.go` bounds
  the whole recovery."
- S2: "The readiness probe is bound to the runtime authority selected for the
  transaction, so a foreign listener cannot satisfy readiness."
- S3: "The transaction reports a rollback failure honestly when a recovery step
  exceeds its deadline."
- S4: "Cover the new behaviour with deterministic tests: deadline-enforced
  restart, occupied-port start, and recovery failure mapping."

## Tasks

- T1 (S1): `ProcessManager.Stop(ctx)` / `Restart(ctx)` bounded by ctx; strict
  restart that refuses to attach to an occupied port. Commit: pending.
  Commit: ad892ed slice 1 (`fix(service): bound ProcessManager Stop/Restart by context and refuse foreign listeners`).
- T2 (S1): systemd / docker adapters run restart via `exec.CommandContext`
  (ctx-aware shims); `DockerService.RestartContext(ctx)`. Commit: ad892ed
  (slice 2, T2+T3 together — the selector change and the ctx shims share one
  behavior: authority-bound, deadline-safe restart).
- T3 (S2): systemd readiness bound to the settings-derived URL instead of the
  fixed default endpoint; selector requires a resolvable endpoint. Commit: ad892ed.
- T4 (S3): recovery error mapping — expired recovery context surfaces as an
  ApplyError rollback failure, never a hang or nil. Commit: verified against
  existing TestRuntimeApplyRecoveryTimeoutUnwrapsRollbackFailure (passes); the
  real adapters now honour the recovery ctx via T1/T2, so no code change was
  needed beyond what T1/T2 deliver.
- T5 (S4): deterministic tests for deadline-enforced restart, occupied-port
  start, recovery failure mapping + regression for changed shared helpers.
  Commit: 9f042b8 + 79826c0. Final verification: new service tests PASS; handler
  (config/settings/system/env) PASS; server suite PASS; recovery mapping PASS.

## Log

- L1: "Realiza un plan para resolver la issue #918" — plan approved with
  "Implementea el plan".
- L2: Verification evidence: T1 tests PASS (process_deadline_test.go, 3 runs);
  T2/T3 tests PASS (apply_adapter_test.go); recovery mapping test PASS;
  internal/server suite PASS (78s); internal/handler suite PASS.
  Pre-existing environment failures (fail identically on origin/main):
  TestRuntimeStructuredAndRawApplyShareFlightGuard,
  TestDashboardApplySharesConfigRuntimeGuard, and a source_patch_fixture
  test that hangs in HostService.Detect → `docker compose version` under
  load. Not caused by this change.
- L3: Slice boundary: slice 1 = 479 authored lines (T1); slice 2 = T2+T3
  (ad892ed); slice 3 = T4/T5 verification and test-fixture fixes.
- L4: Verifier (gentle-ai-verify) findings: B2 (r.Context() in handlers) fixed with context.WithoutCancel (commit 343ef00); B1 (systemd readiness failed only AFTER restart for dynamic-uiPort settings) fixed by validating applyCandidatePort at selection time; docker.go runDockerContext nil-check fixed to the context shim. Added regression tests: TestDockerCustomAdapter_RestartHonoursDeadline, TestSystemdRecoveryHangReportsRollbackFailure, TestSystemdDynamicSettingsFailBeforeRestart. Advisory: manual restart now refuses an occupied port (intended by S2).
