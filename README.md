# NRCC

Local-first control center for Node-RED.

## Current Status

This repository currently contains the phase 0 scaffold:

- Go backend skeleton
- embedded frontend wiring
- React + Vite frontend shell
- starter API routes
- phase 1 CLI environment checks and setup flow
- phase 2 local runtime manager for Node-RED
- phase 3 pivot in progress toward SQLite-backed auth

## Commands

```bash
make build
make run
make frontend-build
go run . doctor
go run . setup
```

## Notes

- `frontend/dist/index.html` is a placeholder so the Go binary can compile before the first frontend build.
- `NRCC_DATA_DIR` can override the default `~/.nrcc/data` location.
- Auth and internal panel state are being moved to SQLite instead of JSON files.
