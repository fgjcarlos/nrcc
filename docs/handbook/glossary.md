# Glossary

> Terms used across the handbook. Every acronym introduced in any
> handbook page must appear here in the same commit. Owned by
> [`governance/ownership.md`](governance/ownership.md) ("Glossary").

## A

### adminAuth

Node-RED setting (top-level key in `settings.js`). When present,
Node-RED requires a username and password to access the editor
(`httpAdminRoot`). See
[`security/auth-surfaces.md § adminAuth`](security/auth-surfaces.md#adminauth-node-red-editor).
Handled in NRCC as the `adminAuthBoundary` surface.

### ADR

Architectural Decision Record. Accepted records live in
[`docs/adr/`](../adr/). The handbook cites ADRs by number whenever
it explains a "why".

### Apply (transactional apply)

The NRCC pipeline that takes a proposed `settings.js` edit through
preview → validation → atomic write → backup → readiness check →
rollback on failure. See
[`configuration/apply-pipeline.md`](configuration/apply-pipeline.md).

## B

### Bootstrap variable

An environment variable read once during NRCC startup
(`main.go`/`runServer()`). Bootstrap variables include `PORT`,
`DATA_DIR`, `JWT_SECRET`, `NRCC_ENCRYPTION_KEY`, `NRCC_*`. They
cannot be changed without restarting the container. See
[`docs/configuration/env-contract.md § Bootstrap variables`](../configuration/env-contract.md).

### Boundary (authentication boundary)

A user-facing authentication surface that NRCC manages separately.
There are four boundaries: NRCC access, `adminAuth`, the combined
`httpNodeAuth`+`httpStaticAuth` boundary, and the Dashboard
boundary. See [`security/auth-surfaces.md`](security/auth-surfaces.md).

## C

### Catalog (setting catalog)

The list of Node-RED 5 settings NRCC can present through its
structured configuration experience. Single source of truth lives
in `internal/service/nodered_compatibility.go` (`nodeRED5Catalog`,
21 entries, `nodeRED5CatalogVersion = "5.0.6"`). Mirror in
[`configuration/setting-catalog.md`](configuration/setting-catalog.md);
sync enforced by `internal/service/catalog_doc_sync_test.go`.

### Compatibility policy

NRCC's behaviour per Node-RED major version: 5.x is editable when
`settings.js` is writable, 4.x is read-only with migration
guidance, 6.x and beyond is read-only until a dedicated adapter is
shipped. See
[`operator/support-matrix.md`](operator/support-matrix.md).

### Configured value

The value the operator has chosen through the visual configuration
flow. Distinguished from `effective value` (what Node-RED is
actually using) and `source value` (the literal text in
`settings.js`).

## D

### Dashboard boundary

The FlowFuse Dashboard HTTP + Socket.IO protection. Handles
operator-facing flows embedded in a Node-RED instance. See
[`security/auth-surfaces.md § Dashboard`](security/auth-surfaces.md#dashboard-boundary).

### Diff (preview / diff)

The redacted preview NRCC shows before applying a change. Secrets
(`adminAuth`, `httpNodeAuth`, `httpStaticAuth`, `https`, etc.) are
replaced with placeholders so the preview can be screen-shared.

## E

### Effective value

The value Node-RED is actually using at runtime. May differ from
the configured value when an environment variable overrides
`settings.js`. Distinct from `source value` and `configured value`.

### Escape hatch (advanced escape hatch)

A way to edit `settings.js` outside the typed catalog while
preserving the rest of the file. Issue #764. See
[`configuration/setting-catalog.md`](configuration/setting-catalog.md)
("advanced" entries) and [`configuration/apply-pipeline.md`](configuration/apply-pipeline.md).

## F

### Fixture (Node-RED settings fixture)

A representative `settings.js` stored under
`internal/service/testdata/` or
`frontend/test/integration/fixtures/` that exercises a slice of the
parser/renderer/apply pipeline. See
[`contributing/testing-fixtures.md`](contributing/testing-fixtures.md).

### Form generation (frontend form generation)

The pipeline that turns a `SettingCatalogEntry` into a typed React
form field. Lives in
`frontend/src/features/configuration/components/FormFields.tsx`.
Documented in
[`contributing/parser-renderer-contract.md`](contributing/parser-renderer-contract.md).

## H

### httpNodeAuth

Node-RED setting that protects HTTP-injected endpoints (nodes
serving HTTP). See
[`security/auth-surfaces.md § httpNodeAuth + httpStaticAuth`](security/auth-surfaces.md#httpnodeauth--httpstaticauth-http-basic-auth).

### httpStaticAuth

Node-RED setting that protects static file serving (`httpStatic`).
Often set together with `httpNodeAuth`. See
[`security/auth-surfaces.md`](security/auth-surfaces.md).

## M

### Managed setting

A `settings.js` top-level key NRCC rewrites through its structured
configuration UI. Listed in `managedSettingKeys`
(`internal/service/source_patch.go`). Anything not on that list is
unmanaged and must round-trip verbatim through every edit.

## N

### NRCC access

The operator-facing authentication surface NRCC itself manages
(JWT cookie, RBAC, MFA). Distinct from `adminAuth` (Node-RED
editor). See [`security/auth-surfaces.md § NRCC access`](security/auth-surfaces.md#nrcc-access).

## P

### Parser

The NRCC component that reads `settings.js` into a typed
representation. The parser must round-trip unmanaged regions
verbatim. See
[`contributing/parser-renderer-contract.md`](contributing/parser-renderer-contract.md).

### Preview (redacted)

See `Diff`.

## R

### RBAC

Role-based access control. NRCC access boundary enforces roles
(read-only, operator, admin). Documented in
[`security/auth-surfaces.md`](security/auth-surfaces.md).

### Renderer

The NRCC component that writes a typed representation back to
`settings.js`, preserving unmanaged regions. See
[`contributing/parser-renderer-contract.md`](contributing/parser-renderer-contract.md).

### Restart-required

A flag on a `SettingCatalogEntry` indicating that changing this
setting requires restarting Node-RED. Currently every managed
setting is restart-required. See
[`configuration/setting-catalog.md`](configuration/setting-catalog.md).

### Rollback

The recovery step of the apply pipeline when readiness fails
post-write. The pre-apply backup is restored and Node-RED is
restarted.

## S

### Secret (catalog flag)

A flag on a `SettingCatalogEntry` indicating that the value is
redacted in previews and never logged in plaintext. Secrets include
`credentialSecret`, `adminAuth`, `httpNodeAuth`, `httpStaticAuth`,
`https`, `functionGlobalContext`, and (when supplied inline)
`NRCC_AI_API_KEY`. See
[`configuration/setting-catalog.md`](configuration/setting-catalog.md).

### Source value

The literal text appearing in `settings.js`. May differ from the
configured value (operator can hand-edit the file) and from the
effective value (an env override).

### Source-preserving

A property of the parser/renderer contract: every byte of
`settings.js` that is not part of a managed entry must round-trip
unchanged. Tested by
`internal/service/source_patch_test.go`.

### Stack (Compose stack)

One Compose service containing one NRCC binary supervising one
Node-RED process. Multiple Node-REDs ⇒ multiple stacks. See
[`docs/architecture/multi-instance-node-red.md`](../architecture/multi-instance-node-red.md)
and [`governance/ownership.md`](governance/ownership.md).

### Support matrix

The versioned compatibility table published in
[`operator/support-matrix.md`](operator/support-matrix.md). Tracks
Node-RED ↔ NRCC ↔ Dashboard ↔ adminAuth strategy ↔
release-channel status.

## U

### Unmanaged setting

A `settings.js` top-level key NRCC does not edit through its
structured UI. Must round-trip verbatim. The complement of
`managed setting`.

## V

### Version gate

A per-setting rule that decides which Node-RED major versions the
catalog entry applies to. Today every entry in the 5.x catalog is
gated to `>=5.0 <6.0`. See
[`configuration/setting-catalog.md`](configuration/setting-catalog.md).
