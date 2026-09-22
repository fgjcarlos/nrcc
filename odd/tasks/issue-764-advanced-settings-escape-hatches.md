# Issue 764 — Safe advanced settings escape hatches (scout)

## Issue body recap

GitHub issue **#764** "feat(configuration): add safe advanced settings escape hatches"
(`status:approved`, OPEN). Scope:

- Curated callbacks/middleware and SSO/reverse-proxy presets with explicit trust
  and channel boundaries.
- `functionGlobalContext`, `nodeDefaults`, WebSocket/TLS options and
  version-cataloged long-tail settings.
- Read-only display + managed patch boundaries for existing custom code.
- Preview, validation, backup, ownership markers, escape back to the original
  source.

Non-goals: arbitrary JS execution in the browser, full visual programming,
snippets from unknown Node-RED major treated as compatible.

Dependencies (already merged/closed):
- #756 versioned long-tail catalog — closed
- #757 executable-source preservation — merged (PRs #777–#781)
- #758 validated transactional apply — merged (slice A/B/C in main)
- #760 auth surfaces separation — CLOSED
- #761 dashboard access surfaces — CLOSED

Acceptance tests called out in the issue:
- `TestAdvancedPatchPreservesUnmanagedCode`
- `TestPresetSurfaceContract`
- `FunctionGlobalContextAndNodeDefaultsFixtureSuite`
- `AdvancedSettingsRollbackE2E`
- Security review confirms previews/logs contain no credential or
  executable-source leakage beyond the authorized view.

## Existing infrastructure already in main

The "easy" parts of #764 already exist; the work is incremental on top.

### Backend (Go)

