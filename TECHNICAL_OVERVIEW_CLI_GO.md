# NRCC CLI And Operator Overview

This document describes the current operator-facing surface on the `main` branch. It is intentionally practical: it lists what is implemented today, what is usable from the CLI, and what still remains deferred.

## Product Shape

NRCC is a local-first control center for one managed Node-RED runtime.

It currently consists of:

- a Go binary that starts the local HTTP server and the managed Node-RED process
- an embedded React frontend served by that same binary
- SQLite-backed auth, sessions, audit logs, log storage, config snapshots, and job history
- file-backed runtime state under the NRCC data directory for Node-RED assets, backups, manifests, and managed environment files

## Usable CLI Commands

The command router currently accepts these subcommands:

```text
setup
start
doctor
logs
support
version
```

The release-ready operator workflow currently uses:

- `nrcc setup`
- `nrcc start`
- `nrcc doctor`
- `nrcc logs`
- `nrcc support`
- `nrcc version`

`nrcc start` runs in the foreground. Stop it with `Ctrl+C` or by sending `SIGTERM` to that process.

### Implemented Flags

`nrcc doctor`

- `--json`
- `--export`

`nrcc logs`

- `--level <level>`
- `--source <prefix>`
- `--lines <n>`
- `--follow`
- `-f`
- `--json`

`nrcc support`

- `--open`

## Current Operator Workflow

The expected workflow on the current branch is:

1. Run `nrcc setup` to prepare the local data directory and first-run environment.
2. Run `nrcc start` to start NRCC and the managed Node-RED runtime.
3. Open the local NRCC web UI, bootstrap the first administrator if needed, and sign in.
4. Perform day-to-day operations from the authenticated dashboard.

The CLI does not yet expose first-class commands for backup/restore, managed environment editing, npm library management, config editing, or updates. Those flows are implemented through the web UI and backend APIs instead.

## Implemented Web UI Flows

The authenticated dashboard currently includes these areas:

- `Overview`: runtime status, health, system information, restart action
- `Logs`: recent runtime logs
- `Diagnostics`: doctor report, application logs, job history, support bundle export
- `Config`: supported Node-RED settings editing, validation, preview, import, apply, snapshots, and snapshot restore
- `Environment`: `.env.managed` editing and validation
- `Backups`: runtime backup creation and restore
- `Libraries`: npm package install and remove for the managed runtime
- `Updates`: installed/available Node-RED version checks and update apply with preventive backup/rollback

## Backend Surface Implemented Today

The backend on the current branch exposes authenticated routes for:

- runtime status, logs, and restart
- system information and operation-lock status
- config load, validate, preview, apply, import, snapshot create/list/restore
- managed environment load and apply
- runtime backup list/create/restore
- npm library list/install/uninstall
- update status and apply
- diagnostics report, diagnostics logs, diagnostics jobs, and support export

That means the release docs should describe backup/restore, managed environment variables, and npm library install/remove as implemented functionality, not future work.

## Deferred Or Partial Areas

- The CLI is still narrower than the UI. It covers startup and diagnostics well, but not most mutating operator tasks.
- Packaging and release hardening are still separate concerns from the operator workflow documented here.

## Contributor Notes

- The repository keeps a placeholder `frontend/dist/index.html` so `go build` and `go run .` work in a fresh checkout.
- Run `make frontend-build` before manual UI testing so the embedded assets match the current frontend source.
- Use `README.md` as the short release summary. Use this file when you need the current operator and CLI surface in one place.
