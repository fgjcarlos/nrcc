# Apply pipeline (source-preserving configuration)

> **Five official stages. Nine observable steps.** The apply
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

## The nine steps

The persistence pipeline has five stages (`validate`, `backup`,
`write`, `reconcile`, `audit`) and nine observable steps. The
production runtime-coordinated path adds `restart` and `ready` before
it reports success. Each step emits the applicable audit event and
returns a typed error.

The table describes the persistence-only write stages; the
production runtime coordinator adds restart, readiness, and recovery
before its final success event.

| # | Step | Stage | Audit event | Failure mode |
|---|------|-------|-------------|--------------|
| 1 | Boundary check | validate | `apply.start` | typed `*ApplyError{Stage: "validate"}` |
| 2 | Path validation | validate | (folded into step 1) | typed `*ApplyError{Stage: "validate"}` |
| 3 | Source parse | validate | (folded into step 1) | `apply.failure` with `failure_stage=validate` |
| 4 | Diff (existing vs. proposed) | validate | (folded into step 1) | `apply.failure` with `failure_stage=validate` |
| 5 | Backup current file | backup | `apply.backup` (success only) | `apply.failure` with `failure_stage=backup` |
| 6 | Atomic write | write | `apply.write` (success only) | `apply.failure` with `failure_stage=write` |
| 7 | Reconcile structured mirror | reconcile | `apply.reconcile` (success only) | `apply.failure` with `failure_stage=reconcile` |
| 8 | Fingerprint + diff redaction | write | `apply.success` | none — failure here is a panic, not an `*ApplyError` |
| 9 | Audit close-out | audit | `apply.success` (close-out only) | none |

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

The atomic writer does not itself signal Node-RED. In production,
the runtime coordinator immediately restarts the pinned runtime
authority and verifies its HTTP readiness before success. The
persistence-only `ApplyService` API remains available to legacy
callers and does not restart a runtime.

