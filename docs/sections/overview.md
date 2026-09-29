<!--
  Overview — operator problem and product mission.
  Inlined into docs/index.html as the "What nrcc is" panel.
  English only (per #768 policy). Keep paragraphs short; the public
  site is read at glance.
-->

Day-to-day Node-RED operation usually means SSH-ing in, hand-editing
`settings.js`, juggling archive files for backups, and tailing logs by
hand. **NRCC** wraps one Node-RED per stack into a persistent control
plane: JWT auth and RBAC, validated `settings.js` edits, persistent
backups, env-var management, npm-node management, and live logs — all
behind a single Go binary with an embedded React UI.

The canonical deployment is a single Docker Compose service with one
NRCC binary supervising one Node-RED process. Persistent volumes keep
flows, settings, env vars, installed npm nodes, and snapshots safe
across restarts. Spin up a second stack with different host ports to
run a second Node-RED — there is no central control plane in this
release.
