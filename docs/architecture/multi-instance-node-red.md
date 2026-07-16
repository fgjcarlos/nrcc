# NRCC deployment model

> Canonical model for the current release. The earlier "multi-instance" framing for one
> NRCC managing multiple Node-REDs was removed in #390 and is preserved in the git
> history of this file for traceability. New work should follow the model below.

## Status

**Accepted** for the current beta line. Encoded in the shipped `docker-compose.yml` and
the production Docker image.

## One stack per Node-RED

The unit of deployment is a single Docker Compose service containing exactly one
NRCC binary supervising one Node-RED process:

```text
Compose service: nodered
┌────────────────────────────────────────────────────────┐
│  Container: nrcc                                        │
│  ┌──────────────────┐   ┌───────────────────────────┐   │
│  │  nrcc binary      │──▶│  node-red process         │   │
│  │  (HTTP / UI / API)│   │  (editor + flows)         │   │
│  └──────────────────┘   └───────────────────────────┘   │
│                                                        │
│  Ports (host): 3001 → :3001 (NRCC)                     │
│                 1880 → :1880 (Node-RED)                 │
│  Volume:       /data  ← persistent volume              │
│  Workdir:      DATA_DIR = /data                        │
└────────────────────────────────────────────────────────┘
```

Multiple Node-RED instances = multiple Compose services, each with its own host ports
and its own persistent volume. There is no central control plane in MVP.

## Per-instance boundaries

Each stack owns its own state and must never share it with another stack:

| Boundary | Owned per stack |
| ---------- | ----------------- |
| Node-RED `userDir` (flows, `settings.js`, `flows_cred.json`, projects) | yes |
| Node-RED `node_modules` | yes |
| NRCC users, sessions, audit log | yes |
| NRCC config, schedule, encrypted secrets | yes |
| Local backup archive volume | yes (recommended isolated sub-path) |
| Host ports `3001` / `1880` | yes (unique per stack) |
| Logs, restart counters, runtime status | yes |
| Container name and Compose project | yes |

Two writable stacks that share a `DATA_DIR` corrupt each other; this is treated as a
configuration error and must be blocked at setup time.

## MVP non-goals (scope fence)

The following are intentionally out of scope and must not be implemented as part of the
core flow:

- **Mounting `/var/run/docker.sock` in the container.** NRCC must not be able to manage
  arbitrary other containers on the host.
- **Managing sibling containers from inside the NRCC container.** Other Compose services
  in the same project are not part of the MVP control plane.
- **Discovering and managing a Node-RED that already runs natively on the Docker host.**
  Native-host Node-RED control is a separate adapter and is not in MVP.
- **Central multi-instance control.** A single NRCC orchestrating several remote
  Node-REDs is deferred. The original design is kept in git history for traceability.
- **Restic / off-host backup provider.** Local ZIP snapshots on a persistent volume are
  the only shipped backup. Restic is planned as a Phase 2 provider.
- **SSH / agent-managed remote Node-RED.** Same reasoning: deferred.

## Customizing the runtime without rebuilding Node-RED

Node-RED does not need a new NRCC image to gain functionality:

- **Function nodes** — code lives inside `flows.json`, persisted by the volume.
- **npm nodes** — installed via the UI's Library section into the persistent
  `node_modules` volume.
- **`settings.js`** — edited from the UI; the previous file is backed up before each save. A Node-RED restart may be required for runtime changes to take effect.

To change NRCC behavior itself (new endpoints, new policies, new UI), publish a new
NRCC image — there is no plugin system in MVP.

## Backups in MVP

- **Storage:** local ZIP archives inside the persistent volume at
  `DATA_DIR/backups/*.zip`.
- **Schedule:** manual + cron, configured in the UI.
- **Pre-restore:** a snapshot of the current state is taken automatically before any
  restore to keep recovery possible if the restore goes wrong.
- **Retention:** configurable per type (manual, auto, pre-restore).
- **Isolation:** bind the `backups` sub-path on its own volume (or copy externally)
  so a `docker compose down` does not destroy snapshots.
- **Off-host copy** is the operator's responsibility (S3, NFS, etc.); NRCC does not push
  backups to remote destinations in MVP.

## Companion documents

- [`README.md`](../../README.md) — the operator-facing canonical deployment model.
- [`docs/index.html`](../index.html) — the GitHub Pages landing page that mirrors the
  same model.
- [`docs/adr/0001-tenant-ready-seams.md`](../adr/0001-tenant-ready-seams.md) — store,
  auth, and process seams that keep mono-tenant deployments simple.
- [`docs/adr/0002-edge-mode-defaults-and-exposure-ux.md`](../adr/0002-edge-mode-defaults-and-exposure-ux.md) —
  edge-friendly defaults and exposure UX for self-hosted deployments.

## Migrating from the old model

Existing single-instance deployments (native or single Docker stack) keep working
without changes. The shipped `docker-compose.yml` already encodes this model. To move
from a native install to the canonical Docker deployment:

1. Stop the existing nrcc service (systemd, native process, or remove the previous Docker container).
2. `docker compose up -d` from the project's `docker-compose.yml`.
3. Confirm the persistent volume carries the existing `DATA_DIR` content (mount it
   explicitly if you want to reuse it).
4. Open `http://localhost:3001`, log in, validate the migrated settings.

No centralized migration tool is shipped; the volume bind is the contract.
