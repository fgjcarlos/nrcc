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
```

The repository keeps a placeholder `frontend/dist/index.html` so `go build` and `go run .` work in a fresh checkout. Run `make frontend-build` before manual UI testing so the embedded assets match the current frontend source.

## Notes

- `NRCC_DATA_DIR` overrides the default data directory location.
- `NRCC_PORT` overrides the NRCC web UI port. `NRCC_NODE_RED_PORT` overrides the managed Node-RED port.
- Config snapshots and runtime backups are different flows: snapshots capture supported NRCC config state, while backups capture the managed runtime state used for restore and update rollback.
