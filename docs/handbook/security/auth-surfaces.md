# Authentication surfaces

> **Stub.** Landed as a skeleton in W1. Four-surface breakdown
> arrives in W3 with Boundaries / Default / Restart impact /
> Common pitfalls per surface.

## Surfaces

1. **NRCC access** — JWT cookie, RBAC, MFA. NRCC-managed.
2. **`adminAuth`** — Node-RED editor login. Top-level
   `settings.js` key.
3. **`httpNodeAuth` + `httpStaticAuth`** — HTTP basic auth
   protecting HTTP-injected endpoints and static file serving.
4. **Dashboard boundary** — FlowFuse Dashboard HTTP + Socket.IO
   protection.

Each surface is independent. Changing one does not affect the
others. The detailed table lands in W3.
