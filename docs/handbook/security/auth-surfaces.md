# Authentication surfaces

> **Four boundaries, four lifecycles.** NRCC manages four distinct
> authentication surfaces. Each one protects a different attack
> surface, uses a different configuration knob, and survives
> independently of the others. This page explains how to recognise
> them, how to configure them, and which mistake is most common
> for each.

## Why four?

Node-RED exposes several HTTP listeners on the same port:

- The editor (`httpAdminRoot`) — used to design flows.
- HTTP-injected endpoints and static file serving
  (`httpNodeRoot`, `httpStatic`) — used by flows at runtime.
- The FlowFuse Dashboard — a long-lived Socket.IO + HTTP surface
  served from inside the editor runtime.

NRCC also exposes its own HTTP API and UI to the operator. Each of
those has its own authentication concern. Treating them as the
same surface (because they happen to live behind the same proxy)
produces the historic "I locked down `adminAuth` and my Dashboard
suddenly stopped asking for a password" confusion.

The four surfaces:

1. **NRCC access** — NRCC's own UI/API authentication.
2. **`adminAuth`** — the Node-RED editor login.
3. **`httpNodeAuth` + `httpStaticAuth`** — HTTP basic auth
   protecting HTTP-injected endpoints and static file serving.
4. **Dashboard boundary** — FlowFuse Dashboard HTTP + Socket.IO.

Each section below is named so its anchor works on GitHub without
rewriting: `nrcc-access`, `adminauth-node-red-editor`,
`httpnodeauth--httpstaticauth-http-basic-auth`,
`dashboard-boundary`.

## NRCC access

### Boundaries

- The HTTP server in `internal/handler/server.go` accepts the
  operator's JWT cookie (or the bearer header in API calls).
- The first login is bootstrap-only: NRCC prompts the operator to
  create the local admin user, then keeps the user in the
  encrypted persisted store.
- MFA is opt-in per user; recovery codes are issued once and
  shown only at enrolment time.

### Default

No users exist at first boot. The setup screen demands an admin
username + password before the rest of the UI is reachable. After
that, the JWT cookie keeps the operator signed in across restarts.

### Restart impact

None. The JWT cookie and the encrypted user store survive a
container restart; the operator does not have to log in again
unless they explicitly sign out or rotate the cookie secret.

### Common pitfalls

- **Sharing `JWT_SECRET` across stacks.** Two stacks that mount the
  same `JWT_SECRET` will accept each other's JWT cookies. Generate
  a fresh secret per stack
  (`openssl rand -base64 48`).
- **Forgetting the setup prompt.** The first call to the API on a
  fresh stack redirects to `/setup`. Operators who front NRCC with
  their own auth proxy sometimes hide that redirect and lock
  themselves out.

## adminAuth (Node-RED editor)

### Boundaries

- The Node-RED editor at `http://<host>:1880/`.
- Backed by the top-level `adminAuth` object in `settings.js`.
- The structured NRCC UI exposes a typed form field for it
  (Secret flag in the [catalog](../configuration/setting-catalog.md)).

### Default

`adminAuth` is **unset** by default in Node-RED 5.0.6 — the editor
is open to anyone who can reach the port. NRCC displays a banner
in the Security Center when this is the case.

### Restart impact

**Required.** Changing `adminAuth` only takes effect after a
Node-RED restart; the editor reads the credentials from
`settings.js` at startup.

### Common pitfalls

- **Hashing the password with the wrong algorithm.** NRCC validates
  that the password hash matches a bcrypt prefix
  (`$2[aby]$`). A SHA-256 hash silently fails to authenticate.
- **Sharing the `adminAuth` object with `httpNodeAuth`.** They have
  different shapes; copying one into the other locks everyone out.
- **Setting `adminAuth` to a strategy without a `type` field.** The
  catalog accepts strategy objects; missing `type` makes Node-RED
  reject the whole `settings.js`.

## httpNodeAuth + httpStaticAuth (HTTP basic auth)

