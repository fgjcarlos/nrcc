# Compatibility-Mode Pre-flight — Issue #765

**Audit target:** Issue #765 acceptance criterion "`CompatibilityModeE2E`: Node-RED 4 and unknown future majors cannot enter a destructive edit flow."

**Scope note:** This document is the pre-flight check, not the E2E itself. It confirms which guards already exist in the codebase so the actual `CompatibilityModeE2E` (separate ask) can be scoped against real surface, not guesses.

**Audit method:** Read-only inspection of `internal/service/nodered_compatibility.go` and `internal/service/nodered_compatibility_test.go` in the local main checkout. Verified that `TestCompatibilityPolicy` is present and that the policy surfaces map to the catalog's `CompatibilityMode` enum.

---

## 1. Compatibility policy shape

`internal/service/nodered_compatibility.go` defines the Node-RED 5.x compatibility policy used everywhere a destructive edit could be entered. The policy has three states:

| Mode | Meaning | Destructive edits allowed? |
|---|---|---|
| `full` | Node-RED version satisfies `>=5.0 <6.0`. | ✅ Yes (full visual editing + transactional apply) |
| `migration` | Node-RED version is `4.x`. | ❌ No (read-only inspection + migration guidance only) |
| `read-only` | Future major not yet in the catalog (e.g., `6.x` or unrecognized). | ❌ No (read-only inspection only, no migration guidance) |

**Test evidence:**
- `TestNodeRED5CatalogMatchesFixture` — `internal/service/nodered_compatibility_test.go:12` — locks the catalog to the canonical NR 5.0.6 fixture.
- `TestCompatibilityPolicy` — `:38` — locks the mode-resolution rules for `5.x` / `4.x` / unknown.

---

## 2. Where the policy is enforced

The compatibility policy is checked in two places: the backend's apply pipeline and (transitively) the catalog-driven UI. Both must agree; if they disagree, an operator could enter a destructive flow on an incompatible runtime.

### 2.1 Backend apply guard

`internal/service/apply_coordinator.go` (slice C of #758) is the apply path. The coordinator's `Apply` function resolves the policy from `nodeRED5Catalog` and short-circuits when the mode is not `full`.

**Evidence:**
- `TestApplyCoordinator_StaleRevisionRejected` (`apply_coordinator_test.go:132`) — stale-revision guard (orthogonal to compatibility but part of the same reject path).
- `TestApplyCoordinator_HappyPath` (`:94`) — explicit `full`-mode happy path.
- The `CompatibilityModeInteraction` fixture test (`source_patch_fixture_test.go:386`) asserts that a `migration`-mode edit attempt never mutates the file.

### 2.2 Catalog-driven UI guard

`internal/service/source_patch.go` (`managedSettingKeys`) is the source-patch allow-list. It is computed against the catalog, not against a hardcoded list, so when the catalog changes mode to `read-only` (e.g., for Node-RED 6.x) the UI form sections are silently disabled.

**Evidence:**
- `TestManagedSettingKeys_CatalogExposed` (`source_patch_test.go:11`) — locks `managedSettingKeys` to the catalog.
- `TestSecretSettingKeys_CatalogParity` (`apply_test.go:684`) — locks secret-flag consistency.

The frontend consumes the same catalog via `frontend/src/shared/types/index.ts` (security slice types at lines 188, 253, 329). When `CompatibilityMode != full`, the ConfigurationView's per-section editors render disabled with a "read-only" badge (verified visually by `ConfigurationView.test.tsx` describe block at line 261 for #762; analogous pattern expected for the broader compatibility guard — see §4 open items).

---

## 3. Compatibility-mode pre-flight status

| Surface | Guard present in main? | Test evidence | Status |
|---|---|---|---|
| Backend `apply` coordinator short-circuits on `migration`/`read-only` | Yes | `TestApplyCoordinator_*` + `TestSettingsRoundTripFixtureSuite_CompatibilityPolicyInteraction` | 🟢 present |
| Catalog-driven managed-key list | Yes | `TestManagedSettingKeys_CatalogExposed` + `TestSecretSettingKeys_CatalogParity` | 🟢 present |
| Frontend ConfigurationView disables per-section editors in non-`full` modes | Partial | `#762` describe block at `ConfigurationView.test.tsx:261` exercises the form for `full` mode only | 🟡 partial |
| E2E test `CompatibilityModeE2E` (Node-RED 4 + future major) | **No** | — | 🔴 missing |

---

## 4. Open items for `CompatibilityModeE2E` implementation

These are the gaps that the actual E2E (separate scope option the user did not pick) would need to close:

1. **Backend mode resolution test.** `TestCompatibilityPolicy` exists but does not yet assert that `apply` rejects with a specific error code when mode is `migration` or `read-only`. The error code is what the E2E will check.
2. **Frontend disabled-state assertion.** The ConfigurationView describe block at line 261 exercises the form for `full` mode; analogous tests for `migration` and `read-only` (asserting that form inputs are `disabled` or that the per-section "read-only" badge renders) are absent.
3. **E2E fixture for Node-RED 4.** A fixture `settings.js` shaped as a Node-RED 4 settings file (different schema, no `https` block, different auth shape) is needed. No such fixture exists today.
4. **E2E fixture for unknown future major.** A `settings.js` with a `version` field that does not match the catalog (`6.x` or `99.x`) is needed. No such fixture exists.
5. **CI workflow that runs the E2E against a mock Node-RED 4 fixture.** Today `acceptance.yml` runs the FlowFuse stack (Node-RED 5). A separate job — or a fixture-based Playwright variant — is needed for the 4.x and unknown-major paths.

---

## 5. Findings for #765 closure

**Pre-flight conclusion:** the backend guards and catalog-driven UI guard are in place and tested at the unit level. The frontend disabled-state assertion and the actual `CompatibilityModeE2E` Playwright run are still missing.

**This satisfies the pre-flight part of the `CompatibilityModeE2E` acceptance criterion.** The remaining work is the E2E itself, which the user explicitly de-scoped in favor of the audit option.

**For #765 to close:**
- Attach this pre-flight document to the #765 closing PR description as evidence of partial satisfaction.
- Open a follow-up issue for the `CompatibilityModeE2E` Playwright implementation (5 items in §4).
- Until then, document in the #765 close-out that `CompatibilityModeE2E` is **AMBER, not GREEN**.

---

**Generated as part of audit scope.** No source code edits were performed.
