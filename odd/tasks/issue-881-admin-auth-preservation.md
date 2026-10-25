# Issue #881 — Preserve all admin users and permissions on unrelated visual edits

> P1 bug fix. Issue: https://github.com/fgjcarlos/nrcc/issues/881
> Status: approved (`status:approved` label present on the issue).

## Objective

An unrelated configuration edit (for example `editorTheme.page.title`)
must not change Node-RED's authentication surfaces. Today the
configuration form rebuilds the `adminAuth` block from two legacy
single-user fields, so the saved payload contains one account with
wildcard permissions — every other account, every `read` permission and
the session expiry are dropped.

## Root cause (verified against the current tree)

`frontend/src/features/configuration/lib/configTransformers.ts`

- `configToFormData` collapses `adminAuth.users` to `users[0].username`
  and throws away `permissions` and `sessionExpiryTime`.
- `formDataToConfigPayload` rebuilds the block from
  `authAdminUser` / `authAdminPassword` and hardcodes
  `permissions: '*'`.

`POST /api/config` replaces the persisted model with that payload
(`ConfigService.Save` / `CommitStructured` assign the candidate over the
stored config), so the destructive block reaches `settings.js` through
`renderAdminAuthBlock`.

The `/security` page already owns these surfaces properly:
`AdminAuthBoundaryCard` edits every user, the per-user `read` / `*`
permission and `sessionExpiryTime`, and applies them through
`POST /api/config/apply`. The configuration form is therefore a second,
lossy writer for the same surface — that is the conflict to remove.

Two further gaps in the same class:

- The configuration payload also emits `nodeHttpAuth` / `staticAuth`
  with the legacy alias keys and a `{type, users}` shape. The backend
  model uses `httpNodeAuth` / `httpStaticAuth` with `{user, pass}`, so
  those keys were silently ignored — the surfaces were never written and
  never read back.
- `SecurityView` reads `config.nodeHttpAuth` / `config.staticAuth`, so
  the basic-auth cards never saw the configured surface even though the
  backend reports it under `httpNodeAuth` / `httpStaticAuth`.

## Design

Ownership split, not a second copy of the model:

1. The **configuration form stops writing the security surfaces**. It
   submits only the capabilities it owns (editor theme, logging, files,
   projects, runtime state, language, TLS). `formDataToConfigPayload`
   no longer emits `adminAuth`, `nodeHttpAuth` or `staticAuth`.
2. The **backend preserves the security surfaces that a payload omits**.
   `POST /api/config` and `POST /api/config/apply` detect an absent
   surface and keep the loaded model (persisted store first, live
   `settings.js` second — the same precedence `ConfigService.Get` uses).
   A carried surface keeps its passwords: an empty password means
   "unchanged" and is filled from the stored/live hash by username.
3. The **security page keeps the real basic-auth keys** so the
   `httpNodeAuth` / `httpStaticAuth` cards read and write the surfaces
   the backend actually exposes.

Explicit clearing stays possible: `adminAuth: null` (as sent by the
security cards when the last user is removed) is *present but empty* and
therefore clears the surface, while an absent key preserves it.

## Tasks

- [x] T1 — Configuration form stops reconstructing the security surfaces.
  - `formDataToConfigPayload` emits no auth surface; `ConfigPayload`
    drops the auth members.
  - `NodeRedConfigResponse` models the real shapes: `adminAuth.users[]`
    with `permissions`, `sessionExpiryTime`, and `httpNodeAuth` /
    `httpStaticAuth` with `{user, pass}`.
  - Unit tests assert that an unrelated visual edit produces a payload
    with no auth surface, and that the payload cannot reintroduce a
    single wildcard account.
- [x] T2 — Backend preserves the security surfaces omitted from a payload.
  - `model.NodeRedConfig` records whether a payload carried each
    security surface (JSON presence, not "zero value").
  - `ConfigService.Save` and `ConfigService.CommitStructured` keep the
    loaded surface when the payload omits it and keep the password hash
    of every unchanged user.
  - Service and handler tests cover: omitted surface preserved, carried
    surface applied, unchanged password preserved, cleared surface
    (`null`) applied.
- [x] T3 — Legacy single-user auth form fields removed.
  - `authEnabled`, `authAdminUser`, `authAdminPassword`,
    `authNodeHttp*`, `authStatic*` leave the configuration form model,
    the per-field diff map, the validator, the field labels and the
    dead `AuthSettings` component (never rendered since #766 slice A).
  - The shared `NodeRedConfig` type stops modelling the legacy
    single-user shapes.
- [x] T4 — Real-stack proof that an unrelated visual edit leaves
  Node-RED authentication unchanged.
  - `frontend/e2e/stack.spec.ts` (real NRCC + real Node-RED):
    configure two admin accounts (one `read`), restart Node-RED, prove
    the `read` account can log in and cannot deploy, apply a
    `editorTheme` change through the configuration save path, then prove
    both accounts, their permissions and the session expiry are
    unchanged and the runtime still enforces them.
- [x] T5 — Settings renderer emits valid JavaScript when it appends a key.
  - Found by the T4 live proof: `replaceScalarKey`, `replaceArrayKey`,
    `replaceBlockKey` and `ensureHTTPStatic` appended their entry right
    after the last property of the file. Node-RED's own default
    `settings.js` ends with `externalModules: {}` and no trailing comma,
    so the rendered file read `externalModules: {} uiPort: 1880,` —
    invalid JavaScript. Every configuration edit therefore failed with
    `SETTINGS_SYNTAX_ERROR` in a fresh stack, which blocks the "an
    unrelated visual edit" scenario the issue asks to prove.
  - `appendBeforeClosingBrace` adds the missing comma and keeps the
    appended entries on their own lines.

## Acceptance criteria (from the issue)

- [x] Security surfaces are no longer reconstructed from single-user
  legacy form fields.
- [x] A payload submits only the edited capabilities; the surfaces it
  omits are preserved with the loaded model and revision.
- [x] Multiple users, `read` vs `*` permissions and the session expiry
  survive an unrelated visual edit.
- [x] A real-stack run proves Node-RED authentication and permissions
  are unchanged after an unrelated visual edit.

## Work units (committed)

| Commit | Work unit |
| --- | --- |
| `5aaea49` | `fix(settings)`: the renderer keeps the settings.js valid when it appends a key (T5) |
| `2134bae` | `fix(backend)`: the surfaces a payload omits are preserved (T2) |
| `efe538b` | `fix(frontend)`: the review panel stays above its backdrop (T4 blocker) |
| `c1a2daf` | `fix(frontend)`: the configuration form no longer rebuilds the auth surfaces (T1) |
| `1352895` | `chore(frontend)`: the legacy single-user auth form fields are gone (T3) |
| `4d18404` | `test(e2e)`: the real-stack proof (T4) |

## Constraints

- Repository artifacts are written in English (AGENTS.md).
- The uncommitted `raw-settings-validation` (#875) work in the tree
  belongs to another workstream: `internal/handler/config.go`,
  `internal/handler/config_apply_test.go`, `internal/handler/settings.go`,
  `internal/handler/settings_test.go`, `internal/model/config.go`,
  `internal/service/config.go`, `internal/service/config_test.go`,
  `internal/service/settings_sandbox.go`,
  `internal/service/settings_sandbox_test.go` and
  `scripts/acceptance/docker-stacks.sh` are already dirty. Changes that
  touch those files must not drop the pending #875 edits, and the #881
  commits must not include them.
- No generated artifact is edited by hand.
- Every claim needs observed evidence (test output, build, real stack).
