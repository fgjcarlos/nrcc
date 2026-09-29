# Issue #771 — NRCC Node-RED 5 Configuration & Security Handbook

## Objective

Publish an English technical handbook (the missing documentation
backbone NRCC never had) explaining how visual controls map to
Node-RED 5 settings, how configuration sources are resolved, which
changes require restart, and how the multiple authentication
boundaries differ. After this lands, contributors stop
reverse-engineering `settings.js` and implementers stop introducing
parallel contracts across frontend / API / parser / renderer.

Closes #771. Documentation companion to umbrella #765.

## Problem statement (issue #771 body)

NRCC has no single maintained documentation structure that explains:

1. The canonical setting catalog (shape, default, validation, version
   gate, configured/effective/source semantics, restart behavior,
   unsupported/raw escape cases).
2. Source-preserving import/edit behavior, validation, preview/diff,
   atomic write, backup, readiness, rollback, recovery.
3. The four authentication surfaces (NRCC users/RBAC/MFA,
   `adminAuth`, `httpNodeAuth`, `httpStaticAuth`, FlowFuse Dashboard
   HTTP + Socket.IO).
4. Instance discovery, environment precedence, secrets, certificates,
   context storage, logging, external modules, operational safety.
5. Architecture and contributor guides (schema, API generation,
   frontend form generation, parser/renderer contracts, testing
   strategy, fixtures).
6. Task-oriented operator guides, troubleshooting, glossary, ADR
   links, versioned support matrix.
7. Documentation ownership rules + link/code-example/release-review
   so the handbook stays synchronised with the code.

Today the closest things are `docs/configuration/env-contract.md`,
`docs/architecture/multi-instance-node-red.md`, `docs/operations/`
guides, and `docs/adr/`. They exist in pieces but do not cover items
1, 2, 3, 5, 6 or 7.

## Architectural choices

- **New top-level directory `docs/handbook/`** so the existing
  `docs/configuration/`, `docs/operations/`, `docs/architecture/`,
  `docs/adr/` stay where they are (operators and contributors already
  link them from README and issue body templates). The handbook is
  the *index*, not a replacement.
- **Setting catalog sync is test-driven** (`internal/service/catalog_doc_sync_test.go`).
  The `[]SettingCatalogEntry{}` in `internal/service/nodered_compatibility.go`
  remains the single source of truth. `docs/handbook/configuration/setting-catalog.md`
  is a hand-written mirror; a Go test parses it and fails CI when
  the keys, shapes, defaults, validations, restart/secret flags or
  UIEditable status drift. The user picked "markdown manual with
  sync test" over auto-generation because every catalog entry has
  prose that an LLM agent can keep readable; auto-generation
  produces tables with empty doc columns.
- **No new design tokens**. The handbook is a `docs/` artefact and
  reuses the same Markdown convention as the rest of `docs/`.
