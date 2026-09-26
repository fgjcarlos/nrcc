# Issue #766 — Slice D: Overview operational tiles

> Slice D of issue #766 (control-plane UI redesign).
> Slices A (PR #837), B (PR #838 → 385ea33), C (PR #839 → 13c134ce) merged.

## Objective

Replace the Overview screen's generic resource cards with **operational decision tiles** that tell the operator *what to do next*, not just *what the metrics say*. Each new tile carries three things: the operational status, the primary action, and a deep link to the screen that owns the decision.

Per the issue body: *"Redesign Overview, Configuration, and Security around operational decisions rather than generic dashboard cards."*

## Audit findings (carried into scope)

- The current Overview renders four pieces that are mostly metric dashboards:
  - `SystemHealthCard` — host readiness + issues list
  - `DashboardStatusCards.RuntimeCard` — Node-RED process state, restart/open actions
  - `DashboardStatusCards.{CpuCard, MemoryCard}` — percentage + sparkline
  - `DashboardDetails.{DiskUsageCard, BackupStatusCard}` — disk bar + backup tiles
- The metric cards (CPU/Memory/Disk) only help if the operator wants raw numbers; the actionable surface is "what is wrong / where do I go". Slice D replaces these with three operational tiles.
- The existing data sources (`useDashboardData`, `systemInfo`, `hostStatus`, `runtime`, `backups`) already carry everything the new tiles need. No backend changes.
- The slice F six-element pattern (configured / effective / source / validation / pending / restart) is owned by slice F. Slice D intentionally ships a simpler per-tile status (chip + CTA + deep link) on Overview; the full six-element per-setting header lands on Configuration.

## Scope

In scope (this slice):

1. **`<SecurityPostureTile>`** — single tile rendering the four authentication surfaces (`adminAuth`, `httpNodeAuth`, `httpStaticAuth`) plus `requireHttps`. Each row is a StatusChip showing configured value + a short "go fix it" link to `/security`. Status variants follow slice B's contract: success / warning / danger / neutral.
2. **`<BackupHealthTile>`** — promotes the existing `BackupStatusCard` content into a top-level Overview tile, drops the disk card that shared its row, and keeps the three glass-panel sub-blocks (last backup / last automatic / storage) as a compact summary. Status chip surfaces the scheduler health.
3. **`<NodeRedHealthTile>`** — folds `SystemHealthCard` + the resource-scope label + the `RuntimeCard`'s restart/open actions + a compact "host resources" indicator (CPU % + Memory % + Disk %) into one tile. Carries a single restart button and the live telemetry link.
4. **Delete generic cards** — `SystemHealthCard.tsx`, `DashboardDetails.tsx`'s `DiskUsageCard`/`BackupStatusCard` go away; their behaviour moves into the new tiles. `DashboardDetails.tsx` is removed entirely.
5. **DashboardView composition** — the page renders `DashboardHeader → DashboardWarnings → SecurityPostureTile → NodeRedHealthTile → BackupHealthTile → RestartConfirmationModal`. Two existing children (`SystemHealthCard`, `DashboardDetails`) are removed from the composition.
6. **i18n keys** — `dashboard:overviewTiles.securityPosture.*`, `dashboard:overviewTiles.backupHealth.*`, `dashboard:overviewTiles.nodeRedHealth.*` under both `en/dashboard.json` and `es/dashboard.json`. The legacy `dashboard:runtimeCard.*` / `dashboard:diskUsage.*` / `dashboard:localBackups.*` / `dashboard:systemHealth` keys remain so existing copy in `DashboardView.test.tsx` keeps working during the migration; the test is updated to assert the new headings.
7. **Tests** — `SecurityPostureTile.test.tsx` (≥ 6 tests covering each surface + the deep link), `BackupHealthTile.test.tsx` (≥ 4 tests covering healthy / unhealthy / no-backup states), `NodeRedHealthTile.test.tsx` (≥ 6 tests covering restart availability + resource indicator + telemetry link), and an updated `DashboardView.test.tsx` that exercises the new tile composition.

Out of scope (deferred to later slices):

- The six-element per-setting header pattern (configured / effective / source / validation / pending / restart) — slice F on Configuration.
- Security surface separation into four boundary cards — slice E.
- A11y + 320 px responsive hardening — slice G.
- Removing the legacy i18n keys for `runtimeCard` / `diskUsage` / `localBackups` / `systemHealth` — they remain for catalog parity. Slice H or a follow-up can clean them up once no consumer references them.

## Architectural choices (locked in)

- **Three tiles, not six.** Slice D is *Overview*; slice F is *Configuration*. We resist the temptation to apply the six-element pattern here.
- **Reuse slice B components.** StatusChip + ds-* palette from slice B, the `ds-focus-ring` utility from slice B, the CVA variant conventions from slice B. No new primitives.
- **No new queries.** Each tile consumes props the parent passes; the parent (`DashboardView`) continues to use `useDashboardData` so no new React Query subscription.
- **No backend changes.** The data is already on `hostStatus`, `runtime`, `system`, `backups`. If a future slice needs finer detail (e.g. validation messages per setting) we add it on the backend then.
- **Deep-link, not embedded form.** Each tile's "primary action" is a `<Link>` to the owning screen (`/security`, `/backups`, `/configuration`). Embedded forms land on Configuration per slice F.

## Slice plan

```text
feat/issue-766-slice-d-overview-operational
├── W1 — SecurityPostureTile + tests (≥ 6)
├── W2 — BackupHealthTile + tests (≥ 4)
├── W3 — NodeRedHealthTile + tests (≥ 6)
├── W4 — DashboardView composition swap + i18n keys
└── W5 — odd/tasks + status + acceptance evidence
```

Each work-unit commit keeps its diff under the < 400 authored-lines PR-review budget.

## Method

1. Read `DashboardView.tsx`, `DashboardStatusCards.tsx`, `DashboardDetails.tsx`, `SystemHealthCard.tsx`, `DashboardView.test.tsx`. Confirm what the existing test asserts and what copy keys it touches.
2. Build `SecurityPostureTile` first (highest-leverage tile; consumes the four auth surfaces + requireHttps). Tests cover each surface's variant.
3. Build `BackupHealthTile` next (mirrors the existing `BackupStatusCard` content but as a top-level tile). Tests cover scheduler healthy / unhealthy / unknown / no-backups.
4. Build `NodeRedHealthTile` last (the most complex — folds restart/open + resource indicator). Tests cover the canRestart gate, the resource indicator's three percentages, and the deep link.
5. Re-compose `DashboardView`. Update `DashboardView.test.tsx` to assert the new headings.
6. Run `npm run typecheck` (expect 0 new errors vs origin/main) and `npm test -- --run` (expect 0 new failing tests vs origin/main).
7. Update `odd/tasks/issue-766-slice-d-overview-operational.md` with the commit log + evidence.
8. Push + open PR.

## Acceptance criteria

- [ ] `SecurityPostureTile.test.tsx` exists with ≥ 6 passing tests.
- [ ] `BackupHealthTile.test.tsx` exists with ≥ 4 passing tests.
- [ ] `NodeRedHealthTile.test.tsx` exists with ≥ 6 passing tests.
- [ ] `DashboardView.tsx` renders the three new tiles (in the documented order) and no longer imports `SystemHealthCard` or `DashboardDetails`.
- [ ] `DashboardView.test.tsx` updated; existing assertions preserved; new tile headings asserted.
- [ ] `dashboard:overviewTiles.*` i18n keys exist in both `en/dashboard.json` and `es/dashboard.json` with non-empty translations.
- [ ] `cd frontend && npm run typecheck` reports 0 new errors vs `origin/main`.
- [ ] `cd frontend && npm test -- --run` reports 0 new failing tests vs `origin/main`.
- [ ] `cd frontend && node scripts/check-no-hardcoded-i18n.mjs` reports 0 violations.

## Constraints

- Read **and** modify only `frontend/src/**`, `odd/tasks/**`. No backend, no CI, no Docker.
- One work-unit commit per W-prefixed step. Conventional Commit messages.
- Branch `feat/issue-766-slice-d-overview-operational` from `origin/main` (HEAD 88a69f2 + 13c134ce).
- Worktree: `nrcc-worktrees/issue-766-slice-d-overview-operational`.
- No new deps.

## TDD

- Mode: enabled (per `AGENTS.md`)
- Runner: `cd frontend && npm test -- --run <file>`
- Sequence: RED → GREEN → REFACTOR per new tile component.

## Delivery strategy

`one-slice-one-pr` — same as slice B and C.

## Status

- [x] Slices A + B + C merged.
- [x] Slice D: planning complete.
- [ ] Slice D: implementation complete (W1..W5).
- [ ] Slice D: tests pass; typecheck parity.
- [ ] Slice D: PR open and CI green.
- [ ] Slice D: merged.

## Follow-ups (tracked here)

- Slice E — Security surface separation (also unwires the 3 SecurityCenter tests skipped in slice A).
- Slice F — Configuration six-element header pattern + safe-apply.
- Slice G — a11y + 320 px responsive hardening.
- Cleanup of legacy `runtimeCard` / `diskUsage` / `localBackups` / `systemHealth` i18n keys once no consumer references them.
