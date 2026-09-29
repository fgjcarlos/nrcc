# Issue 770 — Redesign GitHub Pages for the NRCC control plane

## Objective

Replace `docs/index.html` (currently a "Signal Prime beta" landing page
from #388) with a focused, evidence-based product and documentation
entry point that:

1. **Reflects the approved control-plane product direction** — graphite/cyan
   design system (Signal Prime tokens already in `frontend/tailwind.config.js`),
   NRCC mark, responsive layout, WCAG-AA contrast.
2. **Explains the operator problem and the Node-RED 5.x scope** — single
   supported editing target; Node-RED 4 read-only with migration guidance;
   future majors read-only.
3. **Presents primary workflows** — configuration, security, recovery — and
   explicit non-goals.
4. **Documents the compatibility matrix, Docker-first quick start, security
   model, roadmap status, limitations, and links to detailed documentation.**
5. **Removes stale claims** — "beta · hardening phase", "no production
   guarantees", "API keys may change between minor versions", ad-hoc test
   counts, ad-hoc install snippets.
6. **Defines a publication/check workflow** — releases cannot silently leave
   the site behind; the publication checker fails CI when placeholders or
   stale markers persist.

## Scope (per user decision 2026-09-29)

In scope:

- `docs/index.html` rewrite. Static, single-file, Tailwind CDN (no build),
  English only (per #768), keyboard accessible, mobile friendly, independent
  of application authentication.
- `docs/assets/nrcc-mark.svg` — NRCC mark consistent with the favicon style
  and the Signal Prime cyan accent. Single accent `#0089b4`, graphite frame.
- `docs/assets/favicon.svg` — favicon (currently lives in
  `frontend/public/favicon.svg`; this version uses the NRCC mark so the
  Pages site carries its own brand at the browser tab level).
- `docs/sections/` — markdown section sources (Overview, Configuration,
  Security, Recovery, Roadmap, Compatibility, Limitations, Quick start) so
  the rendered HTML is one inline concatenation and the sources are
  diff-friendly. (Same pattern as the existing `docs/index.html` mirrors
  README/CONTRIBUTING content.)
- `.github/workflows/pages-freshness.yml` — a CI job that fails when
  `data-freshness-stamp` placeholders or the documented beta/no-production
  markers persist in `docs/index.html`. Runs on PRs that touch `docs/**`
  and weekly on `main` via `schedule`.
- Restore the GitHub Pages site at `https://fgjcarlos.github.io/nrcc/`.

Out of scope:

- Real screenshots of Overview / Configuration / Security. The issue body
  asks for them, but the slice G UI only just landed (#855) and is still
  volatile; capturing them now would publish a snapshot that drifts within
  one slice. Per the user decision 2026-09-29 the site ships with
  **placeholders + a freshness checker** that fails CI when the
  placeholders survive past a named release. The placeholder paragraph is
  honest and links to `docs/sections/screenshots.md` which captures the
  capture process.
- Refactor of the `docs/` handbook structure. That is #771.
- i18n catalogs on the public site. That is #767.
- New design tokens. Signal Prime is already canonical in
  `frontend/tailwind.config.js`; the site mirrors the same palette.
- Astro/Vite static build. The single-file CDN approach has served the
  project since #388 and the user wants to keep the surface minimal.

## Method

Single PR sliced into 4 work-units (W-prefixed per repo convention):

- **W1 — Design system mirror + NRCC mark**
  - `docs/assets/nrcc-mark.svg` — 64×64 vector, cyan square mark.
  - `docs/assets/favicon.svg` — same mark at favicon dimensions.
  - `docs/index.html` token block updated from `sp-*` to `ds-*` (mirrors
    `frontend/tailwind.config.js` §"Brand (Signal Prime Red/Cyan)").
  - 1 Vitest unit test for the SVG presence and validity
    (`docs/assets/nrcc-mark.test.mjs`).
  - 1 axe-core a11y test on the live HTML (via Playwright, same harness
    as slice G `e2e/a11y.spec.ts`). Headless playwright is the right tool
    for static HTML; vitest jsdom is not.

- **W2 — Content rewrite (Overview, Configuration, Security, Recovery)**
  - `docs/sections/overview.md`, `configuration.md`, `security.md`,
    `recovery.md`. Each section is a short paragraph + bullet list.
  - `docs/index.html` body inline-renders the four sections in that order.
  - Removes "Beta · hardening phase", "Beta · no production guarantees",
    "API keys may change", "Vitest suites on every PR" stale claims.
  - Adds compatibility matrix (Node-RED 5.x = full editing; 4 = read-only;
    future majors = read-only).
  - 1 Vitest unit test for the section loader (`docs/sections/index.test.mjs`).
  - 1 link-check script for every external link in the rewritten index
    (`scripts/check-pages-links.mjs`, mirrors `scripts/cite_check/main.go`).

- **W3 — Quick start, Roadmap, Limitations**
  - `docs/sections/quick-start.md`, `roadmap.md`, `limitations.md`.
  - Docker-first quick start replaces the three-command block with a
    single `docker compose up -d` paragraph (the older block was
    misaligned with the actual compose file).
  - Roadmap section is a one-row table per cluster with status (🟢/🟡)
    and link to `docs/control-plane.md`.
  - Limitations section enumerates: not a flow editor, not a cluster
    orchestrator, Node-RED 4 read-only, no multi-instance control plane.
  - 1 Vitest unit test asserts the roadmap table renders exactly 9 rows
    with status colors.

- **W4 — Freshness checker CI + placeholders**
  - `docs/sections/screenshots.md` — honest paragraph that documents the
    capture process and the named release when placeholders will be
    replaced.
  - Inline placeholder blocks in `docs/index.html` for Overview,
    Configuration, and Security screenshots. Each placeholder carries a
    `data-freshness-stamp="<release-name>"` attribute.
  - `.github/workflows/pages-freshness.yml` — CI that:
    1. Runs on PRs touching `docs/**`.
    2. Runs weekly on `main` via `cron: "0 6 * * 1"`.
    3. Reads `docs/index.html`, fails when any `data-freshness-stamp`
       placeholder exists and the corresponding release has shipped
       (checked via `gh release list --limit 1` against the stamp).
    4. Fails when forbidden tokens (`Beta ·`, `hardening phase`,
       `no production guarantees`, `API keys may change`,
       `Vitest suites on every PR`) are present.
  - 1 Vitest unit test that asserts the freshness stamp regex extracts
    exactly the placeholder blocks.

## Acceptance criteria

- Site renders end-to-end without console errors at
  `https://fgjcarlos.github.io/nrcc/` (post-deploy).
- `docs/index.html` ≤ 450 lines (slice G readme budget).
- All colors come from `ds-*` tokens (no ad-hoc hex literals outside the
  token block at the top of the file).
- WCAG AA contrast for every text/background pair, verified by axe-core
  in the slice G e2e harness.
- All external links resolve (verified by `scripts/check-pages-links.mjs`).
- All claim tokens removed: grep for `Beta ·`, `hardening phase`,
  `no production guarantees`, `API keys may change`,
  `Vitest suites on every PR` returns zero matches in
  `docs/index.html`.
- Roadmap table renders exactly 9 rows with correct status colors
  (6 🟢 / 3 🟡, matching `docs/control-plane.md`).
- Compatibility matrix covers Node-RED 5.x / 4 / future majors.
- Freshness checker workflow triggers on PR and weekly; passes when
  no forbidden tokens are present.
- All four work-units land as separate commits (Conventional Commits).
- CI green on PR open.

## Constraints

- English only (per #768 policy, see CONTRIBUTING.md).
- Single-file `docs/index.html` with Tailwind CDN (no build).
- All new content lives in `docs/sections/*.md` so the rendered HTML
  is one inline concatenation.
- No new design tokens; reuse `frontend/tailwind.config.js` `ds-*` palette.
- No new dev dependencies for the docs site; reuse Vitest + Playwright
  already in the frontend stack (node-only script for link checks).
- 1 placeholder per screenshot area, never a fake screenshot.
- Roadmap status must match `docs/control-plane.md` exactly — no
  hand-written "✅ done" labels.

## Tracker (status)

Single PR from `docs/issue-770-gh-pages-redesign` -> `main`. Closes #770.

| W | Title | Status |
|---|-------|--------|
| W1 | Design system mirror + NRCC mark | ⏳ |
| W2 | Content rewrite (Overview / Configuration / Security / Recovery) | ⏳ |
| W3 | Quick start / Roadmap / Limitations | ⏳ |
| W4 | Freshness checker CI + placeholders | ⏳ |

## What this issue does NOT do

- Does not introduce real screenshots. Placeholders + freshness checker
  per the user decision 2026-09-29.
- Does not restructure `docs/`. The handbook restructure is #771.
- Does not write i18n catalogs on the public site. That is #767.
- Does not add a CMS or static site generator. The single-file approach
  has served the project since #388.
- Does not change the deploy workflow (`.github/workflows/pages.yml`).
  Only adds the freshness checker (`.github/workflows/pages-freshness.yml`).

## Stop conditions

- If `docs/index.html` exceeds 450 lines after W3, compress the
  capabilities bullets into a one-paragraph summary + table.
- If the freshness checker fails to parse a release tag from `gh`, log
  a warning instead of failing the workflow (we do not want CI to be
  gated on GitHub API rate limits).
