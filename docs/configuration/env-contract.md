# NRCC environment variable contract

Canonical per-instance contract. One Compose stack reads exactly one
`.env` (or one set of Compose `environment:` values); this document
defines which variables belong to that set, which are bootstrap vs.
runtime, and which are secret-managed.

See [ADR 0003 — Docker-first: one Compose stack per Node-RED](../adr/0003-docker-first-one-stack-per-node-red.md)
for why each stack has its own contract instead of sharing one.

## Enforcement

The contract is pinned by `internal/service/env_contract_test.go`:

- `TestEnvContract_BinaryReadsAllCanonicalNames` — every variable in
  the canonical list must be read by `os.Getenv` somewhere in the
  binary. Catches "documented but unused" drift (the old
  `NRCC_BOOTSTRAP_INTERACTIVE` was one such entry).
- `TestEnvContract_NoUndocumentedBootstrapReads` — every `os.Getenv`
  call in non-test code must be in the canonical list. Catches the
  opposite drift: "code reads X but docs don't say so".

Adding a new variable requires extending both the test list and
`.env.example` in the same commit.

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

## Compose forwarding

The canonical `docker-compose.yml` runs one container with one NRCC
binary supervising one Node-RED process. Docker Compose reads the host
`.env` for **interpolation only**: a value reaches the container only
when the service `environment:` block names it (#888). Each contract
variable has exactly one routing decision, pinned by
`internal/service/compose_env_test.go`:

- **Forwarded** as `"${NAME:-}"`: `JWT_SECRET`, `NRCC_CORS_ORIGINS`,
  `NRCC_CORS_UNSAFE_WILDCARD`, `NRCC_TRUSTED_PROXIES`, `NRCC_TLS`,
  `EDGE_MODE`, `NRCC_METRICS_PUBLIC`, `NRCC_BULK_MAX_ENTRIES`,
  `NRCC_BULK_MAX_ENTRY_BYTES`, `NRCC_AI_ENABLED`, `NRCC_AI_PROVIDER`,
  `NRCC_AI_ENDPOINT`, `NRCC_AI_MODEL`, `NRCC_AI_API_KEY`. An unset host
  value reaches the container as the empty string, which every one of
  these readers treats like unset, so the default in the tables below
  applies.
- **Required**: `NRCC_ENCRYPTION_KEY` uses the `:?` form; `docker compose`
  refuses to start without it.
- **Pinned** by the image layout: `PORT=3001`, `DATA_DIR=/data`,
  `NODE_RED_CMD=node-red`, `NODE_RED_USER_DIR=/data`,
  `NODE_RED_SETTINGS=/data/settings.js`. Change the host side with
  `NRCC_HOST_PORT` and `NODE_RED_HOST_PORT` (Compose interpolation only,
  never seen by the binary) instead.
- **Not forwarded**, on purpose. Setting them in `.env` has no effect on
  the canonical stack:

  | Variable | Reason |
  |---|---|
  | `NODE_RED_PORT` | The container port `1880` is fixed by the port mapping; use `NODE_RED_HOST_PORT`. |
  | `NRCC_MANAGE_NODE_RED` | `false` would leave the one-container stack without any Node-RED process to attach to. |
  | `NRCC_BACKUP_DIR`, `NRCC_BACKUP_ROOT` | Pinned to the `nrcc_backups` and `nrcc_data` volumes. |
  | `NRCC_RESTIC_*` | Restic is deferred ([#432](https://github.com/fgjcarlos/nrcc/issues/432)): the image ships no `restic` binary. |
  | `NPM_BIN` | The image ships its own npm. |
  | `NRCC_IMAGE` | Display label only; the stack builds its image locally. |

  Deployments outside the canonical Compose file (bare binary, custom
  Compose overrides) may still set them; the tables below describe the
  binary behavior.

Every bootstrap value is read once at startup. After editing `.env`,
run `docker compose up -d`: Compose recreates the container with the
new environment. `docker compose restart` reuses the old environment and
does not apply the change.

## Bootstrap variables

| Variable | Default | Purpose | Restart? |
|---|---|---|---|
| `PORT` | `3001` | HTTP server port | yes |
| `DATA_DIR` | `./data` | NRCC config + Node-RED userDir root | yes |
| `JWT_SECRET` | generated/persisted | JWT signing secret. Required in prod. | yes |
| `NRCC_ENCRYPTION_KEY` | none | **Required by the canonical Compose stack**, which refuses to start without it (#664). Encrypts the persisted env/secret store. A bare binary started without it boots, logs a warning, and returns 503 for writes that request encryption. | yes |
| `NRCC_CORS_ORIGINS` | empty (deny all cross-origin requests) | Comma-separated allowed origins | yes |
| `NRCC_CORS_UNSAFE_WILDCARD` | `false` | Exactly `true` answers every origin with `*`, even when origins are set (dev only) | yes |
| `NRCC_TRUSTED_PROXIES` | empty | Comma-separated CIDRs/IPs trusted for forwarded headers | yes |
| `NRCC_TLS` | `false` | Mark setup refresh cookies Secure when TLS terminates outside the request path | yes |
| `EDGE_MODE` | `false` | Resource-safe defaults for Pi/NAS (ADR 0002) | yes |
| `NRCC_MANAGE_NODE_RED` | `true` | If `false`, NRCC never starts/stops Node-RED; it only attaches | yes |
| `NRCC_IMAGE` | `nrcc` | Image label reported by `GET /api/docker/status` inside Docker | yes |
| `NRCC_BACKUP_DIR` | `$DATA_DIR/backups` | Backup storage root | yes |
| `NRCC_BACKUP_ROOT` | `$DATA_DIR` | Restore destination containment root for the restic provider restore endpoint | yes |
| `NRCC_BULK_MAX_ENTRIES` | `1000` | Maximum non-comment records accepted by a bulk environment import | yes |
| `NRCC_BULK_MAX_ENTRY_BYTES` | `8192` | Maximum UTF-8 bytes in one parsed bulk key plus value | yes |
| `NRCC_RESTIC_REPO` | empty | **Deferred (#432)** — Restic repository URL. The provider activates when this and one of `_PASSWORD` / `_PASSWORD_FILE` are set; not supported by the canonical Compose stack | yes |
| `NRCC_RESTIC_BINARY` | `restic` (on `PATH`) | Deferred (#432) — path to the restic binary; the image does not ship one | yes |
| `NRCC_RESTIC_PASSWORD` | empty | Deferred (#432) — restic repository password | yes |
| `NRCC_RESTIC_PASSWORD_FILE` | empty | Deferred (#432) — path to a file containing the restic password | yes |
| `NRCC_RESTIC_CACHE_DIR` | `$TMPDIR/nrcc-restic-cache` | Deferred (#432) — local restic cache directory | yes |
| `NPM_BIN` | empty | Override the npm binary path the palette manager invokes | yes |
| `NRCC_AI_ENABLED` | `false` | Enable the in-UI AI assistant | yes |
| `NRCC_AI_PROVIDER` | `offline` | AI provider id (`openai`, `anthropic`, …) | yes |
| `NRCC_AI_ENDPOINT` | OpenAI chat completions URL for `openai`, else empty | Custom AI endpoint URL | yes |
| `NRCC_AI_MODEL` | `gpt-4o-mini` | Model id | yes |
| `NRCC_AI_API_KEY` | none | AI provider API key (prefer secret file) | yes |
| `NRCC_METRICS_PUBLIC` | `false` | When `true`, Prometheus `/metrics` is served unauthenticated. Default `false` gates `/metrics` behind the same JWT as other routes (#671); opt out only on a private metrics network. | yes |

## Node-RED runtime variables

Resolved by `internal/service.resolveNodeRedRuntime` (#429). Defaults
keep today's behavior.

| Variable | Default | Purpose |
|---|---|---|
| `NODE_RED_CMD` | `node-red` | Executable to spawn |
| `NODE_RED_PORT` | `1880` | `--port` and external-attachment probe port |
| `NODE_RED_USER_DIR` | `<DATA_DIR>` | `--userDir` (flows, settings, lib, node_modules) |
| `NODE_RED_SETTINGS` | `<userDir>/settings.js` | `--settings` path |

## Placeholder rejection

`main.go:resolveJWTSecret` (and the equivalent path for
`NRCC_ENCRYPTION_KEY`) refuse to start when the secret is set to one
of three well-known placeholder values:

```
change-me-in-production
cc-secret-change-in-production
dev-secret-not-for-production
```

This catches the common "I copied `.env.example` and forgot to set a
real secret" mistake. Generate a real secret with
`openssl rand -base64 48` or `head -c 48 /dev/urandom | base64`.

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
  via `${VAR_FILE}` indirection (compose `_FILE` convention).
- **File-based inputs** — a future `_FILE` pattern for
  `NRCC_ENCRYPTION_KEY` and `NRCC_AI_API_KEY`.

NRCC does not currently read any `*_FILE` variable for these secrets;
it is a known TODO at the bootstrap layer. The only file-path secret
the binary reads today is `NRCC_RESTIC_PASSWORD_FILE`, which belongs to
the deferred Restic provider. Until then, set `JWT_SECRET`,
`NRCC_ENCRYPTION_KEY` and `NRCC_AI_API_KEY` in the stack's own `.env`
(mode `0600`), which the canonical Compose file forwards.

## Per-stack guarantees

Inside the container every canonical stack uses the same pinned
`PORT`, `NODE_RED_PORT` and `DATA_DIR`. Two stacks on the same host
MUST differ on:

- the Compose project name (`docker compose -p <name>`), which
  namespaces the `nrcc_data`, `nrcc_backups` and `nrcc_node_modules`
  volumes so no state is shared
- `NRCC_HOST_PORT` (host port for the NRCC UI/API)
- `NODE_RED_HOST_PORT` (host port for the Node-RED editor)
- `NRCC_ENCRYPTION_KEY` (independent secret store)
- `JWT_SECRET` (independent auth realm) — when left empty, each stack
  generates its own secret inside its own `/data` volume

The acceptance suite (`scripts/acceptance/docker-stacks.sh`, scenario 2)
resolves both stacks with `docker compose config` and fails when they
share an encryption key or a host port. No runtime preflight detects
this on an operator's host yet; the operator is responsible.

## Validation

- `go test ./internal/service/ -run TestEnvContract` enforces the
  contract against the binary.
- `go test ./internal/service/ -run TestResolveNodeRedRuntime` covers
  the runtime contract end to end.
- Bootstrap values are read once at startup and not currently
  validated centrally; future work should add
  `internal/config/bootstrap.go` with a single typed reader.

## Related

- [.env.example](../../.env.example) — concrete example grouped by layer
- [ADR 0003](../adr/0003-docker-first-one-stack-per-node-red.md) — deployment unit
- [Multi-instance Node-RED](../architecture/multi-instance-node-red.md) — broader model
- #429 — explicit Node-RED runtime (where `resolveNodeRedRuntime` lives)
