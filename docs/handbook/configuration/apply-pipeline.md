# Apply pipeline (source-preserving configuration)

> **Four official stages. Eight observable steps.** The apply
> pipeline is the only path NRCC uses to write `settings.js`. It is
> source-preserving, atomic, audited, and restart-aware. This page
> is the operator's contract: what the pipeline does, what it
> refuses to do, and how to read the audit events.

## Why a pipeline?

`settings.js` is the Node-RED control file. Operators who edit it
by hand outside of NRCC create four classes of problem:

1. The change survives a Node-RED restart but is overwritten by
   the next NRCC edit (no source preservation).
2. Two operators race on the same file and the second write
   silently wins (no transaction coordination).
3. The change breaks Node-RED at the next restart and there is no
   backup to revert to (no atomic write with backup).
4. Nobody can tell who changed what when the editor stops loading
   (no audit trail).

The apply pipeline exists to make all four impossible by
construction.

## The eight steps

Four official stages (`validate`, `backup`, `write`, `audit`),
eight observable steps. Each step emits at most one audit event
and one typed error.

| # | Step | Stage | Audit event | Failure mode |
|---|------|-------|-------------|--------------|
| 1 | Boundary check | validate | `apply.start` | typed `*ApplyError{Stage: "validate"}` |
| 2 | Path validation | validate | (folded into step 1) | typed `*ApplyError{Stage: "validate"}` |
| 3 | Source parse | validate | (folded into step 1) | `apply.failure` with `failure_stage=validate` |
| 4 | Diff (existing vs. proposed) | validate | (folded into step 1) | `apply.failure` with `failure_stage=validate` |
| 5 | Backup current file | backup | `apply.backup` (success only) | `apply.failure` with `failure_stage=backup` |
| 6 | Atomic write | write | `apply.write` (success only) | `apply.failure` with `failure_stage=write` |
| 7 | Fingerprint + diff redaction | write | `apply.success` | none — failure here is a panic, not an `*ApplyError` |
| 8 | Audit close-out | audit | `apply.success` (close-out only) | none |

## Step 1 — Boundary check

Refuses to run when:

- `req.Path == ""` (no path to write to).
- `req.BackupDir == ""` (no place to put the backup).
- `req.Content` is not a UTF-8 string.

Why this matters: every other stage assumes the inputs are sane.
Pushing boundary checks into `validate` keeps the rest of the
pipeline free of nil-pointer guards.

## Step 2 — Path validation

`validateSettingsPath` rejects:

- Paths outside the configured `DATA_DIR`.
- Paths that escape the mount namespace (symlink traversal).
- Paths that do not end in `settings.js`.

The exact rules live in `internal/service/apply.go`. The reason
they are strict: a misconfigured `req.Path` is the single most
common way for an apply transaction to write the wrong file.

## Step 3 — Source parse

The proposed `Content` is parsed as a Node-RED settings module:

- The whole file must be a single CommonJS module
  (`module.exports = { … }`).
- The top-level object must be the literal `{` token
  (a single declaration).
- Top-level keys must come from the
  [catalog](setting-catalog.md) or be classified as
  "operator-managed" through the advanced escape hatch.

A parse error short-circuits the transaction with
`failure_stage=validate`. The operator sees the line/column from
the parser; the audit event records the same cause.

## Step 4 — Diff (existing vs. proposed)

The diff step does three things:

1. **Existing equals proposed** — short-circuits with
   `apply.success` and `failure_stage=ok`. No write, no backup,
   no audit churn.
2. **Revision mismatch** — the operator supplied a `Content`
   whose fingerprint does not match the live `settings.js`
   (someone else edited the file since the operator loaded the
   editor). Returns `*RevisionConflictError` with the diff so the
   operator can decide whether to merge or overwrite.
3. **Managed key missing** — the proposed content dropped a
   catalog-managed key that the live file had. Refused (the
   source-preservation contract forbids it; see
   `InternalService.managedSettingKeys`).

The diff uses `RedactDiff` so secrets in either side never appear
in the audit log.

## Step 5 — Backup current file

The live `settings.js` is copied to
`req.BackupDir/settings-<UTC-timestamp>-<rev>.js`. The backup
includes the live fingerprint in its name so multiple backups
stack without name collisions.

If the backup fails, the transaction stops at
`failure_stage=backup`. The operator sees the I/O error verbatim
(typically disk full or backup directory not mounted).

