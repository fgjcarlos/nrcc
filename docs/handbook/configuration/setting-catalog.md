# Setting catalog

> **Canonical reference.** Single source of truth is the
> `[]SettingCatalogEntry` slice in
> `internal/service/nodered_compatibility.go`
> (`nodeRED5Catalog`, `nodeRED5CatalogVersion = "5.0.6"`). This page
> is a hand-written mirror; the sync is enforced by
> `internal/service/catalog_doc_sync_test.go`.

## How to read this catalog

Each row in the catalog table below is one `SettingCatalogEntry`
defined in `internal/service/nodered_compatibility.go`. The columns
match the struct fields one-to-one:

| Field | Meaning |
|-------|---------|
| `Key` | Top-level `settings.js` key the entry manages. |
| `Shape` | Accepted value shape (string / boolean / object / https-options / …). |
| `Default` | Node-RED 5.0.6 default if the operator does not override. |
| `Validation` | Rule the apply pipeline uses to validate the value. |
| `Secret` | Redacted in previews; never logged in plaintext. |
| `RestartRequired` | Whether changing this setting requires restarting Node-RED. |
| `UIEditable` | Whether the structured UI exposes an editor for it. |

Catalog version is pinned by `nodeRED5CatalogVersion = "5.0.6"`.
Bumping the version is a deliberate operator-visible decision that
must include a sync update and (if shapes change) an ADR.

## Catalog (Node-RED 5.0.6)

| Key | Shape | Default | Validation | Secret | RestartRequired | UIEditable |
|-----|-------|---------|------------|:------:|:---------------:|:----------:|
| flowFile | string | flows.json | non-empty |   | ✓ | ✓ |
| credentialSecret | string-or-false | generated | string-or-false | ✓ | ✓ | ✓ |
| flowFilePretty | boolean | true | boolean |   | ✓ | ✓ |
| userDir | string | ~/.node-red | non-empty-path |   | ✓ | ✓ |
| nodesDir | string |  | path |   | ✓ | ✓ |
| adminAuth | object |  | credentials-or-strategy | ✓ | ✓ | ✓ |
| httpNodeAuth | object |  | bcrypt-password | ✓ | ✓ | ✓ |
| httpStaticAuth | object |  | bcrypt-password | ✓ | ✓ | ✓ |
| uiPort | number-or-expression | process.env.PORT \|\| 1880 | port-or-expression |   | ✓ | ✓ |
| uiHost | string | 0.0.0.0 | host-or-ip |   | ✓ | ✓ |
| httpAdminRoot | string-or-false | / | path-or-false |   | ✓ | ✓ |
| httpNodeRoot | string-or-false | / | path-or-false |   | ✓ | ✓ |
| https | https-options | undefined | https-options | ✓ | ✓ | ✓ |
| requireHttps | boolean | false | boolean |   | ✓ | ✓ |
| httpStatic | string-or-array |  | path-or-static-sources |   | ✓ |   |
| lang | string | en-US | locale |   | ✓ | ✓ |
| runtimeState | object | {enabled:false,ui:false} | runtime-state-options |   | ✓ | ✓ |
| logging | object | {console:{level:info}} | logging-options |   | ✓ | ✓ |
| disableEditor | boolean | false | boolean |   | ✓ | ✓ |
| editorTheme | object |  | editor-theme-options |   | ✓ | ✓ |
| functionGlobalContext | object | {} | object | ✓ | ✓ |   |

`Secret = ✓` cells render with a closed lock glyph in the UI and
with the placeholder `<redacted>` in previews. `UIEditable = ☐`
cells are **managed** (NRCC preserves them through edits) but
**not exposed** as a structured form field; operators reach them
through the advanced escape hatch documented in
[`apply-pipeline.md`](apply-pipeline.md).

## Per-entry notes

### `flowFile`

Path to the active `flows.json` Node-RED loads at startup. Restart
required. The default `flows.json` lives next to `settings.js`.

### `credentialSecret`

Encrypts `credentials.json` at rest. Setting it to `false` disables
encryption (not recommended; the user explicitly opts in).
**Secret**: changing it invalidates stored credentials, so the
catalog treats it as a secret even though the UI accepts the value
through a normal text field. **Restart required**. **Verification**:
editor login still works after restart, and `credentials.json`
cannot be read without the secret.

### `flowFilePretty`

Pretty-prints `flows.json` when Node-RED writes it. Restart
required.

### `userDir`

Root directory Node-RED uses for `flows.json`, `settings.js`,
`lib/`, `node_modules/`. Restart required. Cannot point inside
NRCC's `DATA_DIR`/`backups/` because backups would then be part of
the runtime state.

### `nodesDir`

Search path for additional Node-RED nodes. Restart required.

### `adminAuth`