### Boundaries

- HTTP-injected endpoints (Node-RED nodes serving HTTP) at
  `http://<host>:1880/<httpNodeRoot>/...`.
- Static file serving at `http://<host>:1880/<httpStatic>/...` (or
  the array of static sources configured by `httpStatic`).
- Both are protected by HTTP basic auth with bcrypt-hashed
  passwords.

### Default

Both are **unset** by default. HTTP-injected endpoints are open to
anyone who can reach the port. The same is true for static file
serving. NRCC displays a banner in the Security Center when this is
the case.

### Restart impact

**Required.** Same as `adminAuth`.

### Common pitfalls

- **Different bcrypt cost factor than Node-RED expects.** Node-RED
  reads the bcrypt cost from the hash prefix (`$2b$10$...`); NRCC
  validates the prefix but does not re-hash on save.
- **Setting only one of the two.** Operators who set
  `httpNodeAuth` but forget `httpStaticAuth` leave the static
  surface open. The catalog treats the two as a pair in the
  Security Center.
- **Putting `httpStaticAuth` in front of a public static folder.**
  The basic-auth prompt on a public folder leaks the username
  password through the browser's password manager and through
  every reverse proxy log.

## Dashboard boundary

### Boundaries

- The FlowFuse Dashboard HTTP + Socket.IO surface, served from
  inside the Node-RED process.
- The boundary is independent of `adminAuth`: an operator who can
  reach the editor is not automatically authorised against the
  Dashboard.

### Default

The Dashboard is **not installed** by default in Node-RED 5.0.6.
Operators opt in by installing `@flowfuse/node-red-dashboard` and
restarting. Once installed, the Dashboard listens on the same
port as the editor (`1880`) and uses a different authentication
mechanism.

### Restart impact

**Required** for the initial install/uninstall of the package and
for any change to the Dashboard's authentication configuration.

### Common pitfalls

- **Assuming `adminAuth` protects the Dashboard.** It does not.
  The Dashboard uses its own credentials and exposes a different
  Socket.IO namespace.
- **Forgetting to lock down `httpStatic`.** The Dashboard
  distributes static assets through Node-RED's static handler. An
  open `httpStatic` leaks the Dashboard's JavaScript and CSS to
  anyone who reaches the port.
- **Reverse-proxying the Dashboard with a path rewrite but not a
  separate cookie scope.** The Dashboard's Socket.IO session cookie
  must be scoped to the rewritten path; sharing the editor's cookie
  scope allows a leaked editor cookie to grant Dashboard access.

## Cross-surface guidance

- **Configure each surface independently.** A security incident on
  one surface does not invalidate the others' credentials.
- **Use distinct credentials per surface.** Reusing the
  `adminAuth` username/password for `httpNodeAuth` makes a
  compromise on one surface an instant compromise on the others.
- **Verify each surface after every restart.** The verification
  blocks in the [playbook](../operator/playbook.md) provide
  per-surface curl commands that succeed only when the boundary is
  honoured.
- **Audit the Security Center banners first.** The four banners in
  the Security Center UI correspond one-to-one to the four
  surfaces above; if a banner is yellow or red, that surface is
  the starting point for the diagnosis.

## Related

- [`../configuration/setting-catalog.md`](../configuration/setting-catalog.md)
  — `adminAuth`, `httpNodeAuth`, `httpStaticAuth`, `https` are
  catalog entries with Secret=true and RestartRequired=true.
- [`../configuration/apply-pipeline.md`](../configuration/apply-pipeline.md)
  — what happens when a surface is changed.
- [`../glossary.md`](../glossary.md) — definitions for `adminAuth`,
  `httpNodeAuth`, `httpStaticAuth`, `RBAC`, `Secret`, and
  `Boundary`.
- [`docs/control-plane.md`](../../control-plane.md) — clusters
  #759, #760, and #761 of the roadmap.
- Issue #760 — separating every authentication surface.
- Issue #761 — securing legacy and FlowFuse dashboards.