| File | LOC | Already supports |
| --- | --- | --- |
| `internal/handler/settings.go` | 205 | GET/POST `/api/settings/raw`, admin-gated via `middleware.RequireAdmin`, routes through `ApplyCoordinator` when wired. |
| `internal/service/source_patch.go` | 294 | **Source-preserving managed edits.** `ApplyScalarEdit`, `ApplyBlockEdit`, batch `SourcePatch` with rollback. 21-entry `managedSettingKeys` list already includes `functionGlobalContext`, `editorTheme`, `credentialSecret`, etc. Unmanaged regions round-trip byte-stable. |
| `internal/service/apply_diff.go` | 291 | **Audit redaction.** `RedactSettingsContent`, `RedactDiff`. `secretSettingKeys` already covers `credentialSecret`, `functionGlobalContext`, `adminAuth`, `httpNodeAuth`, `httpStaticAuth`, `https`, plus `dashboard`. |
| `internal/service/apply.go` | 341 | **Apply pipeline** validate → backup → atomic write → audit (slice A of #758). |
| `internal/service/apply_coordinator.go` | 253 | **Single-flight orchestration** with revision precondition + readiness integration. |
| `internal/service/testdata/nodered-5.0.6-catalog.json` | 23 | Version catalog: 21 entries (key + shape + secret flag + version). Already feeds `secretSettingKeys`. |
| `internal/service/settings_sandbox.go` | 232 | goja sandbox for parsing `adminAuth` (NOT directly relevant to #764 but is a precedent for bounded-JS execution). |
| `internal/service/source_patch_test.go` + `source_patch_fixture_test.go` | 367 + 678 | Comprehensive coverage of source-preservation guarantees — directly extensible for the new test cases. |
| `internal/handler/backup_test.go` | — | Already mentions "preset" as a concept (used in backup context). No preset/recipe module exists yet. |

### Frontend (React)

| File | LOC | Already supports |
| --- | --- | --- |
| `frontend/src/features/configuration/components/ConfigurationView.tsx` | ~430+ | Tabbed view: Basic, Auth, Security, Logging, Editor, AI Provider. Plus an existing **"Advanced settings.js"** card with a locked textarea (issue #364 unlocked-on-confirm). This is the natural insertion point for #764 presets. |
| `frontend/src/features/configuration/components/SecurityCenter.tsx` | 9.7K | Admin-gated surface that mirrors the role + preview + backup flow #764 needs. |
| `frontend/src/features/configuration/lib/configTransformers.ts` | — | Existing form↔config converters; will need a presets adapter. |
| `frontend/src/features/configuration/types/index.ts` | — | Re-exports shared types; new preset types fit here. |
| `frontend/src/shared/constants/uiCopy.ts` | — | `advancedSettingsTitle` / `advancedSettingsDescription` strings already exist. |

## What #764 needs that does NOT exist yet

1. **Preset contract** — a Go-side type plus catalog extension that says "preset X touches editor surface A, HTTP surface B, Socket.IO surface C" with trust + channel boundaries. (Today each setting is a flat catalog entry with no preset grouping.)
2. **Preset registry** — curated recipes per Node-RED 5 surface (functionGlobalContext sub-options, nodeDefaults, https, WebSocket options). Each recipe emits one or more `SourceEdit` blocks and is bounded by the surface contract.
3. **Preview + diff UI** — per-preset apply preview that shows exactly what bytes will change, redacts secrets, and exposes a "back to source" escape hatch that reverts the preset's managed keys without touching unmanaged code.
4. **Patch boundary verification** — Go-side test that proves an unrelated structured edit does NOT touch unmanaged callbacks/requires. Maps directly to `TestAdvancedPatchPreservesUnmanagedCode`.
5. **Preset surface contract test** — Go-side test that asserts each preset declares the editor/HTTP/Socket.IO surfaces it touches and never crosses the boundary. Maps to `TestPresetSurfaceContract`.
6. **Rollback E2E** — integration test that drives an invalid preset through the apply pipeline and asserts the source is restored byte-stable. Maps to `AdvancedSettingsRollbackE2E`.
7. **Fixture corpus for functionGlobalContext + nodeDefaults** — the `testdata/` already has callback-middleware and with-extensions fixtures; we need a parallel corpus covering the supported structured cases and the executable/unknown cases. Maps to `FunctionGlobalContextAndNodeDefaultsFixtureSuite`.
8. **Advanced UI surface (frontend)** — a new tab/card that lists presets grouped by surface, each with apply/preview/back-to-source controls. The existing "Advanced settings.js" textarea remains the final escape hatch for fully arbitrary edits.

## Slice candidates (feature-branch-chain)

Slice sizing target: < 400 authored lines each, independently testable.

### Slice 1 — Preset contract + registry core
- **Files**: `internal/model/preset.go` (new), `internal/service/preset_registry.go` (new),
  `internal/service/preset_registry_test.go`, `internal/service/testdata/preset-fixtures/*.js`,
  extensions to `internal/service/testdata/nodered-5.0.6-catalog.json` if needed.
- **What it delivers**: typed `Preset` struct, surface declaration
  (`Editor`, `HTTP`, `SocketIO`), trust tier (`managed`, `curated`), channel
  boundary. First 4 presets: `https-tls-preset`, `functionGlobalContext-strict`,
  `functionGlobalContext-relaxed`, `logging-callback-middleware`.
- **Acceptance**: `TestPresetSurfaceContract` passes; registry returns presets
  with their declared surface and trust tier.
- **Estimated lines**: ~350.
- **Worktree**: `feat/issue-764-preset-contract`.

### Slice 2 — Apply-preview, rollback, unmanaged-region guarantees
- **Files**: `internal/service/preset_apply.go` (new),
  `internal/service/preset_apply_test.go`,
  `internal/service/apply_diff.go` (extend `RedactDiff` for presets),
  extensions to `internal/service/source_patch_test.go` and
  `source_patch_fixture_test.go` for the new fixtures.
- **What it delivers**: `PresetApply` that produces a redacted preview, applies
  through `SourcePatch`, and verifies byte-stable unmanaged regions.
  Implements `TestAdvancedPatchPreservesUnmanagedCode` and
  `FunctionGlobalContextAndNodeDefaultsFixtureSuite`.
- **Acceptance**: rollback path round-trips the original bytes; secrets never
  appear in the preview; structured cases round-trip; executable/unknown cases
  remain preserved.
- **Estimated lines**: ~380.
- **Worktree**: `feat/issue-764-preset-apply` (depends on slice 1).

### Slice 3 — Advanced UI surface + AdvancedSettingsRollbackE2E
- **Files**: `frontend/src/features/configuration/components/AdvancedSettings.tsx`
  (new), `frontend/src/features/configuration/components/AdvancedSettings.test.tsx`,
  extensions to `frontend/src/features/configuration/components/ConfigurationView.tsx`,
  `frontend/src/features/configuration/types/index.ts`,
  `frontend/src/shared/constants/uiCopy.ts`,
  new Playwright spec `frontend/e2e/advanced-rollback.spec.ts`.
- **What it delivers**: the user-facing preset card with grouped recipes,
  preview/back-to-source controls, and the E2E rollback flow.
- **Acceptance**: `AdvancedSettingsRollbackE2E` passes; preview shows redacted
  diff; back-to-source button restores the original settings.js verbatim.
- **Estimated lines**: ~390.
- **Worktree**: `feat/issue-764-advanced-ui` (depends on slice 2).

### Tracker branch
- `feat/issue-764-presets` — fast-forwards to `origin/main`, merges slices in
  order, only the tracker reaches `main`.

## Risks / open questions

1. **Catalog schema extension.** The existing `nodered-5.0.6-catalog.json` is a
   flat list with `key`, `shape`, `secret`, `introduced_in`. Slice 1 needs to
   decide whether preset metadata lives in the same JSON (with a `presets`
   section) or in a separate file (`preset-catalog.json`). Recommendation:
   separate file to keep the per-setting catalog small and immutable.
2. **`https` block surface.** The `https` setting is already a managed key but
   its inner shape (key/cert/passphrase) is large. The first preset
   (`https-tls-preset`) should be opinionated and document its boundaries,
   not cover the full shape.
3. **`nodeDefaults` is NOT in the existing `managedSettingKeys` list.** Slice 1
   needs to either add it (changes the source-patch contract) or treat it as
   a strictly preset-only key (only writable through the preset registry).
   Recommendation: add to managed keys with a dedicated `IsPresetOnly()`
   predicate so the source-patch contract stays narrow.
4. **Security review.** The acceptance test calls out "Security review confirms
   previews and logs contain no credential values or executable-source
   leakage beyond the authorized view." The redaction is already in
   `apply_diff.go`, but the preview UI is new — slice 3 needs a security
   review pass before merge.
5. **Test runner coverage.** Frontend tests run via `npm test -- --run` from
   `frontend/`; Go tests via `go test ./...` from repo root; Playwright E2E
   in `frontend/e2e/`. The odd doc template uses the same pattern.
6. **RDD/gentle-ai availability.** Per the #763 odd doc, `gentle-ai` could not
   be resolved in this worktree last time. RDD is not assumed for #764.

## Recommended plan

1. Open slice 1 worktree (`feat/issue-764-preset-contract`), implement
   `internal/model/preset.go` + `internal/service/preset_registry.go` + tests,
   work-unit commit.
2. Open slice 2 worktree (`feat/issue-764-preset-apply`) branched from slice 1,
   implement apply + preview + rollback tests, work-unit commit.
3. Open slice 3 worktree (`feat/issue-764-advanced-ui`) branched from slice 2,
   implement frontend + E2E, work-unit commit.
4. Open tracker (`feat/issue-764-presets`) on origin/main, merge slices in
   order, push, open single PR to main.
5. Close #764 with reference to the tracker PR.

Stop conditions: if slice 1 exceeds 400 authored lines without including the
catalog extension, split it further; if slice 2 forces a UI preview in Go
test, defer that to slice 3.
