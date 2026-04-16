# NRCC

Local-first control center for Node-RED.

## Overview

NRCC combines:

- a Go backend
- an embedded React + Vite frontend
- local runtime management for Node-RED
- SQLite-backed authentication
- supported configuration editing
- runtime status, health, and logs

## Current Status

The project is past the initial scaffold and currently includes:

- project bootstrap and build wiring
- phase 1 CLI environment checks and setup flow
- phase 2 local runtime manager for Node-RED
- phase 3 auth migration toward SQLite-backed state
- authenticated dashboard for runtime visibility and control

The MVP is still in progress. Core runtime and auth foundations are in place, but backup and restore, environment variable management, npm package management, and some CLI behavior are still pending.

## Repository Layout

```text
cmd/         CLI entrypoints
internal/    backend services, models, middleware, platform code
frontend/    React + Vite app
npm/         local npm package metadata
```

## Commands

### Development

```bash
make build
make run
make frontend-build
```

### CLI

```bash
go run . doctor
go run . setup
go run . start
go run . version
```

`go run . start` runs in the foreground. Stop it with `Ctrl+C` or by sending `SIGTERM` to that process.

## Implemented So Far

- local environment checks and setup flow
- local Node-RED process manager
- runtime status, logs, and restart API
- SQLite-backed initial user bootstrap and login
- cookie sessions with CSRF protection
- supported config load, validate, and apply flow
- embedded frontend served by the Go binary

## MVP Gaps

- backup and restore flows are not implemented yet
- environment variable management is not implemented yet
- npm library install and remove flows are not implemented yet
- the full Linux MVP path is not complete yet

## Notes

- `frontend/dist/index.html` is a placeholder so the Go binary can compile before the first frontend build.
- `NRCC_DATA_DIR` can override the default `~/.nrcc/data` location.
- Auth and internal panel state are being moved to SQLite instead of JSON files.
