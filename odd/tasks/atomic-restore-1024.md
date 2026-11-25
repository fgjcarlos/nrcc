# Feature: atomic, exclusive, complete restore (#1024)

Branch: `fix/1024-atomic-restore`. Milestone D, Data durability. High risk: touches
stored data, recovery at boot and mutual exclusion (Task Size items 1, 3, 4).

## Specs

- **S1** "A restore ends with either the full old set or the full new set".
- **S2** "A test that interrupts the rename loop ends, after restart, with either the full old set or the full new set."
- **S3** "A restore on a clean volume brings back environment variables, MFA enrolment and uploads."
- **S4** "Restore and apply cannot overlap." Also "Nothing prevents a second restore, a scheduled backup or an apply from running at the same moment."
- **S5** "`package.json` is restored without reinstalling `node_modules`, and Node-RED is restarted with no readiness check." → after a restore that replaced `package.json`, dependencies are reinstalled and Node-RED must come back through the same readiness probe apply uses.
- **S6** "It omits `.env`, the MFA store, uploads and `.config.runtime.json`." and "Add the missing files to the archive and bump the manifest version."

## Tasks

- **T1** — Durable restore journal, rollback of the previous set at boot (S1, S2). Route: inline. Commit: `fix(backup): make restore atomic with a durable swap journal (S1, S2)`.
- **T2** — Data-dir operation lock shared by restore, backup creation and apply (S4). Route: inline. Commit: `fix(backup): serialise restore, backup and apply on one data dir (S4)`.
- **T3** — Reinstall dependencies and confirm Node-RED readiness after a restore (S5). Route: inline. Commit: `fix(backup): reinstall dependencies and confirm readiness after restore (S5)`.
- **T4** — Archive `.env`, the MFA store, uploads and `.config.runtime.json`; manifest version 2 (S3, S6). Route: inline. Commit: `fix(backup): back up every file NRCC owns and bump the manifest (S3, S6)`.

Delivery forecast: ~450 authored changed lines across four work units; each unit stays
under the review budget. Delivery strategy: `ask-on-risk` (no publishing without the user).

## Log

- **L1** — User request, verbatim: "Resuelve la issue #1024"
- **L2** — Issue #1024 body read on this run. Acceptance criteria and evidence lines quoted in `## Specs`. Dependencies #1011 (readiness deadline) and #1014 (Stop during backoff) are already closed on `main`.
- **L3** — Exploration: restore path `internal/service/backup.go` (`Restore` 490, `RestoreWithSafetyBackup` 531, `filesForBackup` 905, `verifyArchiveManifest` 1193, `stageRestore` 1273, `swapStagingIntoDataDir` 1317), lock primitive `flockExclusive` (`flowlock_linux.go`), readiness probe `probeAdminReady` (`apply_readiness.go:174`), boot recovery model `RecoverInterruptedApply` (`apply_pending.go:46`, called from `main.go:93`). `.config.runtime.json` has no producer in the tree; it is archived only when present, and absent files stay skipped by `addFileToZip`.
- **L4** — Decisions: (a) interrupted swap rolls **back** to the previous set at boot; the acceptance criterion allows either set and rollback never needs the archive. (b) Exclusion uses an OS flock on `<dataDir>/nrcc-data-op.lock`, so a crash releases it and the scheduler is covered too. (c) Readiness/dependency hooks are optional setters so existing tests keep their contract.
