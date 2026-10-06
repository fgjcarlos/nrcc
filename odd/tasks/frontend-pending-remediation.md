# Frontend security and toolchain completion

## Authorization and delivery

- User selected complete correction in separate units: moderates, lint, then
  Tailwind 4/daisyUI 5 alignment, preserving the current appearance.
- Base: `85cb3be3e83daac0b7bd39060f468cd40b52ea68` (verified H1).
- P1 branch: `fix/nrcc-frontend-moderates`, child of `fix/nrcc-security-highs`.
- Delivery: feature-branch chain; reviewed slices target their immediate parent.
- Human authorized a P3 checkpoint commit and draft PR against P2, publishing
  only the P2/P3 refs needed for that draft; the scoped budget remains unchanged.
- P3 acceptance and merge remain blocked; no waiver or security-gate changes.
- Preserve unstaged completion tracking in `odd/tasks/frontend-security-highs.md`
  and `odd/tasks/nrcc-stabilization.md`, plus untracked `.codegraph/`.
- One source writer at a time. No installs, modules, root lockfile or frontend
  dist in the worktree. Use isolated guarded snapshots and pnpm 11.12.0.
- Preserve production containers/data/secrets, Go/Docker pins and S1–S3 behavior.
- Budget: 400 total additions plus deletions per reviewed slice, including lock
  and tracking. Forecast P1 200–350, P2 80–150; P3 requires a scoped forecast
  before implementation. Split or obtain a scope decision if a coherent slice
  exceeds the agreed budget; never remove tests or compress code to fit.
- P3-only human exception: up to 900 generated lock diff lines and 450 other
  source/test/doc/tracking diff lines, 1300 total. Other slices keep 400 total.

## Verified research baseline

- Fresh npm audit, 2026-10-03 21:44:17 UTC: exit 1, one high/five moderate,
  zero low/critical. Vitest and mocker share one GHSA across two package records.
- Official npm manifests confirm PostCSS 8.5.23, fflate 0.8.3, coordinated
  Vitest/UI/mocker 4.1.11 and js-yaml 5.4.1 are published with integrity metadata.
- Braces 3.0.4 is unpublished (HTTP 404); latest 3.0.3 remains vulnerable.
  All three current audit routes originate in Tailwind 3: chokidar/braces,
  fast-glob/micromatch/braces and direct micromatch/braces.
- Tailwind 3.4.19 plus daisyUI 5.7.37 is not the official supported alignment.
  Build warnings include invalid `[object Object]` CSS; product inputs,
  textareas, toggles and loading indicators use affected class families.
  Neither harmlessness nor a complete visual regression has been demonstrated.
- Frontend modules are not copied from the builder into the final image;
  Node-RED's independent dependency graph remains unassessed by this audit.

## Tasks

### P1 — Remove all five moderate package records

- [x] Verified fixes and local work-unit commit.
- Status: done; verified and committed locally, no publication.
- Route: delegated writer (four non-trivial source surfaces); native ASSESS
  after writer, independent verifier if high/unassessable under RDD off.
- Source scope: `frontend/package.json`, `frontend/pnpm-workspace.yaml`,
  regenerated `frontend/pnpm-lock.yaml`, scoped `CHANGELOG.md` entry.
- Targets: PostCSS `GHSA-fxqj-rqcc-2cmp`, fflate `GHSA-px8p-9vwx-vf98`,
  Vitest/mocker `GHSA-82fw-gwwq-j7x9`, js-yaml `GHSA-r3ph-w7gj-g6xm`.
- Keep Vitest/UI peers coordinated; bound updates to current major families.
  js-yaml 5.2.3 to 5.4.1 is a minor change, not proven compatible by SemVer.
- Preserve H1 floors, generator 7.13.0, Redocly 1.34.15 patch registration/bytes,
  schema and unrelated dependency resolutions; no hand-edited lock/schema.
- RED: observed target IDs in the unchanged baseline audit. Explicitly regenerate
  the candidate lock, then frozen-install and compare identities/routes.
- Acceptance: five moderate records absent, zero new high/critical; braces high
  remains honestly deferred until P3. Overall audit still exits 1 at this step.
- Checks: actual installed cohort/consumer contracts, full fresh frontend suite,
  types, lint/locales/build, gen:api twice plus parser merge/anchor/custom-tag/date
  semantics. Applicable Go 1.26, canonical Docker and runtime checks independently.
  Attribute any failure to the exact base under the same runner, without weakening.
