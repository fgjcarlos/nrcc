# Issue #766 — Slice G: Accessibility, 320 px responsive, loading/empty/error states

> Slice G of issue #766 (control-plane UI redesign).
> Slices A (PR #837), B (PR #838 → 385ea33), C (PR #839 → 13c134ce), D (PR #840 → fe56192), E (PR #850 → 9c9354c), F (PRs #851–#854 → e20cf89, 4e66275, bbe8124, 8a66208) merged.

## Objective

Make the authenticated application **usable for everyone on any screen, in any state**. Slice B standardized the visual primitives, slices C–F standardized the navigation, chrome, Overview, Security, and Configuration surfaces, but they did not enforce the cross-cutting non-functional requirements that the issue body calls out as one sentence: *"Meet keyboard, contrast, screen-reader, 320 px responsive, loading, empty, validation, and error-state requirements."*

Slice G owns those requirements end-to-end so every screen on every viewport, with every data state, is operable, perceivable, understandable, and robust.

Per the issue body, slice G covers the **cross-cutting** half of the requirements. Per-slot token application (typography, density, focus rings) already shipped in slice B.

## Audit findings (carried into scope)

The repo already has partial infrastructure; slice G consolidates and extends it without inventing parallel systems.

- **`frontend/KEYBOARD_NAV.md`** documents the `ConfirmationDialog` and `UserMenu` keyboard patterns. Slice G extends the same contract to the new boundary cards, ReviewChangesPanel, command palette, and any new modal the slices E/F added.
- **`frontend/COLOR_REFERENCE.md`** and **`frontend/docs/SIGNAL_PRIME_TOKENS.md`** define the daisyUI palette. Slice G audits every primary surface against WCAG 2.1 AA (4.5:1 body text, 3:1 large text and UI components) in both themes; any drift is fixed inside the existing `corporateDark` / `corporateLight` tokens (no new token names).
- **`frontend/src/shared/components/ConfirmationDialog.tsx`** already implements focus trap + Escape + auto-focus + focus restoration. Slice G audits that all new dialogs (slice E migration dialog, slice F review drawer) reuse this primitive or match its semantics.
- **`frontend/src/shared/components/ui/StatusChip.tsx`** already provides `success | warning | danger | info | neutral` semantic variants. Slice G does not add a new component; it ensures every status surface uses StatusChip (or a11y-equivalent text + role) so screen readers announce the meaning and not only the colour.
- **`prefers-reduced-motion`** is referenced in Tailwind utilities but not consistently wired. Slice G enforces a single global rule in `frontend/src/index.css` so animations, transitions, and sonner toasts all respect the user setting.
- **320 px responsive** is currently uneven. Slice F's `ReviewChangesPanel` drawer and slice E's boundary cards assume ≥ 768 px. Slice G introduces a `ds-breakpoint-compact` (`max-width: 639px`) layout collapse for the four boundary cards, the review drawer (bottom sheet), the configuration header, and the navigation header strip.
- **Loading / empty / error states** are inconsistent. Some tiles render `null` while loading (slice D), others render skeletons, others block on the query. Slice G codifies a single pattern per state in `frontend/src/shared/components/feedback/` and rewrites the divergent tiles to use it.
- **`StateContainer` already ships** as the legacy loading/error/empty wrapper with four call sites (FilesView, FlowsView, UsersView, BackupListSection). Slice G keeps `StateContainer` unchanged and adds `LoadingBoundary` as a **typed, additive** alternative that drives the new primitives from a discriminated union. W3 will migrate call sites that benefit from the typed contract and leave the others on `StateContainer`.
- **`<html lang>`** is set to `en` in `frontend/index.html`. Slice G keeps `lang` in sync with the active i18n locale via a tiny effect on locale change (mirrors the existing theme toggle pattern).
- **axe-core** is not yet wired into CI. Slice G adds `frontend/scripts/check-axe-violations.mjs` (puppeteer + axe-core) and runs it on the four highest-traffic pages (Login, Overview, Configuration, Security) in the e2e suite. A failing axe audit fails CI.

## Audit findings discovered during W1

- **`Button` variants** are `primary | secondary | tertiary` only — there is no `outline` variant. `ErrorState.tsx` therefore uses `secondary` for the retry CTA.
- **React Query result objects** vary in shape (`data | undefined` for some hooks, `{ data: [] }` for others). `LoadingBoundary`'s `useEmptyState` helper accepts the raw value and treats `null` / `undefined` as "still loading", which matches the React Query contract without forcing the caller to spread the result.
- **`ds-danger` and `ds-border-default` tokens** are already defined in `tailwind.config.js`. No new design tokens introduced in W1.

## Scope

In scope (this slice):

1. **`A11yPrimitives` — shared feedback components.**
   - `frontend/src/shared/components/feedback/Skeleton.tsx` — typed `<Skeleton variant="text|rect|circle" />` with `aria-busy="true"` and reduced-motion fallback (no shimmer animation under `prefers-reduced-motion`).
   - `frontend/src/shared/components/feedback/EmptyState.tsx` — typed `<EmptyState icon title, actionLabel, onAction />` with `role="status"`.
   - `frontend/src/shared/components/feedback/ErrorState.tsx` — typed `<ErrorState title, message, onRetry />` with `role="alert"`.
   - `frontend/src/shared/components/feedback/LoadingBoundary.tsx` — `<LoadingBoundary query={...} children />` that renders Skeleton on `isPending`, EmptyState on `isSuccess && data.length === 0` (or a custom predicate), ErrorState on `isError`, and children otherwise.
2. **Skip link + landmark scaffolding.**
   - `frontend/src/shared/components/layout/SkipLink.tsx` — first focusable element in `<Layout>`, jumps to `#main-content`.
   - `<main id="main-content">` wraps each authenticated view. `<aside aria-label="...">` wraps the sidebar. `<header role="banner">` wraps the topbar (current `<header>` already exists; add role).
3. **320 px responsive collapse.**
   - New Tailwind utility `.ds-compact-collapse` (≤ 639 px) that switches 4-column / 3-column grids to single-column, hides the chrome strip's secondary chips (NodeRedVersion, CompatMode, EdgeModeBadge) into a `<details>` disclosure, and stacks the SecurityView boundary cards vertically.
   - `frontend/src/features/security/components/SecurityView.tsx` applies `.ds-compact-collapse` to its card grid.
   - `frontend/src/features/configuration/components/ReviewChangesPanel.tsx` collapses from a right-side drawer to a bottom sheet at `< sm` breakpoint.
   - `frontend/src/features/dashboard/components/OverviewTiles/{SecurityPosture,BackupHealth,NodeRedHealth}Tile.tsx` already grid-stacked; verify the row + sub-block layouts do not overflow at 320 px.
4. **Loading / empty / error pass on every view.**
   - Every authenticated view renders `<LoadingBoundary>` around its content. Empty / error cases use the typed components above so role + aria-live is consistent.
   - Five views touched: `OverviewView`, `ConfigurationView`, `SecurityView`, `EnvironmentView`, `RecoveryView`. `UsersView`, `LandingView`, `LoginView`, `SetupView` audited but not rewritten (slice G fixes only if there is a blocking bug; otherwise audit notes are filed).
5. **`prefers-reduced-motion` enforcement.**
   - Global rule in `frontend/src/index.css` that zeroes animation-duration, animation-iteration-count, transition-duration, and scroll-behavior under the media query. Includes the sonner toast animations and any framer-motion-like CSS in the chrome strip.
6. **Contrast audit and fixes.**
   - New script `frontend/scripts/check-contrast-tokens.mjs` reads every token used by every DaisyUI theme from `frontend/tailwind.config.js` and computes WCAG 2.1 ratios for `base-content / base-100/200/300`, `primary-content / primary`, `success/warning/danger/info + content` pairs, and the ds-* text tokens from slice B. Writes `frontend/docs/A11Y_CONTRAST_AUDIT.md`.
   - Fix any token that fails. Tokens live in `frontend/tailwind.config.js` or `frontend/src/index.css`; no new token names.
7. **axe-core CI gate.**
   - New spec `frontend/e2e/a11y.spec.ts` that runs `@axe-core/playwright` against Login, Overview, Configuration, Security in both `corporateDark` and `corporateLight`. Any serious or critical violation fails the spec.
   - `playwright.config.ts` includes `a11y.spec.ts` in the default project.
8. **Keyboard contract extension.**
   - Document the keyboard contract for the new components (boundary cards, review drawer, skip link, command palette) in `frontend/KEYBOARD_NAV.md`. Tab/Shift+Tab, Enter/Space, Escape, arrow keys where relevant. Slice G does not change the patterns; it documents them.
9. **i18n keys for feedback components.**
   - `frontend/src/locales/{en,es}/common.json` adds `common:feedback.*` (loadingAria, emptyTitle, emptyMessage, errorTitle, errorRetry, skipToMain, compactViewDisclosure). Existing en/es catalog unchanged.
10. **Tests.**
    - `LoadingBoundary.test.tsx` (≥ 6 tests covering the four render states).
    - `Skeleton.test.tsx`, `EmptyState.test.tsx`, `ErrorState.test.tsx` (≥ 3 tests each).
    - `SkipLink.test.tsx` (≥ 3 tests: visible on focus, jumps to #main-content, hidden when not focused).
    - `SecurityView.compact.test.tsx` (≥ 4 tests using vitest + jsdom with a 320 px viewport stub).
    - `ReviewChangesPanel.compact.test.tsx` (≥ 2 tests covering the bottom-sheet switch at < sm).
    - Updated `DashboardView.test.tsx` and `SecurityView.test.tsx` to assert the LoadingBoundary role + aria-live on the loading path.
    - New `frontend/e2e/a11y.spec.ts` with ≥ 4 axe scans (Login / Overview / Configuration / Security × dark + light = 8, but parameterized as 4 tests × 2 themes).

Out of scope (deferred to follow-up slices):

- Translation of `common:feedback.*` to additional locales (en + es only on this slice).
- AAA contrast (slice G ships AA only).
- Cognitive accessibility / plain-language review (not in the issue body).
- Mobile-app-grade gesture / haptic work (NRCC is a control plane, not a mobile app).
- Screen-reader-specific optimisations beyond what axe-core enforces (e.g., custom VoiceOver rotor instructions).

## Architectural choices (locked in)

- **No new design tokens.** Slice G audits and fixes the existing `corporateDark` / `corporateLight` palettes inside `frontend/tailwind.config.js` and the `--ds-*` CSS variables in `frontend/src/index.css`. New tokens are a follow-up only if a fix cannot be expressed by re-mapping an existing one.
- **`LoadingBoundary` is a view-level wrapper, not a data-fetching primitive.** It accepts a React Query result via a discriminated union (pending | error | empty | success) so each call site decides what "empty" means (`config.users.length === 0`, `backups.length === 0`, etc.). The boundary does not own fetching.
- **axe-core via Playwright, not via vitest.** jsdom lacks layout; Playwright gives us real DOM, real CSS, real themes. Slice G uses `@axe-core/playwright` so the spec reads like any other Playwright spec.
- **320 px first, 1024 px upgrade.** Slice G hard-codes the compact-collapse breakpoint at `max-width: 639px` and tests at 320 px (smallest practical mobile width). The wide layout remains the design source of truth (slice B); the compact layout is a collapse.
- **`prefers-reduced-motion` global rule, no per-component opt-out.** The single rule in `index.css` covers Tailwind animations, sonner toasts, and any future CSS. Components do not need to know about the media query.
- **No new deps unless required.** axe-core already depends on `@axe-core/playwright` which is already a transitive dev dep of Playwright. Slice G adds `@axe-core/playwright` as an explicit dev dep so the import path is stable across Playwright version bumps.
- **No backend changes.** All work is `frontend/**`, `odd/tasks/**`, and CI workflow files. The Go / Docker surface stays untouched.
- **i18n parity.** New i18n keys ship with both `en` and `es` translations. The English-only policy from #768 is preserved (English is the source of truth in source files; locale catalogs carry the operator-facing translation).

## Slice plan

```text
feat/issue-766-slice-g-a11y-responsive-states
├── W1 — Feedback primitives (Skeleton, EmptyState, ErrorState, LoadingBoundary) + tests + i18n
├── W2 — Skip link, main landmark, header role + 320 px ds-compact-collapse utility
├── W3 — Loading / empty / error pass on Overview, Configuration, Security, Environment, Recovery
├── W4 — prefers-reduced-motion global rule + contrast audit script + axe-core e2e spec
└── W5 — docs/odd + status + acceptance evidence + axe-core CI gate wiring
```

## Method

1. **W1 — Feedback primitives.** Build Skeleton, EmptyState, ErrorState, LoadingBoundary with full TypeScript types, vitest tests, and i18n keys. The components are pure; no data fetching. The boundary uses a discriminated union `state: 'pending' | 'error' | 'empty' | 'success'` to render the correct sub-component. RED → GREEN → REFACTOR per primitive.
2. **W2 — Skip link + landmarks + 320 px utility.** Add SkipLink as the first focusable element in `<Layout>`. Wrap each authenticated view's content in `<main id="main-content">`. Add the `ds-compact-collapse` utility to Tailwind. Update SecurityView's card grid to apply it; verify with the 320 px jsdom stub.
3. **W3 — Loading / empty / error pass.** Replace divergent loading / empty / error patterns in the five high-traffic views with `<LoadingBoundary>`. Each call site passes its own React Query result; the boundary decides which sub-component to render. Update existing tests to assert the new roles.
4. **W4 — Reduced motion + contrast audit + axe-core.** Add the global `prefers-reduced-motion` rule to `frontend/src/index.css`. Write the contrast audit script and run it; fix any failures inside existing tokens. Write `frontend/e2e/a11y.spec.ts` with axe-core scans; wire into `playwright.config.ts`.
5. **W5 — docs + status.** Update `KEYBOARD_NAV.md` with the new components' keyboard contract. Update `odd/tasks/issue-766-slice-g-a11y-responsive-states.md` (this file) with the merge evidence. Mark acceptance criteria green.

## Acceptance criteria

- [ ] `frontend/src/shared/components/feedback/{Skeleton,EmptyState,ErrorState,LoadingBoundary}.tsx` exist with full TypeScript types and ≥ 3 vitest tests each (LoadingBoundary ≥ 6).
- [ ] `<LoadingBoundary>` is used by every authenticated view (Overview, Configuration, Security, Environment, Recovery). Five `*.compact.test.tsx` or analogous loading tests assert the role + aria-live on each view.
- [ ] `<SkipLink>` is the first focusable element in `<Layout>` and jumps to `#main-content`. SkipLink.test.tsx has ≥ 3 passing tests.
- [ ] `<main id="main-content">` wraps the content of each authenticated view. Each view's outermost wrapper has a unique `aria-labelledby` referencing the view title.
- [ ] `prefers-reduced-motion` global rule is in `frontend/src/index.css` and verified by a vitest test that asserts the rule exists (or by Playwright with `reducedMotion: 'reduce'`).
- [ ] `frontend/docs/A11Y_CONTRAST_AUDIT.md` exists and reports all primary surface tokens as ≥ AA. Any token that previously failed is fixed in `tailwind.config.js` / `index.css`.
- [ ] `frontend/e2e/a11y.spec.ts` runs `@axe-core/playwright` on Login, Overview, Configuration, Security × both themes. Zero serious or critical violations. The spec is included in the default Playwright project.
- [ ] `frontend/KEYBOARD_NAV.md` documents the keyboard contract for the new components.
- [ ] `frontend/src/locales/{en,es}/common.json` carries the new `common:feedback.*` and `common:layout.*` keys with non-empty translations.
- [ ] `cd frontend && npm run typecheck` reports 0 new errors vs `origin/main`.
- [ ] `cd frontend && npm test -- --run` reports 0 new failing tests vs `origin/main`.
- [ ] `cd frontend && npm run test:e2e` reports 0 new failing tests vs `origin/main` (the a11y spec runs in CI).
- [ ] `cd frontend && node scripts/check-no-hardcoded-i18n.mjs` reports 0 violations.
- [ ] `cd frontend && node scripts/check-contrast-tokens.mjs` reports 0 violations.

## Constraints

- Read **and** modify only `frontend/**`, `odd/tasks/**`, `.github/workflows/**` (for axe-core CI wiring if needed). No backend (`internal/`) changes, no Docker changes, no production Go code.
- One work-unit commit per W-prefixed step. Conventional Commit messages.
- Branch `feat/issue-766-slice-g-a11y-responsive-states` from `origin/main` HEAD (post-#854).
- Worktree: `nrcc-worktrees/issue-766-slice-g-a11y-responsive-states` (created on W1 start).
- New dev deps allowed only with justification: `@axe-core/playwright` (axe-core CI gate; explicitly added). No other deps.
- English source files, English + Spanish locale catalogs.

## TDD

- Mode: enabled
- Runner: `cd frontend && npm test -- --run <file>` for vitest, `cd frontend && npx playwright test e2e/a11y.spec.ts` for axe-core.
- Sequence: RED → GREEN → REFACTOR per primitive and per view rewrite.

## Delivery strategy

`one-slice-one-pr` — same as slices B / C / D / E / F. The slice lands as PR #855 (or whichever number is next) once W1–W5 are green.

## Status

- [x] Slices A + B + C + D + E + F merged.
- [x] Slice G: planning complete.
- [x] Slice G: W1 — feedback primitives (commit `cda616c`).
- [x] Slice G: W2 — skip link, landmarks, 320 px utility.
- [ ] Slice G: W3 — loading / empty / error pass.
- [ ] Slice G: W4 — reduced motion + contrast audit + axe-core.
- [ ] Slice G: W5 — docs + status + acceptance evidence.
- [ ] Slice G: PR open and CI green.
- [ ] Slice G: merged.

## Slice G commit log (work-unit)

> Populated as each W-prefixed step lands. Each row is one Conventional Commit on the feature branch.

| # | Commit | Files | Authored lines | Purpose |
|---|---|---|---|---|
| W1 | `cda616c` | `frontend/src/shared/components/feedback/{Skeleton,EmptyState,ErrorState,LoadingBoundary,loadingBoundaryHelpers,index}.{tsx,ts}` + 4 test files | +443 / -0 (5 components, 25 tests, 0 new deps) | Feedback primitives + tests; pre-existing i18n keys reused |
| W2 | `b3d729c` | `frontend/src/shared/components/a11y/{SkipLink,index}.{tsx,ts}` + test, `layout/{Layout,Header}.tsx`, `configuration/components/ReviewChangesPanel.tsx`, locales en/es | +111 / -4 (1 component, 3 tests, 2 i18n keys, 0 new deps) | Skip link, main landmark, drawer → bottom sheet |
| W2 | (TBD) | `shared/components/layout/SkipLink*`, `features/security/components/SecurityView.tsx`, `tailwind.config.js`, `index.css` | (TBD) | Skip link + landmarks + 320 px utility |
| W3 | (TBD) | five view files + new tests | (TBD) | LoadingBoundary on every authenticated view |
| W4 | (TBD) | `index.css`, `scripts/check-contrast-tokens.mjs`, `e2e/a11y.spec.ts`, `playwright.config.ts`, `package.json` | (TBD) | Reduced motion + contrast audit + axe-core |
| W5 | (TBD) | `KEYBOARD_NAV.md`, `docs/A11Y_CONTRAST_AUDIT.md`, `odd/tasks/issue-766-slice-g-a11y-responsive-states.md` | (TBD) | Docs + status + acceptance evidence |

## Acceptance evidence

> Populated after each W-step lands. Evidence comes from observed test output and CI runs, never from a single local run.

### W1 evidence (commit `cda616c`)

- **TypeScript**: `cd frontend && npm run typecheck` → **0 errors**.
- **Vitest (new tests)**: `cd frontend && npm test -- --run src/shared/components/feedback/` → **25 passed** (7 Skeleton + 6 EmptyState + 6 ErrorState + 6 LoadingBoundary).
- **Vitest (full suite)**: `cd frontend && npm test -- --run` → **572 passed / 2 skipped (83 files)**. No regressions; the 2 skips are pre-existing and unrelated to slice G.
- **ESLint**: `cd frontend && npx eslint src/shared/components/feedback/` → **0 errors / 0 warnings**. The fast-refresh lint warning that initially fired on `LoadingBoundary.tsx` was resolved by extracting `useEmptyState` + `describeError` into `loadingBoundaryHelpers.ts` (mirrors the `securityCenterHelpers.ts` pattern from slice E).
- **Hardcoded copy guard**: `cd frontend && node scripts/check-no-hardcoded-i18n.mjs` → **0 violations**. The four new components reuse existing `common:loading`, `common:errorOccurred`, `common:tryAgain` keys; no new catalog entries needed.
- **Authored lines (W1)**: +443 / -0 across 10 files (5 components + 4 test files + 1 barrel + 1 helpers module).

### W2 evidence (commit `b3d729c`)

- **TypeScript**: `cd frontend && npm run typecheck` → **0 errors**.
- **Vitest (new tests)**: `cd frontend && npm test -- --run src/shared/components/a11y/` → **3 passed**.
- **Vitest (full suite)**: `cd frontend && npm test -- --run` → **575 passed / 2 skipped (84 files)**. No regressions; the +3 vs W1 (572) is exactly the SkipLink tests.
- **ESLint**: 0 errors / 0 warnings on touched paths (`a11y/`, `layout/`, `ReviewChangesPanel.tsx`, `feedback/`).
- **Hardcoded copy guard**: `cd frontend && node scripts/check-no-hardcoded-i18n.mjs` → **0 violations**.
- **i18n coverage**: `cd frontend && pnpm i18n:coverage` → **100% ES coverage (851/851)**. W2 added `common:layout.skipToContent` in both EN and ES with a localised string; no key parity drift.
- **Authored lines (W2)**: +111 / -4 across 8 files (1 new component + 1 test + 1 barrel + 4 layout/drawer edits + 2 locale updates).
- **Dependencies**: **0 new**. Reused `lucide-react`, `react-router-dom`, `@/i18n`.

#### W2 audit findings (carried into scope)

- **`<main>` already existed** in `Layout.tsx`; only the `id` + `tabindex={-1}` were missing. SkipLink + `<main id="main-content" tabindex={-1}">` are the canonical a11y pairing; the `tabindex` is required so the browser moves focus to the landmark, not just scrolls.
- **`<header>` already existed** with implicit `role="banner"` semantics (HTML5 maps it inside `<body>`). The explicit `role="banner"` was added for testability and to survive any future DOM restructuring.
- **`ReviewChangesPanel` used a fixed `max-w-xl` drawer**; the slice F impl assumed ≥ 768 px. The bottom-sheet collapse at `< sm` (640 px) keeps the desktop drawer intact and only restyles below the breakpoint.
- **The Header's runtime-context chips already collapse via `hidden md:flex`** (slice C). Slice G does not introduce a `<details>` disclosure in W2; that refactor is deferred to W3 when call sites move to `LoadingBoundary` and the chips become data-driven.
- **No `ds-compact-collapse` token needed.** Tailwind's default mobile-first breakpoints cover 320–639 px via the unprefixed utility space; the W2 changes use `sm:` and the negation pattern, not a new token name.

## Follow-ups (tracked here)

- Slice H — cleanup of legacy i18n keys (e.g. `dashboard:runtimeCard.*`, `dashboard:diskUsage.*`, `securityCenter.save`) once no consumer references them.
- AAA contrast pass if a regulatory driver appears.
- Locale parity beyond en + es when the i18n roadmap (#767 follow-ups) opens additional locales.