# Spec: Issue #308 — Manage a Node-RED Container from a Native nrcc Host

## Change

`issue-308-native-docker-mgmt`

## Domains

| Domain              | Type | Spec Path                                          |
|---------------------|------|----------------------------------------------------|
| `docker-native-mgmt`| New  | `specs/docker-native-mgmt/spec.md`                 |

## Summary

When `nrcc` is installed on a native host with `--node-red docker`, the
Node-RED container is owned by Docker, not by `nrcc`. Today the Docker
endpoints (`GET /api/docker/status`, `GET /api/docker/info`,
`POST /api/docker/restart`, `POST /api/docker/stop`) all return
`"Docker mode not available"` whenever `/.dockerenv` is missing, so the
`frontend/src/features/docker/` UI is permanently disabled for native
hosts. Issue #308 asks for the smallest production-safe fix: make those
endpoints work from a native host that has the `docker` CLI and a
Node-RED container, and let the existing frontend drive it.

### Capability set (smallest, production-safe)

- **Status (`GET /api/docker/status`)** — when a Node-RED container is
  discoverable on the host, return the full `ContainerInfo` shape the
  frontend already consumes (`id`, `name`, `image`, `status`, `created`,
  `ports`, `state`); otherwise return the existing
  `{ available: false, message }` payload so the UI can keep showing a
  meaningful empty state.
- **Info (`GET /api/docker/info`)** — return a superset of the
  in-Docker payload (`{ available: true, inDocker, source, containerName }`)
  when a Node-RED container is reachable; keep the existing 503-shaped
  payload otherwise.
- **Restart / Stop (`POST /api/docker/restart`, `POST /api/docker/stop`)**
  — accept the action, run `docker restart` / `docker stop` against the
  discovered container, and audit-log the result. Inside-Docker callers
  keep the legacy behaviour (signal shutdown of the parent process so
  Docker's restart policy brings the container back).

### Discovery rules

- Discovery MUST use the `docker` CLI; no Docker socket client is added.
  This matches `host.inspectDockerNodeRed` and keeps the dependency
  surface unchanged.
- A container counts as "the Node-RED container" when its image or name
  contains `node-red` (case-insensitive), mirroring the heuristic in
  `service/host.go::inspectDockerNodeRed`. The first match is used.
- When `nrcc` is itself inside a container (`/.dockerenv`), the
  in-Docker path is preserved and the new native path is bypassed.

### Out of scope (deferred)

- Per-container selection (always the first Node-RED match).
- Docker Compose stack lifecycle (`docker compose up/down`).
- Docker socket TCP/HTTP API access.
- Image pull / update flows.
- Auth model changes — these endpoints inherit the existing
  `middleware.Auth` requirement.

## Requirements Count

| Domain               | Added | Modified | Removed | Scenarios |
|----------------------|-------|----------|---------|-----------|
| docker-native-mgmt   |  4    |  1       |  0      |  9        |
| **Total**            | **4** | **1**    | **0**   | **9**     |