- Existing five lint and CSS warnings remain pending P2/P3, not suppressed.
- Writer `muszdw2p-1c-keb0`: five moderate records removed; remaining braces
  high, audit exit 1. PostCSS 8.5.23, fflate 0.8.3, js-yaml 5.4.1 and coordinated
  Vitest/UI/mocker 4.1.11; lock 100 additions/132 deletions, source total 245.
- Fresh writer suite: 581 passed/two skipped; types/locales/lint/build passed
  with existing warnings; gen:api twice hash-stable, actual consumer/parser
  merge/anchor/binary/omap/pairs/set/plain-date checks passed. Temporary helper
  resolution/fixture corrections preceded the final passing smoke check.
- Evidence: `/tmp/nrcc-audit-DRYCSyHs/p1-writer-dF6qSTSe/evidence/`.
  Protected tracker/patch/schema hashes unchanged; no worktree install artifacts.
- ASSESS: unassessable (untracked declaration), RDD off; independent check required.
- Independent `muszw1dl-1d-8xda`: fresh frozen suite 581 passed/two skipped;
  types/locales/lint/build/gen:api twice/consumer/parser/Go/default Docker passed.
  Audit exit 1, only braces high; hashes and protected documents unchanged.
- Evidence: `/tmp/nrcc-audit-DRYCSyHs/p1-verify-exact-R1-evidence/`.
  Original runtime probes failed on an extra one-ME assertion and an incorrect
  user response shape; these were temporary helper errors, not product regressions.
- Recheck `mut0cozv-1f-bs5y`: helper exit 0, three isolated reloads each exactly
  one refresh 200; four auth/me 200 responses each, retained user, no page errors.
  Both services HTTP 200; protected config writable/editable, runtime 5.0.7;
  updates current/latest 5.0.7, image-local, capability false, image guidance,
  no Update Now/npm notices. `runtime-recheck.json` preserves sanitized proof.
- Owned container/image/label inspected, container stopped/exited; resources and
  failed-probe evidence retained. Final source/protected hashes unchanged.
- Commit: `7df214b1a64dfa717535fbcca8adb521e34391e4`
  (`fix(security): clear frontend moderate advisories`), 374 total diff lines.
  Completion identity is unstaged bookkeeping; global merge remains blocked.

### P2 — Remove five lint warnings without changing behavior

- [x] Verified cleanup and local work-unit commit.
- Status: done; independently verified and committed locally, no publication.
- Route: delegated writer (seven existing source/test files), lint RED/GREEN;
  native ASSESS after writer and follow its independent-verification plan.
- Remove two unused disable directives and three unconsumed value exports,
  updating two barrels; preserve validation, CVA behavior and type-only exports.
- Scope: `useConfigurationSave.test.ts`, `AdminAuthBoundaryCard.tsx`,
  `HttpBasicAuthBoundaryCard.tsx`, `NrccMark.tsx`, `StatusChip.tsx` and the
  shared/components and shared/components/ui index.ts barrels (under frontend/src).
- Mapping by `muszxeu1-1e-ulxz` did not execute any checks or make changes.
  Recheck consumers; observe strict ESLint --max-warnings 0 RED then GREEN
  in a guarded isolated snapshot. Inline replacements count as two diff lines.
- Acceptance: lint zero warnings/errors; relevant tests plus full fresh suite,
  types and build pass. Preserve P1 audit outcome, parser patch and UI behavior.
- Branch: `fix/nrcc-frontend-lint`, child of verified P1 `7df214b`.
- Writer `mut10vd1-1g-6dgi`: seven source files, three additions/ten deletions.
  Strict lint RED exit 1/five warnings/zero errors, GREEN exit 0/zero warnings.
  Focused 44 passed; full 581 passed/two skipped; types/locales/build passed.
  Audit exit 1/only braces high; existing malformed CSS warnings remain for P3.
  Evidence: `/tmp/nrcc-audit-DRYCSyHs/p2-writer-iJduqRk2/evidence/logs/`.
- Parent readback: only unused exports/imports/directives removed; CVA expressions,
  component/type exports and existing test assertions unchanged.
- ASSESS: unassessable (untracked declaration), RDD off; independent check required.
- Independent `mut1eti8-1h-3two`: fresh frozen install, strict lint five-to-zero,
  44 focused/581 full passed/two skipped, types/locales 858/858/build passed.
  Audit exit 1/only braces high; source/snapshot identities and protected inputs
  unchanged, no worktree install artifacts. No source changes by the verifier.
