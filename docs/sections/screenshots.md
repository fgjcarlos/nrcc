<!--
  Screenshots — honest placeholder policy.
  Inlined into docs/index.html as the three screenshot placeholders
  (Overview / Configuration / Security).
-->

Real screenshots of the Overview, Configuration, and Security views
will replace these placeholders once the redesigned UI (issue #766)
stabilises. Until then, the placeholders carry a
`data-freshness-stamp` attribute; the publication checker fails CI
when a placeholder outlives its named release.

- **Capture process.** The captures are rendered with Playwright
  headless against the running frontend. The script lives at
  `scripts/capture-pages-screenshots.mjs` (added in a follow-up
  slice).
- **Replacement policy.** Each placeholder is replaced after the
  slice H release of the redesigned UI ships. The publication
  checker compares the stamp against the latest released tag.
- **Why not fake them.** Stale screenshots attract the wrong users
  and create expectations NRCC cannot safely meet. The
  [issue #770 brief](https://github.com/fgjcarlos/nrcc/issues/770)
  is explicit on this point.
