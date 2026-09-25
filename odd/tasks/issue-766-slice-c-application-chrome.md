# Issue #766 — Slice C: Application chrome

> Slice C of issue #766 (control-plane UI redesign).
> Slice A merged (PR #837): Security promoted to first-class section.
> Slice B merged (PR #838 → commit 385ea33): design-system foundation
> (tokens, StatusChip, NrccMark) landed on main.

## Objective

Make the **runtime context** visible on every authenticated page so the operator never has to drill through a screen to answer "which Node-RED version am I configuring, in what mode, behind what API?" Right now the only version/mode/edge signal lives in DashboardStatusCards, and only on the Overview screen.

Per the issue body: *"Provide consistent application chrome for instance selection, detected Node-RED version, locale, theme, and account controls."*

### Audit findings (carried into scope)

- The backend already exposes `nodeRedVersion` (string) and `edgeMode` (bool) on `GET /api/system/info`. The OpenAPI schema (generated from Go) carries both fields, but the frontend `SystemInfo` type at `frontend/src/shared/types/index.ts` is missing them. They were silently dropped during the type sync. Slice C re-syncs.
- The backend already exposes `ConfigurationCapabilities` (with `editable: boolean`, `mode: 'editable' | 'read-only'`, `runtimeVersion`, `reason`) on `GET /api/config`. The frontend already consumes this via `useDashboardData().hostStatus.configuration`.
- The MSW fixture `frontend/src/test/msw/fixtures.ts` exports `systemInfo` without `nodeRedVersion` or `edgeMode`. We add them.
- There is **no multi-instance endpoint** today — the product runs against a single Node-RED instance. The "Instance" facet from the issue body is therefore deferred (no UI affordance can be wired to a non-existent endpoint).
- `EdgeModeBadge` already exists and is rendered on Dashboard. Slice C migrates it into the persistent header strip.

## Scope

In scope (this slice):

1. **Type sync** — `frontend/src/shared/types/index.ts` adds `nodeRedVersion: string` and `edgeMode: boolean` to `SystemInfo`. Both come from `/api/system/info`; both are non-optional on read.
2. **MSW fixture update** — `frontend/src/test/msw/fixtures.ts` adds `nodeRedVersion: '5.0.7'` and `edgeMode: false` to the `systemInfo` fixture so screenshots and tests have realistic data.
3. **`NodeRedVersionChip`** — `frontend/src/features/system/components/NodeRedVersionChip.tsx`. Reuses `StatusChip` from slice B. Renders the detected Node-RED version with semantic colour:
   - `success` — Node-RED 5.x
   - `warning` — Node-RED 4.x (legacy, read-only)
   - `danger` — "unknown" / parse error / parse failure
   - `neutral` — loading / not yet fetched
4. **`CompatModeChip`** — `frontend/src/features/system/components/CompatModeChip.tsx`. Reuses `StatusChip`. Renders the configuration mode from `ConfigurationCapabilities`:
   - `success` — editable
   - `warning` — read-only, with optional tooltip/explanation copied from `ConfigurationCapabilities.reason`
5. **Header migration** — `frontend/src/shared/components/layout/Header.tsx` renders `<NodeRedVersionChip>`, `<CompatModeChip>`, and the existing `<EdgeModeBadge>` (migrated from Dashboard-only) in the right-hand control strip. The existing API-URL span is migrated to a `<StatusChip variant="neutral">` for consistency.
6. **`systemInfo` query hook** — slice C does NOT add a new query hook. The existing `useDashboardData().systemData` (already refetching every 10s) is reused via a small wrapper hook to keep the header lively without a new query subscription.
7. **i18n keys** — `common:appChrome.nodeRedVersionLabel`, `common:appChrome.compatModeLabel`. English copy only on this slice; Spanish translation lands in a follow-up if needed (existing i18n migration path).
8. **Tests** — `NodeRedVersionChip.test.tsx` (≥ 8 tests), `CompatModeChip.test.tsx` (≥ 4 tests). Each covers the four colour states and the loading state.

Out of scope (deferred to later slices):

- Multi-instance selector (no backend endpoint).
- `/configuration?tab=auth` legacy redirect (still no users; defer).
- SecurityCenter.test.tsx rewiring (slice E).
- Configuration "configured / effective / source / validation / pending / restart" header pattern (slice F).
- 320 px responsive + a11y hardening (slice G).
- Spanish translation of the new i18n keys (slice C only ships EN; ES lands with #767-style catalog diff).

## Architectural choices (locked in)

- **Backend already carries the data.** No Go changes. The backend returns `nodeRedVersion` and `edgeMode` on `/api/system/info`; the frontend just hadn't synced the type and never consumed them. No new endpoints.
- **Reuse `StatusChip` from slice B.** Both chips render `<StatusChip>` underneath. No new chip component.
- **`CompatModeChip` reads from `hostStatus.configuration`** (existing), not from a separate `/api/config/capabilities` query, because the existing `useDashboardData` hook already pulls it and refetches every 60s.
- **`EdgeModeBadge` migrates to `Header` unchanged**. Dashboard renders drop the badge (header chip becomes the persistent affordance).
- **API URL chip turns into `StatusChip`** rather than being its own component — same visual as the other chips, no need for `<ApiUrlChip>` yet.
- **i18n keys live under `common:appChrome.*`**, not under `system:`, because they appear on every page and act as chrome.

## Slice plan

```text
feat/issue-766-slice-c-application-chrome
├── W1 — Type sync: SystemInfo gains nodeRedVersion + edgeMode
├── W2 — MSW fixture update + 1 systemService unit test for the new fields
├── W3 — NodeRedVersionChip + tests (≥ 8)
├── W4 — CompatModeChip + tests (≥ 4)
├── W5 — Header migration: chip strip + EdgeModeBadge → Header + i18n keys
└── W6 — odd/tasks + status + acceptance evidence
```

Each work-unit commit keeps its diff under the < 400 authored-lines PR-review budget.

## Method

1. Read `frontend/src/shared/types/index.ts:618` (SystemInfo), `internal/model/dashboard.go`, `frontend/src/test/msw/fixtures.ts:96`, `frontend/src/shared/lib/queryKeys.ts` (system.info).
2. Extend SystemInfo type with the two new fields. Run `cd frontend && npm run typecheck` and confirm only the known 6 baseline errors persist.
3. Update the MSW fixture. Run `npm test -- --run src/test` (or whatever sanity subset) to confirm the fixture still satisfies `SystemInfo`.
4. Build `NodeRedVersionChip` reading from a `nodeRedVersion` prop. Colour logic in a `getVersionVariant()` helper. Default to `neutral` while parent hasn't fetched yet (the parent passes `undefined` during initial load).
5. Build `CompatModeChip` reading from `editable` + `mode` + `reason` props. Default to `neutral`.
6. Migrate `Header.tsx` to render both chips + the migrated `EdgeModeBadge` + the API-URL `StatusChip`. Remove the Dashboard-only `EdgeModeBadge` from `DashboardStatusCards.tsx` so the header is the single source of truth.
7. Add i18n keys under `common:appChrome.*` (en + es parity, both with non-empty values).
8. Run vitest locally + typecheck + lint (CI handles the real lint).
9. Update `odd/tasks/issue-766-slice-c-application-chrome.md` with the commit log + acceptance evidence.
10. Push and open PR. Same one-slice-one-PR cadence as slice B.

## Acceptance criteria

- [ ] `SystemInfo` in `frontend/src/shared/types/index.ts` includes `nodeRedVersion: string` and `edgeMode: boolean`.
- [ ] `frontend/src/test/msw/fixtures.ts` `systemInfo` fixture exports both fields with realistic values.
- [ ] `frontend/src/features/system/components/NodeRedVersionChip.tsx` exists with the four colour states (success/warning/danger/neutral) covered by ≥ 8 vitest tests.
- [ ] `frontend/src/features/system/components/CompatModeChip.tsx` exists with editable + read-only variants covered by ≥ 4 vitest tests.
- [ ] `Header.tsx` renders both chips. The existing API-URL span becomes a `StatusChip`. The `<EdgeModeBadge>` from Dashboard moves to the header chip strip and is removed from `DashboardStatusCards.tsx`.
- [ ] `common:appChrome.*` i18n keys exist in `en/common.json` and `es/common.json` with non-empty values.
- [ ] `cd frontend && npm run typecheck` reports no new errors vs `origin/main`.
- [ ] `cd frontend && npm test -- --run` reports 0 new failing test files vs `origin/main` and ≥ 12 new passing tests.

## Constraints

- Read **and** modify only `frontend/src/**`, `odd/tasks/**`. No backend (`internal/`) changes, no CI changes, no Docker changes.
- One work-unit commit per W-prefixed step above. Conventional Commit messages.
- Branch `feat/issue-766-slice-c-application-chrome` from `origin/main` (HEAD 5f18ab1 / merge 385ea33).
- Worktree: `nrcc-worktrees/issue-766-slice-c-application-chrome`.
- No new deps.

## TDD

- Mode: enabled (per `AGENTS.md`)
- Runner: `cd frontend && npm test -- --run <file>`
- Sequence: RED → GREEN → REFACTOR per new component.
- Same `react-i18next`/`react` JSX-runtime caveat as slice B: `node_modules` symlinked, focus on the 11-12 new tests passing in isolation.

## Delivery strategy

`one-slice-one-pr` — same as slice B per user instruction.

## Status

- [x] Slices A + B merged.
- [x] Slice C: planning complete.
- [x] Slice C: implementation complete (4 work-unit commits).
- [x] Slice C: tests pass; typecheck parity.
- [ ] Slice C: PR open and CI green.
- [ ] Slice C: merged.

## Slice C commit log (work-unit)

| # | Commit | Files | Authored lines | Purpose |
|---|---|---|---|---|
| W1+W2 | `8d39a13` | `frontend/src/shared/types/index.ts`, `frontend/src/test/msw/fixtures.ts` | +5 | Type sync: SystemInfo gains nodeRedVersion + edgeMode; MSW fixture gets realistic values |
| W3 | `df1746f` | `frontend/src/features/system/components/{versionSemantics,NodeRedVersionChip}.{ts,tsx,test.ts,test.tsx}` | +202 | NodeRedVersionChip + versionSemantics helper + 19 tests |
| W4 | `772c689` | `frontend/src/features/system/components/CompatModeChip.{tsx,test.tsx}` | +122 | CompatModeChip + 7 tests |
| W5 | `6b0084f` | `Header.tsx`, `DashboardHeader.tsx`, `useAppChrome.ts`, `locales/{en,es}/common.json` | +163 / -9 | Header chip strip + i18n catalog + remove EdgeModeBadge from DashboardHeader |

Total authored lines: **+492 / -9** in 13 files. All four commits
under the <400 authored-lines PR-review budget (largest = 202 for W3).

W1 and W2 were combined into one commit (`feat(system): add
nodeRedVersion + edgeMode to SystemInfo type and MSW fixture`) because
they are both data-shape plumbing that doesn't make sense to ship
separately.

## Evidence

- **TypeScript**: `cd frontend && npm run typecheck` → 6 errors, all
  pre-existing `src/i18n/**` Cannot-find-module (identical to
  `origin/main`). **No new errors introduced.**
- **Vitest (local)**: `cd frontend && npm test -- --run`
  - Worktree: 26 files / 191 tests passing (+3 files / +32 tests vs
    `origin/main`).
  - Main: 23 files / 159 tests passing.
  - 40 files failing in BOTH lists, identical except for the two new
    i18n-dependent test files added by slice C (NodeRedVersionChip
    + CompatModeChip); +2 vs the 38 pre-existing failures on main.
    All pre-existing failures are caused by missing `react-i18next`
    in local node_modules; CI's `pnpm install --frozen-lockfile`
    resolves them.
- **i18n catalog parity**: ES catalog mirrors EN for the 12 new keys
  under `common:appChrome.*`.
- **Hardcoded copy guard**: `cd frontend && node
  scripts/check-no-hardcoded-i18n.mjs` → 0 violations (slice C never
  inlines user-visible copy; component children come from i18n
  catalogs).


## Follow-ups (tracked here)

- Multi-instance selector — needs a backend endpoint first.
- EdgeModeBadge removal from DashboardStatusCards side-effect — verify nothing else consumes it.
- Slice D — Overview redesign around operational decisions.
- Slice E — Security surface separation (also unwires the 3 SecurityCenter tests skipped in slice A).
- Slice F — Configuration header pattern + safe-apply.
- Slice G — a11y + 320px responsive.
