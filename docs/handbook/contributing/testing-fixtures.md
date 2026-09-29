# Testing fixtures

> **The corpus is the contract.** Every fixture in
> `internal/service/testdata/settings-*.js` is a regression test
> for one source-preservation rule. Adding a new fixture without
> a corresponding parser/renderer rule is wasted work; adding a
> new rule without a fixture is invisible work. They land together.

## The two directories

The fixtures live in two places:

- `internal/service/testdata/` — `.js` files NRCC parses and
  round-trips through the parser/renderer pair.
- `internal/service/testdata/golden/` — expected outputs for
  diff-style tests. Golden files are committed; the diff between
  the actual output and the golden file must be empty.

The fixtures are not configuration examples for end users. They
are regression inputs for the contract documented in
[`parser-renderer-contract.md`](parser-renderer-contract.md).

## Required fixtures

Every fixture must exist for one of these reasons:

1. **Round-trip guarantee.** A `.js` file that NRCC must
   round-trip without changes. Used by `TestParseRendererRoundtrip`.
2. **Reject contract.** A `.js` file that NRCC must refuse to
   parse. Used by `TestParserRejects`. The expected error message
   is recorded in a sibling `.err.txt` file.
3. **Render contract.** A `ParsedSettings` JSON file plus an
   expected output `.js` file. Used by `TestRendererProducesExpected`.

A fixture that fits none of these reasons is deleted in review.

## Required fixture set (today)

The corpus today covers:

| Fixture | Reason |
|---------|--------|
| `settings-minimal.js` | round-trip: smallest valid settings.js |
| `settings-full.js` | round-trip: every catalog entry at default value |
| `settings-with-comments.js` | round-trip: comments inside body |
| `settings-unicode.js` | round-trip: Unicode identifier and string values |
| `settings-nested-blocks.js` | round-trip: deeply nested object literals |
| `settings-trailing-comma.js` | round-trip: trailing comma inside array |
| `settings-multiple-exports.js` | reject: two `module.exports = …` |
| `settings-top-level-let.js` | reject: top-level `let foo = …` |
| `settings-require-fs.js` | reject: top-level `require('fs')` |
| `settings-renderer-insert-key.js` | render: append a managed key to a minimal file |
| `settings-renderer-update-value.js` | render: change one scalar in a minimal file |
| `settings-renderer-remove-key.js` | render: remove a managed key with empty default |

Adding a new catalog entry requires:

1. Adding `settings-full.js` to include the new key at its default.
2. Adding one fixture per round-trip rule that the new key
   stresses (e.g. a deeply nested block for `editorTheme`).

Removing a catalog entry requires removing every fixture that
exercises it.

## Authoring rules

A new fixture must:

1. **Have a single reason.** A file labelled
   `settings-with-comments-and-unicode.js` is two fixtures glued
   together; split it.
2. **Be the smallest input that exercises the rule.** Five lines
   is better than fifty. The contract fails on small files first.
3. **Round-trip on the current code.** A fixture that does not
   round-trip today is not a fixture; it is a TODO. Either fix
   the contract or delete the file.
4. **Use real-world syntax.** Synthetic syntax invented for the
   fixture does not test the contract end-to-end.
5. **Carry a `// fixture: <reason>` header comment.** Reviewers
   use the header to triage fixture churn during refactors.

## Golden files

A render-style fixture has a sibling `.golden.js` file. The test
that runs it produces a string and compares it byte-for-byte to
the golden file:

```
$ go test ./internal/service/ -run TestRendererProducesExpected
```

A diff in the output means the renderer broke a contract. Either
the contract changed (and the golden file is updated in the same
commit) or the renderer has a regression (and the test fails).

Golden files are never edited by hand. The update flow is:

1. `go test ./internal/service/ -update` — writes the new golden.
2. `git diff internal/service/testdata/golden/` — review the diff
   with the same scrutiny as a parser/renderer patch.
3. Commit the golden in the same commit as the rule change.

## Failure attribution

When a fixture fails:

- **Round-trip fixture fails** → either the parser dropped content
  or the renderer rewrote content. Bisect by adding `t.Log` of the
  parser output before the renderer call.
- **Reject fixture fails** → either the parser accepts an
  unexpected input (too permissive) or the renderer rejects a
  valid input (too strict).
- **Render fixture fails** → the renderer diverged from the
  golden. If the divergence is intentional, update the golden; if
  not, fix the renderer.

## Related

- [`parser-renderer-contract.md`](parser-renderer-contract.md) —
  the contract these fixtures enforce.
- [`style.md`](style.md) — author-facing conventions for new
  fixtures.
