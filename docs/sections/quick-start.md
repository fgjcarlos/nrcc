<!--
  Quick start — Docker-first, single command.
  Inlined into docs/index.html as the "Quick start" panel.
-->

The canonical install path is Docker Compose. One `docker-compose.yml`,
one command, one stack.

1. Drop the [`docker-compose.yml`](https://github.com/fgjcarlos/nrcc/blob/main/docker-compose.yml)
   into a fresh directory:
   `curl -fsSLO https://raw.githubusercontent.com/fgjcarlos/nrcc/main/docker-compose.yml`.
2. Create the encryption key the stack requires:
   `printf 'NRCC_ENCRYPTION_KEY=%s\n' "$(openssl rand -hex 32)" > .env`.
3. Run `docker compose up -d`. It pulls the latest stable image; to run
   a release candidate, add `NRCC_IMAGE_TAG=1.0.0-rc.1` (the version
   without the `v`) to `.env` first.
4. Open `http://localhost:3001`, create the admin, and edit
   `settings.js` from the Configuration view.

Upgrade with `docker compose pull && docker compose up -d`. A stack pinned
with `NRCC_IMAGE_TAG`, as the installer does, upgrades by re-running the
installer with the new version.
Or let the installer do steps 1–3:
`curl -fsSL https://raw.githubusercontent.com/fgjcarlos/nrcc/main/docs/install.sh | sh`.

The image is published as
[`ghcr.io/fgjcarlos/nrcc`](https://github.com/fgjcarlos/nrcc/pkgs/container/nrcc)
for `linux/amd64` and `linux/arm64` (on 32-bit ARM, use the
`nrcc-linux-armv7` binary). Tagged builds are
cut from `main`; see the
[Releases](https://github.com/fgjcarlos/nrcc/releases) page for the
latest.
