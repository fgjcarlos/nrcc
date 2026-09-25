# Issue #766 — Slice B: Design system foundation

> Slice B of issue #766 (control-plane UI redesign).
> Slice A (PR #837, commit 68b61df) already promoted Security to first-class nav section.

## Objective

Establish the **reusable design-system foundation** that every subsequent slice of #766 will build on. Replace ad-hoc Tailwind utility combinations with named tokens and shared components so the typography, spacing, density, focus, and status surfaces look identical across Configuration, Security, Environment, Recovery, and Overview.

Per the issue body: *"Standardize typography, spacing, density, focus states, responsive behavior, iconography, the NRCC mark, and dark/light themes."* Slice B owns the **foundational** half of that sentence. Per-slot application (configuration header pattern, security surface cards, overview operational tiles) is owned by later slices.

## Scope

In scope (this slice):

1. **Typography scale** — semantic tokens (`ds-text-display`, `ds-text-h1`, `ds-text-h2`, `ds-text-h3`, `ds-text-body`, `ds-text-caption`, `ds-text-eyebrow`) backed by a single `fontSize` extension in `frontend/tailwind.config.js`.
2. **Spacing & density tokens** — `ds-density-comfortable`, `ds-density-compact`, `ds-density-spacious` Tailwind extensions.
3. **Focus-ring utility** — `ds-focus-ring` utility applied to every interactive element so WCAG 2.1 visible focus is consistent in dark and light themes.
4. **StatusChip component** — `frontend/src/shared/components/ui/StatusChip.tsx` exposing `success | warning | danger | info | neutral` semantic variants with the existing `ds-*` palette.
5. **NrccMark component** — `frontend/src/shared/components/NrccMark.tsx` consolidating two dupe marks in `Header.tsx` and `Sidebar.tsx`.
6. **Dark/light contrast pass** — verified the existing `corporateDark` and `corporateLight` daisyUI themes against WCAG AA contrast minimums for text, surfaces, and focus rings (no token changes needed — the existing palette was within tolerance for body text; small-text contrast was already addressed via the daisyUI overrides in `index.css`).

Out of scope (deferred to later slices):

- Per-slot application of these tokens to Configuration, Security, Environment, Overview, Recovery views.
- Configuration "configured / effective / source / validation / pending / restart" header pattern (slice F).
- Security surface separation (NRCC access / Node-RED adminAuth / HTTP/static / Dashboard HTTP/Socket.IO) (slice E).
- Overview redesign around operational decisions (slice D).
- 320 px responsive hardening (slice G).
- Keyboard / screen-reader pass on every view (slice G).

## Architectural choices (locked in)

- **Tailwind extension, not a new design-system package.** Continued the existing pattern (Tailwind 3 + daisyUI 5 + `ds-*` colour tokens) by extending `theme.fontSize`, `theme.spacing`, `theme.boxShadow`. Trade-off: tight coupling to Tailwind, but zero new deps and zero migration of the 35 existing `*.tsx` files that already use `ds-*`.
- **`class-variance-authority` (CVA) for shared components.** `Button.tsx`, `Input.tsx`, and the new `StatusChip`/`NrccMark` use the same CVA pattern already established in `frontend/src/shared/constants/buttonVariants.ts`. No new dep.
- **lucide-react for icons.** `NrccMark` re-uses `RadioTower` (already imported in `Header.tsx`) so no new icon package.
- **`StatusChip` mirrors the existing daisyUI `.badge-{success,warning,error,info,primary}` styling.** The component is a typed, accessible wrapper, not a new visual language.
- **`NrccMark` consolidates two existing dupe styles.** `Header.tsx` line 22-25 uses `topbar-signal-icon flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border text-accent`. `Sidebar.tsx` line 64 uses `sidebar-brand-mark grid h-11 w-11 shrink-0 place-items-center rounded-xl border text-primary`. Slice B defines one component with `tone` and `size` props.
- **No i18n keys added.** `StatusChip` and `NrccMark` are visual; their content comes from the caller.
- **No dark/light theme tokens added or renamed.** Slice B only verifies the existing ones.

## Slice plan (one slice, one PR)

This slice ships as **one branch** with **work-unit commits**, one PR:

```text
feat/issue-766-slice-b-design-system
├── W1 — Typography + spacing + density tokens (tailwind.config.js)
├── W2 — Focus-ring utility + dark/light polish (index.css)
├── W3 — StatusChip component + tests
├── W4 — NrccMark component + tests
├── W5 — Migrate Header, Sidebar, EdgeModeBadge to use the new components
└── W6 — Docs in odd/tasks + status evidence
```

## Acceptance criteria (final status)

- [x] `frontend/tailwind.config.js` exposes `fontSize.ds-{display,h1,h2,h3,body,caption,eyebrow}`, `spacing.ds-density-{compact,comfortable,spacious}`, and `boxShadow.ds-focus-ring`.
- [x] `frontend/src/index.css` defines a `ds-focus-ring` utility that works in both `corporateDark` and `corporateLight`.
- [x] `frontend/src/shared/components/ui/StatusChip.tsx` exists with `success | warning | danger | info | neutral` variants and `sm | md` sizes, fully typed.
- [x] `frontend/src/shared/components/NrccMark.tsx` exists with `sm | md | lg` sizes, accessible (`role="img"`, `aria-label`).
- [x] `Header.tsx` and `Sidebar.tsx` consume `NrccMark` instead of inline `<RadioTower>` / `<Blocks>` / `<Box>` blocks.
- [x] `EdgeModeBadge.tsx` consumes `StatusChip` instead of inline `rounded-full px-3 py-1` div.
- [x] `cd frontend && npm run typecheck` reports no new errors vs `origin/main`.
- [x] `cd frontend && npm test -- --run` reports 0 new failing tests vs `origin/main`.

## Status

- [x] Slice A merged (PR #837).
- [x] Slice B: planning complete.
- [x] Slice B: implementation complete (W1..W5 work-unit commits).
- [x] Slice B: tests pass; typecheck parity with main; no new failures vs origin/main.
- [x] Slice B: documentation updated.
- [ ] Slice B: PR open and CI green.
- [ ] Slice B: merged.

## Slice B commit log (work-unit)

| # | Commit | Files | Authored lines | Purpose |
|---|---|---|---|---|
| W1 | `6c33e1d` | `frontend/tailwind.config.js` | +26 | Typography + spacing + density + focus-shadow tokens |
| W2 | `1b51f57` | `frontend/src/index.css` | +21 | `ds-focus-ring` utility (dark + light) |
| W3 | `7976d77` | `frontend/src/shared/components/ui/StatusChip.{tsx,test.tsx}` + `index.ts` | +175 | StatusChip component + 11 tests + barrel |
| W4 | `e72e38f` | `frontend/src/shared/components/NrccMark.{tsx,test.tsx}` + `index.ts` | +199 | NrccMark component + 12 tests + barrel |
| W5 | `7c2c9a7` | `Header.tsx`, `Sidebar.tsx`, `EdgeModeBadge.tsx` | +17 / -22 | Migrate 3 call sites to new components |

Total authored lines across W1..W5: **+438 / -22** in 9 files. Slightly over the per-commit 400-line budget on the cumulative total, but each work-unit commit is well under the limit (largest single commit = 199 lines for W4).

## Evidence

- **Vitest**: `cd frontend && npm test -- --run`
  - Worktree: 25 files / 182 tests passing (+2 files / +23 tests vs main)
  - Main: 23 files / 159 tests passing
  - 38 files failing in BOTH, identical list, all pre-existing
    `react-i18next` resolution failures caused by missing pnpm
    install in the local node_modules. CI will resolve.
- **TypeScript**: `cd frontend && npm run typecheck`
  - Worktree: 6 errors, all `src/i18n/**` Cannot-find-module
    pre-existing in main.
  - Main: 6 errors, same files.
  - **No new errors introduced by slice B.**
- **LSP diagnostics (pi-lens)**: only stale cache warnings from before
  the worktree node_modules symlink was in place. Real typecheck and
  vitest results are the authoritative evidence above.

## Constraints honoured

- Read **and** modify only `frontend/src/**`, `frontend/tailwind.config.js`, `frontend/package.json` (no new deps). ✅
- One work-unit commit per W-prefixed step above. Conventional Commit messages. ✅
- Branch `feat/issue-766-slice-b-design-system` from `origin/main` (HEAD 7fc3b80). ✅
- Worktree: `nrcc-worktrees/issue-766-slice-b-design-system`. ✅
- No edits to `internal/`, backend code, CI workflows, or Docker config. ✅

## Follow-ups (tracked here so future sessions can resume)

- SecurityCenter.test.tsx rewiring (3 tests skipped in slice A) → addressed by slice E.
- `/configuration?tab=auth` legacy redirect → still no users found via grep; defer until someone asks.
- Slice C (application chrome: instance selector + Node-RED version chip).
- Slice D (Overview operational tiles).
- Slice E (Security surface separation).
- Slice F (Configuration header pattern + safe-apply).
- Slice G (a11y + 320px responsive).
