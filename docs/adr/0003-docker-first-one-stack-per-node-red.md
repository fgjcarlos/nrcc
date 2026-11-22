# ADR 0003 — Docker-first: one Compose stack per Node-RED

## Status

Accepted. Resolves #428.

## Context

NRCC exposes three deployment paths: an all-in-one Docker image, the
static binary on a host, and Docker-container discovery (`DOCKER_*`
envs). The runtime code, however, assumes a single process owning a
single `DATA_DIR` and a single Node-RED lifecycle (`internal/service/process_linux.go`,
`main.go`). That ambiguity is what #144 / `docs/architecture/multi-instance-node-red.md`
called out — there is no explicit instance boundary yet.

We are not building a multi-instance control plane now. But we should
make the canonical deployment unit unambiguous: a Compose stack is one
NRCC binary + one Node-RED process + one persistent volume. Future
multi-instance work will then compose N copies of this unit instead of
re-discovering the boundary.

## Decision

Adopt the Docker-first, one-stack-per-Node-RED model as the canonical
deployment contract:

- One Compose service owns exactly one `nrcc` binary and exactly one
  Node-RED process.
- Each stack binds unique host ports (defaults `3001` for nrcc, `1880`
  for Node-RED) and mounts an isolated `DATA_DIR` volume.
- The `nrcc` process inside the container manages Node-RED as a child
  process only — it does not reach sibling containers, the host, or a
  remote Node-RED.
- The static binary on a host remains supported as an alternative for
  single-host, single-instance deployments (see
  `docs/production-install-launch-guide.md`). It is not the canonical
  multi-instance model. There is no `nrcc install` command.
- NRCC never attaches to a Node-RED it did not start. When it manages
  Node-RED (`NRCC_MANAGE_NODE_RED=true`, the default) and another process
  already binds the Node-RED port, start, restart and the automatic
  restart fail with `ErrForeignListener`. To run beside a Node-RED that another supervisor
  owns, set `NRCC_MANAGE_NODE_RED=false` (#918, #1018).
- The Compose file at `docker-compose.yml` is the reference shape; the
  image at `ghcr.io/fgjcarlos/nrcc` is the reference binary.

Out of scope for the MVP contract:

- A central NRCC orchestrating remote Node-RED instances (requires
  credentials, capability discovery, authorization).
- Separate NRCC and Node-RED containers per stack (deferred until a
  controller interface exists).
- Sibling-container control, Docker socket access, and native-host
  control from inside a container.

## Consequences

- Adding a second Node-RED on the same host means a second Compose
  stack with remapped host ports and a new volume. Not a config flag.
- Backups, env vars, flows, logs, and library installs are scoped to
  the stack that owns the volume. Cross-stack operations are not part
  of the MVP.
- The existing all-in-one image and `docker-compose.yml` already match
  this contract. No breaking changes are required for current users.

## Follow-up issues

This ADR is the parent contract. Implementation lives in the focused
slices below; each one tightens one boundary without expanding scope:

- #429 — make Node-RED ports and paths explicit in the runtime.
- #430 — define the canonical per-instance environment variable contract.
- #431 — harden per-instance backup snapshots and safe restore.
- #432 — optional Restic off-host provider (still one stack = one repo).
- #433 — persist custom Node-RED nodes across image upgrades.
- #434 — isolated multi-stack Compose acceptance coverage.

## Related

- `docs/architecture/multi-instance-node-red.md` — the broader model
  this ADR selects the canonical unit from.
- ADR 0001 — tenant-ready seams (orthogonal; this ADR is about
  deployment unit shape, not tenancy).