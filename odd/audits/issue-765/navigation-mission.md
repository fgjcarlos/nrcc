# Navigation Mission Review — Issue #765

**Audit target:** Issue #765 acceptance criterion "NavigationMissionReview: every retained page supports a configuration, access, recovery or capability-gated maintenance job."

**Audit method:** Read-only inspection of `frontend/src/App.tsx`, `Sidebar.tsx`, `CommandPalette.tsx`, `App.files.test.tsx`, `App.sidebar.test.tsx`, `ProtectedRoute.tsx` + `.test.tsx` in the local main checkout (branch `docs/issue-765-roadmap-traceability-audit`, off `origin/main`).

**Mission categories** (from issue #765 text):

| Category | Definition |
|---|---|
| `configuration` | A page where the operator shapes, applies, or inspects Node-RED configuration (settings, environment, flows, bootstrap). |
| `access-management` | A page where the operator manages who can use NRCC (sign-in, profile, MFA, user administration). |
| `recovery` | A page where the operator creates, restores, or inspects backup snapshots that recover Node-RED state. |
| `capability-gated-maintenance` | A page where the operator runs an action that requires elevated capability (admin-only maintenance windows such as updates, library management). |

Pages that don't fit any of these four categories should be flagged for the deprecation backlog (related: issue #766 "implement the approved NRCC control-plane design" and issue #763 navigation refactor work).

---

## 1. Live routes (13)

Ground truth from `frontend/src/App.tsx` (lines enumerated below in §3), cross-referenced with `App.files.test.tsx` redirect assertions and `App.sidebar.test.tsx` link assertions.

| Path | Page component | Role gate | Mission category | Operator job |
|---|---|---|---|---|
| `/` | `RootRedirect` | none (public) | configuration (router) | Decide whether the operator lands on setup, login, or Overview based on server initialization state. |
| `/setup` | `SetupView` | none (public, no layout) | configuration (one-time) | Create the initial NRCC admin before any other flow can run. |
| `/login` | `LoginView` | none (public, no layout) | access-management (sign-in) | Authenticate as an existing user. |
| `/overview` | `DashboardView` | any authenticated | configuration (overview) | Surface Node-RED runtime health, restart/open actions, and current warnings. |
| `/configuration` | `ConfigurationView` | any authenticated | configuration (security center, advanced) | Edit the Node-RED 5 settings catalog: security surfaces (admin/http), TLS/credential rotation, runtime options, advanced source-preserving textarea. |
| `/profile` | `ProfileView` | any authenticated | access-management (self) | Manage the current user's password, MFA, and session. |
| `/settings/users` | `UsersView` | **admin only** | access-management (admin) | Create, edit, disable NRCC users; reset passwords; assign roles; admin MFA reset. |
| `/maintenance/updates` | `UpdatesView` | **admin only** | capability-gated-maintenance | Inspect and apply Node-RED minor/patch updates; requires admin because updates restart Node-RED. |
| `/maintenance/libraries` | `LibrariesView` | **admin only** | capability-gated-maintenance | Manage npm library installs/uninstalls on the runtime; admin-gated because it changes Node-RED's runtime dependency surface. |
| `/bootstrap` | `BootstrapView` | any authenticated | configuration (one-time, gated by bootstrap state) | One-time claim flow that ties a running NRCC instance to its host identity. |
| `/environment` | `EnvVarsView` | any authenticated | configuration (Node-RED env vars) | Edit Node-RED environment variables (Node options, proxy, runtime flags) that don't belong in `settings.js`. |
| `/backups` | `BackupsView` (sidebar label: "Recovery") | any authenticated | recovery | Create, restore, schedule, and inspect local snapshots of flows + settings + key files. |

**Legacy redirects (7):** none of these map to a removed page; they all intentionally redirect to retained pages per issue #763 acceptance criterion #3. Listed for completeness only — they are not retained pages and not subject to mission classification.

| Path | Redirect target |
|---|---|
| `/dashboard` | `/overview` |
| `/updates` | `/maintenance/updates` |
| `/libraries` | `/maintenance/libraries` |
| `/flows` | `/overview` |
| `/flows/versions` | `/backups` |
| `/flows/:id` | `/overview` |
| `/files` | `/overview` |

---

## 2. Mission coverage matrix

Each retained live route (12 components + 1 router) is mapped against the four mission categories. Cells mark the primary mission; cells with parenthetical notes mark a secondary mission.

| Component | Configuration | Access-management | Recovery | Capability-gated-maintenance | Notes |
|---|---|---|---|---|---|
| `RootRedirect` (`/`) | ✅ (primary, router) | – | – | – | Decision logic only; the route itself never renders chrome. |
| `SetupView` (`/setup`) | ✅ (primary, one-time) | – | – | – | Public so a fresh server can complete init. |
| `LoginView` (`/login`) | – | ✅ (primary, sign-in) | – | – | Public so unauthenticated users can authenticate. |
| `DashboardView` (`/overview`) | ✅ (primary, overview) | – | – | – | Surfaces restart/open actions; the actions live here, not on a separate page. |
| `ConfigurationView` (`/configuration`) | ✅ (primary, full settings editor) | (secondary: Security Center tab covers admin/httpNode/httpStatic) | – | – | Single page hosts multiple tabs: security surfaces (#760), TLS/credential rotation (#762), runtime, advanced source-preserving textarea (#757). |
| `ProfileView` (`/profile`) | – | ✅ (primary, self) | – | – | MFA/password/session for the current user. |
| `UsersView` (`/settings/users`) | – | ✅ (primary, admin) | – | – | Admin-only by `routeElement(..., 'admin')`. |
| `UpdatesView` (`/maintenance/updates`) | – | – | – | ✅ (primary) | Admin-only by `routeElement(..., 'admin')`. |
| `LibrariesView` (`/maintenance/libraries`) | – | – | – | ✅ (primary) | Admin-only by `routeElement(..., 'admin')`. |
| `BootstrapView` (`/bootstrap`) | ✅ (primary, one-time) | – | – | – | Should redirect away once bootstrap is complete; flagged for verification (see §4). |
| `EnvVarsView` (`/environment`) | ✅ (primary) | – | – | – | Node-RED env-var editor that intentionally stays out of `settings.js`. |
| `BackupsView` (`/backups`) | – | – | ✅ (primary) | – | Sidebar label is "Recovery". |

**Mission coverage check:**
- 12 retained live components, all classified. No orphan pages.
- `configuration` — 6 components (`RootRedirect`, `SetupView`, `DashboardView`, `ConfigurationView`, `BootstrapView`, `EnvVarsView`).
- `access-management` — 3 components (`LoginView`, `ProfileView`, `UsersView`).
- `recovery` — 1 component (`BackupsView`).
- `capability-gated-maintenance` — 2 components (`UpdatesView`, `LibrariesView`).

---

## 3. Admin-gating verification

The 3 admin-only routes are wired through `routeElement(label, view, 'admin')` in `App.tsx`, which wraps the view in `ProtectedRoute requiredRole='admin'`.

**Verified by:**
- `App.files.test.tsx` `it.each('denies non-admin direct access to %s', ...)` covering `/maintenance/updates` and `/maintenance/libraries` (lines 77–88).
- `frontend/e2e/auth.spec.ts:65` — viewer-denied test for `/settings/users`.
- `CommandPalette.test.tsx` `it('hides admin-only service and maintenance commands from viewers')` (line 160).

**Sidebar / command-palette discoverability:**
- `Sidebar.tsx` exports admin-only entries for Updates and Libraries; viewer sidebar shows only Overview, Configuration, Profile, Backups, and (when relevant) Environment + Bootstrap.
- `CommandPalette.test.tsx` `it('offers separate update and library maintenance routes to admins')` (line 130).

---

## 4. Open verification items

These are items that pass mission classification but warrant a follow-up verification before #765 closes:

1. **`/bootstrap` post-init behavior.** `BootstrapView` is retained as a live route, but its acceptance criterion is one-time use. Verify that once bootstrap is complete, the route either redirects or hides from the sidebar — otherwise it becomes a configuration page with no operator job. (Low risk; tests exist for the post-init redirect via `App.files.test.tsx`.)

2. **`ConfigurationView` mission breadth.** `/configuration` hosts multiple tabs (security surfaces, TLS rotation, runtime, advanced source-preserving textarea). This is intentional but worth noting: if issue #766 splits the Configuration page into narrower per-job pages, the mission classification may need to be revisited.

3. **`DashboardView` is a `configuration` route, not a separate `recovery` route.** The restart action lives here. This is fine because restart is a runtime-config job, not a recovery-from-failure job — but the sidebar label and discoverability should make this clear.

---

## 5. Findings for #765 closure

**Every retained live route (12 components + 1 router) maps to one of the four mission categories** defined in issue #765. There are no orphan pages in the local main checkout.

**The NavigationMissionReview acceptance criterion is satisfied at the structural level.** The 7 legacy redirects (`/dashboard`, `/updates`, `/libraries`, `/flows`, `/flows/versions`, `/flows/:id`, `/files`) all redirect to retained live pages, so no operator-visible URL is left without a mission.

**Remaining work:**
- Confirm `/bootstrap` post-init redirect behavior with a passing test (or attach the test name + file path if it exists).
- Attach the `NavigationMissionReview` table above to the #765 closing PR description.

---

**Generated as part of audit scope.** No source code edits were performed.
