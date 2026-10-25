<!--
  Recovery — backups, restore, rollback, monitoring.
  Inlined into docs/index.html as the "Recovery" panel.
-->

When a configuration change breaks a flow or the Node-RED instance,
NRCC offers three recovery paths in order of cost.

- **Restore from snapshot.** Every manual or scheduled backup is one
  click away. A pre-restore safety snapshot is taken before the new
  snapshot is applied, so the restore itself is reversible.
- **Recover a settings apply.** The production runtime-coordinated
  apply path backs up the active `settings.js` and stages structured
  configuration before restarting the selected runtime and checking
  HTTP readiness. A failed restart/readiness restores both apply-owned
  settings and structured fields before restarting that same authority.
  A recovered transaction is still reported as a failed apply; if
  recovery fails, operator intervention is required. The persistence-only
  legacy apply API does not restart Node-RED. Referenced TLS files are
  outside the apply write-set.
- **Live logs and metrics.** A ring buffer of Node-RED stdout/stderr
  with level filtering and SSE streaming reads failures as they happen.
  CPU, memory, disk, and uptime are sampled every 30 s, plus a
  Prometheus-compatible `/metrics` endpoint for external scrapers.

Recovery is documented in
[`docs/setup-recovery.md`](https://github.com/fgjcarlos/nrcc/blob/main/docs/setup-recovery.md).
