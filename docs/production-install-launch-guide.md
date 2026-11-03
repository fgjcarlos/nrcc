# Production install and launch (Linux, Docker-first)

nrcc ships in two supported production forms, both built from the same source
and carrying the same version:

| Form | Use it when | Version is visible in |
| --- | --- | --- |
| Container image (recommended) | You run the canonical stack | `org.opencontainers.image.version` label, startup banner |
| Static Linux binary | You already own the host stack | `go version -m <binary>`, startup banner |

There is **no `nrcc install` command and no `nrcc --version` flag**. The `nrcc`
binary is a server-only process (see [ADR 0003 — Docker-first](../adr/0003-docker-first-one-stack-per-node-red.md)),
and the project is Linux-only at compile time (see [ADR 0004 — Linux-only build](../adr/0004-linux-only-build.md)).
This guide replaces the older bootstrap-installer checklist, which described a
CLI installer and Darwin/Windows binaries that the project does not ship.

## Quick path — container image (recommended)

```bash
git clone https://github.com/fgjcarlos/nrcc.git
cd nrcc
printf 'NRCC_ENCRYPTION_KEY=%s\n' "$(openssl rand -hex 32)" > .env
docker compose up -d --build
```

Then open:

- nrcc UI and API — <http://localhost:3001>
- Node-RED editor — <http://localhost:1880>
- Health probe — <http://localhost:3001/healthz> (returns `200`)

Ports are bound to `127.0.0.1` by default. Change `NRCC_HOST_PORT` /
`NODE_RED_HOST_PORT` (or the port bindings in `docker-compose.yml`) to expose
them elsewhere.

`NRCC_ENCRYPTION_KEY` is mandatory — Compose refuses to start without it, so
encrypted environment variables are never stored in plaintext. `JWT_SECRET` is
optional; when it is absent the container generates and persists one under
`/data`.

### Bootstrap installer

The canonical POSIX shell installer clones the selected revision, prepares
`NRCC_ENCRYPTION_KEY` before Compose starts, and waits for the stack to become
healthy (or for `/healthz` to respond when container health is unavailable).
It requires Git, Docker, and a reachable Docker daemon.

Supported command forms:

```sh
# Latest published release
curl -fsSL https://get.nrcc.dev/install.sh | sh

# Explicit version as a positional argument
curl -fsSL https://get.nrcc.dev/install.sh | sh -s -- v1.2.3

# Explicit version through the installer process environment
NRCC_VERSION=v1.2.3 sh -c "$(curl -fsSL https://get.nrcc.dev/install.sh)"
```

`NRCC_VERSION=… curl … | sh` is **not** supported: the assignment binds to
`curl`, not to the installer. The custom domain is published from GitHub
Pages; when it is not live yet, fetch the same script from the repository:

```sh
curl -fsSL https://raw.githubusercontent.com/fgjcarlos/nrcc/main/docs/install.sh | sh
```

Version selection is positional argument, then `NRCC_VERSION`, then the latest
published GitHub release for `NRCC_REPO` (default `fgjcarlos/nrcc`). If that
release cannot be resolved, the installer visibly warns and falls back to
`main`. `NRCC_REPO` and `INSTALL_DIR` can be overridden.

An exported `NRCC_ENCRYPTION_KEY` is used as-is. Otherwise an existing
`$INSTALL_DIR/.env` key and all other lines are preserved; if no key exists,
the installer creates a 32-byte random key (64 hexadecimal characters) in
`.env` with mode `0600`. An existing but empty key is replaced, because
Compose refuses to start without one. After `docker compose up -d --build`,
it waits up to `NRCC_READY_TIMEOUT` seconds (default 300). Compose startup
failures and readiness timeouts print `docker compose ps` and the last 50 log
lines before exiting non-zero.

Running the installer again over an existing install replaces the checkout
with the selected revision and keeps the operator's `.env`. A directory that
is not an nrcc install is refused, so an unrelated `INSTALL_DIR` is never
overwritten.

## Quick path — static Linux binary

```bash
# 1. download and check the artifact
curl -fsSLO https://github.com/fgjcarlos/nrcc/releases/download/v1.2.3/nrcc-linux-amd64
curl -fsSLO https://github.com/fgjcarlos/nrcc/releases/download/v1.2.3/nrcc-linux-amd64.sha256
# the sidecar records the digest under the normalized file name "nrcc",
# so compare the digest directly (or check it from a directory that
# contains a file named "nrcc")
cat nrcc-linux-amd64.sha256
sha256sum nrcc-linux-amd64   # must print the same digest

# 2. install it into a bin directory on PATH
sudo install -m 0755 nrcc-linux-amd64 /usr/local/bin/nrcc

# 3. run it with isolated state
sudo install -d -m 0750 /var/lib/nrcc
sudo env DATA_DIR=/var/lib/nrcc PORT=3001 NRCC_MANAGE_NODE_RED=false \
  /usr/local/bin/nrcc
```

The startup banner prints the version, the listen port and the data directory:

```
┌─ nrcc ─────────────────────────────┐
│ Version: v1.2.3                    │
│ Listening on: :3001                │
│ Data directory: /var/lib/nrcc      │
└────────────────────────────────────┘
```

