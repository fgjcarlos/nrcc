<!--
  Quick start — Docker-first, single command.
  Inlined into docs/index.html as the "Quick start" panel.
-->

The canonical install path is Docker Compose. One `docker-compose.yml`,
one command, one stack.

1. Drop the [`docker-compose.yml`](https://github.com/fgjcarlos/nrcc/blob/main/docker-compose.yml)
   from the repo into a fresh directory.
2. Run `docker compose up -d`.
3. Open `http://localhost:3001`, create the admin, and edit
   `settings.js` from the Configuration view.

The image is published as
[`ghcr.io/fgjcarlos/nrcc`](https://github.com/fgjcarlos/nrcc/pkgs/container/nrcc)
for `linux/amd64`, `linux/arm64`, and `linux/armv7`. Tagged builds are
cut from `main`; see the
[Releases](https://github.com/fgjcarlos/nrcc/releases) page for the
latest.
