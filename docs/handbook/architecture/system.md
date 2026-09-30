# System architecture

> **One diagram, three runtimes.** NRCC is the control plane that
> sits between the operator and Node-RED. It owns `settings.js`,
> the audit log, and the operator's session. It does not own
> Node-RED's flows, the dashboard's state, or the editor's runtime.

## Layers

```
┌──────────────────────────────────────────────────────────────┐
│                Operator (browser / curl / automation)        │
└──────────────────────────────────────────────────────────────┘
                          ▲                ▲
        structured UI     │                │  raw JSON / OpenAPI
                          │                ▼
┌──────────────────────────────────────────────────────────────┐
│                    NRCC control plane (Go)                    │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐  │
│  │  HTTP server   │  │  ApplyService  │  │   Audit hook   │  │
│  │ (handler/)     │  │ (service/)     │  │ (service/)     │  │
│  └────────────────┘  └────────────────┘  └────────────────┘  │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐  │
│  │  UserStore     │  │  Backups       │  │  Settings      │  │
│  │  (JWT + RBAC)  │  │  (snapshots)   │  │  (filesystem)  │  │
│  └────────────────┘  └────────────────┘  └────────────────┘  │
└──────────────────────────────────────────────────────────────┘
                          ▲                ▲
                          │ apply          │ read / write
                          │                │
┌──────────────────────────────────────────────────────────────┐
│                Node-RED runtime (Node.js)                     │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐  │
│  │  settings.js   │  │  flows.json    │  │  credentials   │  │
│  │  (managed)     │  │  (untouched)   │  │  (.json, AES)  │  │
│  └────────────────┘  └────────────────┘  └────────────────┘  │
│  ┌────────────────┐  ┌────────────────┐                      │
│  │  editor        │  │  HTTP-injected │                      │
│  │  (adminAuth)   │  │  endpoints     │                      │
│  └────────────────┘  └────────────────┘                      │
└──────────────────────────────────────────────────────────────┘
```

Three runtimes:

1. **Browser** — the structured UI in
   `frontend/src/features/`.
2. **NRCC** — the Go binary in `cmd/nrcc` and `internal/`.
3. **Node-RED** — the long-lived Node.js process.

The rest of this page walks each runtime top-down.

## Runtime 1 — Browser

The structured UI is a React 19 SPA built with Vite and Vitest.
It ships four feature areas that touch the control plane:

- `features/configuration/` — the typed form for catalog entries.
- `features/security/` — the Security Center with the four
  authentication banners.
- `features/backups/` — the snapshot history and restore UI.
- `features/dashboard/` — the FlowFuse Dashboard tunnel view
  (read-only mirror of editor state).

Every feature talks to NRCC through the JSON API documented in
`docs/openapi.yaml`. The OpenAPI spec is the contract; the
frontend consumes a generated client (`frontend/src/api/`).

The browser does not own any state. Refreshing the page is safe;
the audit log captures every state transition.

## Runtime 2 — NRCC

The Go binary boots in this order:

1. **Config load.** `cmd/nrcc/main.go` reads
   `DATA_DIR` and the bootstrap environment variables.
2. **Migrations.** `internal/storage/migrate.go` runs any pending
   SQLite migrations. NRCC refuses to boot if a migration fails.
3. **HTTP server.** `internal/handler/server.go` mounts the JSON
   API, the static UI bundle (`frontend/dist`), and the
   `/healthz` and `/readyz` endpoints.
4. **Apply service.** `internal/service/apply.go` initializes the
   settings filesystem driver and the audit hook.
5. **Backups.** `internal/service/backups.go` starts the periodic
   snapshot loop on a goroutine.
6. **User store.** `internal/user/store.go` decrypts the user
   database using the local encryption key.

After boot, NRCC responds to:

- Operator requests (HTTP, structured UI).
- Node-RED readiness probes (so the operator sees when Node-RED
  is back up after a restart).

NRCC does NOT poll Node-RED, does not push configuration changes,
and does not consume Node-RED's event bus. Communication is one
way: NRCC writes `settings.js`, Node-RED reads it on restart.

### Long-lived goroutines

- `internal/service/backups.go` — periodic snapshot loop.
- `internal/handler/server.go` — the HTTP server itself.

Every other component is request-scoped and goroutine-local.

### Audit hook

The audit hook (`internal/service/audit.go`) is the only writer
to the audit log. It is called by:

- `ApplyService` (apply transactions).
- `UserStore` (login, logout, MFA events).
- `Backups` (snapshot creation, snapshot deletion).
- `ApplyCoordinator` (concurrent apply coordination events).

The audit log is append-only and exported as JSON lines.

## Runtime 3 — Node-RED

Node-RED boots normally; NRCC is invisible to it. Node-RED reads
`settings.js` once on startup. While running, Node-RED serves:

- The editor at `http://<host>:1880/` (the `adminAuth` surface).
- HTTP-injected endpoints at `http://<host>:1880/<httpNodeRoot>/`
  (the `httpNodeAuth` surface).
- Static file serving at `http://<host>:1880/<httpStatic>/`
  (the `httpStaticAuth` surface).
- The FlowFuse Dashboard (when installed) at the same port with
  a different Socket.IO namespace (the Dashboard surface).

NRCC has no read access to Node-RED's runtime state (`flows.json`,
`credentials.json`, etc.). The control plane is a write-only
authority over `settings.js` plus a read-only mirror for the
Security Center.

## Boundaries

### What NRCC owns

- `settings.js` (via the apply pipeline).
- The audit log (`audit.jsonl`).
- The encrypted user store.
- The snapshots directory.
- The structured UI bundle.

### What NRCC does not own

- `flows.json`.
- `credentials.json`.
- `node_modules/`.
- The FlowFuse Dashboard package (an add-on under
  `~/.node-red/node_modules/`).
- Node-RED's runtime logs.

Crossing these boundaries is a defect. If a feature requires
NRCC to write to `flows.json`, the right answer is to open a
discussion, not to add a side-channel write.

## Failure domains

| Failure | Detected by | Operator sees |
|---------|-------------|---------------|
| NRCC crashes | NRCC `healthz` goes red, container restarts | UI 502 |
| Node-RED crashes | NRCC `readyz` flips red | banner in Security Center |
| `settings.js` corrupts | apply pipeline `apply.failure` | revert from backup |
| Audit log full | apply pipeline `audit.failure` | offline dump and rotate |
| Backup directory full | apply pipeline `backup.failure` | operator must free space |

## Related

- [`docs/control-plane.md`](../../control-plane.md) — umbrella
  traceability for #765.
- [`../configuration/setting-catalog.md`](../configuration/setting-catalog.md)
  — what NRCC can write to `settings.js`.
- [`../configuration/apply-pipeline.md`](../configuration/apply-pipeline.md)
  — the write path itself.
- [`../security/auth-surfaces.md`](../security/auth-surfaces.md)
  — the four boundaries NRCC brokers between the operator and
  Node-RED.
- [`../contributing/parser-renderer-contract.md`](../contributing/parser-renderer-contract.md)
  — the source-preservation contract.
- [`docs/architecture/multi-instance-node-red.md`](../../architecture/multi-instance-node-red.md)
  — how multiple Node-RED instances share one NRCC.