Why a backup even when atomic write is in place: an atomic write
protects against partial writes, not against semantic breakage.
The backup is the rollback path for the next Node-RED restart
that fails to parse `settings.js`.

## Step 6 — Atomic write

`AtomicWriteSettings` writes the new content to
`req.Path + ".nrcc-tmp"`, fsyncs, then renames over `req.Path`.
The rename is atomic on POSIX filesystems. A crash before the
rename leaves the live `settings.js` untouched.

The write step does NOT touch Node-RED. The restart is a separate
concern — see step 7.

## Step 7 — Fingerprint + diff redaction

After a successful write, the pipeline computes:

- `Revision = FingerprintSource(req.Content)` — the SHA-256 of the
  new content with all secret values redacted. The fingerprint is
  what `*RevisionConflictError` compares against on the next
  apply.
- `Diff = RedactDiff(liveContent, req.Content)` — the line-by-line
  diff with secret cells replaced by `<redacted>`. The diff goes
  to the audit log and to the operator's success response.

Both pieces of information are computed BEFORE `apply.success` is
emitted so the audit record and the operator-visible result are
guaranteed to agree.

## Step 8 — Audit close-out

The final `apply.success` event is the close-out. After it is
emitted:

- `req.Path` holds the new content.
- `req.BackupDir` holds at least one new backup file.
- The audit log has a complete record (`apply.start` → up to three
  intermediate events → `apply.success`).
- The structured UI clears its "unsaved changes" banner.

There is no "undo" step. Recovery uses the backup file from
step 5 — see [`../operator/troubleshooting.md`](../operator/troubleshooting.md)
for the recovery playbook.

## Failure modes and what the operator sees

| Stage | Failure | Operator-visible error | Audit |
|-------|---------|------------------------|-------|
| validate | empty path | `apply path is required` | `apply.failure` `failure_stage=validate` |
| validate | empty backup dir | `apply backup directory is required` | `apply.failure` `failure_stage=validate` |
| validate | invalid path | `validateSettingsPath: …` | `apply.failure` `failure_stage=validate` |
| validate | parse error | `parser: …` | `apply.failure` `failure_stage=validate` |
| validate | revision mismatch | `*RevisionConflictError` | `apply.failure` `failure_stage=validate` |
| backup | I/O error | `os: …` | `apply.failure` `failure_stage=backup` |
| write | I/O error | `AtomicWriteSettings: …` | `apply.failure` `failure_stage=write` |
| audit | panic | (operator sees 500) | (audit incomplete — recovery from log) |

## Restart policy

The apply pipeline never restarts Node-RED. Restart is the
operator's responsibility:

- The structured UI shows a banner listing the catalog entries
  whose `RestartRequired=true` changed in the just-applied
  transaction.
- The operator clicks "Restart Node-RED" (or accepts the deferred
  restart prompt on shutdown).
- The restart uses the same audit hook and the same readiness
  probe; a failed restart emits `restart.failure` with the
  underlying cause.

This decoupling is deliberate: the apply pipeline is local I/O
only, the restart is a process lifecycle event, and conflating the
two would make audit recovery impossible.

## Reading the audit log

The audit log is the source of truth. The shape:

```
apply.start     stage=validate result=ok
apply.backup    stage=backup   result=ok    (only if backup succeeded)
apply.write     stage=write    result=ok    (only if write succeeded)
apply.success   stage=write    result=ok
```

A failed transaction looks like:

```
apply.start     stage=validate result=ok
apply.failure   stage=validate result=error   cause="parser: line 42, …"
```

`apply.failure` is always the LAST event for a failed transaction.
A transaction that emits `apply.failure` after `apply.backup` is
the classic "operator got an error after the backup" race; the
operator can roll back from the backup file.

## Related

- [`setting-catalog.md`](setting-catalog.md) — the catalog entries
  that drive the parse step.
- [`../security/auth-surfaces.md`](../security/auth-surfaces.md) —
  the four surfaces whose changes go through this pipeline.
- [`../operator/playbook.md`](../operator/playbook.md) — restart
  policy in production.
- [`../contributing/parser-renderer-contract.md`](../contributing/parser-renderer-contract.md)
  — what the parser must accept (used in step 3).
- [`../glossary.md`](../glossary.md) — `apply`, `Revision`,
  `BackupDir`, `AtomicWriteSettings`, `RedactDiff`.
