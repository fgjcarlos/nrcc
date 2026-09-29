# Troubleshooting

> **Symptom → diagnosis → fix → verification.** This page is the
> diagnosis layer between the [playbook](playbook.md) (which
> solves recurring scenarios) and the [`../glossary.md`](../glossary.md)
> (which defines terms). Use it when the symptom does not match
> one of the playbook's eight scenarios.

## How to use this page

1. Find the symptom in the index.
2. Run the diagnosis command.
3. Apply the fix.
4. Run the verification command.
5. If the fix does not stick, follow the postmortem pointer.

The index is sorted by the layer of the stack the symptom lives
in:

- **Browser** — the operator sees the symptom in the UI.
- **NRCC** — the operator sees it in the audit log or the JSON
  API.
- **Node-RED** — the operator sees it on the Node-RED side
  (`/healthz`, editor, dashboard).

## Browser-layer symptoms

### The structured UI shows "Cannot reach NRCC"

**Diagnosis.**

```
curl -sI http://<nrcc-host>:8080/healthz
```

If the request times out, NRCC is unreachable from the browser's
network. If it returns `200 OK`, the issue is between the browser
and NRCC (often a reverse proxy).

**Fix.**

- NRCC unreachable: `docker ps | grep nrcc` and `docker logs
  <nrcc> --tail 50`.
- Reverse proxy: check the proxy logs for the request and the
  NRCC response.

**Verification.** The browser banner turns green within the next
poll (typically 5 seconds).

### The structured UI shows "Audit log full"

**Diagnosis.** Same as
[playbook scenario 5](playbook.md#scenario-5--audit-log-fills-the-disk).

### A typed form field is missing

**Diagnosis.** The field is unmanaged. See
[`playbook § scenario 8`](playbook.md#scenario-8--operator-cannot-find-a-settingsjs-field).

## NRCC-layer symptoms

### The audit log shows `apply.failure` with `failure_stage=validate`

**Diagnosis.**

```
docker exec -it <nrcc> tail -n 1 /data/audit.jsonl | jq
```

Look at `.cause`. The pipeline rejected the proposed content.
See
[`../configuration/apply-pipeline.md § failure modes`](../configuration/apply-pipeline.md#failure-modes-and-what-the-operator-sees).

**Fix.** Per
[`playbook § scenario 2`](playbook.md#scenario-2--apply-pipeline-refuses-with-failure_stagevalidate).

### The audit log shows `apply.failure` with `failure_stage=backup`

**Diagnosis.** Disk full or backup directory not mounted.

```
df -h /data
mount | grep /data
```

**Fix.** Free disk or mount the backup directory. Re-apply.

### The audit log shows `apply.failure` with `failure_stage=write`

**Diagnosis.** `AtomicWriteSettings` failed. Almost always a
disk-full or permission error.

**Fix.** Same as the backup failure above. Re-apply.

### The audit log shows `apply.failure` with `failure_stage=audit`

**Diagnosis.** Audit hook failed (panic or disk full on the
audit log itself).

**Fix.** Per
[`playbook § scenario 5`](playbook.md#scenario-5--audit-log-fills-the-disk)
and
[`playbook § scenario 7`](playbook.md#scenario-7--applysuccess-is-missing-from-the-audit-log).

### NRCC returns `503 Service Unavailable` for every endpoint

**Diagnosis.**

```
docker logs <nrcc> --tail 100 | grep -iE "panic|fatal|migration"
```

**Fix.**

- Migration failure: `docker exec -it <nrcc> nrcc migrate
  status` and consult the error.
- Panic: capture the stack trace and open an issue.

## Node-RED-layer symptoms

### The editor returns `401 Unauthorized` for everyone

**Diagnosis.** `adminAuth` changed; the bcrypt hash is wrong;
or `adminAuth` was removed and the default open-editor state
takes over (which would be `200 OK`, not `401`).

**Fix.** Per
[`playbook § scenario 1`](playbook.md#scenario-1--locked-out-of-the-editor-after-a-adminauth-edit).

### The editor returns `404 Not Found`

**Diagnosis.** `httpAdminRoot` is set to `false`, or
`disableEditor` is `true`, or the wrong port is configured.

```
curl -sI http://<host>:1880/ | head -1
```

**Fix.** Reset the affected field through the NRCC UI. Restart
Node-RED.

### Node-RED logs `Error: Cannot find module 'X'`

**Diagnosis.** An `npm install` was run outside of NRCC; the
resulting `package.json` was overwritten by the next NRCC edit
because the unmanaged `nodesDir` was preserved but the
managed `functionGlobalContext` was not.

**Fix.** Re-install the module inside the NRCC-controlled
directory. Operators should not run `npm install` against
`~/.node-red` directly; that path is managed by NRCC.

### Node-RED logs `Error: EACCES: permission denied`

**Diagnosis.** The container is running as a non-root user but
the mounted volume is owned by root.

**Fix.** Chown the mounted volume to the container's UID:GID.
The recommended UID is `1000` (the Node-RED default).

### Node-RED readiness probe fails after a restart

**Diagnosis.**

```
docker logs <node-red> --tail 50 | grep -iE "ready|listen|error"
```

Common causes:

- The port is wrong (`uiPort` mismatch).
- The `httpNodeRoot` collides with a static path.
- `credentialSecret` was changed without rotating
  `credentials.json`.

**Fix.** Per
[`playbook § scenario 3`](playbook.md#scenario-3--node-red-does-not-come-back-after-a-restart).

## Cross-layer symptoms

### Two operators see different revisions for the same file

**Diagnosis.** Operator A and Operator B loaded the editor at
different times; the underlying `settings.js` changed between
their loads.

```
docker exec -it <nrcc> sha256sum /data/settings.js
```

**Fix.** The structured UI forces a re-fetch when the revision
in memory does not match the revision on disk. The operator who
edited last is correct; the other operator must refresh.

### A `flows.json` change disappeared after a Node-RED restart

**Diagnosis.** NRCC does not touch `flows.json`. The change
disappeared because Node-RED rejected the file at startup
(`flows.json` could not parse).

**Fix.** This is a Node-RED issue, not an NRCC one. Open the
Node-RED log, find the parse error, fix the file in the editor.

### Secret rotation

NRCC uses three secrets:

- `JWT_SECRET` — signs the operator's cookie.
- `DATA_ENCRYPTION_KEY` — encrypts the user store.
- `credentialSecret` (Node-RED) — encrypts Node-RED
  `credentials.json`.

Each has a different rotation policy:

- `JWT_SECRET`: rotating it logs every operator out once. Plan
  for the brief outage. Re-login is automatic on next request.
- `DATA_ENCRYPTION_KEY`: rotating it locks out every operator and
  the NRCC refuses to boot. Re-encrypt the user store first;
  see the migration guide in
  [`../governance/ownership.md`](../governance/ownership.md).
- `credentialSecret`: rotating it invalidates every stored
  Node-RED credential. Re-issue credentials before rotating.

## Related

- [`playbook.md`](playbook.md) — the eight recurring scenarios.
- [`support-matrix.md`](support-matrix.md) — what NRCC
  guarantees.
- [`../configuration/apply-pipeline.md`](../configuration/apply-pipeline.md)
  — the audit log shape.
- [`../security/auth-surfaces.md`](../security/auth-surfaces.md)
  — per-surface failure attribution.
