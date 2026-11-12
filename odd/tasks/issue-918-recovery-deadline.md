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
- T2 (S1): systemd / docker adapters run restart via `exec.CommandContext`
  (ctx-aware shims); `DockerService.RestartContext(ctx)`. Commit: pending.
- T3 (S2): systemd readiness bound to the settings-derived URL instead of the
  fixed default endpoint; selector requires a resolvable endpoint. Commit: pending.
- T4 (S3): recovery error mapping — expired recovery context surfaces as an
  ApplyError rollback failure, never a hang or nil. Commit: pending.
- T5 (S4): deterministic tests for deadline-enforced restart, occupied-port
  start, recovery failure mapping + regression for changed shared helpers.
  Commit: pending.

## Log

- L1: "Realiza un plan para resolver la issue #918" — plan approved with
  "Implementea el plan".
