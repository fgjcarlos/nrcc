# Node-RED Control Center (nrcc)

[![PR Validation](https://github.com/fgjcarlos/nrcc/actions/workflows/pr.yml/badge.svg)](https://github.com/fgjcarlos/nrcc/actions/workflows/pr.yml)
[![Release](https://github.com/fgjcarlos/nrcc/actions/workflows/release.yml/badge.svg)](https://github.com/fgjcarlos/nrcc/actions/workflows/release.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go 1.25+](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](go.mod)

A single-binary management UI for one Node-RED instance. The canonical deployment model is **one Docker Compose service containing one NRCC and one Node-RED runtime**, backed by persistent per-instance volumes. Spin up multiple stacks to run multiple Node-RED instances; each stack has its own credentials, settings, backups, and host ports.

> **Status: Beta · hardening phase.**
> nrcc is feature-complete for its MVP scope but it is **not** 1.0. There is no SLA, no promised compatibility between minor versions, and no external security audit has been performed. Run it behind your firewall and review the security notes in [`SECURITY.md`](SECURITY.md) before exposing it publicly. Tag releases are published from `main` — see the [Releases page](https://github.com/fgjcarlos/nrcc/releases).

## Features

- **Authentication** — JWT-based auth with multi-user support (admin/user roles)
- **Process Management** — Start, stop, restart Node-RED inside the stack
- **Live Logs** — near-real-time view of Node-RED output (polled)
- **Configuration Editor** — Edit Node-RED `settings.js` with validation
- **Backup & Restore** — One-click backups, timestamped archives on a persistent volume
- **Environment Management** — Manage Node-RED environment variables from the UI
- **npm Library Management** — Install/update/remove npm packages into the persistent Node-RED userDir
- **Flow Viewer & Export** — View and export Node-RED flows
- **System Monitoring** — CPU, memory, disk, uptime stats
- **File Upload/Download** — Manage user files and uploads

## Requirements

- Docker 24+ and Docker Compose (canonical stack deployment)
- Go 1.25+ (to build from source)
- For source/frontend builds: Node.js 22+ and pnpm 11+

## Quick Start — Docker (canonical)

One stack = one NRCC + one Node-RED + one persistent volume. Distinct
host ports per stack.

### 1. Bring up the stack (local build)

The shipped image is built locally from this repo — the registry image
publish flow is wired in `.github/workflows/release.yml` and will
appear on GHCR/Docker Hub as the release workflow runs.

```bash
git clone https://github.com/fgjcarlos/nrcc.git
cd nrcc
cp .env.example .env
# edit .env — set JWT_SECRET and NRCC_ENCRYPTION_KEY to real values
docker compose up -d --build
```

Open <http://localhost:3001> for the NRCC UI and
<http://localhost:1880> for the Node-RED editor.

The full stack contract (every env var, every restart requirement)
lives in [docs/configuration/env-contract.md](docs/configuration/env-contract.md).
For day-to-day operations (multi-instance, backup, upgrade, uninstall)
see [docs/operations/docker-stack.md](docs/operations/docker-stack.md).

### 2. Multi-instance

A second instance = a second Compose service with different host ports
and its own volumes. Copy `docker-compose.yml` and change the service
name, host ports, and volume names:

```yaml
  nrcc-b:
    build: .
    ports:
      - "127.0.0.1:3002:3001"   # nrcc UI / API (Tailscale-reachable)
      - "127.0.0.1:1881:1880"   # Node-RED editor (Tailscale-reachable)
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

Two stacks on the same host MUST differ on `PORT`, `NODE_RED_PORT`,
`DATA_DIR` (volume), `JWT_SECRET`, and `NRCC_ENCRYPTION_KEY`. The
contract enforces this — see
[docs/configuration/env-contract.md#per-stack-guarantees](docs/configuration/env-contract.md).

## Host port binding

The shipped `docker-compose.yml` binds to `127.0.0.1` so the same
ports are reachable via Tailscale without exposing them to the LAN.
Change the binding to `"0.0.0.0"` (or remove the prefix) if you
need LAN access.

### With Tailscale (recommended for VPS)

```bash
sudo tailscale serve --bg --https=3001 http://localhost:3001
sudo tailscale serve --bg --https=1880 http://localhost:1880
```

URLs become `https://<tailnet-hostname>:3001/` and
`https://<tailnet-hostname>:1880/` with HTTPS terminated by
Tailscale. See `docs/operations/docker-stack.md` for full
persistence and revert steps.

### What persists per instance

Each stack owns its own persistent data:

- Node-RED flows, credentials, `settings.js`, projects, function nodes
- Installed npm nodes (`node_modules`)
- NRCC users, configuration, encrypted secrets
- Local backup snapshots

### Customizing Node-RED inside a stack

You do **not** need a new NRCC image to extend Node-RED functionality:

- Function node code lives inside the persistent flow files.
- npm nodes install into the persistent `node_modules` volume via the UI's Library section.
- `settings.js` edits persist in the same volume.

To change NRCC's own behavior you still rebuild and publish a new NRCC image.

## Configuration

All configuration is via environment variables. Create `.env` from
[`.env.example`](.env.example):

```bash
cp .env.example .env
# Generate real secrets:
echo "JWT_SECRET=$(openssl rand -base64 48)" >> .env
echo "NRCC_ENCRYPTION_KEY=$(openssl rand -base64 48)" >> .env
```

The shipped `.env.example` rejects three well-known placeholder
values (`change-me-in-production` and friends) — the binary refuses
to start until you replace them.

The complete contract — every variable, its default, restart
requirement, and precedence rule — lives in
[docs/configuration/env-contract.md](docs/configuration/env-contract.md).
Highlights:

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `3001` | nrcc HTTP port |
| `DATA_DIR` | `/data` | Where nrcc stores config, users, sessions, backups |
| `JWT_SECRET` | (auto-generated, persisted) | JWT signing secret |
| `NRCC_ENCRYPTION_KEY` | empty (encryption disabled) | Encrypts the persisted env store |
| `NODE_RED_PORT` | `1880` | Node-RED editor port |
| `NODE_RED_USER_DIR` | `<DATA_DIR>` | Where Node-RED keeps flows, `settings.js`, `node_modules` |
| `NRCC_BACKUP_DIR` | `<DATA_DIR>/backups` | Local backup snapshot root |
| `NRCC_RESTIC_REPO` + `_BINARY` + `_PASSWORD` + `_PASSWORD_FILE` + `_CACHE_DIR` | empty | Off-host backups (all five required to enable) |
| `NRCC_CORS_ORIGINS` | `*` | CORS allowed origins |
| `EDGE_MODE` | `false` | Resource-safe defaults (ADR 0002) |

> **Operator self-check.** nrcc is in beta hardening — confirm these
> before you put it on the public internet: set strong `JWT_SECRET`
> and `NRCC_ENCRYPTION_KEY`, persist `DATA_DIR` on durable storage,
> place it behind a reverse proxy with HTTPS, and restrict
> `NRCC_CORS_ORIGINS`.

## Data Directory Layout

Within the persistent volume, NRCC stores all runtime state under `DATA_DIR/`:

```
data/
├── config.json              # NRCC configuration
├── cc-users.json            # NRCC user accounts (bcrypt-hashed)
├── flows.json               # Node-RED flows (managed by Node-RED)
├── settings.js              # Generated Node-RED settings
├── backup-settings.json     # Backup scheduler configuration
├── backups/                 # Timestamped backup archives (.zip)
│   ├── backup-20260424-100234.zip
│   └── backup-20260424-110456.zip
└── node_modules/            # Installed npm nodes (persistent)
```

**Backup strategy.** Backups are complete snapshots stored inside the persistent volume. Mount the backups subdirectory on a separate volume (or copy to an external location such as S3 or NFS) so they survive a stack recreation.

## Future Work (not in MVP)

These are **not** part of the current release. Treat the following as backlog, not as documented behavior:

- **Central multi-instance control plane.** A single NRCC that orchestrates multiple remote Node-RED instances. The design rationale is preserved in [`docs/architecture/multi-instance-node-red.md`](docs/architecture/multi-instance-node-red.md) and implementation is tracked in [#428](https://github.com/fgjcarlos/nrcc/issues/428).
- **Restic as an off-host backup provider.** Encrypted, deduplicated, incremental backups to S3/SFTP/B2. Phase 2, tracked in [#432](https://github.com/fgjcarlos/nrcc/issues/432). The current shipped backup is the local ZIP snapshot stored inside the persistent volume.
- **Sibling-container / Docker socket management.** NRCC in one container does not control other containers, and the image does not mount `/var/run/docker.sock`.
- **Native-host Node-RED control from containerized NRCC.** Cross-host wiring is out of scope.

## Data Directory Cross-references

- The model that current routes assume is defined by [ADR 0001 — Tenant-ready seams](docs/adr/0001-tenant-ready-seams.md).
- Edge-friendly defaults and exposure UX are defined by [ADR 0002](docs/adr/0002-edge-mode-defaults-and-exposure-ux.md).

## Operations

- [docs/operations/docker-stack.md](docs/operations/docker-stack.md) — bring-up, multi-instance, backup, upgrade, uninstall
- [docs/production-install-launch-guide.md](docs/production-install-launch-guide.md) — manual GitHub Pages, DNS, and release validation steps

## Development

### Local development (Docker)

End-to-end dev stack with hot reload, driven by [Taskfile.yml](Taskfile.yml):

```bash
task build     # build the nrcc image
task up        # start the stack (--build on first run)
task down      # stop the stack (volumes preserved)
task logs      # stream container logs
```

Install `task` from <https://taskfile.dev/installation/>.

### Run tests

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

## API Reference

The complete API lives in [docs/openapi.yaml](docs/openapi.yaml) (OpenAPI 3.1). Preview locally:

```bash
npx @redocly/cli preview-docs docs/openapi.yaml
```

All responses use the standard envelope:

```json
{"success": true, "data": {}, "timestamp": "2026-05-24T10:00:00Z"}
{"success": false, "error": {"code": "ERROR_CODE", "message": "..."}, "timestamp": "..."}
```

Authentication: `Authorization: Bearer <jwt-token>` header (short-lived JWT, refreshed via httpOnly cookie).

## Troubleshooting

**Q: Node-RED process won't start**

- Check `NODE_RED_CMD` points to a valid executable inside the container.
- Verify `node-red` is on PATH inside the container image.
- Check logs: the Logs view in the UI, or `docker compose logs`.

**Q: "JWT_SECRET not set" warning**

- Set `JWT_SECRET` env var before running in production.
- Default insecure value is for development only.

**Q: Backups disappear on `docker compose down`**

- Confirm `nrcc_backups` is a named volume rather than a container-local path.
- Re-run the stack; backups return once the volume is mounted.

**Q: Backup/restore fails**

- Ensure `DATA_DIR` has write permissions.
- Check disk space: `df -h $(DATA_DIR)`.
- Review errors in the UI or logs.

## License

Licensed under the [Apache License 2.0](LICENSE).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, the workflow we follow, and how to open an issue or a PR.
