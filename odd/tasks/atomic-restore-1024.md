# Feature: atomic, exclusive, complete restore (#1024)

Branch: `fix/1024-atomic-restore`. Milestone D, Data durability. High risk: touches
stored data, recovery at boot, concurrency and shared backup creation (Task Size
items 1, 3, 4).

## Specs

- **S1** "A restore ends with either the full old set or the full new set".
- **S2** "A test that interrupts the rename loop ends, after restart, with either the full old set or the full new set."
- **S3** "A restore on a clean volume brings back environment variables, MFA enrolment and uploads."
- **S4** "Restore and apply cannot overlap." Also "Nothing prevents a second restore, a scheduled backup or an apply from running at the same moment."
- **S5** "`package.json` is restored without reinstalling `node_modules`, and Node-RED is restarted with no readiness check." → after a restore that replaced `package.json`, dependencies are reinstalled and Node-RED must come back through the same readiness probe apply uses.
- **S6** "It omits `.env`, the MFA store, uploads and `.config.runtime.json`." and "Add the missing files to the archive and bump the manifest version."

## Tasks

- [x] **T1** — Durable restore journal (each line written and fsynced before dataDir moves), rollback of the previous set at boot (S1, S2). Route: inline. Commit: `4a8bb05`.
- [x] **T2** — Data-dir operation lock shared by restore, backup creation and apply (S4). Route: inline. Commit: `fbdd8ac`.
- [x] **T3** — Reinstall dependencies and confirm Node-RED readiness after a restore (S5). Route: inline. Commit: `f78ae2b`.
- [x] **T4** — Archive `.env`, the MFA store, uploads and `.config.runtime.json`; manifest version 2 (S3, S6). Route: inline. Commit: `3aba1f0`.
- [x] **T5** — Record the verification evidence, the advisories and the residual-window analysis (docs only). Route: inline. Commit: this one.

Delivery forecast: ~450 authored changed lines; actual ~1380 across five commits. The user
chose `chain_strategy=feature-branch-chain` when publishing. Slice boundaries: PR1 = `4a8bb05`
(journal + boot rollback, ~590), PR2 = `fbdd8ac` (data-dir lock, ~220), PR3 = `f78ae2b`
(dependency reinstall + readiness, ~240), PR4 = `3aba1f0` (archive coverage + manifest v2,
~205), PR5 = this documentation commit. Tracker branch: `feature/1024-atomic-restore`.

Evidence:

- `go test ./internal/service -run 'TestRestore|TestRecoverInterruptedRestore' -count=1` — PASS.
- `go test ./internal/service -run 'TestRestoreAndApplyCannotOverlap|TestDataDirLockIsReentrantFree|TestSafetyBackupRestoreDoesNotDeadlockOnItsOwnLock' -count=1` — PASS.
- `go test ./internal/service -run 'TestRestoreReinstallsDependenciesAndWaitsReady|TestRestoreSkipsDependencySyncWithoutPackageJSON|TestRestoreReadinessFailureIsReported' -count=1` — PASS.
- `go test ./internal/service -run 'TestBackupCoversEverythingNRCCOwns|TestBackupManifestVersionIsCurrent|TestRestoreAcceptsVersionOneArchive' -count=1` — PASS.
- Differential check (shared creation/restore code): `go test ./internal/handler -run 'TestBackup|TestRestore|TestUpdate'` and `go test ./internal/service -run 'TestBackup|TestUpdate|TestRestore'` on `origin/main` and on the branch — same stdout tail, exit 0 both.
- `go test ./...` — one failure, `TestSaveConfig_WritesExplicitRuntimeSettingsAndRestarts` in `internal/handler`; reproduced unchanged on `origin/main` (environment reports `Source:env`), so it is pre-existing, not a regression.

## Log

