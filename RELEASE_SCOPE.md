# NRCC MVP — Release Scope & Smoke-Test Gate

## Release Scope Decision

### In MVP

| Feature | Status | Notes |
|---------|--------|-------|
| CLI: `nrcc setup` | ✅ Implemented | Full bootstrap flow via `internal/service/environment.go` |
| CLI: `nrcc start` | ✅ Implemented | Starts backend + Node-RED process |
| CLI: `nrcc doctor` | ✅ Implemented | 13-check diagnostic service |
| CLI: `nrcc logs`, `nrcc support` | ✅ Implemented | Log viewer + support bundle export |
| Auth: register first user, login, logout | ✅ Implemented | Cookie sessions, bcrypt, audit logging |
| Dashboard: runtime status, uptime, system info | ✅ Implemented | Frontend overview page with restart |
| Config: edit supported settings, validate, apply | ✅ Implemented | Full config service with settings.js generation, JSON editor, snapshots, import |
| Environment variables: CRUD + apply | ✅ Implemented | `.env.managed` with atomic write |
| Backups: create, restore, preventive backup | ✅ Implemented | Compressed snapshots with manifest |
| NPM libraries: install, uninstall, list | ✅ Implemented | Job manager with operation locking |
| Node-RED updates: check + apply | ✅ Implemented | Status check + apply with backup |
| Diagnostics: logs, jobs, doctor, export | ✅ Implemented | Phase 10 fully complete with tests |
| Local access via portless | ✅ Implemented | Merged PR #43 |
| CI pipeline | ✅ Implemented | Backend tests + frontend build (#22) |
| Release packaging | ✅ Implemented | Automated artifacts (#25) |
| Frontend polish: design system, animations, responsive, accessibility | ✅ Implemented | PRs #39 merged (issues #31–#37) |
| Frontend tests | ✅ Implemented | Vitest baseline (#29) |

### Explicitly Deferred

| Feature | Issue | Reason |
|---------|-------|--------|
| Branding asset uploads | #15 | URL-based config works; file upload pipeline not built |
| Secret encryption for env vars | #14 | Vars stored in plaintext `.env.managed`; acceptable for local-only tool |
| Multi-user / roles | #13 | Single admin user is sufficient for MVP |
| CSRF protection, rate limiting, password policy | #38 | Local-only tool; security hardening is post-MVP |
| Backend hardening (op locking, error format) | #16, #30 | Basic operation lock exists; full hardening deferred |
| Empty states & error boundaries | #33 | UX polish, not blocking |
| Flow management module | #18 | Post-MVP (Phase 11) |
| AI flow analysis | #19 | Post-MVP (Phase 12) |
| macOS / Windows support | — | Linux-only for MVP per plan |
| `nrcc stop` as public command | — | Removed (#21); stop happens via signal/process exit |

---

## Smoke-Test Checklist

Each item is a concrete test a human can run on a clean Linux machine.

### ✅ Must Pass Before Release

- [ ] **Setup**: Run `nrcc setup` on a machine with Node.js/npm. Confirm it creates `~/.nrcc/data`, installs Node-RED, reports success.
- [ ] **Start**: Run `nrcc start`. Confirm backend serves UI at printed URL. Confirm Node-RED process is running.
- [ ] **First user**: Open UI → registration form appears (no existing users). Create user. Confirm redirect to dashboard.
- [ ] **Login/Logout**: Log out. Log back in with created credentials. Session persists across page reload.
- [ ] **Dashboard**: Dashboard shows Node-RED status (running), version, uptime. System info panel populates.
- [ ] **Restart**: Click restart from UI. Confirm Node-RED restarts and dashboard updates.
- [ ] **Runtime logs**: Navigate to logs page. Confirm recent Node-RED stdout/stderr lines appear.
- [ ] **Config apply**: Navigate to config page. Change a supported setting (e.g. flow file name). Validate → Apply. Confirm settings.js is regenerated. Restart Node-RED and confirm change took effect.
- [ ] **Environment vars**: Navigate to environment page. Add a variable `TEST_VAR=hello`. Apply. Restart Node-RED. Confirm var is in process environment (use a flow's `env.get()` or check logs).
- [ ] **Backup create**: Navigate to backups. Create backup. Confirm it appears in list with timestamp and size.
- [ ] **Backup restore**: Restore a backup. Confirm preventive backup is created first. Confirm files are restored.
- [ ] **NPM install**: Navigate to libraries. Install a package (e.g. `node-red-contrib-moment`). Confirm it appears in list after job completes.
- [ ] **NPM uninstall**: Uninstall the package. Confirm removal.
- [ ] **Doctor**: Run `nrcc doctor` from CLI. Confirm all checks pass on a healthy system.
- [ ] **Diagnostics page**: Open diagnostics in UI. Confirm doctor, logs, and jobs tabs load data.

### ⚠️ Nice to Have (non-blocking)

- [ ] **Update check**: Updates page shows current vs available Node-RED version.
- [ ] **Update apply**: If a newer version exists, apply update and confirm Node-RED restarts on new version.
- [ ] **Config snapshots**: Create config snapshot, make changes, restore snapshot.
- [ ] **Config import**: Import a `settings.js` from another Node-RED instance.
- [ ] **Support export**: Run `nrcc support` or use UI export. Confirm bundle downloads with sanitized data.
- [ ] **Portless local access**: If portless is available, confirm hostname is published on start.
- [ ] **Dark mode**: Toggle theme. Confirm UI renders correctly in both modes.

### ⏭️ Explicitly Deferred

- [ ] Branding image uploads (#15)
- [ ] CSRF / rate limiting / password policy (#38)
- [ ] Multi-user management (#13)
- [ ] Encrypted env var secrets (#14)
- [ ] Flow viewer/analysis (#18, #19)
- [ ] macOS / Windows

---

## Known Gaps & Acceptable Limitations

1. **No `nrcc stop` command** — Intentional (#21). The process is stopped via Ctrl+C / SIGTERM. This is documented.
2. **No file uploads for branding** — Editor theme config accepts URLs, not file uploads. Functional but limited (#15).
3. **Env vars stored in plaintext** — `.env.managed` is not encrypted. Acceptable for a local-only tool where the filesystem is the trust boundary (#14).
4. **Single admin user only** — No role-based access. First registered user is the only user (#13).
5. **No CSRF tokens** — Session uses HttpOnly cookies but no CSRF double-submit. Low risk for local-only tool (#38).
6. **Operation locking is basic** — `OperationLock` prevents concurrent mutations but error reporting for locked operations may not be user-friendly (#16).

---

## Release Gate

**Shipping is blocked until ALL of these are true:**

1. Every item in the "Must Pass" checklist above passes on a clean Ubuntu/Debian machine
2. `make build` produces a working binary
3. CI is green (backend tests + frontend build)
4. No open issues labeled `high-priority` that affect core flows (currently: #38 is `high-priority` but scoped to security hardening — **decision needed**: ship without it or promote to blocker?)
5. README and operator docs are current (#24 — already merged)

### Decision Needed

> **#38 (CSRF + rate limiting + password policy)** is labeled `high-priority` + `security`. For a local-only tool, this is arguably acceptable to defer. But if NRCC will be exposed over a network (even LAN), rate limiting on login and basic CSRF become more important. **Decide: is network exposure a supported use case for MVP?**
