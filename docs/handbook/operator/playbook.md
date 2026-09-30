# Operator playbook

> **Eight recurring scenarios.** Most NRCC operators see the same
> eight scenarios week after week. This playbook walks each one
> from symptom to fix in five steps or fewer, and links to the
> canonical explanation in the rest of the handbook.

## Conventions used in this playbook

Every scenario follows the same shape:

1. **Symptom** — what the operator sees.
2. **Diagnosis** — the one curl / log / config line that
   confirms the cause.
3. **Fix** — the exact change to make.
4. **Verification** — the curl / banner / readiness probe that
   confirms the fix.
5. **Postmortem** — what to do if the fix does not stick.

The postmortem is short on purpose. If the fix does not stick,
open an issue and link the audit event ID.

## Scenario 1: Locked out of the editor after a `adminAuth` edit

**Symptom.** The Node-RED editor at `http://<host>:1880/` returns
`401 Unauthorized` for every operator, including the one whose
credentials are saved in the NRCC UI.

**Diagnosis.**

```
curl -sI http://<host>:1880/ | head -1
# HTTP/1.1 401 Unauthorized
```

```
docker exec -it <nrcc> nrcc cat /data/settings.js | head -20
# confirm adminAuth block hash and username
```

**Fix.** Edit `settings.js` directly (temporarily) to remove
`adminAuth`, then apply the same change through the NRCC UI so
the audit log captures the operator identity:

1. `docker exec -it <nrcc> sed -i '/adminAuth:/,/^    }/d' /data/settings.js`
2. Restart Node-RED: `docker restart <node-red>`.
3. Apply a fresh `adminAuth` value through the NRCC UI.
4. Verify the structured UI banner is green.

**Verification.** `curl -u user:pass http://<host>:1880/` returns
`HTTP/1.1 200 OK`.

