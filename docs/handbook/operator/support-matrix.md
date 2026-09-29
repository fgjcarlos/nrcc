# Support matrix

> **What NRCC supports today. What it does not.** This page is
> the contract between NRCC and its operators. Every cell is
> either backed by a passing test or by an explicit non-goal.

## Runtime support

| Runtime | Supported | Notes |
|---------|:---------:|-------|
| Node-RED 5.0.x (LTS line) | ✓ | full read/write; [catalog](../configuration/setting-catalog.md) pinned to 5.0.6 |
| Node-RED 5.1.x | ✓ | catalog version bump is operator-driven; same shape |
| Node-RED 4.x | read-only | `nodered_compatibility.go` recognises the 4.x settings shape and refuses to write to it; see [`../compatibility.md`](#node-red-version-support) below |
| Node-RED ≥ 6.0 | read-only | shape drift is the operator's risk; NRCC will not auto-upgrade |
| Node-RED 3.x and below | not supported | not in the compatibility matrix |

## Deployment support

| Deployment | Supported | Notes |
|------------|:---------:|-------|
| Docker (single instance) | ✓ | the canonical deployment |
| Docker Compose stack | ✓ | `docs/operations/docker-stack.md` |
| Kubernetes (single pod) | ✓ | same env contract as Docker |
| Kubernetes (HA, multiple replicas) | ✗ | non-goal: active-active settings write is undefined |
| Bare metal | ✗ | non-goal: NRCC assumes a container filesystem |
| Windows host | ✗ | non-goal: paths assume POSIX semantics |

## Browser support

The structured UI is built with React 19 + Vite and targets
evergreen browsers.

| Browser | Supported | Notes |
|---------|:---------:|-------|
| Chrome / Edge (Chromium ≥ 110) | ✓ | primary target |
| Firefox ≥ 110 | ✓ | primary target |
| Safari ≥ 16 | ✓ | verified manually on each release |
| IE 11 | ✗ | non-goal |
| Chrome < 110 | ✗ | non-goal |

The structured UI does not depend on JavaScript modules the
browser does not ship.

## Authentication

| Method | Supported | Notes |
|--------|:---------:|-------|
| Local username/password (bcrypt) | ✓ | first-boot setup screen |
| TOTP MFA | ✓ | opt-in per user; recovery codes issued at enrolment |
| Recovery codes | ✓ | one-time use |
| LDAP / SAML / OIDC | ✗ | non-goal: deferred to flowfuse.com for hosted users |
| API tokens | ✓ | scoped per operator; documented in the OpenAPI spec |

## Storage

| Storage | Supported | Notes |
|---------|:---------:|-------|
| SQLite (local) | ✓ | the canonical store |
| PostgreSQL | ✗ | non-goal: deferred to flowfuse.com |
| MySQL | ✗ | non-goal |
| In-memory | ✗ | non-goal: ephemeral state is not supported |

## Backups

| Format | Supported | Notes |
|--------|:---------:|-------|
| Local snapshot (`/data/backups/`) | ✓ | the canonical path |
| S3-compatible object store | ✗ | non-goal: deferred to flowfuse.com |
| Encrypted snapshots | ✗ | non-goal: NRCC relies on volume-level encryption |

## Observability

| Surface | Supported | Notes |
|---------|:---------:|-------|
| JSON-lines audit log | ✓ | the canonical record |
| Prometheus metrics endpoint | ✗ | non-goal: deferred to flowfuse.com |
| OpenTelemetry traces | ✗ | non-goal |
| Healthz / readyz endpoints | ✓ | documented in `docs/openapi.yaml` |

## Tested configurations

The CI matrix exercises the following configurations on every PR
and every merge to `main`:

| Layer | Configuration |
|-------|---------------|
| Go | 1.26.8 (golangci-lint, go test, govulncheck) |
| Node.js | 22 (frontend unit tests) |
| pnpm | 10.x (frontend dependencies) |
| Node-RED | 5.0.6 (docker stack acceptance) |
| SQLite | bundled (no external dependency) |

Anything outside this matrix is operator-managed; the support
contract does not extend to it.

## Node-RED version support

NRCC recognises three Node-RED versions:

1. **5.0.x — full support.** The catalog mirrors the 5.0.6
   settings shape; NRCC reads and writes every catalog entry.
2. **4.x — read-only.** `nodered_compatibility.go` parses 4.x
   `settings.js` for the Security Center banners but refuses
   every apply transaction with
   `failure_stage=validate` and
   `cause="incompatible Node-RED version"`.
3. **≥ 6.0 — read-only.** The parser tolerates the new shape
   but the catalog is pinned to 5.0.6; apply transactions are
   refused with a hint to upgrade the catalog.

The full migration policy is documented in
[`../../architecture/multi-instance-node-red.md`](../../architecture/multi-instance-node-red.md).

## Out of scope (explicit non-goals)

The following are intentionally not built:

- **Flow editor.** Operators design flows in the Node-RED
  editor; NRCC has no UI for flow editing.
- **Cluster orchestrator.** NRCC writes one `settings.js`; the
  multi-instance story lives in
  [`../../architecture/multi-instance-node-red.md`](../../architecture/multi-instance-node-red.md).
- **npm library browser.** Operators use the Node-RED palette
  manager; NRCC does not expose a separate npm catalog.
- **Node-RED 4 support.** Operators must upgrade to 5.x before
  adopting NRCC for write operations.
- **Docker socket.** NRCC does not call the Docker API; the
  container lifecycle is managed by the orchestrator (Compose,
  Kubernetes, systemd).
- **Local backups beyond `DATA_DIR`.** Backups live inside
  `DATA_DIR`; off-host backups are the operator's responsibility.

## Related

- [`playbook.md`](playbook.md) — what to do when something
  breaks.
- [`troubleshooting.md`](troubleshooting.md) — symptom-driven
  diagnosis.
- [`../governance/ownership.md`](../governance/ownership.md) —
  who decides what gets supported.
- [`docs/control-plane.md`](../../control-plane.md) — the
  roadmap umbrella.
