<!--
  Configuration — what NRCC configures and how.
  Inlined into docs/index.html as the "Configuration" panel.
-->

NRCC configures the high-value Node-RED 5.x settings from a single
validated editor. Every save round-trips through a real `settings.js`,
creates a backup of the previous file, applies the change
transactionally, and surfaces the effective state after the Node-RED
restart.

- **Persistent per-instance data.** Flows, credentials, `settings.js`,
  env vars, npm `node_modules`, users, and snapshots all live in named
  volumes. The stack survives restarts without losing state.
- **Edit `settings.js` safely.** Validated editor that round-trips
  through a real `settings.js` and creates a backup of the previous
  file before every save. Restart prompt included.
- **Manage env vars and secrets.** Typed environment variables
  (string, number, boolean, secret) with encryption at rest. Changes
  propagate into Node-RED on the next start.
- **npm nodes via the UI.** Install, search, and uninstall npm
  packages from the same panel that uses them. The package lands in
  the persistent `node_modules` volume.
