# Docker Native Management Specification

## Purpose

Defines how `nrcc` exposes container actions for a Docker-managed
Node-RED when `nrcc` itself runs natively on the host (no
`/.dockerenv`). This complements the existing in-container path; it
does not replace it. The intent is to unblock the
`frontend/src/features/docker/` UI for hosts that were installed with
`sudo nrcc install --node-red docker`.

Sections "Existing Behaviour" are written **retroactively** to lock in
the in-Docker code path that already ships in `main`. Sections "New
Behaviour" specify the native-host capability introduced by this
change.

---

## Existing Behaviour (retroactive — code already present)

### Requirement: In-Docker Container Status

When `nrcc` runs inside a Docker container, `GET /api/docker/status`
MUST return a `ContainerInfo`-shaped payload
(`{ status, image, inDocker: true }`) and a 200 status code. The image
MUST be read from the `NRCC_IMAGE` env var and fall back to
`"nrcc"` when unset.

#### Scenario: Status inside a container

- GIVEN `nrcc` is running inside a Docker container
- AND `NRCC_IMAGE` is unset
- WHEN `GET /api/docker/status` is called by an authenticated user
- THEN the response is 200
- AND the body is `{ status: "running", image: "nrcc", inDocker: true }`

#### Scenario: Status with custom image env

- GIVEN `nrcc` is running inside a Docker container
- AND `NRCC_IMAGE=ghcr.io/composedof2/nrcc:1.2.3`
- WHEN `GET /api/docker/status` is called
- THEN the body is `{ status: "running", image: "ghcr.io/composedof2/nrcc:1.2.3", inDocker: true }`

### Requirement: In-Docker Restart/Stop Trigger Parent Shutdown

When `nrcc` runs inside a Docker container, `POST /api/docker/restart`
and `POST /api/docker/stop` MUST audit-log the action, respond
`200 { message }` immediately, and then asynchronously stop the
Node-RED process and signal shutdown so Docker's restart policy can
recreate the container. Non-in-Docker callers MUST receive
`503 DOCKER_NOT_AVAILABLE`.

#### Scenario: Restart outside a container

- GIVEN `nrcc` is running natively on the host
- WHEN `POST /api/docker/restart` is called by an authenticated user
- THEN the response is 503
- AND the error code is `DOCKER_NOT_AVAILABLE`

---

## New Behaviour

### Requirement: Discover a Node-RED Container on a Native Host

When `nrcc` runs natively on a host that has the `docker` CLI and a
Node-RED container (image or name matches `node-red`, case-insensitive),
the system MUST be able to discover that container. Discovery MUST use
`docker ps -a --format` (matching the heuristic in
`service/host.go::inspectDockerNodeRed`); the first matching container
is authoritative.

#### Scenario: Discover a running container

- GIVEN `nrcc` runs natively
- AND `docker ps -a` returns one line with `nodered/node-red:latest` image and name `nrcc-node-red`
- WHEN the discovery routine is invoked
- THEN it returns the container `id`, `name`, `image`, the human-readable
  `Status` line from `docker ps`, and the parsed `state` (`running`,
  `restartCount`) when available

#### Scenario: No Node-RED container present

- GIVEN `nrcc` runs natively
- AND no container has an image or name containing `node-red`
- WHEN the discovery routine is invoked
- THEN it returns a "not found" result
- AND the HTTP layer responds 200 with
  `{ available: false, message: "No Node-RED container found" }`

### Requirement: Native Status Returns ContainerInfo

`GET /api/docker/status` MUST return the full `ContainerInfo` shape
(`id`, `name`, `image`, `status`, `created`, `ports`, `state`) the
frontend already consumes, when a Node-RED container is discoverable on
a native host. The response MUST remain 200.

#### Scenario: Status with a running container

- GIVEN `nrcc` runs natively
- AND a container `nrcc-node-red` (image `nodered/node-red:latest`) is up
- WHEN `GET /api/docker/status` is called
- THEN the response is 200
- AND the body includes `id`, `name: "nrcc-node-red"`,
  `image: "nodered/node-red:latest"`, `status: "running"`, `created`,
  `ports` (with at least the host/container 1880 mapping), and
  `state: { running: true, restartCount: N }`

#### Scenario: Status with an exited container

- GIVEN a Node-RED container that is stopped
- WHEN `GET /api/docker/status` is called
- THEN `status` is `"exited"`
- AND `state.running` is `false`

### Requirement: Native Info Endpoint

`GET /api/docker/info` MUST return
`{ available: true, inDocker: false, source: "docker-cli", containerName }`
when a Node-RED container is discoverable on a native host. When no
container is discoverable, the response MUST keep the existing
`{ available: false, message }` shape so the frontend can render a
sensible empty state.

#### Scenario: Info with a discoverable container

- GIVEN `nrcc` runs natively
- AND a container `nrcc-node-red` exists
- WHEN `GET /api/docker/info` is called
- THEN the response is 200
- AND the body is
  `{ available: true, inDocker: false, source: "docker-cli", containerName: "nrcc-node-red" }`

### Requirement: Native Restart and Stop

`POST /api/docker/restart` and `POST /api/docker/stop` MUST invoke
`docker restart <name>` and `docker stop <name>` respectively against
the discovered Node-RED container when `nrcc` runs natively. The action
MUST be audit-logged; on success the response is `200 { message }`. On
failure (Docker not present, container not found, non-zero exit) the
response MUST be `503 DOCKER_NOT_AVAILABLE` with a human-readable
`message`.

#### Scenario: Restart succeeds

- GIVEN `nrcc` runs natively
- AND `docker restart nrcc-node-red` exits 0
- WHEN `POST /api/docker/restart` is called
- THEN the response is 200
- AND the audit log records `CONTAINER_RESTART` with outcome `ok`

#### Scenario: Stop succeeds

- GIVEN `nrcc` runs natively
- AND `docker stop nrcc-node-red` exits 0
- WHEN `POST /api/docker/stop` is called
- THEN the response is 200
- AND the audit log records `CONTAINER_STOP` with outcome `ok`

#### Scenario: Docker CLI missing

- GIVEN `nrcc` runs natively
- AND the `docker` binary is not in `$PATH`
- WHEN `POST /api/docker/restart` is called
- THEN the response is 503
- AND the error code is `DOCKER_NOT_AVAILABLE`
- AND the audit log records `CONTAINER_RESTART` with outcome `error`

### Requirement: In-Docker Behaviour is Unchanged

The change MUST NOT alter the in-Docker path. When `/.dockerenv` is
present, `nrcc` continues to expose its in-Docker status, info, restart,
and stop endpoints with the same payloads, status codes, and audit
events as before.

#### Scenario: In-Docker path still works

- GIVEN `nrcc` runs inside a Docker container
- WHEN `GET /api/docker/status` is called
- THEN the response is 200 with `{ status: "running", image, inDocker: true }`
