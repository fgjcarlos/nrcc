# NRCC environment variable contract

Canonical per-instance contract. One Compose stack reads exactly one
`.env` (or one set of Compose `environment:` values); this document
defines which variables belong to that set, which are bootstrap vs.
runtime, and which are secret-managed.

See [ADR 0003 — Docker-first: one Compose stack per Node-RED](../adr/0003-docker-first-one-stack-per-node-red.md)
for why each stack has its own contract instead of sharing one.

## Layers

Two layers, two restart requirements:

- **Bootstrap** (`NRCC_*`, plus the framework-shaped `PORT`, `DATA_DIR`,
  `JWT_SECRET`, `EDGE_MODE`): read once during `main.go` /
  `runServer()` / `internal/server/server.go` startup. Changing them
  requires restarting the container.
- **Node-RED runtime** (`NODE_RED_*`): resolved by
  `ProcessManager.startLocked` on every `Start()`. Mutable per-start;
  most users keep these pinned in Compose.

Mutable Node-RED environment variables (e.g. credentials consumed by
flow nodes) live in NRCC's encrypted persisted store, NOT in `.env`.
`ProcessManager.envForChild` injects them with the precedence
OS env → EnvService → `.env`.

## Bootstrap variables

| Variable | Default | Purpose | Restart? |
|---|---|---|---|
| `PORT` | `3001` | HTTP server port | yes |
| `DATA_DIR` | `./data` | NRCC config + Node-RED userDir root | yes |
| `JWT_SECRET` | generated/persisted | JWT signing secret. Required in prod. | yes |
| `NRCC_ENCRYPTION_KEY` | none (encryption disabled) | Encrypts the persisted env/secret store | yes |
| `NRCC_CORS_ORIGINS` | `*` | Comma-separated allowed origins | yes |
| `NRCC_CORS_UNSAFE_WILDCARD` | `false` | Allow `*` even when origins are set (dev only) | yes |
| `NRCC_TRUSTED_PROXIES` | empty | Comma-separated CIDRs/IPs trusted for forwarded headers | yes |
| `EDGE_MODE` | `false` | Resource-safe defaults for Pi/NAS (ADR 0002) | yes |
| `NRCC_MANAGE_NODE_RED` | `true` | If `false`, NRCC never starts/stops Node-RED; it only attaches | yes |
| `NRCC_BOOTSTRAP_INTERACTIVE` | auto-detect TTY | Force the setup wizard on/off | yes |
| `NRCC_IMAGE` | release image | Image tag the install/update flow tracks | yes |
| `NRCC_AI_ENABLED` | `false` | Enable the in-UI AI assistant | yes |
| `NRCC_AI_PROVIDER` | none | AI provider id (`openai`, `anthropic`, …) | yes |
| `NRCC_AI_ENDPOINT` | provider default | Custom AI endpoint URL | yes |
| `NRCC_AI_MODEL` | provider default | Model id | yes |
| `NRCC_AI_API_KEY` | none | AI provider API key (prefer secret file) | yes |

## Node-RED runtime variables

Resolved by `internal/service.resolveNodeRedRuntime` (#429). Defaults
keep today's behavior.

| Variable | Default | Purpose |
|---|---|---|
| `NODE_RED_CMD` | `node-red` | Executable to spawn |
| `NODE_RED_PORT` | `1880` | `--port` and external-attachment probe port |
| `NODE_RED_USER_DIR` | `<DATA_DIR>` | `--userDir` (flows, settings, lib, node_modules) |
| `NODE_RED_SETTINGS` | `<userDir>/settings.js` | `--settings` path |

## Compatibility aliases

These legacy names are still honored by the bootstrap read paths but
emit a one-time deprecation warning at startup. Migrate before the
next minor release.

| Legacy | Canonical | Notes |
|---|---|---|
| `ENCRYPTION_KEY` | `NRCC_ENCRYPTION_KEY` | Bootstrap secret |
| `CORS_ORIGINS` | `NRCC_CORS_ORIGINS` | Bootstrap CORS |

## Precedence (Node-RED env injected into the child)

When NRCC starts Node-RED, the child process sees variables in this
order, last write wins:

1. OS environ (base layer)
2. `EnvService` — encrypted persisted store (`<DATA_DIR>/cc-config.json`)
3. `<DATA_DIR>/.env` file (highest priority)

Bootstrap reads only layer 1 (OS env / Compose `environment:` /
`.env`). The persisted store is not consulted for bootstrap values.

## Secrets

Production should source secrets from:

- **Docker Secrets** — mount under `/run/secrets/<name>` and reference
  via `${NRCC_JWT_SECRET_FILE}` indirection (compose `_FILE` convention).
- **File-based inputs** — same `_FILE` pattern for `NRCC_ENCRYPTION_KEY_FILE`,
  `NRCC_AI_API_KEY_FILE`, etc.

NRCC does not currently auto-apply `_FILE` expansion; declare it as a
known TODO at the bootstrap layer. Until then, compose `secrets:` +
`environment:` substitution is the recommended path.

## Per-stack guarantees

Two Compose stacks running on the same host MUST differ on at least:

- `PORT` (NRCC UI/API)
- `NODE_RED_PORT` (Node-RED editor)
- `DATA_DIR` (mounted volume)
- `NRCC_JWT_SECRET` (independent auth realm)
- `NRCC_ENCRYPTION_KEY` (independent secret store)

A preflight check that detects two stacks sharing `DATA_DIR` or
`NRCC_ENCRYPTION_KEY` is the natural follow-up; for now the operator
is responsible.

## Validation

`go test ./internal/service/ -run TestResolveNodeRedRuntime` covers
the runtime contract end to end. Bootstrap values are read once at
startup and not currently validated centrally; future work should
add `internal/config/bootstrap.go` with a single typed reader.

## Related

- [.env.example](../../.env.example) — concrete example grouped by layer
- [ADR 0003](../adr/0003-docker-first-one-stack-per-node-red.md) — deployment unit
- [Multi-instance Node-RED](../architecture/multi-instance-node-red.md) — broader model
- #429 — explicit Node-RED runtime (where `resolveNodeRedRuntime` lives)