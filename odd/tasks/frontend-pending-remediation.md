# Frontend security and toolchain completion

## Authorization and delivery

- User selected complete correction in separate units: moderates, lint, then
  Tailwind 4/daisyUI 5 alignment, preserving the current appearance.
- Base: `85cb3be3e83daac0b7bd39060f468cd40b52ea68` (verified H1).
- P1 branch: `fix/nrcc-frontend-moderates`, child of `fix/nrcc-security-highs`.
- Delivery: feature-branch chain; verified local Conventional Commits only.
  No push, PR, merge, publication, waiver, or security-gate changes.
- Preserve unstaged completion tracking in `odd/tasks/frontend-security-highs.md`
  and `odd/tasks/nrcc-stabilization.md`, plus untracked `.codegraph/`.
- One source writer at a time. No installs, modules, root lockfile or frontend
  dist in the worktree. Use isolated guarded snapshots and pnpm 11.12.0.
- Preserve production containers/data/secrets, Go/Docker pins and S1–S3 behavior.
- Budget: 400 total additions plus deletions per reviewed slice, including lock
  and tracking. Forecast P1 200–350, P2 80–150; P3 requires a scoped forecast
  before implementation. Split or obtain a scope decision if a coherent slice
  exceeds the agreed budget; never remove tests or compress code to fit.

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

- [ ] Verified fixes and local work-unit commit.
- Status: in progress; all targeted checks verified, local commit pending.
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
- Commit: pending; target unit verified, global merge still blocked by P2/P3.

### P2 — Remove five lint warnings without changing behavior

- [ ] Verified cleanup and local work-unit commit.
- Status: pending; seven-file read-only scope mapped, no helper extraction needed.
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
- Branch/commit/evidence: pending, child of verified P1.

### P3 — Align Tailwind 4 and daisyUI 5; close braces and CSS defects

- [ ] Verified alignment and local work-unit commit(s).
- Status: pending; map exact config/theme/CSS and full graph before writer.
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
- Branch/slices/commit/evidence: pending, child chain after P2.

## Recovery and evidence

- Full Engram mirror: `odd/frontend-pending-remediation/tasks`.
- Previous isolated evidence: `/tmp/nrcc-audit-DRYCSyHs/h1-verify-exact-R1-evidence/`.
- New dependency graph requires fresh evidence; previous GREEN is not a substitute.
- Rollback each unit's exact source/tracking slice without reverting S1–S3/H1.
- Record observed command exits, failures, skips, hashes and owned stopped runtime
  resources after every task transition. Commit identities are recorded afterward
  as unstaged completion bookkeeping; never stage the two earlier tracking files.
