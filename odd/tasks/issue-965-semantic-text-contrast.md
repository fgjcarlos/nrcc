# Issue #965 — Semantic text colours meet AA on every surface in both themes

Issue: <https://github.com/fgjcarlos/nrcc/issues/965> · Branch: `fix/issue-965-semantic-text-contrast` · Base: `main` (4e7e082).
`text-primary|error|success|warning|info|accent` reuse the solid-fill `--color-*` tokens, which cannot satisfy both white button labels and text on dark surfaces.

## Specs

- S1 (issue AC1): "Semantic text colours meet 4.5:1 on every documented surface in both themes, without changing the solid button fills."
- S2 (issue AC2): "The strict contrast guard audits the semantic text pairs."
- S3 (issue AC3): "axe (light and dark projects) stays free of serious colour-contrast violations."
- S4 (issue expected behavior): "Semantic text is readable (≥ 4.5:1, or ≥ 3:1 only where the text is verified large) on every surface it is rendered on, in both themes."
- S5 (plan, L2, refined L3): a readable semantic text token per theme (`--ds-on-tint-X`, L3), routed centrally so `text-X` (incl. opacity/hover variants and `@apply`) uses them while `bg-`/`border-`/fills keep `--color-X`; the compiled CSS mapping is pinned by a test.
- S6 (plan, L2): utilities that bypass `text-X` (`.sidebar-nav-link-active svg`) and hover opacity variants that lower contrast (`hover:text-X/80`, `/90`) use the readable token.

## Tasks

- [x] T1 — S2: auditor audits `ds-on-tint-X` (the token behind `text-X`, L3) on every surface and on the 18% tint of X over every surface; `--ds-on-tint-accent` tint pair; fixture tests RED→GREEN. Route: inline. Commit: a69c892 (T1+T2 together; guard would fail CI alone)
- [x] T2 — S1, S4, S5: readable semantic tokens (`--ds-on-tint-*` + new accent) in both themes; central routing of `text-X` to them; compiled-CSS pin test. Route: inline. Commit: a69c892
- [x] T3 — S6: bypasses and hover opacity variants. Route: inline. Commit: b3ad7fc
- [ ] T4 — S3: full checks (contrast, unit, css/appearance, e2e dark+light with axe). Route: inline. Commit: —

## Log

- L1 (user, verbatim): "Realiza un plan para resolver la issue #965" → "Ok adelante"
- L2 (plan accepted): dedicated text tokens + central routing via Tailwind `--text-color-*` namespace (fallback `theme.extend.textColor`); icons share the token; `hover:text-accent-hover` (undefined `--color-accent-hover`) is out of scope → follow-up issue.
- L3 (decision/evidence): `--ds-text-primary` already names neutral body text, so new `--ds-text-X` tokens would collide. The #889 `--ds-on-tint-X` values already pass on plain surfaces too (min dark 5.25–6.47, light 5.64–6.19 on surfaces; 4.62–4.89 on the 18% tint), so `text-X` is routed to them via `@theme inline { --text-color-X: var(--ds-on-tint-X) }`; new `--ds-on-tint-accent` dark #5db4cf / light #0a6080. Engram mirror unavailable (runtime contract incompatible); the task file is the only tracker.
- L4 (evidence): T1 RED 11 failed assertions → GREEN; real tree exit 2 (missing accent) before T2. T2 pin in `test:css` RED (missing `text-primary on --ds-on-tint-primary`) → GREEN; built CSS: `.text-primary{color:var(--ds-on-tint-primary)}`, `hover:text-error/80` → color-mix of the on-tint token, `.bg-primary`/`.btn-primary` keep `--color-primary`. `check:contrast` 194 pairs pass; `test:contrast`, `test:tailwind-vars`, `test:css`, `test:appearance` pass; vitest 602 passed / 2 skipped; typecheck ok; lint 0 errors (1 pre-existing warning).
- L5 (evidence): T3 — `.sidebar-nav-link-active svg` uses `--ds-on-tint-primary`; UserTable text actions `hover:text-X/80` → `hover:underline`; UserMenu logout `hover:text-error/90` → `hover:text-error` (on its `bg-error/15` tint). BackupListSection icon `hover:text-error/80` kept (icon, 3:1 rule). No `text-X/NN` text left in `.tsx`.
- L6 (evidence): native RDD review `review-0af7e0475f226ac5` (base 4e7e082, committed-only, lenses readability/reliability, tier high via the UserTable auth path) approved and acknowledged (authority burned). Non-blocking follow-ups: dark accent equals info; colour list duplicated between SEMANTIC_TEXT and TINTED_LABELS; `on-tint` name now also covers plain surfaces; CSS pin does not cover modifier variants; sidebar active icon colour unpinned. Stale T1 wording fixed here.
