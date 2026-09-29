<!--
  Compatibility matrix — Node-RED version policy.
  Inlined into docs/index.html as the "Compatibility" table.
-->

The compatibility matrix below is the operator contract. NRCC matches
the [Node-RED 5.x compatibility cluster](https://github.com/fgjcarlos/nrcc/issues/756).

| Node-RED version | NRCC behaviour |
|------------------|----------------|
| `>=5.0 <6.0` | **Full editing.** All `settings.js` fields, env vars, npm nodes, backups, auth surfaces. |
| `4.x` | **Read-only.** Detection, inspection, and migration guidance. Destructive edit flows are blocked. |
| `>=6.0` (future) | **Read-only** until a dedicated adapter / catalog is verified for that major. |

The Docker image pins Node-RED `5.0.7-24-minimal` by digest; Dependabot
updates the digest weekly. See
[`Dockerfile`](https://github.com/fgjcarlos/nrcc/blob/main/Dockerfile)
for the pinned reference.