Node-RED editor login. Top-level object. **Secret**. **Restart
required**. See
[`security/auth-surfaces.md § adminAuth`](../security/auth-surfaces.md#adminauth-node-red-editor).

### `httpNodeAuth`

HTTP basic auth protecting HTTP-injected endpoints (Node-RED nodes
serving HTTP). **Secret**. **Restart required**. See
[`security/auth-surfaces.md § httpNodeAuth + httpStaticAuth`](../security/auth-surfaces.md#httpnodeauth--httpstaticauth-http-basic-auth).

### `httpStaticAuth`

HTTP basic auth protecting static file serving. **Secret**.
**Restart required**. See [`security/auth-surfaces.md`](../security/auth-surfaces.md).

### `uiPort`

Port Node-RED listens on. Accepts a literal number (`1880`) or a
JS expression evaluated at startup
(`process.env.PORT || 1880`). Restart required.

### `uiHost`

Interface Node-RED binds to. Default `0.0.0.0` (all interfaces).
Restart required.

### `httpAdminRoot`

Mount path for the editor (`/` by default). Setting to `false`
disables the editor entirely (paired with `disableEditor: true`).
Restart required.

### `httpNodeRoot`

Mount path for HTTP-injected endpoints (`/` by default). `false`
disables them. Restart required.

### `https`

TLS listener block (key/cert/ca/port/passphrase options). When set,
Node-RED serves HTTPS instead of HTTP. **Secret**: cert/key paths
leak server identity, and the private key must never appear in
plaintext logs. **Restart required**. **Verification**:
`openssl s_client -connect host:port` returns the configured
certificate. See [`../security/auth-surfaces.md`](../security/auth-surfaces.md)
and the TLS playbook in [`../operator/playbook.md`](../operator/playbook.md).

### `requireHttps`

When `true` Node-RED redirects `http://` requests to `https://`.
Restart required. **Verification**: an editor URL without `https`
redirects to `https://<same path>`.

### `httpStatic`

Static file serving config (path or array of
`{path, root}` sources). Restart required. **Not UI-editable**:
operators reach it through the advanced escape hatch because the
shape is too variable for a typed form field.

### `lang`

Default UI language. Restart required. ISO locale code (`en-US`,
`es-ES`, …).

### `runtimeState`

Node-RED `runtimeState` block (`enabled`, `ui`). Restart required.

### `logging`

Node-RED `logging` block. Restart required. Defaults to
`{console:{level:info}}`.

### `disableEditor`

When `true` Node-RED refuses to mount the editor even with a
working `httpAdminRoot`. Restart required. **Verification**:
`GET http://host:1880/` returns `404 Not Found`.

### `editorTheme`

Node-RED `editorTheme` block (projects, code completion,
monacoOptions, …). Restart required. The structured UI exposes the
common keys only; everything else goes through the advanced escape
hatch.

### `functionGlobalContext`

Initial values for `global.get()`/`global.set()` keys. Object.
**Secret**: keys often carry credentials used by flow nodes.
**Restart required**. **Not UI-editable**: reached through the
advanced escape hatch because the value shape is fully operator-
defined.

## Verification

The catalog is verified by
`internal/service/catalog_doc_sync_test.go` (W2). The test parses
the table above and asserts:

- Exactly 21 rows.
- The `Key` column matches `ManagedSettingKeys()` exactly (set
  equality, not order).
- For every catalog entry, the row's `Shape` / `Default` /
  `Validation` / `Secret` / `RestartRequired` / `UIEditable`
  columns match the Go struct fields byte-for-byte.
- The `Catalog version` cell of the table header matches
  `nodeRED5CatalogVersion`.

A drift in any column fails `go test ./internal/service/`.

## Adding a new entry

1. Add the `SettingCatalogEntry` to `nodeRED5Catalog` in
   `internal/service/nodered_compatibility.go`.
2. Add the corresponding row to the catalog table above in the
   same commit.
3. Add the entry to `managedSettingKeys` in
   `internal/service/source_patch.go` if the parser/renderer must
   rewrite it.
4. Add a frontend form field under
   `frontend/src/features/configuration/components/FormFields.tsx`
   if `UIEditable: true`.
5. Document the entry above (Per-entry notes block).
6. Run `go test ./internal/service/ -run TestCatalogDocSync` and
   `go test ./internal/service/ -run TestManagedSettingKeys` and
   the frontend test suite.

A future slice may replace this process with code generation
(decision deferred to W2 mid-way checkpoint per the plan); for now
the manual sync test is the contract.

## Related

- [`docs/control-plane.md`](../../control-plane.md) — umbrella
  traceability for #756 (this catalog is one of its closing
  artefacts).
- [`apply-pipeline.md`](apply-pipeline.md) — what happens when a
  catalog entry changes.
- [`../glossary.md`](../glossary.md) — definitions for every term
  used here.
- [`../security/auth-surfaces.md`](../security/auth-surfaces.md) —
  per-surface authentication breakdown.
