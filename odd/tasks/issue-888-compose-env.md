# Issue #888 — Forward the documented operational environment settings through Compose

Issue: <https://github.com/fgjcarlos/nrcc/issues/888> · Branch: `fix/issue-888-compose-env` · Base: `main` (8c20edf).
The canonical `docker-compose.yml` forwards only a subset of the documented env contract; host `.env` values only reach the container when the service `environment:` block interpolates them.

## Specs

- S1 (issue AC1): "Explicitly forward supported settings or document an intentional alternative deployment configuration."
- S2 (issue AC2): "Test defaults, non-default values, restart requirements and effective CORS/edge behavior."
- S3 (issue AC3): "Ensure Restic activation and secret/file paths are usable if advertised as supported; otherwise clearly defer them."
  User decision (L2): defer Restic — do not forward `NRCC_RESTIC_*` in Compose and document it as deferred (#432); no Dockerfile change.
- S4 (issue AC4): "Reconcile encryption requirements and one-container topology across documentation."
- S5 (issue AC5): "Validate two stacks without shared secrets/state or unintended host-port assumptions."
- S6 (issue expected behavior): "Supported documented options reach the container and have the advertised effective behavior."
- S7 (L2): `.env.example` must be reconciled with the forwarded set. The safety policy blocks agent access to that path, so the change is handed to the user as an exact snippet.

## Tasks

- [x] T1 — S1, S2, S6: forward supported settings in `docker-compose.yml`; static + `docker compose config` contract test; extract and test the CORS env parsing. Route: inline. Commit: 03e926d
- [x] T2 — S2, S5, S6: acceptance suite uses a distinct encryption key per stack, asserts no shared key / host port, and adds scenario 8 (live EDGE_MODE + CORS effect after `up -d` recreate). Route: inline. Commit: 15f66f7
- [x] T3 — S3, S4: reconcile env-contract, docker-stack, README, architecture docs (Compose forwarding, Restic deferred, encryption, topology, multi-stack). Route: inline. Commit: (this commit)
- [ ] T4 — S7: hand the `.env.example` snippet to the user. Route: user. Commit: —

## Log

- L1 (user, verbatim): "Resuelve la issue #888"
- L2 (user decisions): Restic → "Diferir Restic (Recomendado)". `.env.example` → "Me das acceso puntual"; the read tool remained blocked by the Gentle AI safety policy after the grant, so the agent does not bypass it (no `git show` workaround).
- L3 (evidence): every documented variable treats an empty string like unset (explorer map), so `"${VAR:-}"` forwarding keeps defaults. `NRCC_MANAGE_NODE_RED=false` would leave the one-container stack without Node-RED; `NRCC_BACKUP_DIR`/`NRCC_BACKUP_ROOT` are pinned by the volume layout; the image ships no restic binary.
- L4 (evidence): RED — `TestComposeEnv_*` failed with 13 forwarded variables missing from the resolved container env; `TestCORSConfigFromEnv` failed to compile (`undefined: corsConfigFromEnv`). GREEN after the compose and main.go change. Slice boundary: T1 alone (466 lines incl. this document); T2 and T3 are separate commits.
- L5 (evidence): first live acceptance run failed scenario 8 because the test read `.edgeMode` while the API wraps payloads in `{success,data}`; fixed to `.data.edgeMode` (test bug, not product). Scenario 2 already passed the new key/port distinctness check.
