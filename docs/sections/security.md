<!--
  Security — authentication, RBAC, auth surfaces, dashboard.
  Inlined into docs/index.html as the "Security" panel.
-->

NRCC separates every authentication surface and gives the operator
explicit, auditable control over each one.

- **Auth and RBAC.** JWT login with bcrypt, admin and viewer roles,
  MFA support, and a last-admin guard so you can never lock yourself
  out of your own stack.
- **Authentication surfaces.** Node-RED admin auth, the legacy
  dashboard, and the FlowFuse dashboard each have a named control
  surface with independent policy.
- **TLS and credential secret.** `credentialSecret`, `https`
  (`requireHttps`), `httpsKey` / `httpsCert` settings are first-class
  fields in the configuration editor; misconfigurations are blocked at
  validation time.
- **Persistent backups.** Manual + scheduled snapshots on a dedicated
  volume. Pre-restore safety backup, retention policies, and integrity
  checks. Restore is a single click.

The full security model is documented in
[`docs/architecture/`](https://github.com/fgjcarlos/nrcc/tree/main/docs/architecture)
and traced to the [#760](https://github.com/fgjcarlos/nrcc/issues/760)
cluster of the control-plane roadmap.