`NRCC_MANAGE_NODE_RED=false` disables the managed Node-RED child process, which
is what you want when Node-RED is provided by another supervisor. Leave it
enabled (the default) only when `node-red` is on `PATH`.

## What a release publishes

| Artifact | Target | Notes |
| --- | --- | --- |
| `nrcc-linux-amd64` | `linux/amd64` | static (`CGO_ENABLED=0`) |
| `nrcc-linux-arm64` | `linux/arm64` | static (`CGO_ENABLED=0`) |
| `nrcc-linux-armv7` | `linux/arm` v7 | static (`CGO_ENABLED=0`, `GOARM=7`) |
| `SHA256SUMS` | — | digest manifest over the three binaries |
| `nrcc-linux-<arch>.sha256` | — | per-binary sidecar using the normalized `nrcc` filename |

No Darwin or Windows binaries are published: the codebase uses Linux-only
syscalls, so those targets cannot compile.

## Verify a downloaded release

1. Check the manifest and the sidecars against the bytes on disk:

   ```bash
   # the manifest names the published files, so it can be checked directly
   sha256sum -c SHA256SUMS            # add --ignore-missing for a partial download
   ```

   Each sidecar names the normalized file `nrcc` (the convention the install
   scripts rely on), so its digest is checked either against a file named
   `nrcc` or by comparing the two digest strings directly:

   ```bash
   cat nrcc-linux-amd64.sha256        # <digest>  nrcc
   sha256sum nrcc-linux-amd64          # must print the same digest
   ```

2. Check the version and the target embedded in the binary:

   ```bash
   go version -m /usr/local/bin/nrcc
   ```

   The output must show `GOOS=linux`, the expected `GOARCH` (`GOARM=7` for the
   `armv7` build), `CGO_ENABLED=0`, and the version in the build flags:

   ```
   build   -ldflags="-s -w -X main.Version=v1.2.3 -extldflags=-static"
   build   CGO_ENABLED=0
   build   GOARCH=amd64
   build   GOOS=linux
   ```

3. Start the binary and confirm the banner and the health endpoint (see the
   Quick path above).

## Package and verify locally

Both scripts are self-contained and take the version and the output directory:

```bash
# build the three Linux artifacts plus SHA256SUMS and the sidecars
scripts/release/package.sh v1.2.3 dist

# verify checksums, embedded build metadata and an isolated install/start
scripts/release/verify.sh v1.2.3 dist
```

`package.sh` reuses an existing `frontend/dist` and builds it with `pnpm` when
it is missing (set `NRCC_SKIP_FRONTEND_BUILD=1` to fail instead). `verify.sh`
re-computes the checksums, reads the Go build metadata out of every binary,
installs the native artifact into a temporary directory, starts an isolated
server, probes `/healthz`, checks the banner, and then removes the temporary
state and stops the process.

The focused test suite for both scripts is standard-library only:

```bash
python3 scripts/release/test_package.py
```

## Docker image versioning

The image build takes the version as a build argument that defaults to `dev`:

```bash
docker build --build-arg VERSION=v1.2.3 -t nrcc:1.2.3 .
```

The same value is used twice and the two must agree:

- embedded in the binary — `-X main.Version=${VERSION}`;
- published as the OCI label — `org.opencontainers.image.version="${VERSION}"`.

Inspect the label of a built image with:

```bash
docker image inspect --format '{{ index .Config.Labels "org.opencontainers.image.version" }}' nrcc:1.2.3
```

## Operator checklist

- [ ] `NRCC_ENCRYPTION_KEY` is set before `docker compose up`.
- [ ] The stack is reachable on `http://localhost:3001` and `/healthz` returns `200`.
- [ ] Node-RED is reachable on `http://localhost:1880`.
- [ ] The persistent `/data` volume is backed up (it holds the settings, users and secrets).
- [ ] For the binary path: the digest in `SHA256SUMS` and in the `*.sha256` sidecar matches the installed artifact.
- [ ] For the binary path: `go version -m` reports the expected version and target.
- [ ] The startup banner reports the version you intended to deploy.

## Troubleshooting

### The checksum does not match

Re-download the artifact: the manifest, the sidecar and the binary must come
from the same release. Confirm the sidecar carries the digest under the
normalized `nrcc` filename and that the digest was compared against the file
you actually installed.

### `go version -m` reports a different version

The binary was rebuilt locally, or the wrong artifact was downloaded. Compare
`-X main.Version=…` in the build flags with the release tag you expect.

### The binary exits immediately

Check that `DATA_DIR` exists and is writable, that `PORT` is free, and that
`node-red` is on `PATH` unless `NRCC_MANAGE_NODE_RED=false` is set. The banner
and any error are written to standard output.

### The image label and the binary disagree

The image was built without `--build-arg VERSION=…` (or with a different
value). Rebuild it with the argument and re-check the label and the banner.

## Next steps

1. Publish the release and let the release workflow attach the artifacts above.
2. Deploy through the Compose path for the canonical stack.
3. Keep this guide aligned with the ADRs when the deployment model changes.
