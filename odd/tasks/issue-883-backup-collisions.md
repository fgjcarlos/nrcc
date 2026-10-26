# Issue #883 — Prevent settings snapshot collisions between rapid applies (P1)

## Objective

Issue #883 (`fix(backups): prevent settings snapshot collisions between rapid applies (P1)`)
reports that two successful applies that land inside the same second share one
backup path, so the second transaction overwrites the recovery snapshot of the
first one.

- Issue: <https://github.com/fgjcarlos/nrcc/issues/883>
- Branch: `fix/883-backup-collisions`
- Base: `origin/main` = `3c0fb02`

## Problem and why it matters

Every mutation of `settings.js` keeps a recovery snapshot of the state it
replaced. The snapshot is the rollback target: when a post-write stage fails,
the pipeline restores the bytes the snapshot captured (see
`internal/service/apply_rollback.go`).

The snapshot name is built from a one-second-resolution timestamp and the file
is written with an overwrite-capable write:

- `internal/service/apply.go` — `applyBackup()` builds
  `settings-<YYYYMMDD-HHMMSS>.js.bak` and writes it with `os.WriteFile`.
- `internal/service/config.go` — `backupSettingsFile()` builds the same name
  and writes it with `os.WriteFile` into the same `backups/settings`
  directory.

Consequences on the current baseline:

1. Two applies inside the same second return the same `BackupPath` and write
   the same file: the second transaction destroys the first one's recovery
   point.
2. The operator who has to undo the first change has no snapshot of the
   state that existed before it.
3. The rollback of the first transaction would restore the *second* change's
   preceding state — the recovery point is silently rolled forward.

The rollback path itself is not broken: `ApplyResult.BackupPath` always
points at a readable snapshot. What is broken is the guarantee that the
snapshot is the state *that transaction* replaced.

## Authorized policy and constraints

- Every transaction keeps **its own** immutable recovery snapshot.
- The snapshot name is collision-resistant: it keeps the UTC timestamp (so
  the directory listing stays readable and sorts chronologically) and adds a
  random token.
- The snapshot is created **exclusively** (`O_EXCL`). An existing snapshot is
  never overwritten, not even partially. When a name is already taken the
  writer picks a new token and tries again; it never falls back to an
  overwrite-capable write.
- The snapshot bytes are durable: the file is fsync'ed before the transaction
  continues.
- The discovery tooling keeps working: the new names still match the
  `settings-*.js.bak` / `*.js.bak` patterns, still sort chronologically and
  are still found by `latestBackupFile()` (which reads the modification
  time, not the name).
- The same defect exists in `ConfigService.backupSettingsFile()`, which
  writes into the *same* `backups/settings` directory. Both paths share one
  writer so a structured save and an apply can never clobber each other.
- The post-mortem `.failed-` sibling the rollback keeps next to the live
  file had the same defect (one-second timestamp, overwrite-capable
  write). It is not a recovery point, but it is per-transaction evidence,
  so it gets the same guarantee.
- Repository language rule: every artifact in this feature is written in
  English (see `AGENTS.md`).
- No merge, no issue-label change, no issue closure: those stay human
  decisions.

## Tasks

- [x] **T1 — Baseline audit (RED).** Run the issue's reproduction test on
      `origin/main` and record the failure.
- [x] **T2 — Snapshot writer.** A collision-resistant, exclusive-creation
      snapshot writer shared by every settings backup.
- [x] **T3 — Wiring.** The apply pipeline and the structured save use the
      shared writer.
- [x] **T4 — Regression coverage.** Uniqueness, exact previous bytes,
      creation failures, recovery from each snapshot and the discovery
      compatibility of the new naming scheme.
- [x] **T5 — Verification.** Go build/vet/test, golangci-lint, the language
      scanner and the frontend gates.

## Implementation record

### Backend