- **L1** — User request, verbatim: "Resuelve la issue #1024"
- **L2** — Issue #1024 body read on this run. Acceptance criteria and evidence lines quoted in `## Specs`. Dependencies #1011 (readiness deadline) and #1014 (Stop during backoff) are already closed on `main`.
- **L3** — Exploration: restore path `internal/service/backup.go` (`Restore` 490, `RestoreWithSafetyBackup` 531, `filesForBackup` 905, `verifyArchiveManifest` 1193, `stageRestore` 1273, `swapStagingIntoDataDir` 1317), lock primitive `flockExclusive` (`flowlock_linux.go`), readiness probe `probeAdminReady` (`apply_readiness.go:174`), boot recovery model `RecoverInterruptedApply` (`apply_pending.go:46`, called from `main.go:93`). `.config.runtime.json` has no producer in the tree; it is archived only when present, and absent files stay skipped by `addFileToZip`.
- **L4** — Decisions: (a) an interrupted swap rolls **back** to the previous set at boot; the acceptance criterion allows either set and rollback never needs the archive. (b) Exclusion uses an OS flock on `<dataDir>/nrcc-data-op.lock`, so a crash releases it and the scheduler is covered too. (c) Readiness/dependency hooks are optional setters so existing tests keep their contract.
- **L5** — T1 landed with four tests: interruption mid-swap, idempotent rollback, a failed swap undone in the same run, and a journal-only crash that must leave the old set untouched. `RecoverInterruptedRestore` is wired in `main.go` next to `RecoverInterruptedApply`, before Node-RED starts. The journal line is written and fsynced *before* any rename, which is what makes the rollback total and lets it work from the journal alone — the earlier draft had to walk `.replaced` and produced a gosec G122 finding.
- **L6** — T2 landed: `ErrDataDirBusy`, `acquireDataDirLock` (flock on Linux, mutex fallback elsewhere). `Restore`/`RestoreWithSafetyBackup`/`Create`/`CreateTyped` and `ApplyCoordinator.Apply` take it once; internal helpers (`restore`, `restoreWithSafetyBackup`, `createBackup`) never do. A busy data dir surfaces as `ErrApplyInFlight` on the apply path so the existing 409 contract holds.
- **L7** — T3 landed: `SetRestoreRecoveryHooks`, `NpmPackageManager.InstallDependencies`, exported `ProbeNodeRedReady`. Server wiring reinstalls in `BackupService.DataDir()` and probes `127.0.0.1:1880/settings`. Restart and readiness failures are recorded as events and now also returned to the caller; the files stay restored.
- **L8** — T4 landed: payload always contains `.env`, `mfa.json`, `.config.runtime.json` and `uploads/**`; writer emits manifest version 2, reader accepts 1 and 2. README documents what an archive holds and the restore guarantees.
- **L9** — Known limits, deliberately out of scope: a v1-only NRCC build rejects a v2 archive (no downgrade path); restore has no UI progress surface; `backup-settings.json` in the README data layout still does not match `backup_config.json` (pre-existing doc drift).
- **L10** — Independent verification (`gentle-ai-verify`, high risk items 1/3/4) returned **S1–S6 met, no blockers**, with adversarial probes beyond the committed tests: crash at every rename point k=0..6 including nested `uploads/sub/x`, crash inside the rollback, torn journal lines, a real cross-process lock probe (HELD → BUSY → ACQUIRED), reader rejection of symlink/traversal/unknown/oversized/v0/v3 entries, and restart-failure propagation.
- **L11** — Advisories resolved or recorded. (1) *Fixed inside T1*: the journal line is durable before any rename, so no crash leaves a destructive step unrecorded and the rollback reads only the journal; `TestRollbackAfterJournalOnlyCrashKeepsOldSet` pins it. (2) A crash between the last journal append and staging removal yields the old set — S1 permits either, no action. (4) A hand-made `restore-staging-*` directory in dataDir is treated as journal and removed — acceptable, documented. (5) Several leftover staging dirs: one is rolled back per boot, each boot making progress.
- **L12** — Advisory left open on purpose: `ResticProvider.Restore` (`backup_restic.go:235`, driven from `handler/backup.go:493`) can write into the live dataDir without the new lock. That path is pre-existing, untouched by this branch, and outside issue #1024's evidence; closing it needs a deliberate data-lock API decision, so it is reported as a follow-up rather than widened here.
- **L13** — First CI round on the chain: the five PRs passed everything except `Lint (golangci-lint)`, which reported two findings and failed on every slice. G122 (walk-based rollback) was fixed by making the rollback journal-driven as described in L5. G703 (`flowlock_linux.go`) was introduced by routing the new data-dir lock through `flockExclusive`, so `acquireDataDirLock` now opens its own `<dataDir>/nrcc-data-op.lock` with a justified `#nosec`, leaving `flowlock_linux.go` untouched. Both fixes were folded into the slices that introduced them (T1 and T2) and the chain was rebuilt so every PR is lint-clean on its own.
