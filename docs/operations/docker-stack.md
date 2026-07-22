# Operating the NRCC Docker stack

End-to-end guide for the operator. Assumes you have Docker 24+ and
Docker Compose installed and a clone of this repo.

## Bring up

```bash
git clone https://github.com/fgjcarlos/nrcc.git
cd nrcc
cp .env.example .env

# Generate real secrets — required in production.
echo "JWT_SECRET=$(openssl rand -base64 48)"           >> .env
echo "NRCC_ENCRYPTION_KEY=$(openssl rand -base64 48)"  >> .env

docker compose up -d --build
```

The first `docker compose up` builds the image from the local
`Dockerfile`. Subsequent ups reuse the cached image; pass `--build`
to force a rebuild after a code change.

Once the stack is up:

- nrcc UI: <http://localhost:3001>
- Node-RED editor: <http://localhost:1880>

The first login prompts you to create the admin user. After that,
the JWT cookie keeps you signed in across restarts.

## Verify the contract is honoured

The env contract is enforced by CI:

```bash
go test -count=1 -timeout 60s -run TestEnvContract ./internal/service/
```

If you add a new `os.Getenv` call in the binary, this test fails
until you extend `canonicalBootstrap` in
`internal/service/env_contract_test.go` (and update
[`.env.example`](../../.env.example) +
[docs/configuration/env-contract.md](../configuration/env-contract.md)
in the same commit).

## Multi-instance

Two or more NRCC stacks on the same host MUST differ on:

- `PORT` (the host port mapped to nrcc's container port `3001`)
- `NODE_RED_PORT` (the host port mapped to Node-RED's `1880`)
- `DATA_DIR` (the named volume mounted at `/data` inside the container)
- `JWT_SECRET` (independent auth realm)
- `NRCC_ENCRYPTION_KEY` (independent secret store)

Procedure:

1. Copy `docker-compose.yml` to a second file (e.g. `docker-compose.b.yml`).
2. Rename the service (`nrcc` → `nrcc-b`).
3. Remap host ports: `"127.0.0.1:3001:3001"` → `"127.0.0.1:3002:3001"`,
   `"127.0.0.1:1880:1880"` → `"127.0.0.1:1881:1880"`.
4. Rename volumes: `nrcc_data` → `nrcc_data_b`, etc.
5. Set fresh `JWT_SECRET` and `NRCC_ENCRYPTION_KEY` in the new file's
   `environment:` block.
6. `docker compose -f docker-compose.b.yml up -d --build`.

A preflight check that detects two stacks sharing `DATA_DIR` or
`NRCC_ENCRYPTION_KEY` is the natural follow-up; for now the operator
is responsible. See
[the Per-stack guarantees section](../configuration/env-contract.md#per-stack-guarantees)
for the full contract.

## Host port binding

The shipped `docker-compose.yml` binds to `127.0.0.1` so the same
ports are reachable via Tailscale without exposing them to the LAN.
If you do not use Tailscale and need LAN access, change the host
binding to `"0.0.0.0"` or remove the prefix.

### With Tailscale (recommended for VPS)

Tailscale routes to the host's loopback via `tailscale serve`,
which terminates HTTPS with a Tailscale-issued cert. No need to
expose the host port publicly.

```bash
# One-time per host. Re-run after `tailscaled` restarts.
sudo tailscale serve --bg --https=3001 http://localhost:3001
sudo tailscale serve --bg --https=1880 http://localhost:1880
```

URLs become (replace the hostname with your tailnet one):

- nrcc UI:     `https://ubuntu-1.tail03b606.ts.net:3001/`
- Node-RED:    `https://ubuntu-1.tail03b606.ts.net:1880/`

To persist across reboots, add a `tailscaled`-post-up script or
systemd drop-in that runs the two commands above. See
<https://tailscale.com/kb/1242/tailscale-serve>.

To undo:

```bash
sudo tailscale serve --https=3001 off
sudo tailscale serve --https=1880 off
```

### Without Tailscale

If you do not use Tailscale and want LAN access, change the host
binding to `"0.0.0.0"` (or remove the prefix) on both ports in
`docker-compose.yml`. Put a reverse proxy in front for HTTPS.

## Backup

NRCC ships two backup surfaces:

1. **Local snapshots** (always on) — timestamped ZIP archives in
   `<DATA_DIR>/backups/`. Manage them from the **Backups** UI panel.
   These cover flows, settings.js, env store, and the npm package
   list. They are **not** off-host; if the volume is destroyed the
   snapshots go with it.
2. **Restic (optional)** — encrypted, deduplicated, off-host
   incremental backups. Enable by setting all five `NRCC_RESTIC_*`
   variables in `.env` and restarting. See
   [the contract](../configuration/env-contract.md#bootstrap-variables).

Recommended: mount `<DATA_DIR>/backups` on a separate volume (the
shipped `docker-compose.yml` does this — `nrcc_backups`) so a stack
recreation does not lose history.

## Restore

Restores happen from the **Backups** UI panel. Pick a snapshot, pick
a restore mode (full restore or selective), and the server replaces
the corresponding files in `DATA_DIR`. A restart is required for
`settings.js` or `flows.json` changes to take effect on the
Node-RED process.

Encrypted backups are decrypted server-side using the passphrase
that was set at backup creation time. The passphrase is **not**
recoverable from the NRCC configuration; lose it, lose the backup.

## Upgrade

When a new image is available (a tagged release):

```bash
docker compose pull           # only works after a registry image is published
docker compose up -d --build  # rebuilds from local source otherwise
```

Until a registry image is published (issue #508 — Docker Hub secrets
pending), `docker compose pull` is a no-op. Use
`docker compose up -d --build` to pick up the latest code.

After the upgrade the database schema may have changed. The release
notes call out any required migrations; most upgrades are
non-breaking.

## Health check

```bash
docker compose ps                     # STATUS column shows "(healthy)" after start_period
curl -fsS http://localhost:3001/healthz
```

The image ships with a built-in `HEALTHCHECK` directive (added in
#504) — the container reports `healthy` once `/healthz` returns 200.

## Logs

```bash
docker compose logs -f                # all logs, follow
docker compose logs -f nrcc           # only nrcc process logs
docker compose logs --tail=200 nrcc   # last 200 lines
```

NRCC polls Node-RED output and surfaces it in the in-UI **Logs**
view. For long-running investigation, stream the container logs
directly.

## Uninstall

Preserve volumes (e.g. to inspect data later):

```bash
docker compose down
```

Destroy everything (irreversible — all backups, flows, users, and
settings gone):

```bash
docker compose down -v
```

## Troubleshooting

See the [README](../../README.md#troubleshooting) and
[`docker/entrypoint.sh`](../../docker/entrypoint.sh) for the
self-heal logic that fixes `/data` ownership on container start
(issue #484).

Common pitfalls:

- **`JWT_SECRET is set to a known placeholder`** — the binary
  rejects `change-me-in-production` and two other well-known dev
  strings. Generate a real secret with `openssl rand -base64 48`.
- **Two stacks sharing `DATA_DIR`** — the second one will corrupt
  the first one's database. Use distinct volumes.
- **`docker compose pull` does nothing** — by design, the shipped
  compose file uses `build:` not `image:`. Use
  `docker compose up -d --build` to pick up source changes.
- **Node-RED won't start** — check `docker compose logs` for the
  child process output. Verify `NODE_RED_CMD` (default `node-red`)
  is on the container's `$PATH`.