**Postmortem.** If the lockout recurs, the bcrypt cost factor in
the hash is incompatible with Node-RED. See
[`../security/auth-surfaces.md § adminAuth`](../security/auth-surfaces.md#adminauth-node-red-editor).

## Scenario 2: Apply pipeline refuses with `failure_stage=validate`

**Symptom.** The structured UI shows "Apply rejected" with a
redacted error from the parser.

**Diagnosis.**

```
docker exec -it <nrcc> tail -n 1 /data/audit.jsonl | jq '.failure_stage,.cause'
# "validate"
# "parser: line 42, …"
```

**Fix.** Open `settings.js` in the editor, find the line the
parser flagged, and decide:

- If the line is a typo in the proposed content, fix the value
  in the UI.
- If the line is an operator-managed block, NRCC cannot validate
  it; use the advanced escape hatch with the original syntax
  and re-apply.
- If the parser flagged a managed key, the key is missing or
  has the wrong shape; consult
  [`../configuration/setting-catalog.md`](../configuration/setting-catalog.md).

**Verification.** Re-apply and watch the audit log:

```
docker exec -it <nrcc> tail -f /data/audit.jsonl | grep apply.
```

**Postmortem.** If the parser repeatedly refuses a valid file,
open an issue with the parser error and the offending file.

## Scenario 3: Node-RED does not come back after a restart

**Symptom.** The NRCC readiness probe shows "Node-RED
unreachable" for more than 60 seconds.

**Diagnosis.**

```
docker logs <node-red> --tail 50 | grep -iE "error|fatal|settings"
```

Common causes:

- `settings.js` is syntactically invalid (Node-RED crashes on
  startup).
- `credentialSecret` was changed without rotating
  `credentials.json`.
- `httpStatic` points at a non-existent path.

**Fix.** Restore the last good backup:

```
docker exec -it <nrcc> nrcc backup list --path /data/settings.js
# pick the most recent backup
docker exec -it <nrcc> nrcc backup restore --backup <id> --path /data/settings.js
docker restart <node-red>
```

**Verification.** `curl -sI http://<host>:1880/` returns
`HTTP/1.1 200 OK`.

**Postmortem.** Once Node-RED is back, open the apply pipeline
audit log to see which apply caused the breakage; the operator
identity is in the audit record. See
[`troubleshooting.md`](troubleshooting.md).

## Scenario 4: Dashboard returns 404 after a `httpStatic` change

**Symptom.** The FlowFuse Dashboard was working yesterday;
today every URL returns 404.

**Diagnosis.** `httpStatic` is either empty or pointing at a
directory that does not exist. The `httpStaticAuth` banner in
the Security Center turns red.

**Fix.** Reset `httpStatic` to its operator-defined value through
the NRCC UI. If the operator never set `httpStatic`, leave it
empty and the catalog default is in effect.

**Verification.**

```
curl -sI http://<host>:1880/<httpStaticPath>/ | head -1
# HTTP/1.1 200 OK or 401 Unauthorized (if httpStaticAuth is set)
```

**Postmortem.** See
[`../security/auth-surfaces.md § Dashboard boundary`](../security/auth-surfaces.md#dashboard-boundary).

## Scenario 5: Audit log fills the disk

**Symptom.** Apply transactions return `apply.failure` with
`failure_stage=audit` and a `disk full` error.

**Diagnosis.**

```
df -h /data
docker exec -it <nrcc> du -sh /data/audit.jsonl
```

**Fix.**

1. Snapshot the audit log: `cp /data/audit.jsonl
   /data/backups/audit-<date>.jsonl`.
2. Truncate the live log: `> /data/audit.jsonl`.
3. Apply a fresh transaction to confirm the pipeline is healthy.

**Verification.** A fresh apply transaction completes and emits
`apply.success`.

**Postmortem.** Add a logrotate entry for `/data/audit.jsonl` so
this does not recur. The audit log is rotated automatically by
the periodic snapshot loop, but only when the snapshot loop is
healthy.

## Scenario 6: JWT cookie rejected after a stack restart

**Symptom.** Every browser refresh logs the operator out, even
within the cookie expiry window.

**Diagnosis.** The `JWT_SECRET` was regenerated. The encrypted
cookie and the secret no longer match.

```
docker exec -it <nrcc> env | grep JWT_SECRET
docker inspect <nrcc> --format '{{ .Config.Env }}' | grep -o JWT_SECRET=[^ ]*
```

If the two outputs differ, the secret was rotated.

**Fix.** Restore the original `JWT_SECRET` (the one the operator
generated on first boot). Re-login once. Going forward, the
operator must commit the secret to the secrets manager before
rotating it.

**Verification.** The operator can sign in and refresh without
being logged out.

**Postmortem.** See
[`troubleshooting.md § secret rotation`](troubleshooting.md#secret-rotation)
for the rotation policy.

## Scenario 7: `apply.success` is missing from the audit log

**Symptom.** The structured UI shows the transaction succeeded,
but `tail /data/audit.jsonl` does not contain a matching
`apply.success` event.

**Diagnosis.** The audit hook panicked (panic recovery writes
the partial state and aborts). Inspect the operator-visible
result: the JSON response from the apply endpoint will show the
revision fingerprint and the backup path even if the audit
close-out failed.

**Fix.**

1. Restart NRCC: `docker restart <nrcc>`. The audit log is
   append-only and the panic recovery guarantees the partial
   transaction is captured.
2. Replay the transaction manually if needed: read the live
   `settings.js`, compare against the backup, restore if the
   apply was incomplete.
3. File an issue with the panic stack trace from
   `docker logs <nrcc>`.

**Verification.** A fresh apply transaction emits
`apply.start` → `apply.backup` → `apply.write` → `apply.success`
in order.

## Scenario 8: Operator cannot find a `settings.js` field

**Symptom.** The structured UI does not expose a field the
operator needs (e.g. a custom `functionGlobalContext` key).

**Diagnosis.** The field is unmanaged (not in the
[setting catalog](../configuration/setting-catalog.md)). NRCC
preserves unmanaged fields through edits but does not expose
them as a typed form field.

**Fix.** Use the advanced escape hatch to edit the raw
`settings.js`. The escape hatch is the only path to add or
remove unmanaged keys; it ships the change through the same
apply pipeline.

**Verification.** The new key survives the next NRCC edit
(round-trip guarantee).

**Postmortem.** If the operator believes the key deserves a
catalog entry, open an issue with the use case. The catalog is
extended conservatively (see
[`../contributing/style.md`](../contributing/style.md)).

## Related

- [`troubleshooting.md`](troubleshooting.md) — symptom-driven
  recovery procedures.
- [`support-matrix.md`](support-matrix.md) — what NRCC
  guarantees in production.
- [`../configuration/apply-pipeline.md`](../configuration/apply-pipeline.md)
  — the audit log shape.
- [`../security/auth-surfaces.md`](../security/auth-surfaces.md)
  — per-surface pitfalls.
