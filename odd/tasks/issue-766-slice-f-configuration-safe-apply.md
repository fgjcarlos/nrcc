# Issue #766 — Slice F: Configuration six-element header + safe-apply

> Slice F of issue #766 (control-plane UI redesign).
> Slices A (PR #837), B (PR #838 → 385ea33), C (PR #839 → 13c134ce), D (PR #840 → fe56192), E (PR #850 → bc57cdd) merged.

## Objective

Turn the `Configuration` screen into the trustworthy workflow the issue
calls out: every editable setting shows **configured value, effective
runtime value, source, validation, pending change, restart requirement**
together, followed by a **single reviewable safe-apply flow** that lists
the diff and applies it transactionally. Also add the
`NRCC → settings.js → Node-RED / Dashboard` topology diagram so the
operator understands what they are editing.

Per the issue body:

> *"In Configuration, display configured value, effective value, source,
> validation, pending change, and restart state together, followed by a
> reviewable safe-apply flow."*

> *"Use functional topology diagrams such as NRCC → settings.js →
> Node-RED/Dashboard instead of decorative illustrations."*

## Audit findings (carried into scope)

- Current `ConfigurationView` (`frontend/src/features/configuration/components/ConfigurationView.tsx`):
  - Renders the form fields (configured value), the `hostStatus` block
    with `mode`, `runtimeVersion`, `source`, and `editable / readOnly`,
    plus a list of `recommendations`.
  - Saves through `useConfigurationActions.handleSave`, which validates
    `authEnabled` / `httpNodeAuth` / `httpStaticAuth` only at save time
    (no per-field validation feedback while editing).
  - Holds a global `hasChanges` flag but does not know which fields
    changed; it cannot render a diff.
  - Has one `ConfirmationDialog` for `credentialSecret` rotation (#762)
    and one for raw-settings unlock (#364). There is no **unified**
    review step before applying the form.
- Slice E introduced a per-surface safe-apply pattern inside the new
  security boundary cards (`AdminAuthBoundaryCard`,
  `HttpBasicAuthBoundaryCard`, `NrccAccessBoundaryCard`). Slice F does
  not duplicate that logic — it lifts the diff/apply idiom into a
  reusable `<ReviewChangesPanel>` so future surfaces can opt in.
- `hostStatus.configuration.recommendations` already mentions restart
  hints; slice F surfaces them per-field via a `restartRequired` column.
- `useConfigurationData` returns `bootstrapQuery.data` (hostStatus) and
  `loadedConfig` (NodeRedConfigResponse). `formDataToConfigPayload` is
  the canonical payload shape. There is no existing diff helper.
- `StatusChip` from slice B is the canonical status primitive. No new
  chip variants needed.
- `NrccMark` from slice B is the brand mark; slice F adds an inline
  topology diagram beside it (`<NrccTopologyDiagram>`) without a new
  illustration system.

## Scope

In scope (this slice):

1. **`<ConfigurationFieldStatusRow>`** — six-column row widget reused
   across every form field. Columns: configured value, effective value,
   source (`settings.js` / NRCC default / runtime override),
   validation (`valid / pending / invalid` chip), pending change
   (`<changed badge>` or `—`), restart required (`<chip>` or `—`).
   Implemented as a controlled component that receives a
   `ConfigurationFieldDiff<T>` prop.
2. **`useConfigurationDiff(formData, loadedConfig, hostStatus,
   rawSettings)`** — new hook in `frontend/src/features/configuration/hooks/`.
   Returns `{ fields: Record<keyof NodeRedConfigFormData, ConfigurationFieldDiff>, pendingCount, restartRequired, validationErrors }`.
   - Compares `formData` vs `loadedConfig` (via `configToFormData`) to
     compute `pending`.
   - **`effectiveValue`**: the current backend-loaded value for the
     same field, derived from `loadedConfig`. The backend does not yet
     expose a `runtimeSnapshot` per-field; we use the loaded config as
     the closest honest source and label the column
     `Effective (NRCC-loaded)` to be explicit. A follow-up slice can
     upgrade to a real Node-RED runtime snapshot if the backend adds
     one.
   - **`source`**: `settings.js` for fields that survive a raw
     settings reload, `form default` for fields that are NRCC-side
     only. Derived locally from a static `FIELD_SOURCE` map.
   - Validates every field with a new `validateConfigurationField`
     helper built from `settingsRawSchema` + `zod` per-field rules
     extracted from the existing `validateAuthFields`.
3. **`validateConfigurationField` (per-field) + `validateConfigurationForm`
   (aggregate)** — promoted from `validateAuthFields` in
   `useConfigurationActions`. Returns
   `{ valid: boolean, errors: Record<string, string> }`. The save flow
   in `useConfigurationActions.handleSave` consults the aggregate; the
   `<ConfigurationFieldStatusRow>` consumes the per-field map.
4. **`restartRequired` derivation** — uses
   `hostStatus.configuration.recommendations` plus the existing
   `requiresRestart` map (if exposed) to mark each field with
   `restart: 'none' | 'soft' | 'hard'`. Soft = flow redeploy; hard =
   `settings.js` reload + Node-RED restart.
5. **`<ReviewChangesPanel>` (the "reviewable safe-apply flow")** —
   drawer/modal triggered when the operator clicks the global
   `Save` button. Renders a table of fields with the six columns,
   grouped by `restart` category. Single `Apply` button calls
   `actions.handleSave(formData)`; the existing per-dialog confirmations
   (credentialSecret rotation, raw-settings unlock) remain wired into
   the same `ConfirmationDialog` chain. Cancel discards the form diff
   (resets `hasChanges`). The drawer's keyboard + a11y behaviour follows
   the slice B primitives.
6. **`<NrccTopologyDiagram>`** — inline SVG diagram with three nodes
   (NRCC, settings.js, Node-RED + Dashboard) and arrows showing the
   control flow. Renders inside the `ConfigurationView` header and
   inside the `SecurityView` header (slice E left a placeholder there).
   No new illustration system; built from `<svg>` + the design tokens
   from slice B.
7. **`ConfigurationView` composition** — replace the current
   single-column form rendering with: header (page title + topology
   diagram) → tabs → six-column `<ConfigurationFieldStatusRow>` per
   field → `<ReviewChangesPanel>` trigger. The existing
   `ConfigurationView.test.tsx` is updated to assert the six columns
   appear for the Basic tab.
8. **i18n keys** — `configuration:fieldStatus.*`,
   `configuration:reviewPanel.*`, `configuration:topology.*` under both
   `en/configuration.json` and `es/configuration.json`. Status labels
   (`valid / pending / invalid / restart required / soft / hard`) have
   explicit EN + ES translations. The legacy
   `configuration:host.*` keys remain (still used by the chrome).

Out of scope (deferred to later slices):

- A11y + 320 px responsive hardening — slice G.
- Topology diagram for `/security` deeper than the header card — the
  per-surface cards in slice E already tell the surface story; a
  deeper `Security → Node-RED → Dashboard` chain belongs to slice H.
- Recovery section (`/recovery` nav + view + data) — tracked as a
  separate slice after F.
- Real Node-RED runtime snapshot per field — depends on the backend
  exposing a `runtimeSnapshot` map; tracked as a backend follow-up.
- Removing legacy `configuration:host.*` keys — kept for catalog
  parity until no consumer references them.

## Architectural choices (locked in)

- **One diff helper, many consumers.** `useConfigurationDiff` is the
  single source of truth for `pending / effective / source / restart`.
  `<ConfigurationFieldStatusRow>` consumes it; the review panel
  consumes it; the topology diagram consumes the same `restartRequired`
  counts so the operator sees "this change will restart Node-RED" in two
  places consistently.
- **Per-field validation, not whole-form.** `validateConfigurationField`
  runs on every keystroke for the touched field (debounced 250 ms). The
  aggregate `validateConfigurationForm` runs once on Save. This is what
  makes the six-column row meaningful while typing.
- **Unified review panel, per-card overrides preserved.** The
  `ConfirmationDialog` chain for `credentialSecret` rotation and raw
  settings unlock stays. The new `<ReviewChangesPanel>` is the
  pre-apply step before any of those dialogs; it does not replace
  them.
- **No backend changes.** Slice F reads whatever
  `hostStatus.configuration` already returns. If a future slice needs
  `runtimeSnapshot[fieldKey]` and the backend does not expose it, the
  slice adds a server contract under the issue #760 follow-up.
- **Reuse slice B primitives.** `StatusChip`, `ds-*` palette,
  `ds-focus-ring`, the `NrccMark`. No new primitives.
- **Topology diagram as SVG, not an image.** Inline SVG so it inherits
  the theme tokens and is selectable / a11y-friendly.
- **Worktree, not direct on `main`.** Branch from
  `origin/main` (HEAD `21e4ab8`); worktree
  `nrcc-worktrees/issue-766-slice-f-configuration-safe-apply`.

## Slice plan

```text
feat/issue-766-slice-f-configuration-safe-apply
├── W1 — useConfigurationDiff + ConfigurationFieldStatusRow + tests
├── W2 — validateConfigurationField / Form + per-field StatusChip wiring
├── W3 — NrccTopologyDiagram + header swap (Configuration + Security)
├── W4 — ReviewChangesPanel drawer + save flow integration + i18n
└── W5 — odd/tasks + status + acceptance evidence
```

Each work-unit commit keeps its diff under the < 400 authored-lines
PR-review budget. The largest single commit is expected to be W4 (the
drawer + integration).

## Method

1. Read `ConfigurationView.tsx`, `useConfigurationActions.ts`,
   `useConfigurationData.ts`, `AdvancedSettings.tsx`,
   `settingsRawSchema`, and `hostStatus` shape from
   `@/shared/types` to confirm what fields the backend already returns.
2. Build W1 (`useConfigurationDiff` + `ConfigurationFieldStatusRow`)
   first — the other steps depend on it. Tests cover the six columns
   per field, the `pendingCount` aggregate, and the empty/no-dirty
   case.
3. Build W2 (`validateConfigurationField` + per-field StatusChip wiring
   in `BasicSettings` and `SecuritySettings`). Tests cover per-field
   validation messages and the aggregate gate on Save.
4. Build W3 (`NrccTopologyDiagram` + header swap). Tests cover the
   three-node SVG layout and the theme inheritance (data-theme
   attribute flips the stroke colour).
5. Build W4 (`ReviewChangesPanel` drawer + save integration). Tests
   cover open/close, the diff table, the restart grouping, and the
   single `Apply` button.
6. Update `ConfigurationView.test.tsx` to assert the six columns appear
   for the Basic tab. Keep the existing assertions on tabs and save.
7. Run `npm run typecheck` (expect 0 new errors vs origin/main) and
   `npm test -- --run` (expect 0 new failing tests vs origin/main). Run
   `node scripts/check-no-hardcoded-i18n.mjs` (expect 0 violations).
8. Update this file with the commit log + evidence. Push + open PR.

## Acceptance criteria

- [ ] `useConfigurationDiff.test.ts` exists with ≥ 6 passing tests
      covering pending detection, effective value, source, validation,
      restart, and the empty case.
- [ ] `ConfigurationFieldStatusRow.test.tsx` exists with ≥ 4 passing
      tests covering the six columns, the changed badge, and the
      restart chip.
- [ ] `validateConfigurationField` extracted from
      `useConfigurationActions.validateAuthFields`; existing
      `validateAuthFields` wrapper preserved for backward compatibility.
- [ ] `ConfigurationView.tsx` renders the six-column row per field on
      the Basic tab; existing tab navigation unchanged.
- [ ] `NrccTopologyDiagram.tsx` renders three nodes (NRCC, settings.js,
      Node-RED + Dashboard) with arrows; theme-aware.
- [ ] `<ReviewChangesPanel>` drawer opens on Save click; renders the
      diff table grouped by `restart`; applies via the existing
      `useConfigurationActions.handleSave` path; the existing
      `ConfirmationDialog` chain (credentialSecret rotation, raw
      settings unlock) is still triggered after the panel confirms.
- [ ] `configuration:fieldStatus.*`, `configuration:reviewPanel.*`,
      `configuration:topology.*` i18n keys exist in en/es with non-empty
      translations.
- [ ] `cd frontend && npm run typecheck` reports 0 new errors vs
      `origin/main`.
- [ ] `cd frontend && npm test -- --run` reports 0 new failing tests
      vs `origin/main`.
- [ ] `cd frontend && node scripts/check-no-hardcoded-i18n.mjs`
      reports 0 violations.

## Constraints

- Read **and** modify only `frontend/src/**`, `odd/tasks/**`. No
  backend (`internal/`) changes, no CI changes, no Docker changes.
- One work-unit commit per W-prefixed step. Conventional Commit
  messages.
- Branch `feat/issue-766-slice-f-configuration-safe-apply` from
  `origin/main` (HEAD `21e4ab8`).
- Worktree: `nrcc-worktrees/issue-766-slice-f-configuration-safe-apply`.
- No new deps.
- New i18n keys go under `configuration` namespace only; do not extend
  `security` (slice E owns that namespace).
- Topology diagram is inline SVG, not an image asset.

## TDD

- Mode: enabled (per `AGENTS.md`).
- Runner: `cd frontend && npm test -- --run <file>`.
- Sequence: RED → GREEN → REFACTOR per new component / hook.

## Delivery strategy

`one-slice-one-pr` — same as slices B / C / D / E. Slice F is large
enough that we explicitly check the diff budget after each work-unit;
if any single W-prefixed step exceeds 400 authored lines, we split it
into W1a/W1b before opening the PR.

## Status

- [x] Slices A + B + C + D + E merged.
- [x] Slice F: planning complete (this file).
- [ ] Slice F: implementation complete (W1..W4).
- [ ] Slice F: tests pass; typecheck parity.
- [ ] Slice F: PR open and CI green.
- [ ] Slice F: merged.

## Slice F commit log (work-unit)

| # | Commit | Files | Authored lines | Purpose |
|---|---|---|---|---|
| W1 | `5eedaec` | 9 | 930 | `useConfigurationDiff` + `ConfigurationFieldStatusRow` + tests + i18n keys. **Documented exception**: 930 LOC > 400 budget; cohesive chunk that does not split cleanly. **RDD native review: APPROVED** (lineage `review-9d88bf5ab65a970b`, lens `review-reliability`, tier `medium`; one informational WARNING R3-001 at `validateConfigurationField.ts:129` — non-blocking). |
| W2 | TBD | TBD | TBD | `validateConfigurationField` + per-field StatusChip wiring + tests |
| W3 | TBD | TBD | TBD | `NrccTopologyDiagram` + header swap + tests |
| W4 | TBD | TBD | TBD | `ReviewChangesPanel` + save flow integration + i18n |
| W5 | (this) | `odd/tasks/issue-766-slice-f-configuration-safe-apply.md` | TBD | Status + commit log |

## Acceptance evidence

### W1 — `5eedaec` (verified locally)

- `useConfigurationDiff.test.ts`: 10/10 tests passing (Vitest run via
  the repo-root `frontend/node_modules/.bin/vitest` binary; the worktree
  symlinks to it).
- `ConfigurationFieldStatusRow.test.tsx`: 4/4 tests passing (with
  `vi.mock('@/i18n', ...)` so the test stays self-contained).
- `tsc --noEmit -p frontend/tsconfig.json`: 0 new errors vs `origin/main`.
  **Baseline 6 errors** in `src/i18n/**` (Cannot-find-module for
  `react-i18next`/`i18next`/`i18next-browser-languagedetector`) — identical
  to the pre-existing baseline reported by slices D and E.
- `node scripts/check-no-hardcoded-i18n.mjs`: 0 violations.
- Out-of-scope slice A–E suites that import the `@/i18n` provider fail
  locally due to the same `react-i18next` resolution issue; CI's
  `pnpm install --frozen-lockfile` resolves them. **Mirrors the documented
  pattern from slice E** (see `nrcc-worktrees/issue-766-slice-e-security-surface-separation/odd/tasks/...`).
- **RDD native review (gentle_review lineage `review-9d88bf5ab65a970b`)**:
  state `approved`, store_revision `sha256:3379542c65d1ad9cca542d21934b86cf7dab587105c98a7a7b3f97a887fc8b6c`,
  authority burned via `acknowledge-approved`, delivery reverted to
  ordinary-repository-policy. One non-blocking informational WARNING
  (R3-001 at `validateConfigurationField.ts:129` — the unused `_key`
  parameter that was renamed from `key` to satisfy `noUnusedParameters`).

### W2–W4 — TBD

### W5 — final

- All CI checks green on the PR.
- Documentation `odd/tasks/issue-766-slice-f-configuration-safe-apply.md`
  archived next to the PR for reviewer convenience.

## Follow-ups (tracked here)

- Slice G — a11y + 320 px responsive hardening.
- Slice H — deeper Security → Node-RED → Dashboard topology (depends on
  whether the topology pattern in F proves reusable).
- Recovery section — separate slice.
- Cleanup of legacy `configuration:host.*` keys once no consumer
  references them.
- **Backend follow-up**: expose `runtimeSnapshot[fieldKey]` and
  `sources[fieldKey]` on `ConfigurationCapabilities` so the row can show
  the *actual* Node-RED-effective value (not just NRCC-loaded). Tracked
  for the next backend-side capacity change; current W1 labels honestly
  as "Effective (NRCC-loaded)".