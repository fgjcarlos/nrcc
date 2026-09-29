# System architecture

> **Stub.** Landed as a skeleton in W1. Process model, parser,
> renderer, API, and a Mermaid diagram arrive in W5.

## Process model

One NRCC binary supervising one Node-RED process inside one Compose
service. See [`docs/architecture/multi-instance-node-red.md`](../../architecture/multi-instance-node-red.md)
for the deployment model.

## Components

- **HTTP server** — Go HTTP API + static UI.
- **Process manager** — supervise, restart, readiness, logs.
- **Parser** — read `settings.js` losslessly.
- **Renderer** — write `settings.js` losslessly.
- **Apply pipeline** — orchestrate the transaction (preview,
  validate, atomic write, backup, readiness, rollback).
- **Encrypted persisted store** — EnvService, secrets.
- **Backup provider** — in-volume snapshots + optional Restic
  off-host.
- **Frontend** — React + TS + Vite, talks to the HTTP API.

The detailed contracts land in W5.
