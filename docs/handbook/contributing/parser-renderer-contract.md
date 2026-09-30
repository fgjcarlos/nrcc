# Parser / renderer contract

> **Source preservation is the contract.** The NRCC parser reads
> `settings.js` and produces a JSON document. The renderer writes
> the JSON back to a syntactically valid `settings.js` that is
> byte-for-byte equivalent to the original on the unchanged keys.
> Any round-trippable setting must round-trip exactly.

## Why this matters

The pipeline in [`../configuration/apply-pipeline.md`](../configuration/apply-pipeline.md)
is the only path NRCC uses to write `settings.js`. If the parser
loses comments, the renderer rewrites them away. If the renderer
mangles the indentation, the next hand-edited diff is unreadable.
If the parser rejects a key NRCC does not know about, the
operator cannot add their own `functionGlobalContext` keys.

This page is the contract between the parser and the renderer.
Both halves of the contract live in `internal/service/`:

- `source_patch.go` — the comment-preserving block editor.
- `nodered_compatibility.go` — the catalog (input validation,
  shape taxonomy).
- `apply.go` — the orchestrator that calls the parser and the
  renderer.

## The parser contract

The parser accepts the following inputs and produces the
following outputs.

### Inputs accepted

1. A CommonJS module whose body is a single `module.exports = { … }`
   block.
2. The top-level object literal must use standard JSON syntax
   (double quotes, no trailing commas) inside the braces.
3. Comments (`//`, `/* … */`) and whitespace are preserved
   verbatim in the source map.

### Inputs rejected

1. Top-level statements other than the `module.exports = { … }`
   assignment (e.g. an immediate function call, a `require()` call
   that mutates the export).
2. Top-level `require()` calls that NRCC cannot statically analyse
   (`require('fs')` is rejected; `require('./some-local-config')`
   is allowed as long as the required file is in `DATA_DIR`).
3. Top-level identifiers other than `module` (no `let foo = …` at
   the top level; NRCC refuses to track dynamic top-level state).
4. Multiple `module.exports = …` assignments (the file must declare
   the export exactly once).

### Output

The parser returns a `ParsedSettings` struct:

```
ParsedSettings {
  Source:        string  // original verbatim content
  Body:          string  // body of the object literal
  Keys:          []string
  ManagedKeys:   []string  // subset of Keys that NRCC recognises
  UnmanagedKeys: []string  // subset of Keys that NRCC does NOT recognise
  SourceMap:     SourceMap
}
```

The `SourceMap` records the (line, column) span of every key and
every value so the renderer can rewrite a single key without
touching the rest of the file.

## The renderer contract

The renderer accepts a `ParsedSettings` and a `[]SourceEdit` and
returns a new `settings.js` string.

### Inputs accepted

1. `SourceEdit { Key, Value }` for a single key.
2. `SourceEdit { Key, Block }` for a block-style key (object or
   array).
3. `SourceDelete { Key }` for a managed key whose value is the
   empty value (NRCC removes the key but only if it is in the
   catalog).

### Inputs rejected

1. `SourceEdit { Key }` where `Key` is not in the catalog and not
   in the `UnmanagedKeys` of the parser (NRCC refuses to invent a
   key the operator never had).
2. `SourceEdit { Key, Value }` where the value violates the
   catalog `Validation` rule (NRCC validates before rewriting).
3. `SourceDelete { Key }` for a managed key that has a non-empty
   default in the catalog (NRCC refuses to remove the key, but
   does accept an explicit reset to the default).

### Output

The renderer returns the new content as a string. The renderer
guarantees:

- Every byte of every comment in the original file is preserved
  verbatim.
- Whitespace inside the object literal is preserved verbatim.
- The indentation style of the original file is detected and
  reused.
- The `module.exports = { … }` wrapper is preserved verbatim.

## Round-trip guarantee

For every input file, parsing and then rendering without any
edits must produce a string equal to the original input on the
keys that exist in both. A regression test enforces this.

The test corpus is documented in
[`testing-fixtures.md`](testing-fixtures.md) and lives in
`internal/service/testdata/settings-*.js`.

## Source-preservation rules

The parser/renderer pair enforces the following source-preservation
rules. A violation is a defect.

1. **Comments.** `//` and `/* … */` comments inside the body are
   preserved. Comments before the body (license header, docblock)
   are preserved. The renderer never adds or removes comments.
2. **Whitespace.** Spaces, tabs, and newlines are preserved as-is.
   The renderer never reflows lines.
3. **Trailing commas.** NRCC follows Node-RED's convention of no
   trailing commas. The parser accepts trailing commas inside
   arrays but emits a warning; the renderer never emits trailing
   commas inside objects.
4. **Quotation style.** The parser accepts double-quoted strings
   only. The renderer always emits double-quoted strings.
5. **Key order.** The renderer preserves the key order from the
   original file. New keys are appended at the end of the body.
6. **Numeric formatting.** `1880` and `1.88e3` are different
   strings; the renderer emits the original formatting verbatim.
7. **Block delimiters.** `{}` and `{ … }` on one line are
   preserved as-is. The renderer never collapses a multi-line
   block into a single line.

## Failure modes

| Failure | Operator-visible error | Audit |
|---------|------------------------|-------|
| Unknown top-level statement | `parser: only module.exports = { … } is allowed` | `apply.failure` `failure_stage=validate` |
| Comment in middle of object key | `parser: unexpected token` | `apply.failure` `failure_stage=validate` |
| Managed key removed | `parser: managed key "X" is missing` | `apply.failure` `failure_stage=validate` |
| Renderer whitespace collapse | (silently breaks round-trip) | caught by round-trip test |
| Renderer comment loss | (silently breaks round-trip) | caught by round-trip test |

The round-trip test catches renderer defects that the parser does
not surface. The fixtures are deliberately adversarial: comments
in unusual places, Unicode identifiers, deeply nested blocks,
and operator-defined keys interleaved with managed keys.

## Related

- [`../configuration/setting-catalog.md`](../configuration/setting-catalog.md)
  — what the parser/renderer must recognise.
- [`../configuration/apply-pipeline.md`](../configuration/apply-pipeline.md)
  — the orchestrator.
- [`testing-fixtures.md`](testing-fixtures.md) — the corpus the
  round-trip test runs against.
- [`style.md`](style.md) — author-facing conventions for new
  pipeline code.
