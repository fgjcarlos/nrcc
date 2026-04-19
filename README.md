# NRCC

Local-first control center for a single Node-RED runtime.

## Overview

NRCC ships as:

- a Go backend and CLI
- an embedded React + Vite frontend
- a local Node-RED process manager
- SQLite-backed auth, audit, logs, and job history
- operator workflows for config, environment, backups, libraries, updates, and diagnostics

NRCC is intended to run on the operator's machine. It is not a hosted multi-tenant service.

## What The Product Does Today

The current main branch supports this end-to-end workflow:

- prepare the local data directory and runtime prerequisites with `nrcc setup`
- start NRCC and the managed Node-RED runtime with `nrcc start`
- bootstrap the first local administrator, then sign in through the web UI
- inspect runtime status, health, logs, and system information
- restart the managed Node-RED runtime from the UI
- edit supported Node-RED settings, validate changes, preview generated `settings.js`, import existing `settings.js`, and apply changes
- create and restore config snapshots inside the config workflow
- manage `.env.managed` variables from the UI
- create and restore full runtime backups
- install and remove npm libraries in the managed runtime
- check for Node-RED updates and apply them with preventive backup and rollback handling
- run diagnostics, inspect audit/log/job history, and export support bundles

## Current Release Surface

### Web UI

The authenticated dashboard currently includes these operator pages:

- Overview
- Logs
- Diagnostics
- Config
- Environment
- Backups
- Libraries
- Updates

### CLI

These commands are part of the usable workflow on the current main branch:

```bash
go run . setup
go run . start
go run . doctor
go run . logs
go run . support
go run . version
```

`go run . start` runs in the foreground. Stop it with `Ctrl+C` or by sending `SIGTERM` to that process.

Relevant command flags implemented in the code today:

- `doctor --json`
- `doctor --export`
- `logs --level <level>`
- `logs --source <prefix>`
- `logs --lines <n>`
- `logs --follow` or `logs -f`
- `logs --json`
- `support --open`

## Deferred Or Not Release-Ready

These items should not be described as supported MVP workflows yet:

- the standalone CLI is not a full operator replacement for the web UI; backup, restore, environment, library, update, and config editing flows are currently UI/API driven
- Linux-first runtime management is implemented, but broader packaging and release hardening are still incomplete

## Repository Layout

```text
cmd/         CLI entrypoints
internal/    backend services, models, middleware, platform code
frontend/    React + Vite app
npm/         local npm package metadata
```

## Development

```bash
make build
make run
make frontend-build
make release-package
```

The repository keeps a placeholder `frontend/dist/index.html` so `go build` and `go run .` work in a fresh checkout. Run `make frontend-build` before manual UI testing so the embedded assets match the current frontend source.

## Release Packaging

The MVP release artifacts are platform-specific archives containing:

- the `nrcc` Go binary for that target platform
- the frontend already embedded into the binary at build time
- the top-level `README.md`

The packaging flow intentionally builds `frontend/dist` as part of release assembly instead of relying on checked-in frontend assets.

Create the full release artifact set locally with:

```bash
make release-package
```

Artifacts are written to `dist/release/` with this naming scheme:

- `nrcc_<version>_linux_amd64.tar.gz`
- `nrcc_<version>_linux_arm64.tar.gz`
- `nrcc_<version>_darwin_amd64.tar.gz`
- `nrcc_<version>_darwin_arm64.tar.gz`
- `nrcc_<version>_windows_amd64.zip`
- `nrcc_<version>_windows_arm64.zip`
- `checksums.txt`

Override the version label or target list when needed:

```bash
VERSION=v0.1.0 make release-package
TARGETS="linux/amd64 windows/amd64" ./scripts/package-release.sh
```

A GitHub Actions workflow at `.github/workflows/release-package.yml` runs the same script on tag pushes matching `v*` and on manual dispatches.

## npm Wrapper Status

The `npm/` directory is not part of the current release artifact set. The wrapper remains intentionally out of scope until it does more than print a placeholder message, so releases should be built from the Go packaging flow above.

## Notes

- `NRCC_DATA_DIR` overrides the default data directory location.
- `NRCC_PORT` overrides the NRCC web UI port. `NRCC_NODE_RED_PORT` overrides the managed Node-RED port.
- `NRCC_LOCAL_HOSTNAME` and `NRCC_LOCAL_TLD` override the stable local URL published through `portless` when it is available.
- Config snapshots and runtime backups are different flows: snapshots capture supported NRCC config state, while backups capture the managed runtime state used for restore and update rollback.

## Stable Local Access

NRCC can expose a stable local hostname when `portless` is installed.

- default hostname: `https://nrcc.localhost`
- fallback when `portless` is missing or cannot be configured: `http://127.0.0.1:<NRCC_PORT>`
- install `portless` with `npm install -g portless`

At startup NRCC now:

- detects whether `portless` is available
- tries to start the `portless` proxy
- registers a stable alias for the current NRCC web port
- surfaces the resulting status in the overview and diagnostics flows

If the alias cannot be configured because of trust, permissions, or host-environment constraints, NRCC keeps running on the direct localhost URL and reports the fallback clearly.