- Evidence: `/tmp/nrcc-audit-DRYCSyHs/p2-verify-WhmEPvhz/evidence/`.
- Not rerun: Go, Docker, generator, browser/e2e and runtime. This unit changes
  only unused exports/imports/directives, not render logic, styles or dependency
  inputs; focused/full frontend checks cover the affected component contracts.
  P1 checks are not represented as fresh P2 verification; P3 requires new proof.
- Commit: `222f3f5adf0d6715dda05df02149ebf5c42cd6d6`
  (`chore(frontend): remove unused lint directives and exports`), 45 total diff lines.
  Completion identity is unstaged bookkeeping; global merge remains blocked.

### P3 — Align Tailwind 4 and daisyUI 5; close braces and CSS defects

- [ ] Verified alignment and local work-unit commit(s).
- Status: writer checked; read-only diagnosis complete; independent acceptance pending.
- Branch: `fix/nrcc-tailwind-alignment`, child of P2 `222f3f5`.
- Preparation: map config/theme/CSS and browser compatibility; measure dependency
  graph changes only in isolated snapshots, with no repository source changes.
- Official upgrade guides confirm explicit @config support, CSS daisyUI plugin
  registration, changed border defaults and custom-utility registration risks.
  Human accepted Safari 16.4+/Chrome 111+/Firefox 128+ as the P3 browser minimum.
- Graph probe `mut228nm-1k-cvvx`: Vite package +3/-2, lock +362/-486,
  853 total; PostCSS package +3/-2, lock +361/-480, 846 total. These are dependency
  lower bounds, excluding CSS/config/tests/tracking, not net line counts.
- Both isolated lock generations/frozen lock-only validations exited 0; lock hashes
  stable, audit exit 0/zero advisories. This is not an installed-build/runtime pass.
  Both routes include oxide and 12 optional packages; Vite plugin peers include Vite 8.
  DaisyUI 5.7.37/security overrides/patch records preserved; jiti 1.21.7 to 2.7.0
  changes eight direct dev-dependency peer references and needs functional checks.
- Evidence: `/tmp/nrcc-audit-DRYCSyHs/p3-graph-a8d38kAi/`; parent independently
  recounted both diffs and read audit counts. Repository source/index unchanged.
- Human replied `Autorizo` to Vite integration, up to 900 generated lock diff lines
  plus 400 other source/test/doc/tracking diff lines (1300 total), preserving
  appearance and full verification, with the documented browser minimum.
  Stop for a new decision if either sub-limit or the total is exceeded.
- Writer scope: package/lock, Vite/PostCSS/Tailwind config, index.css, targeted
  CSS regression tests and contrast-token scanner/tests, plus changelog.
  No JSX redesign, backend/settings changes or unrelated dependency updates.
- Independent installed graph/parser/suite/Go/default Docker passed; appearance
  regressed and full UI/reload verification remains incomplete. No merge clearance.
- Route: delegated preparation/writer; single-threaded source changes.
- Preserve appearance, theme switching, focus/disabled/loading states and
  responsive behavior. No redesign, settings changes or unrelated backend work.
- Candidate: published compatible Tailwind 4/daisyUI 5 integration. Do not assume
  selective manifest absence proves the complete graph has no braces.
- Acceptance: all vulnerable braces routes removed, current audit high/moderate
  records eliminated, no malformed CSS warnings; complete build and browser proof.
- Checks: new frozen graph, suite/types/locales/lint/generator/parser semantics,
  canonical final Docker; desktop/mobile affected controls and themes, three
  isolated authenticated reloads, one refresh 200 each, both services HTTP 200.
- Any remaining blocker, changed browser support or necessary budget exception
  requires a focused parent decision, not an automatic waiver or redesign.
- Writer `mutghslr-1-mgiz`: CSS diagnostic and CSS-palette RED/GREEN observed;
  fresh full suite 581 passed/two skipped, types/lint/locales 858/858/build passed,
  no malformed CSS warnings, gen:api twice schema-stable, audit exit 0/zero alerts.
  Existing light muted-text contrast drift remains report-only, not repaired/waived.
- Parent recount: lock 848, other 220 including the 46-line new regression test
  and existing tracker diff, total 1068 before these notes. Writer's 1054 count
  omitted 14 final test lines; both observed counts stay within authorized limits.
- Writer could not read the supplied historical parser helper (ENOENT); actual
  YAML semantic checks and browser/computed appearance/runtime remain pending.
- Incident diagnosis `muth5jlc-2-sbzb`: helper absent in bounded searched paths,
  not a demonstrated permission error; retain historical artifacts, do not restore.
  External `/tmp/tailwind-alignment-build-jlWi52` remains untouched (47 files).