| File | Change |
|------|--------|
| `internal/service/apply_snapshot.go` (new) | The shared snapshot writer: collision-resistant name, exclusive creation, durable bytes. |
| `internal/service/apply.go` | `applyBackup` uses the shared writer instead of `os.WriteFile`. |
| `internal/service/config.go` | `backupSettingsFile` uses the shared writer instead of `os.WriteFile`. |
| `internal/service/apply_rollback.go` | The post-mortem `.failed-` sibling gets the same treatment: a random token in the name and an exclusive creation. |
| `internal/service/apply_snapshot_test.go` (new) | The writer tests: uniqueness, exact bytes, exclusive creation, creation failure, discovery compatibility. |
| `internal/service/apply_test.go` | The apply pipeline keeps a distinct snapshot per transaction. |
| `internal/service/apply_rollback_test.go` | Two rollbacks keep two distinct post-mortem artifacts. |
| `internal/handler/config_apply_test.go` | The issue's reproduction test. |

## Acceptance criteria

- [x] Every successful apply returns a distinct snapshot path. —
      `TestReleaseAuditRapidAppliesMustKeepDistinctSnapshots`,
      `TestApplyService_RapidAppliesKeepDistinctSnapshots`
- [x] Each snapshot holds the exact bytes of the state that transaction
      replaced. — `TestApplyService_RapidAppliesKeepDistinctSnapshots`,
      `TestApplyService_SnapshotKeepsExactPreviousBytes`
- [x] An existing snapshot is never overwritten. —
      `TestWriteSettingsSnapshot_ExistingFileIsNeverOverwritten`,
      `TestApplyService_ExistingSnapshotIsNotOverwritten`
- [x] A backup creation failure rejects the transaction and leaves the live
      file untouched. — `TestApplyService_BackupCreationFailureKeepsLiveFile`
- [x] Each snapshot can be used to recover the state it captured. —
      `TestApplyService_EachSnapshotRestoresItsOwnState`
- [x] Two rollbacks inside the same second keep two distinct post-mortem
      artifacts. — `TestExecuteRollback_KeepsDistinctFailedSiblings`
- [x] The new names stay compatible with the discovery tooling. —
      `TestSettingsSnapshotName_StaysDiscoverable`

## Verification evidence

Baseline (RED), on `origin/main` = `3c0fb02`, before the change:

```
--- FAIL: TestReleaseAuditRapidAppliesMustKeepDistinctSnapshots (0.06s)
    config_apply_test.go: two successful transactions reused and overwrote snapshot /tmp/.../backups/settings-20261014-120000.js.bak
```

After the change:

| Check | Command | Result |
|-------|---------|--------|
| Backend build | `go build ./...` | pass |
| Backend vet | `go vet ./internal/...` | pass |
| Backend tests | `go test ./internal/...` | pass |
| Backend lint | `golangci-lint run --config=.golangci.yml ./...` | `0 issues.` |
| Frontend lint | `corepack pnpm lint` | pass |
| Frontend typecheck | `corepack pnpm typecheck` | pass |
| Frontend tests | `corepack pnpm test --run` | pass |
| Language policy | `go run ./tools/langscan .` | `[]` |

## Delivery

- Pull request: <https://github.com/fgjcarlos/nrcc/pull/935>
- Branch: `fix/883-backup-collisions`, based on `origin/main`
- CI: green.

- The merge is the human's decision. The issue closes when the pull request
  is merged.

## Known boundaries

- The snapshot directory grows by one file per transaction. Nothing prunes
  the `backups/settings` directory today (the retention policy of the backup
  service applies to the `.zip` archives of the full-backup provider, not to
  the settings snapshots), so the change does not alter the retention
  behaviour.
- Old snapshots created before this change keep their legacy
  `settings-<timestamp>.js.bak` name. They are still discovered by the
  existing patterns and are never overwritten by the new writer.
- The random token comes from `crypto/rand`. When the system entropy source
  fails, the transaction fails at the backup stage instead of falling back to
  a predictable name.
