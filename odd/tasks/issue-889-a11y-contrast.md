# Issue #889 — Correct light-theme contrast and enforce both-theme acceptance

Issue: <https://github.com/fgjcarlos/nrcc/issues/889> · Branch: `fix/issue-889-a11y-contrast` · Base: `main` (b2cfecf).
The static contrast guard is report-only, audits only base-100 backgrounds, treats small button labels as UI (3:1), and browser checks never pin a theme or a mobile viewport.

## Specs

- S1 (issue AC1): "Correct light-theme tokens and verify computed styles without regressing the corporate palette."
- S2 (issue AC2): "Make an appropriate strict contrast guard required rather than report-only."
  User decision (L2): strict by default; report-only needs an explicit `--report` flag; CI runs the default mode.
- S3 (issue AC3): "Test dark/light, representative mobile widths, focus visibility and modal keyboard behavior."
- S4 (issue AC4): "Check real small button text against the correct WCAG threshold rather than automatically treating labels as large text."
- S5 (issue AC5): "Keep appearance regression evidence distinct from accessibility evidence."
- S6 (issue expected behavior): "Maintained themes meet the appropriate contrast rules and essential controls remain usable by keyboard and on mobile."
- S7 (L2): dark-theme solid primary/error buttons: slightly darken the red background until white text reaches >= 4.5:1, keeping the corporate look.
- S8 (L2): make Tailwind `ds-*` colours theme-aware (`var(--ds-*)`) inside #889; dark-theme computed colours must stay identical.
- S10 (L4): semantic-coloured text on surfaces (`text-primary|error|success|warning|info|accent`, ~230 uses) is out of scope for #889; open a separate issue with the measured table. Badges, StatusChip and solid button labels stay in scope.
- S9 (L2): the Sidebar (768px JS) vs drawer (`lg`, 1024px) breakpoint mismatch goes to a separate issue; fix it in #889 only if the mobile test proves navigation unusable.

## Tasks

- [ ] T1 — S2, S4: auditor classifies labels as text (4.5:1), audits every real surface in both themes, strict by default with `--report`; fixture tests wired to `pnpm test:contrast`. Route: inline. Commit: —
- [ ] T2 — S1, S4, S6, S7: correct the failing tokens (muted text, solid primary/error/success, badges) in both themes; computed-style checks. Route: inline. Commit: —
- [ ] T3 — S1, S8: Tailwind `ds-*` colours resolve through CSS variables; dark theme unchanged. Route: inline. Commit: —
- [ ] T4 — S2: run the strict contrast guard and its tests in the PR (and therefore release) workflow. Route: inline. Commit: —
- [ ] T5 — S3, S6, S9: Playwright dark/light/mobile projects; axe per theme; focus visibility; modal keyboard behaviour; focus fixes. Route: inline. Commit: —
- [ ] T6 — S5: appearance checks run as a separate CI step/artifact; docs explain accessibility vs appearance evidence. Route: inline. Commit: —

## Log

- L1 (user, verbatim): "Realiza un plan para resolver la issue #889" → "Realizame preguntas con las decisiones" → "ok implementa"
- L2 (user decisions): brand red → "Oscurecer levemente el rojo (Recomendado)"; ds-* scope → "Incluir en #889 (Recomendado)"; breakpoint → "Issue aparte (Recomendado)"; auditor → "Strict por defecto + --report (Recomendado)".
- L3 (evidence, exploration): light muted #6f7d90 is 3.69–4.19 on light surfaces (worst: thead gradient ~3.57); dark muted fails on base-200/surface/elevated/overlay (3.18–4.42); dark #fff on #d4472b 4.43, on #e54233 4.07; light .btn-success 4.11; Button/StatusChip/badge labels are 12–16px (not large text); tailwind.config.js hardcodes dark ds-* hex; no contrast/appearance step in CI; e2e has no theme pinning and no mobile project.
- L4 (user decision): semantic text colours on surfaces → "Issue aparte (Recomendado)". Evidence: light warning text 2.6–3.0, light success 3.6–4.1, dark primary 3.0–3.9 on surface/overlay; dark primary text vs. button background need opposite adjustments.