- Cleanup writer `muthblpa-3-26vd`: lifecycle RED leaked one new directory;
  GREEN leaves none, and a real missing-Vite failure also cleans its fresh output.
  try/finally removes only this run's exact mkdtemp directory; all earlier outputs
  retained. Test is 50 lines; current slice 848 lock +241 other =1089 before these
  notes, with independent verification and final tracking still pending.
- Writer evidence: `/tmp/nrcc-audit-DRYCSyHs/p3-writer-z2MlwWff/evidence/`.
  Semantic fixtures must be reconstructed privately against actual installed modules.
- ASSESS: unassessable (untracked declaration), RDD off; independent check required.
- Independent `muthul4w-4-kvon`: fresh frozen base/candidate installs; CSS baseline
  RED/candidate GREEN, 581 passed/two skipped, types/strict lint/locales/build/gen
  twice/audit zero passed. Actual Redocly production parseYaml passed merge, binary,
  omap/pairs/set/plain-date semantics; Go 1.26/default final Docker passed.
- Blocker: Chromium computed styles lost panel/action backdrop blur in both themes;
  light panel also lost its existing translucent gradient/light shadow. Identical
  private markup/CSS and desktop/mobile screenshots establish candidate causality.
- Evidence: `/tmp/nrcc-audit-DRYCSyHs/p3-verify-h5gRx82l/evidence/`.
  Chromium 153.0.8010.12 only; Firefox/WebKit and minimum-version matrix not run.
- Runtime probes failed setup-status/helper assertions and second reload window;
  no retained per-window counts or full UI acceptance. Three newly owned containers
  stopped/exited; their image/volumes retained. No production resources touched.
- Parent recount: 848 lock +248 other =1096 before these notes; verifier's 231
  other count is inconsistent. Include every scoped file/new test and tracking.
- Diagnosis `mutix7co-5-xy1j`: setup is 201; helper two called a string method
  property; helper three failed a compound assertion before persisting rows.
  Refresh counts/user retention remain unknown, not an inferred auth regression.
  Replacement probes must persist request/response windows before each assertion.
- CSS fixture omitted danger; correction tests must cover all four action classes,
  both themes/viewports, backdrop, light surface/shadow, pointer and disabled states.
- Budget diagnosis confirmed the omitted 17 scanner-test additions; current recount
  must include them. Delegated correction remains bounded to the approved limits.
- Corrective writer stopped before writes on a forecast exceeding the non-lock cap;
  its 295 count included 25 prior-task tracking lines. Parent confirmed 270 other.
- Human approved raising only the non-lock ceiling to 450; lock ceiling 900 and
  total ceiling 1300 remain unchanged, with the same scope and all checks retained.
- Corrective writer `mutpjbm9-7-7ua6`: computed panel RED/GREEN, Chromium 153
  two themes/viewports/four actions; 581 passed/two skipped, lint/types/locales/build
  and audit zero passed. New appearance test is 98 lines; installed graph unchanged.
- Evidence: `/tmp/nrcc-audit-DRYCSyHs/p3-css-fix-fa0JLiUq/evidence/`.
  Parent scoped count: 397 other +848 lock =1245 before these notes, within caps.
- Final-candidate ASSESS remains unassessable/RDD off; separate verifier required.
- Diagnosis `mutqezn7-8-h5og` is complete: emitted CSS has standard/WebKit
  backdrop declarations, not executed Safari support proof. Retain unknown pnpm
  purge/locator provenance and the unexplained historical action-filter discrepancy.
- Publication: human approved a P3 checkpoint/draft against P2 and issue #865;
  GitHub confirmed `bug,status:approved`. Final functional acceptance stays pending.
- Next: publish the draft without claiming completion; subsequently independently
  check corrected CSS/image/full UI and all three persisted reload windows.
- Historical private audit snapshots are now unavailable; prior recorded results
  are not fresh verification. No cause or product regression is inferred.
- Branch: `fix/nrcc-tailwind-alignment`; checkpoint and final verification pending.

## Recovery and evidence

- Full Engram mirror: `odd/frontend-pending-remediation/tasks`.
- Previous isolated evidence: `/tmp/nrcc-audit-DRYCSyHs/h1-verify-exact-R1-evidence/`.
- New dependency graph requires fresh evidence; previous GREEN is not a substitute.
- Rollback each unit's exact source/tracking slice without reverting S1–S3/H1.
- Record observed command exits, failures, skips, hashes and owned stopped runtime
  resources after every task transition. Commit identities are recorded afterward
  as unstaged completion bookkeeping; never stage the two earlier tracking files.