- **Link integrity** is checked by the existing
  `scripts/check-pages-links.mjs` (landed in #770). The handbook
  routes any new cross-link through that checker. The freshness
  workflow from #770 (`pages-freshness.yml`) will run on every
  handbook change because of the `docs/**` path filter.
- **No screenshots**. The UI slice G from #766 just landed; the
  public site freshness stamp policy from #770 already says we use
  placeholders + named release markers (currently `slice-h`) until
  the UI stabilises. The handbook follows the same policy.
- **English source files only** per #768. No Spanish copy.
- **Conventional Commits**, one work-unit per `W` step, ≤400 authored
  lines per commit. Glossary + ADRs + handbook pages naturally
  produce larger files (the ADR template is ~50 lines, the glossary
  grows incrementally), so some commits will be documentation-heavy.

## Slice plan (W1–W6)

| W | Title | Scope | Commit target |
|---|-------|-------|----------------|
| W1 | Handbook shell + index + glossary skeleton | New directory `docs/handbook/`, `index.md`, `glossary.md`, navigation cross-links, MkDocs-light pattern (no build, just `index.md` links). | `<450` lines `index.md`; cross-link check passes. |
| W2 | Setting catalog + sync test | `docs/handbook/configuration/setting-catalog.md` + `internal/service/catalog_doc_sync_test.go`. | 21-entry sync passes (Key/Shape/Default/Validation/Secret/RestartRequired/UIEditable). |
| W3 | Authentication surfaces separation | `docs/handbook/security/auth-surfaces.md`. Four sections: NRCC access, `adminAuth`, `httpNodeAuth` + `httpStaticAuth`, Dashboard. | Each surface has its own `Boundaries`, `Default`, `Restart impact`, `Common pitfalls` blocks. |
| W4 | Apply pipeline (source-preserving) | `docs/handbook/configuration/apply-pipeline.md` describing import → edit → preview/diff → validation → atomic write → backup → readiness → rollback. | Flow diagrams + every step cross-referenced to its code path. |
| W5 | Architecture + contributor guide + fixtures | `docs/handbook/architecture/system.md`, `docs/handbook/contributing/parser-renderer-contract.md`, `docs/handbook/contributing/testing-fixtures.md`, `docs/handbook/contributing/style.md`. | Tests reference fixtures; style guide cites existing repos. |
| W6 | Operator playbook + troubleshooting + support matrix + ownership | `docs/handbook/operator/playbook.md`, `troubleshooting.md`, `support-matrix.md`, `docs/handbook/governance/ownership.md`. Final cross-link + freshness gate run. | Every playbook entry ends with a "Verify" block. |

Final acceptance:

- All 6 W-units land.
- `pages-freshness.yml` green (forbidden tokens + stamp + link
  integrity + 3 test files).
- `go test ./internal/service/ -run TestCatalogDocSync` green.
- New `docs/handbook/index.md` links resolve.
- New `go run scripts/handbook-lint` (optional) reports zero broken
  anchors.

Single PR from `docs/issue-771-handbook` → `main`. Closes #771.

| W | Title | Status |
|---|-------|--------|
| W1 | Handbook shell + index + glossary skeleton | ⏳ |
| W2 | Setting catalog + sync test | ⏳ |
| W3 | Authentication surfaces separation | ⏳ |
| W4 | Apply pipeline (source-preserving) | ⏳ |
| W5 | Architecture + contributor guide + fixtures | ⏳ |
| W6 | Operator playbook + troubleshooting + support matrix + ownership | ⏳ |

## What this issue does NOT do

- Does NOT rewrite `docs/index.html` (the public site). That is #770.
- Does NOT write i18n catalogs. That is #767.
- Does NOT ship a static site generator. The handbook is plain
  Markdown served by the existing Pages workflow.
- Does NOT add a new `go doc` extraction tool. The setting catalog
  stays a single Go struct slice; the doc is hand-written.
- Does NOT retire the existing `docs/configuration/`,
  `docs/architecture/`, `docs/operations/`, `docs/adr/` pages. The
  handbook *links* to them.
- Does NOT add screenshots. Placeholders + freshness stamp `slice-h`
  per #770 policy.

## Stop conditions

- If `docs/handbook/index.md` exceeds 400 lines after W6, split into
  `handbook/{getting-started,operations,security}.md` sections.
- If `catalog_doc_sync_test.go` reports drift more than once per
  slice, fall back to W2 strategy of generating the catalog MD from
  Go via `go run ./cmd/catalog-doc` (decision deferred to W2 mid-way
  checkpoint).
- If `pages-freshness.yml` fails because the new handbook pages
  introduce a forbidden token by accident, fix the token, do not
  weaken the gate.

## Acceptance evidence (per requirement)

| # | Issue body requirement | Evidence |
|---|------------------------|----------|
| 1 | Setting catalog (shape, default, version gate, restart, escape hatches) | `docs/handbook/configuration/setting-catalog.md` + `catalog_doc_sync_test.go` (W2) |
| 2 | Source-preserving import/edit/validation/preview/diff/atomic/backup/readiness/rollback | `docs/handbook/configuration/apply-pipeline.md` (W4) |
| 3 | Four authentication surfaces separated | `docs/handbook/security/auth-surfaces.md` (W3) |
| 4 | Instance discovery, env precedence, secrets, certs, context storage, logging, modules, safety | split across `architecture/system.md`, `configuration/setting-catalog.md`, `operator/playbook.md`, `support-matrix.md` (W5/W6) |
| 5 | Architecture + contributor guides (schema, API gen, form gen, parser/renderer, testing, fixtures) | `architecture/system.md` + `contributing/{parser-renderer-contract,testing-fixtures,style}.md` (W5) |
| 6 | Task-oriented operator guides, troubleshooting, glossary, ADRs, support matrix | `operator/{playbook,troubleshooting}.md` + `glossary.md` + `support-matrix.md` + cross-links to `adr/` (W1/W6) |
| 7 | Documentation ownership, link check, code-example test, release review | `governance/ownership.md` (W6) — reuses `scripts/check-pages-links.mjs` + `pages-freshness.yml` |

## Pre-PR checks

- `go test ./internal/service/ -run TestCatalogDocSync` PASS
- `node scripts/check-pages-links.mjs` PASS
- `node docs/sections/freshness-stamp.test.mjs` PASS (handbook
  `slice-h` markers do not regress)
- `pages-freshness.yml` (in `gh pr checks`) green
- `wc -l docs/handbook/index.md` ≤ 400
