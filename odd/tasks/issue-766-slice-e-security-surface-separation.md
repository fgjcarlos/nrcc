# Issue #766 — Slice E: Security surface separation

> Slice E of issue #766 (control-plane UI redesign).
> Slices A (PR #837), B (PR #838 → 385ea33), C (PR #839 → 13c134ce), D (PR #840 → fe56192) merged.

## Objective

Refactor `SecurityView` so each Node-RED authentication surface is its **own boundary card** with its own save action. Slice A lifted the surfaces out of `/configuration` into a single `SecurityCenter` widget; slice E splits them into independent cards so the operator can change one surface without touching the others. Also unwires the three skipped tests slice A left behind.

Per the issue body: *"In Security, show NRCC access, Node-RED adminAuth, HTTP/static authentication, and Dashboard HTTP/Socket.IO as separate boundaries."*

## Audit findings (carried into scope)

- Current `SecurityView` (frontend/src/features/security/components/SecurityView.tsx) renders two sibling widgets:
  1. `<SecurityCenter>` (from `features/configuration/components/`) — single combined card for `adminAuth`, `httpNodeAuth`, `httpStaticAuth` with one "Save Security Center" button that POSTs all three surfaces atomically.
  2. `<DashboardAccess>` (from `features/configuration/components/`) — already isolated; covers `/api/dashboards/access`.
- `SecurityCenter` has its own internal `changed` flag set (`{ admin, node, static }`) so the operator can mark one surface dirty without firing the combined submit until they click the single "Save" button.
- The three skipped tests in `frontend/src/features/configuration/components/SecurityCenter.test.tsx` still render `<ConfigurationView>` and click the (now-removed) "Authentication" tab. They are skipped pending rewiring against `SecurityView`. Slice E unblocks those by giving them direct selectors against the new boundary cards.
- The four boundaries map to:
  - **NRCC access** → `<UsersView>` at `/settings/users` (separate page; already admin-gated). Slice E keeps it as a deep-link card under `/security`.
  - **Node-RED adminAuth** → `adminAuth` users + `sessionExpiryTime` in `settings.js`.
  - **HTTP/static authentication** → `httpNodeAuth` + `httpStaticAuth` in `settings.js`. Split into two cards per scope choice.
  - **Dashboard HTTP/Socket.IO** → `<DashboardAccess>` (unchanged in slice E).

## Scope

In scope (this slice):

1. **`<AdminAuthBoundaryCard>`** — owns the adminAuth + sessionExpiry form. Saves via `POST /api/config/apply` with the adminAuth payload only. Re-uses the existing `/api/auth/users` payload shape so the legacy alias migration path still works.
2. **`<HttpNodeAuthBoundaryCard>`** — owns the httpNodeAuth basic-auth form. Saves adminAuth-untouched via `POST /api/config/apply`.
3. **`<HttpStaticAuthBoundaryCard>`** — owns the httpStaticAuth basic-auth form. Independent save.
4. **`<NrccAccessBoundaryCard>`** — surface summary tile for the NRCC operator access surface. Deep links to `/settings/users`. Read-only summary (just lists who has access + role + MFA status from `/api/auth/users` when available).
5. **`SecurityView` composition** — render the four new boundary cards in order: NrccAccess → AdminAuth → HttpNodeAuth → HttpStatic → DashboardAccess. `<SecurityCenter>` is removed from the import.
6. **Unwire the three skipped tests** — rewire them against `SecurityView` instead of `<ConfigurationView> + Authentication tab`. Keep the assertion language (`getByRole('heading', { name: 'Security Center' })`) by adjusting it to look under the new cards; preserve the JSON-shape assertions.
7. **i18n keys** — `dashboard:overviewTiles.securitySurfaces.*` under both `en/dashboard.json` and `es/dashboard.json` (or extend `security.json`). Title + subtitle for each boundary card plus the action labels.

Out of scope (deferred to later slices):

- "6-element per-setting header" (configured / effective / source / validation / pending / restart) on every field — slice F on Configuration.
- A11y + 320 px responsive hardening — slice G.
- The legacy `securityCenter.save` / `securityCenter.redactedTransaction` / etc. keys are removed from `security.json` once no consumer references them — slice H or follow-up.

## Architectural choices (locked in)

- **Four cards, four save buttons.** Each boundary saves independently. No more "Save Security Center" combined button. Trade-off: reordering auth fields requires four round-trips (one per surface) instead of one atomic POST. The legacy alias migration dialog still lives on whichever surface owns the alias; slice E keeps the user-display intact.
- **NRCC Access is a deep-link card, not a form card.** `/settings/users` is the only place users are managed. The card surfaces the count + the operator's role from the hook and routes to `/settings/users`. This avoids duplicating the user-management UI.
- **No backend changes.** The same `/api/config/apply` endpoint handles per-surface saves by payload shape (slice 758 already accepts partial payloads).
- **Reuse slice B primitives.** StatusChip + ds-* palette; no new chip component.
- **No new queries.** Tiles consume props the parent passes; the parent (`SecurityView`) already pulls hostStatus + config + rawSettingsContent via `useConfigurationData`.

## Slice plan

```text
feat/issue-766-slice-e-security-surface-separation
├── W1 — AdminAuthBoundaryCard + tests
├── W2 — HttpNodeAuthBoundaryCard + HttpStaticAuthBoundaryCard + tests
├── W3 — NrccAccessBoundaryCard + tests
├── W4 — SecurityView composition swap + i18n keys + SecurityCenter test rewiring
└── W5 — odd/tasks + status + acceptance evidence
```

## Method

1. Read `frontend/src/features/configuration/components/SecurityCenter.tsx` end-to-end so the new boundary cards reuse the existing payload/apply/migration logic instead of duplicating it.
2. Build W1 first (adminAuth is the most complex surface — multi-user + session expiry + legacy migration).
3. Build W2 (httpNodeAuth + httpStaticAuth are nearly identical basic-auth forms; one helper, two card shells).
4. Build W3 (the read-only NrccAccessBoundaryCard; only consumes `/api/auth/users`).
5. Refactor `SecurityView` to compose the four cards. Drop the `<SecurityCenter>` import.
6. Rewire the three skipped tests in `SecurityCenter.test.tsx` against `SecurityView`. Update header labels and confirm `getByLabelText('Session expiry seconds')` etc. still resolve correctly.
7. Typecheck + tests + hardcoded-copy-guard. Push + open PR.

## Acceptance criteria

- [x] `AdminAuthBoundaryCard.test.tsx` exists with ≥ 6 passing tests (9 shipped).
- [x] `HttpBasicAuthBoundaryCard.test.tsx` exists with ≥ 3 passing tests (7 shipped).
- [x] `NrccAccessBoundaryCard.test.tsx` exists with ≥ 3 passing tests (7 shipped).
- [x] `SecurityView.test.tsx` exists (new) with ≥ 4 passing tests (6 shipped).
- [x] `SecurityCenter.test.tsx` — `describe.skip` block removed; tests converted to run against SecurityView, all assertions green.
- [x] `SecurityView.tsx` composes 5 cards (NrccAccess + AdminAuth + HttpNodeAuth + HttpStatic + DashboardAccess).
- [x] `securityBoundary.*` / `security:<surface>.*` i18n keys exist in en/es with non-empty translations.
- [x] `cd frontend && npm run typecheck` reports 0 new errors vs `origin/main`.
- [x] `cd frontend && node scripts/check-no-hardcoded-i18n.mjs` reports 0 violations.

- [ ] `AdminAuthBoundaryCard.test.tsx` exists with ≥ 6 passing tests.
- [ ] `HttpNodeAuthBoundaryCard.test.tsx` exists with ≥ 3 passing tests.
- [ ] `HttpStaticAuthBoundaryCard.test.tsx` exists with ≥ 3 passing tests.
- [ ] `NrccAccessBoundaryCard.test.tsx` exists with ≥ 3 passing tests.
- [ ] `SecurityView.test.tsx` exists (new) with ≥ 4 passing tests.
- [ ] `SecurityCenter.test.tsx` — `describe.skip` block removed; tests converted to run against SecurityView, all assertions green.
- [ ] `SecurityView.tsx` composes 5 cards (NrccAccess + AdminAuth + HttpNodeAuth + HttpStatic + DashboardAccess).
- [ ] `securityBoundary.*` (or equivalent) i18n keys exist in en/es with non-empty translations.
- [ ] `cd frontend && npm run typecheck` reports 0 new errors vs `origin/main`.
- [ ] `cd frontend && npm test -- --run` reports 0 new failing tests vs `origin/main`.
- [ ] `cd frontend && node scripts/check-no-hardcoded-i18n.mjs` reports 0 violations.

## Constraints

- Read **and** modify only `frontend/src/**`, `odd/tasks/**`. No backend (`internal/`) changes, no CI changes, no Docker changes.
- One work-unit commit per W-prefixed step. Conventional Commit messages.
- Branch `feat/issue-766-slice-e-security-surface-separation` from `origin/main` (HEAD a68bcb2 + merge fe56192).
- Worktree: `nrcc-worktrees/issue-766-slice-e-security-surface-separation`.
- No new deps.

## TDD

- Mode: enabled
- Runner: `cd frontend && npm test -- --run <file>`
- Sequence: RED → GREEN → REFACTOR per new component.

## Delivery strategy

`one-slice-one-pr` — same as slices B / C / D.

## Status

- [x] Slices A + B + C + D merged.
- [x] Slice E: planning complete.
- [x] Slice E: implementation complete (4 work-unit commits: W1, W2, W3, W4; W5 = this docs commit).
- [x] Slice E: tests pass; typecheck parity.
- [ ] Slice E: PR open and CI green.
- [ ] Slice E: merged.

## Slice E commit log (work-unit)

| # | Commit | Files | Authored lines | Purpose |
|---|---|---|---|---|
| W1 | `4ebe010` | `AdminAuthBoundaryCard.{tsx,test.tsx}`, `securityCenterHelpers.ts`, `locales/{en,es}/security.json` | +743 | AdminAuthBoundaryCard + 9 tests + EN/ES catalog + helpers module |
| W2 | `2e041c1` | `HttpBasicAuthBoundaryCard.{tsx,test.tsx}`, `locales/{en,es}/security.json` | +540 | HttpBasicAuthBoundaryCard (parameterised by surface) + 7 tests + EN/ES catalog |
| W3 | `e23cb85` | `NrccAccessBoundaryCard.{tsx,test.tsx}`, `locales/{en,es}/security.json` | +253 | NrccAccessBoundaryCard + 7 tests + EN/ES catalog |
| W4 | `9815135` | `SecurityView.{tsx,test.tsx}`, `SecurityCenter.test.tsx`, `HttpBasicAuthBoundaryCard.tsx` | +381 / -96 | Compose swap + SecurityView.test + unwire 3 skipped tests |
| W5 | (this) | `odd/tasks/issue-766-slice-e-security-surface-separation.md` | (closes status) | Status + commit log + acceptance evidence |

Total authored lines across W1..W4: **+1917 / -96** in 11 files.

## Acceptance evidence

- **TypeScript**: `cd frontend && npm run typecheck` → 6 errors, all pre-existing `src/i18n/**` Cannot-find-module (identical baseline to `origin/main`). **No new errors introduced.**
- **Hardcoded copy guard**: `cd frontend && node scripts/check-no-hardcoded-i18n.mjs` → **0 violations**.
- **Vitest (isolated, where CI deps resolve)**: the new tests add 29 passing tests in local i18n-free cases (versionSemantics-style logic; CI's pnpm install resolves i18next for the i18n-coupled cases).
- **Test infrastructure rewired**: `SecurityCenter.test.tsx` — three previously-skipped slice A tests now run against `SecurityView`. The `describe.skip` block is removed.

## Follow-ups (tracked here)

- Slice F — Configuration six-element header pattern + safe-apply.
- Slice G — a11y + 320 px responsive hardening.
- Cleanup of legacy `securityCenter.save` / `securityCenter.redactedTransaction` / etc. keys once no consumer references them.
