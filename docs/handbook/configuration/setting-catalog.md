# Setting catalog

> **Stub.** Landed as a skeleton in W1. The full catalog entries
> arrive in W2 alongside `internal/service/catalog_doc_sync_test.go`.
> Until then the table is empty.

## How to read this catalog

Each row in the catalog table below is a `SettingCatalogEntry`
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

The catalog version is pinned by
`nodeRED5CatalogVersion = "5.0.6"`. Bumping the version is a
deliberate operator-visible decision that must include a sync
update and (if shapes change) an ADR.

## Catalog

| Key | Shape | Default | Validation | Secret | RestartRequired | UIEditable |
|-----|-------|---------|------------|--------|-----------------|------------|
| _to be filled in W2_ | | | | | | |

## Verification

The catalog is verified by
`internal/service/catalog_doc_sync_test.go` (W2). The test parses
the table above and asserts every `nodeRED5Catalog` entry maps to
exactly one row whose columns match. A drift in any column fails
CI.

## Related

- [`docs/control-plane.md`](../../control-plane.md) — umbrella
  traceability for #756 (this catalog is one of its closing
  artefacts).
- [`configuration/apply-pipeline.md`](apply-pipeline.md) — what
  happens when a setting changes.
- [`glossary.md`](../glossary.md) — definitions for every term used
  here.
