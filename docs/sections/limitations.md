<!--
  Limitations — explicit non-goals.
  Inlined into docs/index.html as the "What NRCC does NOT do" panel.
-->

NRCC's mission is narrow. The following are explicit non-goals and will
not change in the control-plane roadmap.

- **Not a flow editor.** Use the Node-RED editor (port 1880) to
  author flows. NRCC configures; it does not author.
- **Not a cluster orchestrator.** A central NRCC that manages
  multiple remote Node-REDs is deferred — see
  [#428](https://github.com/fgjcarlos/nrcc/issues/428). Each NRCC
  instance supervises exactly one Node-RED.
- **Not a npm library browser.** NRCC installs, searches, and
  uninstalls the npm packages that flows need, but it does not replace
  npm itself.
- **Read-only on Node-RED 4 and unknown future majors.** NRCC
  supports full editing on Node-RED `>=5.0 <6.0`. Earlier majors and
  future majors are detected, inspected read-only, and routed to a
  migration guide.
- **No Docker socket.** NRCC does not manage sibling containers and
  does not mount `/var/run/docker.sock`. Bring up a second stack as a
  second Compose service with different host ports.
- **Local backups only.** Off-host encrypted backups (Restic / S3 /
  SFTP / B2) are deferred — see
  [#432](https://github.com/fgjcarlos/nrcc/issues/432).
