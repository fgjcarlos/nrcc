# Node-RED Control Center (nrcc)

[![PR Validation](https://github.com/fgjcarlos/nrcc/actions/workflows/pr.yml/badge.svg)](https://github.com/fgjcarlos/nrcc/actions/workflows/pr.yml)
[![Release](https://github.com/fgjcarlos/nrcc/actions/workflows/release.yml/badge.svg)](https://github.com/fgjcarlos/nrcc/actions/workflows/release.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go 1.25+](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](go.mod)

A single-binary management UI for Node-RED. Run it alongside Node-RED to get a web dashboard for configuration, backups, logs, and more.

> **Status: Beta · hardening phase.**
> nrcc is feature-complete for its MVP scope but it is **not** 1.0. There is no SLA, no promised compatibility between minor versions, and no external security audit has been performed. Run it behind your firewall and review the security notes in [`SECURITY.md`](SECURITY.md) before exposing it publicly. Tag releases are published from `main` — see the [Releases page](https://github.com/fgjcarlos/nrcc/releases).

## Features

- **Authentication** — JWT-based auth with multi-user support (admin/user roles)
- **Process Management** — Start, stop, restart Node-RED remotely
- **Live Logs** — near-real-time view of Node-RED output (polled)
- **Configuration Editor** — Edit Node-RED config with validation
- **Backup & Restore** — One-click backups, timestamped archives, restore with integrity checks
- **Environment Management** — Manage Node-RED environment variables
- **npm Library Management** — Install/update/remove npm packages
- **Flow Viewer & Export** — View and export Node-RED flows
- **System Monitoring** — CPU, memory, disk, uptime stats
- **File Upload/Download** — Manage user files and uploads
- **Portless integration** — Install Portless and expose nrcc or Node-RED with named local HTTPS URLs

## Requirements

- Go 1.25+ (to build from source)
- For source/frontend builds: Node.js 22+ and pnpm 11+
- For `nrcc install`: a supported Linux package manager when dependencies must be installed automatically (`apt-get`, `dnf`, `yum`, `pacman`, `zypper`, or `apk`)

## Quick Install

The fastest way to get nrcc running as a system service:

### Prerequisites

- `curl` — for downloading the installer
- `sha256sum` (Linux) or `shasum` (macOS) — for checksum verification (part of GNU coreutils)

### One-Liner Install

```bash
curl -fsSL https://raw.githubusercontent.com/fgjcarlos/nrcc/main/scripts/install.sh | sh
```

Then set up the system service:

```bash
sudo nrcc install
```

In an interactive terminal, `sudo nrcc install` opens a guided wizard that detects the host, lets you choose the Node-RED path, and can configure Portless local HTTPS in-flow. If you select Portless quick setup, open **https://nrcc.localhost**; otherwise open **http://localhost:3001** and set up your admin user.

### Version Pinning

Install a specific version:

```bash
NRCC_VERSION=v0.1.0 curl -fsSL https://raw.githubusercontent.com/fgjcarlos/nrcc/main/scripts/install.sh | sh
```

### Custom Install Directory

Install to a non-root directory (no `sudo` needed):

```bash
INSTALL_DIR=$HOME/.local/bin curl -fsSL https://raw.githubusercontent.com/fgjcarlos/nrcc/main/scripts/install.sh | sh
```

Make sure `$HOME/.local/bin` is in your `$PATH`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

For the manual production rollout steps for GitHub Pages, DNS, and release validation, see [docs/production-install-launch-guide.md](docs/production-install-launch-guide.md).

Architecture decisions for future implementation slices live in [docs/adr/](docs/adr/). Additional architecture notes live in [docs/architecture/](docs/architecture/), including the [multi-instance Node-RED management model](docs/architecture/multi-instance-node-red.md).

### Uninstall

To remove nrcc service and clean up:

```bash
sudo nrcc uninstall          # Remove service and binary
sudo nrcc uninstall --purge  # Also remove config and data files
```

## Quick Start

### Build from source
```bash
git clone https://github.com/fgjcarlos/nrcc.git
cd nrcc
make build        # builds frontend + Go binary (~30s)
./nrcc            # local one-shot run on :3001
```

Visit **http://localhost:3001** → Set up admin user → Login

To install the locally built binary as a managed Ubuntu/systemd service, run:

```bash
sudo ./nrcc install
# or: make install-local
```

`nrcc install` copies the currently running binary to `/usr/local/bin/nrcc`, checks the host, prepares Node-RED according to the selected/detected mode, writes `/etc/nrcc/nrcc.env`, installs the systemd unit, and starts the service. Interactive TTY installs use the guided wizard; non-TTY installs or any explicit install flag keep the existing flag-driven path. For automation, pass flags such as `--node-red skip|native|docker`, `--with-portless`, `--portless-quick-setup`, and `--portless-trust`.

Dependency handling by Node-RED mode:

- `native`: if Node.js or npm is missing, nrcc installs both with the detected Linux package manager before running `npm install -g node-red`.
- `docker`: if Docker is missing, nrcc installs Docker with the detected Linux package manager and verifies that the Docker daemon is reachable before pulling/running Node-RED.
- `skip`: nrcc does not install Node-RED or Node-RED-specific dependencies.
- `--with-portless`: if Portless is requested and Node.js/npm is missing, nrcc prepares Node.js/npm before installing the Portless CLI globally.

If the host has no supported package manager, or Docker installs but its daemon cannot be reached, the installer fails with an explicit prerequisite error instead of continuing with a broken service setup.

### Binary release

Download the binary for your platform from [Releases](https://github.com/fgjcarlos/nrcc/releases).

```bash
chmod +x nrcc-linux-amd64
sudo ./nrcc-linux-amd64 install
```

For a local one-shot run without system service installation:

```bash
./nrcc-linux-amd64
```

## Configuration

All configuration is via environment variables. Create `.env` from `.env.example`:

```bash
cp .env.example .env
```

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 3001 | HTTP server port |
| `DATA_DIR` | `./data` | Directory for config, users, backups |
| `EDGE_MODE` | `false` | Opt-in edge mode for constrained deployments (resource-safe defaults). See [ADR 0002](docs/adr/0002-edge-mode-defaults-and-exposure-ux.md) |
| `JWT_SECRET` | (insecure) | ⚠️ **Must set in production** |
| `NODE_RED_CMD` | `node-red` | Path to node-red executable |
| `NODE_RED_PORT` | 1880 | Node-RED HTTP port (for health checks) |
| `ENCRYPTION_KEY` | (insecure) | For encrypted env vars |
| `CORS_ORIGINS` | `*` | CORS allowed origins |

## Portless

nrcc can install and use [Portless](https://portless.sh) to expose local services with stable names instead of raw ports. Portless provides HTTPS `.localhost` URLs by default, optional LAN `.local` URLs, and Tailscale/Funnel sharing when the Tailscale CLI is installed and connected.

### Quick Start with Portless

Prerequisites: nrcc is installed and `npm` is available on the host.

Install Portless, register the default aliases, start the Portless proxy, and trust local HTTPS:

```bash
nrcc portless install
nrcc portless quick-setup
nrcc portless setup-trust
```

After setup, open:

- `https://nrcc.localhost`
- `https://node-red.localhost`

When installing nrcc as a systemd service, choose the level of Portless automation you want:

```bash
sudo nrcc install
sudo nrcc install --node-red native
sudo nrcc install --node-red docker
sudo nrcc install --node-red skip
sudo nrcc install --with-portless
sudo nrcc install --with-portless --portless-quick-setup
sudo nrcc install --with-portless --portless-quick-setup --portless-trust
```

`nrcc install` now decides how to handle Node-RED before starting the service:

- It runs host detection first, before install side effects, so the installer can adopt an existing Node-RED instead of reinstalling it.
- Existing native Node-RED installs are frozen into `/etc/nrcc/nrcc.env` with `NODE_RED_CMD`, `NODE_RED_USER_DIR`, and `NODE_RED_SETTINGS` so systemd uses the same detected paths.
- If `--node-red skip|native|docker` is provided, that mode is used without prompting.
- If Node-RED is already detected on the host, install continues without prompting.
- If Node-RED is missing and the install runs in an interactive terminal, nrcc prompts for `skip`, `native`, or `docker`.
- If Node-RED is missing and there is no interactive terminal, nrcc falls back to `--node-red skip` and prints a warning.

The install flow is normalized into an `InstallPlan` before mutations start. Pre-systemd steps use a rollback stack and the commit point is the systemd unit write, so an abort before that point cleans up newly-created install files/directories where possible. The guided installer model lives under `internal/wizard/`; its pure model is the TTY-gated foundation for future richer huh/Bubble Tea rendering while preserving the same `InstallPlan` execution path.

`--portless-quick-setup` registers `nrcc -> 3001` and `node-red -> 1880`, then starts the Portless proxy required for `https://*.localhost` to respond. `--portless-trust` is explicit because it may prompt for permission to install the local HTTPS CA.

For advanced LAN, Tailscale, or Funnel usage, follow the official [Portless documentation](https://portless.sh) and run Portless directly. nrcc only prints guidance for public exposure: verify `tailscaled` is running, MagicDNS is enabled, and HTTPS certificates are enabled in the Tailscale admin console before turning on Tailscale Funnel. These prerequisites are outside nrcc's local installer and are not automated by `nrcc install`.

### HTTPS Trust & Certificate Errors

Portless uses a local CA certificate for HTTPS on `*.localhost`. On first use, your browser may show a certificate warning until that local CA is trusted.

Run the nrcc wrapper:

```bash
nrcc portless setup-trust
```

You can also run Portless directly with `portless trust`. Browser or OS certificate stores may require a browser restart before certificate warnings disappear.

### Managing Aliases

View registered aliases:

```bash
nrcc portless status
```

Expose additional services:

```bash
nrcc portless expose --name <name> --port <port>
```

Re-run the default setup and overwrite existing definitions:

```bash
nrcc portless quick-setup --force
```

Uninstall Portless:

```bash
nrcc portless uninstall
nrcc portless uninstall --clean-aliases
```

### Before deploying

This is an operator self-check, not a vendor production-readiness claim. nrcc is in beta hardening — confirm these before you put it on the public internet.

1. Set `JWT_SECRET` to a strong random value
2. Set `ENCRYPTION_KEY` to a strong random value
3. Use `DATA_DIR` on persistent storage (not `/tmp`)
4. Run behind a reverse proxy with HTTPS
5. Set restrictive `CORS_ORIGINS` if needed

## Data Directory Layout

nrcc stores all data in `DATA_DIR/`:

```
data/
├── config.json              # Node-RED configuration
├── cc-users.json            # Control Center user accounts (bcrypt-hashed)
├── flows.json               # Node-RED flows (managed by Node-RED)
├── settings.js              # Generated Node-RED settings
├── backup-settings.json     # Backup scheduler configuration
├── backups/                 # Timestamped backup archives (tar.gz)
│   ├── backup-20260424-100234.tar.gz
│   └── backup-20260424-110456.tar.gz
└── uploads/                 # User-uploaded files
```

**Backup strategy:** Backups are complete snapshots. Store backups in a safe location (S3, NFS, etc.).

## Development

### Start dev servers

```bash
make dev
# Go backend:   http://localhost:3001
# Vite frontend: http://localhost:5173 (HMR enabled)
```

The dev server proxies API requests to the Go backend.

### Local development (Docker)

If you don't want to install Go and Node on the host, the dev stack runs end-to-end in Docker with hot reload:

```bash
make dev-up        # build the dev image + start stack in detached mode
make dev-logs      # stream backend + frontend logs
make dev-shell     # open a shell in the backend container
make dev-down      # stop the stack (keeps volumes)
make dev-reset     # stop the stack AND remove volumes (clean slate)
```

Open in browser:

| URL | What |
| --- | --- |
| <http://localhost:5173> | React UI (Vite HMR) |
| <http://localhost:3001> | Go API + Node-RED control plane |
| <http://localhost:1880> | Node-RED editor |

Source is bind-mounted into the containers, so edits to `*.go` (rebuilt by `air`) and `frontend/**` (HMR via Vite) show up without rebuilding the image. Run `make dev-build` only after changing `Dockerfile.dev` itself.

### Run tests

```bash
make test            # Go test suite
make test-frontend   # Vitest frontend suite
make e2e             # Playwright e2e suite (boots its own preview server)
go run -race .       # Go with race detector
```

Frontend quality gates live under `frontend/`:

```bash
cd frontend
pnpm install --frozen-lockfile
pnpm run typecheck          # TypeScript project check
pnpm test -- --run          # Vitest unit/integration tests; MSW starts from vitest.setup.ts
pnpm run test:e2e           # Playwright Chromium smoke tests with fixture API responses
pnpm run build              # Production Vite build
```

Test layering:

- Unit tests cover pure utilities, hooks, and isolated components.
- Integration tests render feature flows against MSW handlers in `frontend/src/test/msw/`, so auth, setup, status, backups, libraries, files, and runtime endpoints never call a real host.
- E2E smoke tests live in `frontend/e2e/` and exercise setup/login, runtime start/stop, backup create, restore dry path, and critical navigation with Playwright route fixtures instead of destructive host operations.

### Build frontend only

```bash
make frontend-build
```

### Cross-compile for all platforms

```bash
make release
# Output: dist/
#   ├── nrcc-linux-amd64
#   ├── nrcc-linux-arm64
#   ├── nrcc-darwin-amd64
#   ├── nrcc-darwin-arm64
#   └── nrcc-windows-amd64.exe
```

## Project Structure

```
nrcc/
├── main.go                      # Entry point, CLI setup
├── embed.go                     # Frontend embedding directive
├── go.mod / go.sum              # Go dependencies
├── Makefile                     # Build, dev, release targets
├── README.md                    # This file
├── .env.example                 # Example environment config
│
├── frontend/                    # React 19 + Vite + TS SPA (feature-folders layout)
│   ├── src/
│   │   ├── App.tsx              # Root component, routing
│   │   ├── main.tsx             # Entry point
│   │   ├── features/            # One folder per domain
│   │   │   ├── auth/            # login, setup, users, profile
│   │   │   ├── backups/         # list, schedule, retention, restore
│   │   │   ├── bootstrap/       # first-run wizard
│   │   │   ├── configuration/   # Node-RED settings editor
│   │   │   ├── dashboard/       # overview, host/system metrics
│   │   │   ├── docker/          # container status & control
│   │   │   ├── env-vars/        # environment variables editor
│   │   │   ├── flows/           # flow viewer & export
│   │   │   ├── libraries/       # npm package management
│   │   │   ├── logs/            # streaming logs viewer
│   │   │   ├── patterns/        # flow pattern analysis
│   │   │   └── updates/         # self-update checker
│   │   │       Each feature contains: components/, hooks/, services/, types/, lib/
│   │   └── shared/              # Cross-feature primitives
│   │       ├── components/      # ui/, layout/, ConfirmationDialog, StateContainer
│   │       ├── hooks/           # shared React hooks
│   │       ├── lib/             # api client, formatters
│   │       ├── types/           # cross-feature TypeScript types
│   │       └── constants/       # UI copy, defaults
│   ├── package.json
│   ├── pnpm-lock.yaml
│   ├── vite.config.ts
│   ├── vitest.config.ts
│   └── dist/                    # Built SPA (embedded into binary)
│
├── internal/
│   ├── model/                   # Data types (User, Config, etc.)
│   │   ├── user.go
│   │   ├── response.go
│   │   └── ...
│   │
│   ├── store/                   # JSON file storage
│   │   ├── json_store.go        # Generic JSON store with file locks
│   │   └── ...
│   │
│   ├── service/                 # Business logic
│   │   ├── auth.go              # JWT, user management
│   │   ├── process_manager.go   # Node-RED process control
│   │   ├── config.go            # Configuration management
│   │   ├── backup.go            # Backup/restore
│   │   ├── env.go               # Environment variables
│   │   ├── flow.go              # Flow viewer/export
│   │   ├── library.go           # npm library management
│   │   ├── update.go            # Self-update checks
│   │   ├── log_buffer.go        # Ring buffer for logs
│   │   └── ...
│   │
│   ├── handler/                 # HTTP request handlers
│   │   ├── auth.go
│   │   ├── config.go
│   │   ├── backup.go
│   │   └── ...
│   │
│   ├── middleware/              # HTTP middleware
│   │   ├── auth.go              # JWT verification
│   │   ├── logger.go            # Request logging
│   │   └── ...
│   │
│   └── server/                  # Chi router setup, SPA handler
│       └── server.go
│
└── data/                        # Runtime data (gitignored)
    ├── config.json
    ├── cc-users.json
    ├── flows.json
    ├── settings.js
    ├── backups/
    └── uploads/
```

## API Reference

The complete API is documented in [docs/openapi.yaml](docs/openapi.yaml) (OpenAPI 3.1). To browse it interactively, paste the file into [Swagger Editor](https://editor.swagger.io) or run a local viewer:

```bash
npx @redocly/cli preview-docs docs/openapi.yaml
```

All responses use a standard envelope:

```json
{"success": true, "data": {}, "timestamp": "2026-05-24T10:00:00Z"}
{"success": false, "error": {"code": "ERROR_CODE", "message": "..."}, "timestamp": "..."}
```

Authentication: `Authorization: Bearer <jwt-token>` header (short-lived JWT, refreshed via httpOnly cookie).

### Generated TypeScript Client (future)

The OpenAPI spec enables generating a type-safe frontend client via [orval](https://orval.dev) or [openapi-typescript](https://github.com/openapi-ts/openapi-typescript). Adoption path:

1. `pnpm add -D @openapitools/openapi-generator-cli` or `pnpm add -D orval`
2. Generate client from `docs/openapi.yaml`
3. Replace manual Axios calls in `frontend/src/shared/lib/api.ts` incrementally

This is tracked but not yet adopted — the current frontend uses hand-written Axios services per feature.

## Deployment

### Systemd service (Linux)

Create `/etc/systemd/system/nrcc.service`:

```ini
[Unit]
Description=Node-RED Control Center
After=network.target

[Service]
Type=simple
User=nrcc
WorkingDirectory=/home/nrcc/nrcc
Environment="PORT=3001"
Environment="DATA_DIR=/var/lib/nrcc"
Environment="JWT_SECRET=your-secret-here"
ExecStart=/home/nrcc/nrcc-linux-amd64
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable nrcc
sudo systemctl start nrcc
```

### Docker (optional)

```dockerfile
FROM golang:1.25 as builder
WORKDIR /build
COPY . .
RUN make release

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y node-red && rm -rf /var/lib/apt/lists/*
COPY --from=builder /build/dist/nrcc-linux-amd64 /usr/local/bin/nrcc
EXPOSE 3001 1880
CMD ["nrcc"]
```

## Troubleshooting

**Q: Node-RED process won't start**
- Check `NODE_RED_CMD` points to a valid executable
- Verify `node-red` is installed: `which node-red`
- Check logs: `tail -f data/*.log`

**Q: "JWT_SECRET not set" warning**
- Set `JWT_SECRET` env var before running in production
- Default insecure value is for development only

**Q: Can't access frontend**
- Verify port is open: `netstat -tlnp | grep 3001`
- Check logs: `./nrcc 2>&1 | tail -20`
- Clear browser cache and reload

**Q: Backup/restore fails**
- Ensure `DATA_DIR` has write permissions
- Check disk space: `df -h $(DATA_DIR)`
- Review error in UI or logs

## License

Licensed under the [Apache License 2.0](LICENSE).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, the workflow we follow, and how to open an issue or a PR.
