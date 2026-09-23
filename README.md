# NRCC — Node-RED 5 Control Center

[![PR Validation](https://github.com/fgjcarlos/nrcc/actions/workflows/pr.yml/badge.svg)](https://github.com/fgjcarlos/nrcc/actions/workflows/pr.yml)
[![Release](https://github.com/fgjcarlos/nrcc/actions/workflows/release.yml/badge.svg)](https://github.com/fgjcarlos/nrcc/actions/workflows/release.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go 1.26+](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go)](go.mod)
[![Language policy (advisory)](https://github.com/fgjcarlos/nrcc/actions/workflows/language-policy.yml/badge.svg)](https://github.com/fgjcarlos/nrcc/actions/workflows/language-policy.yml)

A trustworthy visual control plane for **one Node-RED 5.x instance at a time**. NRCC discovers the deployment, imports `settings.js` losslessly, walks you through edits with a diff, validates the result, snapshots a backup, applies the change, and rolls back on readiness failure — without you hand-editing JSON or shelling into the container.

> **Status: Beta · hardening phase.** nrcc is feature-complete for the Node-RED 5 scope but is not 1.0. There is no SLA, no promised compatibility between minor versions, and no external security audit. Run it behind a firewall and read [`SECURITY.md`](SECURITY.md) before exposing it publicly. Tag releases are published from `main`; see the [Releases page](https://github.com/fgjcarlos/nrcc/releases).

## Why NRCC

Configuring a Node-RED instance is risky: `settings.js` controls credentials, HTTPS, security middleware, context stores, and runtime limits. One misplaced comma locks an operator out of their own editor. NRCC exists to remove that risk for the parts of `settings.js` the operator actually touches, and to keep the parts they do touch under versioned, reversible change.

NRCC is for solo operators, small teams, and integrators who run one Node-RED instance per server and want a control plane that explains what it is about to do before it does it.

## What NRCC does

- **Visual configuration of Node-RED `settings.js`** — typed catalog of high-value settings (#762), source-preserving advanced escape hatches (#764), transactional apply (#758) with backups (#431) and atomic rollback.
- **Access administration** — NRCC users and roles (#759), separate surface for Node-RED `adminAuth` (#760), Dashboard authentication (#761).
- **Instance management** — lifecycle, live logs, environment, npm library, flow import/export, system metrics.
- **Backup and restore** — in-volume timestamped snapshots (default) and optional Restic off-host encrypted backups (#432).

NRCC configures. It does **not**:

- Run more than one Node-RED instance per stack.
- Orchestrate Node-RED across multiple hosts.
- Mount `/var/run/docker.sock` or manage sibling containers.
- Duplicate the Node-RED flow editor.

## Compatibility policy

| Node-RED version | NRCC behavior |
| --- | --- |
| `>=5.0, <6.0` | **Full editing** — visual catalog, advanced surface, transactional apply. |
| `>=4.0, <5.0` | Detection + read-only inspection + migration guidance only. |
| Future majors (`>=6.0`) | Read-only until a dedicated adapter and catalog are verified. |

Single-stack: one NRCC container + one Node-RED container + one persistent volume. Multi-instance: spin up additional stacks on distinct host ports with distinct volumes.

## Core workflow

Every configuration change follows the same seven steps. None of them are optional; each is gated by the previous one:

1. **Discover / import** — NRCC reads the live `settings.js` and shows the catalogued values.
2. **Edit visually** — the typed catalog surfaces validated controls; advanced escape hatches (#764) expose anything outside the catalog with a preview.
3. **Review the redacted diff** — secrets are masked before the diff is shown; nothing the operator types in plain text is sent to logs.
4. **Validate** — the pending change is checked against Node-RED 5 shape rules (#756) before any file write.
5. **Backup snapshot** — a timestamped archive is written to the persistent volume (#431).
6. **Apply transactionally** — the change is staged, Node-RED is asked to reload, and the readiness probe is watched.
7. **Readiness check / rollback** — if readiness fails, the previous `settings.js` is restored and NRCC returns the operator to the review screen with the failure surfaced.

## Distinct security surfaces

| Surface | Auth | Mediated by | NRCC controls | NRCC does **not** control |
| --- | --- | --- | --- | --- |
| NRCC UI + API | `JWT_SECRET` (per-stack) | NRCC | Yes | — |
| Node-RED editor + `adminAuth` API | `httpNodeAuth`, `adminAuth` (#760) | Node-RED | Yes (config surfaced) | Runtime credential rotation |
| HTTP/static endpoints (`/foo`) | `httpNodeAuth` | Node-RED | Yes (config surfaced) | Custom middleware outside NRCC's catalog |
| FlowFuse / legacy Dashboard | `httpNodeAuth` (#761) | Node-RED | Yes (visibility and access) | Dashboard content styling |

Each surface is independent. Compromising one does not automatically compromise the others.

## Quick start (Docker)

One stack = one NRCC + one Node-RED + one persistent volume. Distinct host ports per stack.

### 1. Bring up the stack

```bash
git clone https://github.com/fgjcarlos/nrcc.git
cd nrcc
cp .env.example .env
# Generate real secrets:
echo "JWT_SECRET=$(openssl rand -base64 48)" >> .env
echo "NRCC_ENCRYPTION_KEY=$(openssl rand -base64 48)" >> .env
docker compose up -d --build
```

Open <http://localhost:3001> for the NRCC UI and <http://localhost:1880> for the Node-RED editor. The compose binds to `127.0.0.1`; reach those ports over Tailscale Serve (`sudo tailscale serve --bg --https=3001 http://localhost:3001`) or change the binding to `0.0.0.0` for LAN access.

The full env contract (every variable, defaults, restart requirements, per-stack guarantees) lives in [docs/configuration/env-contract.md](docs/configuration/env-contract.md). For day-to-day operations, see [docs/operations/docker-stack.md](docs/operations/docker-stack.md).

### 2. Add a second instance

```yaml
  nrcc-b:
    build: .
    ports:
      - "127.0.0.1:3002:3001"
      - "127.0.0.1:1881:1880"
    environment:
      PORT: "3001"
      DATA_DIR: /data
      JWT_SECRET: "<a different random string>"
      NRCC_ENCRYPTION_KEY: "<a different random string>"
    volumes:
      - nrcc_data_b:/data
      - nrcc_backups_b:/data/backups
      - nrcc_node_modules_b:/data/node_modules
```

Two stacks on the same host MUST differ on `PORT`, `NODE_RED_PORT`, `DATA_DIR` (volume), `JWT_SECRET`, and `NRCC_ENCRYPTION_KEY`. The env contract enforces this.

## Configuration

All configuration is via environment variables. The shipped `.env.example` rejects three well-known placeholders; the binary refuses to start until they are replaced. Highlights:

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `3001` | NRCC HTTP port |
| `DATA_DIR` | `/data` | Where NRCC stores config, users, sessions, backups |
| `JWT_SECRET` | (auto-generated, persisted) | JWT signing secret |
| `NRCC_ENCRYPTION_KEY` | empty (encryption disabled) | Encrypts the persisted env store (required from #664) |
| `NODE_RED_PORT` | `1880` | Node-RED editor port |
| `NODE_RED_USER_DIR` | `<DATA_DIR>` | Where Node-RED keeps flows, `settings.js`, `node_modules` |
| `NRCC_BACKUP_DIR` | `<DATA_DIR>/backups` | Local backup snapshot root |
| `NRCC_RESTIC_REPO` + `_BINARY` + `_PASSWORD` + `_PASSWORD_FILE` + `_CACHE_DIR` | empty | Off-host backups (all five required to enable, #432) |
| `NRCC_CORS_ORIGINS` | `*` | CORS allowed origins |
| `EDGE_MODE` | `false` | Resource-safe defaults (ADR 0002) |

> **Operator self-check.** Before public exposure: set strong `JWT_SECRET` and `NRCC_ENCRYPTION_KEY`, persist `DATA_DIR` on durable storage, place NRCC behind a reverse proxy with HTTPS, and restrict `NRCC_CORS_ORIGINS`.

## Data directory layout

Within the persistent volume, NRCC stores all runtime state under `DATA_DIR/`:

```
data/
├── config.json              # NRCC configuration
├── cc-users.json            # NRCC user accounts (bcrypt-hashed)
├── flows.json               # Node-RED flows (managed by Node-RED)
├── settings.js              # Generated Node-RED settings
├── backup-settings.json     # Backup scheduler configuration
├── backups/                 # Timestamped backup archives (.zip)
└── node_modules/            # Installed npm nodes (persistent)
```

Mount the `backups/` subdirectory on a separate volume (or copy to S3/NFS) so snapshots survive a stack recreation.

## Operations

- [docs/operations/docker-stack.md](docs/operations/docker-stack.md) — bring-up, multi-instance, backup, upgrade, uninstall
- [docs/production-install-launch-guide.md](docs/production-install-launch-guide.md) — manual GitHub Pages, DNS, and release validation steps

## Development

### Prerequisites

- **Linux only** at compile time (see [ADR 0004 — Linux-only build](docs/adr/0004-linux-only-build.md)). On macOS or Windows, use the project's Docker image.
- Docker 24+ and Docker Compose (canonical stack).
- Go 1.26+ (to build from source).
- Node.js 22+ and pnpm 11+ (frontend sources and tests).

### Local development (Docker)

End-to-end dev stack with hot reload, driven by [Taskfile.yml](Taskfile.yml):

```bash
task build     # build the nrcc image
task up        # start the stack (--build on first run)
task down      # stop the stack (volumes preserved)
task logs      # stream container logs
```

Install `task` from <https://taskfile.dev/installation/>.

### Tests

```bash
go test ./...                          # Go suite
cd frontend && pnpm test --run         # Vitest frontend suite
cd frontend && pnpm run test:e2e       # Playwright Chromium smoke
```

Frontend gates:

```bash
cd frontend
pnpm install --frozen-lockfile
pnpm run typecheck          # tsc --noEmit
pnpm test -- --run          # Vitest
pnpm run build              # Production Vite build
```

### Coding agents

If you are an AI coding agent (Claude Code, OpenAI codex, Gemini CLI, Cursor, …), start with [`AGENTS.md`](AGENTS.md). It defines the entry path for AI codegen on this repo, the language policy, and the work-unit commit convention.

## API reference

The complete API lives in [docs/openapi.yaml](docs/openapi.yaml) (OpenAPI 3.1). Preview locally:

```bash
npx @redocly/cli preview-docs docs/openapi.yaml
```

All responses use the standard envelope:

```json
{"success": true, "data": {}, "timestamp": "2026-05-24T10:00:00Z"}
{"success": false, "error": {"code": "ERROR_CODE", "message": "..."}, "timestamp": "..."}
```

Authentication: `Authorization: Bearer <jwt-token>` header (short-lived JWT, refreshed via `httpOnly` cookie).

## Troubleshooting

**Q: Node-RED won't start inside the stack**

- Check `NODE_RED_CMD` points to a valid executable inside the container.
- Confirm `node-red` is on `PATH` in the image.
- See the Logs view in the UI, or run `docker compose logs nrcc`.

**Q: "JWT_SECRET not set" warning**

- Set `JWT_SECRET` before running in production. The default is for development only.
- If the warning persists after setting the variable, rebuild the image so `docker compose` re-reads `.env`.

**Q: Backups disappear on `docker compose down`**

- Confirm `nrcc_backups` is a named volume, not a bind-mount into a path the container owns.
- Re-run the stack; backups return once the volume is mounted.

**Q: Backup or restore fails**

- Ensure `DATA_DIR` is writable inside the container.
- Check disk space: `df -h $(DATA_DIR)`.
- For Restic off-host backups, see [#432](https://github.com/fgjcarlos/nrcc/issues/432) and the related env vars in [docs/configuration/env-contract.md](docs/configuration/env-contract.md).

**Q: Edits to `settings.js` are silently dropped (#707, #715)**

- Verify the field is part of NRCC's catalog or is being set through the advanced escape hatch surface (#764).
- If a setting is non-catalogued, use the advanced surface so the change is previewable and reversible.

**Q: Save returns "password must be a bcrypt hash" (#706)**

- NRCC stores credential hashes only. Re-enter the plain-text password; the binary will hash it before persisting.

## Roadmap

NRCC is repositioned around the Node-RED 5 control plane described in [#765](https://github.com/fgjcarlos/nrcc/issues/765). The roadmap tracker lists the active capability work (settings catalog, advanced surface, transactional apply, recovery) and the planned UX overhaul (#766). A first-class Node-RED 5 handbook is being written in [#771](https://github.com/fgjcarlos/nrcc/issues/771).

The multi-instance control plane (#428) and Restic off-host backups (#432) are tracked but **not** part of the current release — they describe adjacent work, not shipped behavior.

## Read this before opening an issue or PR

Release-relevant facts in this README must stay accurate. Reviewers and the maintainer rotation should confirm each of these in the README diff before merge:

- The compatibility policy table covers the currently supported Node-RED majors.
- The "Distinct security surfaces" section still names each independent auth surface.
- The env-contract highlights table is in sync with [docs/configuration/env-contract.md](docs/configuration/env-contract.md).
- The Quick Start command produces a stack that boots on a clean clone of `main`.
- The Roadmap section names the open tracker issue (#765) and explains what is and is not shipped.
- The screenshots / diagrams referenced in any future contributor docs exist in `docs/` and are not stale.

If any of these drift, the change must update both the README and the linked doc in the same commit.

## License

Licensed under the [Apache License 2.0](LICENSE).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, the language policy, the work-unit commit convention, and how to open an issue or a PR. The change history lives in [CHANGELOG.md](CHANGELOG.md); release announcements are linked from the [Releases page](https://github.com/fgjcarlos/nrcc/releases).
