# Forbidden tokens (handbook enforcement)

This file is the single source of truth for tokens that the
freshness gate from #770 forbids inside `docs/`. It lives outside
`docs/` so the gate does not scan it; the handbook
(`docs/handbook/`) and the public site (`docs/index.html` + its
sections) are scanned and must not contain any token listed below.

The scan itself is a one-liner per pattern that runs inside
`.github/workflows/pages-freshness.yml`:

```bash
grep -RInE --include='*.html' --include='*.md' "$pattern" docs/
```

A non-zero exit for any pattern fails the gate. Fix the token in
place; do not silence the gate.

## Why this lives here and not in the handbook

The handbook is itself a `docs/**/*.md` file, so anything written
inside `docs/handbook/` is subject to the scan. Listing the literal
tokens there would make the handbook a permanent source of gate
failures. Keeping the enumeration in `odd/handbook/forbidden-tokens.md`
keeps the policy auditable while the prose that *describes* the
policy can stay compliant.

## Enumeration

The freshness gate currently forbids the following patterns:

| # | Pattern | Category | Why forbidden |
|---|---------|----------|---------------|
| 1 | `sp-(bg|text|border|brand|accent)-` | Legacy palette tokens | Signal Prime leftovers. The NRCC design system is `ds-*`; `sp-*` was the previous-generation name and never shipped in this repo's design tokens. |
| 2 | `Beta ·` | Stale beta-era copy | NRCC ships a 5.x catalog. The "beta ·" framing was an early-2025 placeholder; the era it described is closed. |
| 3 | `hardening phase` | Stale beta-era copy | Same era as #2. The hardening phase is finished; production-grade runs every PR. |
| 4 | `no production guarantees` | Stale beta-era copy | Same era as #2. The handbook pages and the public site state the policy accurately; this phrase is misleading. |
| 5 | `API keys may change` | Stale beta-era copy | Same era as #2. The 5.x API surface is frozen; keys do not change without a SemVer bump. |
| 6 | `Vitest suites on every PR` | Incorrect promise copy | The catalog policy is one Vitest run per PR, gated on the affected scope. "Suites on every PR" is a broader promise than the actual gate. |

## How to add or remove a forbidden token

To **add** a token: edit `.github/workflows/pages-freshness.yml`
under the `forbidden-token scan` step (the `patterns=(...)` array),
then add a row to the table above. Both edits go in the same commit;
the gate is the source of truth, the table is documentation.

To **remove** a token: edit the same workflow file, remove the row
from the table above, and verify the gate no longer references the
pattern. Do not orphan a row in the table without removing the gate
entry — that confuses future contributors.

## Cross-links

- The handbook's `index.md` and `contributing/style.md` reference
  this file from inside `docs/handbook/`. Both links use a relative
  path (`../../../odd/handbook/forbidden-tokens.md`) so the link
  resolves both in GitHub's web view and when the handbook is
  rendered by the Pages workflow.
- The freshness gate lives at
  `.github/workflows/pages-freshness.yml`. The `forbidden-token
  scan` step is a single block; its `patterns` array is the contract
  this file documents.