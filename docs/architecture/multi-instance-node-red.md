# Multi-instance Node-RED management model

## Status

Proposed architecture for issue #144. **Note (2026-07-14):** the first read-only slice (`Instance`, `InstanceStore`, `GET /api/instances`) shipped on `origin/main` at `5393042` was removed in PR #409 (issue #390). This document remains as the design rationale. Re-introduce the code only with a real consumer.

**Note (2026-07-15):** the canonical deployment unit is now locked by
[ADR 0003 — Docker-first: one Compose stack per Node-RED](../adr/0003-docker-first-one-stack-per-node-red.md),
which closes #428. Multi-instance behavior will compose N copies of that
unit; this document continues to describe the eventual control plane
behind them.

## Context

NRCC currently manages one local Node-RED runtime. The existing implementation is intentionally simple:

- `main.go` wires one `dataDir` into the service graph.
- `internal/service/process.go` owns one `ProcessManager` for one Node-RED process.
- `internal/service/config.go`, backups, env vars, flows, logs, and library management operate against the current data directory.
- Frontend routes assume one active runtime and one set of status/configuration resources.

That model should remain the default. Multi-instance support should be added through explicit instance boundaries rather than by making every service infer global mutable state.

## Goals

- Let one NRCC manage more than one Node-RED instance over time.
- Preserve today's single-instance local deployment as the zero-config path.
- Support local instances first, then leave room for Docker, SSH, and agent-managed remote instances.
- Keep secrets, logs, backups, and lifecycle operations scoped to a selected instance.

## Non-goals

- Do not turn the current release into a SaaS control plane.
- Do not require tenants or organizations to support local multi-instance use.
- Do not auto-discover and control arbitrary Node-RED processes without operator approval.
- Do not share one data directory across writable instances.

## Instance model

A future `Instance` record should describe the control boundary:

```go
type InstanceKind string

const (
    InstanceKindLocal  InstanceKind = "local"
    InstanceKindDocker InstanceKind = "docker"
    InstanceKindSSH    InstanceKind = "ssh"
    InstanceKindAgent  InstanceKind = "agent"
)

type Instance struct {
    ID          string       `json:"id"`
    Name        string       `json:"name"`
    Kind        InstanceKind `json:"kind"`
    DataDir     string       `json:"dataDir,omitempty"`
    BaseURL     string       `json:"baseUrl,omitempty"`
    Health      string       `json:"health"`
    AuthContext string       `json:"authContext,omitempty"`
    CreatedAt   time.Time    `json:"createdAt"`
    UpdatedAt   time.Time    `json:"updatedAt"`
}
```

Recommended constraints:

- `ID` should be stable, URL-safe, and validated.
- `DataDir` must resolve inside an allowed root unless explicitly configured by an admin.
- `BaseURL` should be read-only for remote instances until write operations are designed safely.
- `AuthContext` should reference stored credentials indirectly, never inline secrets.

## Process abstraction

Do not make the existing `ProcessManager` internally multi-instance. Instead, introduce a registry that owns one manager per local writable instance:

```go
type InstanceProcessRegistry struct {
    managers map[string]*ProcessManager
}
```

Each local instance gets its own:

- `ProcessManager`
- data directory
- env service
- log buffer
- restart tracker
- backup/config/flow services scoped to that instance

Remote instances should use a different controller interface rather than pretending to be local processes:

```go
type InstanceController interface {
    Status(ctx context.Context, instanceID string) (ProcessStatus, error)
    Start(ctx context.Context, instanceID string) error
    Stop(ctx context.Context, instanceID string) error
    Restart(ctx context.Context, instanceID string) error
}
```

Local, Docker, SSH, and agent controllers can then implement only the operations that are safe for their transport.

## Data and service boundaries

Current service constructors should remain available for the default instance. Multi-instance work should add resolver/factory layers around them:

- `InstanceStore` — persists the list of configured instances.
- `InstanceResolver` — validates instance IDs and returns metadata.
- `InstanceServices` — builds per-instance service dependencies.
- `InstanceController` — performs runtime lifecycle operations.

For the default path, `default` maps to today's current `DATA_DIR`. This avoids a forced migration for existing installs.

## API routing model

Recommended routing shape:

- Existing routes remain valid and operate on the default instance.
- New multi-instance routes use `/api/instances` and `/api/instances/{id}/...`.
- Mutating operations must verify permissions and instance capabilities.

Examples:

- `GET /api/instances`
- `POST /api/instances`
- `GET /api/instances/{id}/status`
- `POST /api/instances/{id}/process/start`
- `GET /api/instances/{id}/logs`
- `POST /api/instances/{id}/backups`

Compatibility rule: existing routes should not gain a required `instanceId` parameter until the UI and API clients have a migration path.

## UI navigation model

The first UI version should use an instance switcher in the app shell:

- Show the active instance name and health.
- Keep `default` selected for existing users.
- Persist the selected instance in URL/search params or local storage.
- Disable actions that the selected instance cannot perform.
- Show clear labels for local, Docker, SSH, and agent-managed targets.

Recommended path strategy:

- Keep current routes for the default instance.
- Later add `/instances/:instanceId/...` only when deep-linking and permissions are designed.

## Migration path

1. Add instance model, store, and read-only API.
2. Seed a `default` instance that points to the existing `DATA_DIR`.
3. Add a read-only instance switcher that always shows `default` first.
4. Introduce local multi-instance process registry behind feature flags.
5. Add Docker/remote read-only status before any remote lifecycle mutations.

## Risks and mitigations

- **Secrets leakage:** store remote credentials by reference and redact values in logs and UI.
- **Wrong-target operations:** every destructive action should display the target instance and require confirmation where appropriate.
- **Shared data directories:** validate local instance paths and block two writable instances from using the same directory.
- **Partial controller support:** expose capabilities per instance so UI does not offer unsupported actions.
- **Process collisions:** local instances need explicit ports and preflight checks before start.
- **Permission confusion:** role checks should apply per operation and eventually per instance.

## First implementation slice

A safe first PR should be read-only and backwards-compatible:

- Add `Instance` model and validation tests.
- Add `InstanceStore` that seeds the default instance from existing config.
- Add `GET /api/instances` returning the default instance only.
- Add a small UI switcher/status chip that displays the default instance.
- Do not add multi-process start/stop behavior yet.

This proves the model without changing the current single-instance runtime.

## Related issue

Resolves #144.
