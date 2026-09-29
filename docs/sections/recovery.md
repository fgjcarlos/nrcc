<!--
  Recovery — backups, restore, rollback, monitoring.
  Inlined into docs/index.html as the "Recovery" panel.
-->

When a configuration change breaks a flow or the Node-RED instance,
NRCC offers three recovery paths in order of cost.

- **Restore from snapshot.** Every manual or scheduled backup is one
  click away. A pre-restore safety snapshot is taken before the new
  snapshot is applied, so the restore itself is reversible.
- **Rollback a settings apply.** The transactional apply pipeline
  records the effective state before and after each apply. A failed
  restart triggers a rollback to the previous settings file.
- **Live logs and metrics.** A ring buffer of Node-RED stdout/stderr
  with level filtering and SSE streaming reads failures as they happen.
  CPU, memory, disk, and uptime are sampled every 30 s, plus a
  Prometheus-compatible `/metrics` endpoint for external scrapers.

Recovery is documented in
[`docs/setup-recovery.md`](https://github.com/fgjcarlos/nrcc/blob/main/docs/setup-recovery.md).