The persistence-only API computes its fingerprint and closes out
its audit after the durable write. In the production
runtime-coordinated path, that is not the final success boundary:
`apply.success` is emitted only after the selected runtime has
restarted and passed readiness. See [Runtime-coordinated apply](#runtime-coordinated-apply).

## Step 7 — Reconcile the structured mirror

`settings.js` is the authoritative source for the values Node-RED
reads; `config.json` is NRCC's structured mirror of the values its
model represents. A raw apply therefore has to re-derive the mirror
from the bytes it just wrote, otherwise a structured read keeps
reporting the pre-apply values and the next visual save writes those
stale values back into the source.

The reconciliation is a single atomic commit of the mirror:

- every managed value the source declares wins;
- every managed value the source does not declare keeps its mirrored
  value — the source speaks for the values it declares, it does not
  reset the ones it is silent about;
- the NRCC-owned state the mirror holds (the environment metadata,
  the function libraries, the runtime state) is never taken from the
  source;
- the operator-owned blocks the structured model does not represent
  stay in `settings.js`; the reconciliation never rewrites the source.

A structured apply has nothing to reconcile: it committed the mirror
before the write.

If the reconciliation fails, the transaction is rejected as a whole.
The pre-write source bytes are restored, so neither representation
keeps a partial effect. The failure is reported with
`failure_stage=reconcile`.

## Step 8 — Fingerprint + diff redaction

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

## Step 9 — Audit close-out

On the persistence-only API, `apply.success` closes the write. On
the production runtime-coordinated path, the close-out happens only
after restart and readiness. The structured UI clears its unsaved
changes banner only after that successful result. A failed runtime
transition is reported as a failed apply even when automatic
recovery succeeds.

## Revision precondition on the structured surfaces

Every structured save surface accepts an optional `expectedRevision`:
the `settings.js` fingerprint the operator read before composing the
edit. The revision travels in the request body, next to the
configuration payload.

| Surface | Endpoint | Revision source |
|---------|----------|-----------------|
| Visual save | `POST /api/config` | `revision.fingerprint` of the settings document the form loaded |
| Structured apply | `POST /api/config/apply` | `revision.fingerprint` of the settings document the form loaded |
| Raw editor | `POST /api/settings/raw` | `revision.fingerprint` of the settings document the editor loaded |
| Security center | `POST /api/config/apply` | same as the structured apply |

When the supplied revision does not match the live `settings.js`, the
save is rejected with HTTP 409 `SETTINGS_REVISION_CONFLICT`. The
response carries the live and the provided fingerprint so the operator
UI can render one conflict dialog for every surface.

A rejected save has no durable effect: the revision precondition is
checked under the apply coordinator's single-flight guard, before the
structured configuration or any file is written. The same holds for the
other rejected stages — a validation failure, a backup failure and a
write failure and a reconciliation failure all leave `config.json`
and `settings.js` as they were, because the structured commit is
restored when a later stage fails and the pre-write source bytes are
restored when the reconciliation fails.

The precondition is optional. A client that never learned a revision
(no field in the body) keeps the historical behaviour: the server
compares the live revision against itself and the save proceeds. The
operator UI always sends the revision it read, so two browser sessions
editing the same deployment cannot silently overwrite each other.

## Runtime-coordinated apply

The production coordinator resolves a supported runtime adapter
before it mutates configuration or files. Unsupported or ambiguous
authority is rejected during validation, before mutation. Once
accepted, the coordinator retains the same runtime authority through
candidate restart, readiness, and any recovery; it does not switch
to a different process midway through the transaction.

The ordered transaction is:

1. Validate the requested settings and revision; stage structured
   fields when requested.
2. Back up the live `settings.js`, then persist the candidate
   atomically.
3. Restart the selected runtime and probe its admin HTTP endpoint.
   Only a ready runtime receives `apply.success`.
4. If restart or readiness fails, restore the settings snapshot and
   structured fields first, then restart and probe the same runtime.
   The original apply remains a failure even when recovery succeeds.

Recovery uses an independent bounded context, so cancellation of
the caller cannot interrupt restoration. Candidate readiness uses
the request's context and the readiness probe's bounded retry
window; recovery defaults to 30 seconds unless configured by the
request. These are separate budgets. If recovery itself fails, the
failure is recorded and surfaced for operator intervention; inspect
the restored files and runtime before attempting another apply.

The referenced TLS key/certificate paths are not part of the apply
write-set. NRCC stores their paths and Node-RED reads the files at
startup; recovery restores `settings.js` and structured configuration
but does not mutate referenced TLS files. See
[`../operator/troubleshooting.md`](../operator/troubleshooting.md)
for diagnosis and manual recovery.

## Partial patches on `POST /api/config/apply`

The structured apply endpoint takes a **patch**: the payload carries
only the fields the operator changed. This is what the security
boundary cards send — the admin-auth card owns `adminAuth`, the
basic-auth cards own `httpNodeAuth` and `httpStaticAuth` — and it is
what keeps an edit to one surface from dropping the rest of the
configuration.

The merge runs inside the same transaction as the write: the apply
coordinator holds the single-flight slot and has already checked the
revision precondition, and the merge itself runs under the
configuration store's update lock. The patch is therefore merged onto
the state the store holds at commit time, never onto a stale copy
read before the transaction started.

| Payload | Meaning |
|---------|---------|
| field absent | keep the live value |
| field present, JSON object | merge key by key; nested arrays are replaced |
| field present, `null` | disable the surface the field configures |
| field present, value | replace the live value |

Examples:

- `{"adminAuth": {"users": [...]}}` replaces the user list and keeps
  the auth type and the session expiry.
- `{"httpNodeAuth": null}` disables the HTTP-node basic auth and
  leaves the admin surface and the static surface alone.
- `{"adminAuth": {"users": [{"username": "operator", "password": "", ...}]}}`
  keeps the stored password hash: an empty password means "the
  operator did not retype it", never "clear it".
- A write-only secret echoed back as `[redacted]` (or as an empty
  string) keeps the stored value, exactly like a full configuration
  save does.

`POST /api/config` keeps its full-replacement contract: the
configuration editor posts the whole configuration, and the editor
must be able to clear a surface by leaving it out.

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
| reconcile | mirror commit error | `reconcile: …` | `apply.failure` `failure_stage=reconcile` |
| audit | panic | (operator sees 500) | (audit incomplete — recovery from log) |

## Restart policy

Production configuration apply is runtime-coordinated: after the
validated, backed-up, atomic persistence, the coordinator restarts
the selected supported authority and requires its HTTP readiness
before reporting success. The accepted authority remains pinned
through recovery. Unsupported or ambiguous runtime authority is
rejected before configuration or files are mutated.

The persistence-only `ApplyService` API is retained for legacy
callers and does not restart Node-RED. Do not infer production UI
apply behavior from that persistence-only contract. Restart-required
catalog metadata continues to describe which settings need runtime
application; the runtime coordinator applies the change as part of
the guarded transaction rather than relying on a later unscoped
restart.

## Reading the audit log

The audit log is the source of truth. Persistence-only audit
records use the shape below; runtime-coordinated apply also emits
restart/readiness outcomes and rollback events where applicable:

```
apply.start     stage=validate  result=ok
apply.backup    stage=backup    result=ok    (only if backup succeeded)
apply.write     stage=write     result=ok    (only if write succeeded)
apply.reconcile stage=reconcile result=ok    (raw apply only)
apply.success   stage=write     result=ok
```

A validation failure looks like:

```
apply.start     stage=validate result=ok
apply.failure   stage=validate result=error   cause="parser: line 42, …"
```

Runtime recovery uses the actual `apply.rollback.started`,
`apply.rollback.succeeded`, and `apply.rollback.failed` events.
Their metadata includes `apply_rollback_stage`, adapter identity,
operation, and relevant backup/target basenames. A recovered
candidate failure therefore has rollback events followed by the
original `apply.failure`; it does not produce `apply.success`. The
failure event is not necessarily the last event when recovery itself
emits audit metadata.

## Related

- [`setting-catalog.md`](setting-catalog.md) — the catalog entries
  that drive the parse step.
- [`../security/auth-surfaces.md`](../security/auth-surfaces.md) —
  the four surfaces whose changes go through this pipeline.
- [`../operator/playbook.md`](../operator/playbook.md) — production
  runtime recovery scenario and opt-in real acceptance test.
- [`../contributing/parser-renderer-contract.md`](../contributing/parser-renderer-contract.md)
  — what the parser must accept (used in step 3).
- [`../glossary.md`](../glossary.md) — `apply`, `Revision`,
  `BackupDir`, `AtomicWriteSettings`, `RedactDiff`.
