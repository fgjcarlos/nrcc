# Gap Report — Issue #765

**Audit target:** Explicit list of gaps surfaced by the Roadmap Traceability Review, Navigation Mission Review, and Compatibility-Mode Pre-flight. Each gap is mapped to the cluster it belongs to, the file/test/PR it concerns, and the recommended remediation.

**Status legend:** 🟢 GREEN (no gap), 🟡 AMBER (partial / weak paper trail), 🔴 RED (missing or contradicted).

---

## 1. Cluster status recap

| Sub-issue | Status | Closing PR | Code in main | Named tests in main | Documentation |
|---|---|---|---|---|---|
| #756 | 🟡 AMBER | unknown | yes | 4/4 | none |
| #757 | 🟢 GREEN | #777–#781 (fetched) | yes | 15/15 | none |
| #758 | 🟡 AMBER | unknown | yes | 14/14 | none |
| #759 | 🟡 AMBER | unknown | yes | 3/5 | none |
| #760 | 🟡 AMBER | unknown | yes | 2/4 | none |
| #761 | 🟢 GREEN | unknown | yes | 10/10 | in-CI |
| #762 | 🟢 GREEN | unknown | yes | 10/10 | none |
| #763 | 🟢 GREEN | #827/#828/#829 | yes | all 5 ACs | `odd/tasks/issue-763-navigation.md` |
| #764 | 🔴 **RED** | unknown | **NO** | **0/4 in main** (3/4 in worktree only) | none |

**Overall:** 4 GREEN, 4 AMBER, 1 RED.

---

## 2. Critical gaps

### Gap G1 — #764 advanced settings escape hatches are not merged

- **Severity:** 🔴 blocker for #765 closure.
- **Claim under audit:** `odd/tasks/issue-767-i18n-en-es-localization.md` asserts "#764 presets — merged ✅".
- **Evidence contradicting the claim:** `find internal frontend scripts -path '*preset*'` in the local main checkout returns zero hits (excluding `node_modules`). The 3 of 4 named tests (`TestAdvancedPatchPreservesUnmanagedCode`, `TestPresetSurfaceContract`, `FunctionGlobalContextAndNodeDefaultsFixtureSuite`) live only in `nrcc-worktrees/issue-764-presets/internal/service/`. `AdvancedSettingsRollbackE2E` is absent from any worktree.
- **Recommended remediation:**
  1. Open a follow-up issue for #764 status confirmation. The cluster should not be re-marked "merged" until the code, the 3 named tests, the `AdvancedSettingsRollbackE2E`, and the security-review attestation are all in `main`.
  2. If the #764 work is intentionally reverted or scope-cut, attach an explicit "scope changed" rationale to the cluster and update the `RoadmapTraceabilityReview` entry for `functionGlobalContext` to classify it as **read-only** (downgrade from **preserved-advanced (recipe-managed)**).
  3. Update `odd/tasks/issue-767-i18n-en-es-localization.md` to remove the "#764 presets — merged ✅" claim.

### Gap G2 — #760 missing 2 named tests

- **Severity:** 🟡 AMBER.
- **Claim under audit:** "SecurityCenter UI applies and round-trips Node-RED authentication surfaces."
- **Evidence:** `TestLegacyAuthFieldMigration` and `SecuritySurfaceIsolationE2E` are not present anywhere in the codebase.
- **Recommended remediation:**
  1. `TestLegacyAuthFieldMigration` — write the unit test in `internal/service/auth_surfaces_test.go` covering the legacy NRCC field-name → canonical mapping (legacy `adminAuth`/`httpAuth` → canonical `adminAuth`/`httpNodeAuth`).
  2. `SecuritySurfaceIsolationE2E` — write a Playwright spec in `frontend/e2e/security-center.spec.ts` that proves NRCC, editor, HTTP node, and static-resource credentials each protect only their declared surface.

### Gap G3 — #759 missing 2 acceptance items

- **Severity:** 🟡 AMBER.
- **Claim under audit:** "Access administration is reliable; MFA is delivered if shipped."
- **Evidence:** MFA shipped (`internal/handler/mfa.go` + `_test.go`, `internal/service/mfa.go` + `_test.go` + `mfa_atomic_update_test.go`, `internal/model/mfa.go`). Yet the `access-administration E2E` and `MFA lifecycle E2E` named in the issue are not in `frontend/e2e/`.
- **Recommended remediation:**
  1. Add `frontend/e2e/access-administration.spec.ts` covering: create user, edit user role, disable user, admin MFA reset. Use the existing `frontend/e2e/auth.spec.ts` fixture pattern.
  2. Add `frontend/e2e/mfa-lifecycle.spec.ts` covering: enroll MFA, sign-in challenge, recovery code flow.

### Gap G4 — No documentation pointers for any cluster

- **Severity:** 🟡 AMBER.
- **Claim under audit:** "Each managed catalog entry has a UI/recipe or explicit preserved-advanced classification and named verification evidence."
- **Evidence:** No cluster has a `docs/` entry pointing at the evidence. Only #763 has a plan doc in `odd/tasks/`. README.md does not reference any cluster.
- **Recommended remediation:**
  1. Add a section in `README.md` (or a new `docs/control-plane.md`) that lists the 9 clusters with one-line summaries and links to the per-cluster evidence in `odd/audits/issue-765/`.
  2. OpenAPI spec already references `catalogVersion` at line 2887 — expand that section to list the per-catalog-entry classification.

### Gap G5 — Closing PR numbers unknown for 5 clusters

- **Severity:** 🟡 AMBER.
- **Clusters affected:** #756, #758, #759, #760, #762.
- **Recommended remediation:** Before #765 closes, attach the closing PR numbers to the per-cluster evidence rows in `roadmap-traceability.md`. This requires `gh` access; the audit could not verify because no shell tool was available in the scout session.

---

## 3. Compatibility-Mode gaps (forward-looking)

These do not block #765 closure today (the user de-scoped the E2E build), but they should be tracked in a follow-up issue.

| Item | Severity | Notes |
|---|---|---|
| `CompatibilityModeE2E` Playwright run | 🔴 missing | 5 items in `compatibility-mode-pre-flight.md` §4 |
| Frontend disabled-state assertion for `migration`/`read-only` modes | 🟡 partial | `#762` describe block exercises `full` mode only |
| Backend apply-coordinator mode error code (machine-readable) | 🟡 partial | current rejection is implicit |

---

## 4. Documentation gaps

| Doc | State | Notes |
|---|---|---|
| `README.md` reference to clusters | missing | add `docs/control-plane.md` and link from README |
| `docs/control-plane.md` | missing | new file |
| `odd/tasks/issue-767-i18n-en-es-localization.md` claim "#764 presets — merged ✅" | **incorrect** | should be corrected to "#764 presets — **NOT merged; verify before close**" |
| Per-cluster evidence cross-link | partial | this audit adds the evidence; needs to be linked from the cluster issue threads |

---

## 5. Process gaps

| Item | State | Recommendation |
|---|---|---|
| Audit without `gh`/`bash` access for scout | known limitation | future audits should run with `bash` access so closing PR numbers and `git log --grep` cross-checks are first-class |
| "X is merged" claim contradicted by file-system evidence | happened with #764 | introduce a "merged-claim verification" step before marking a cluster closed in any doc |
| Private issue bodies (`odd/tasks/*.md`, `.issue-N-private.md`, `.private/issue-N.md`) | scattered | consolidate into `odd/issues/<number>.md` for audit-friendliness |

---

## 6. Recommended remediation order (lowest risk first)

1. **Correct the #764 claim in `odd/tasks/issue-767-i18n-en-es-localization.md`.** Pure doc fix, no code risk. (1 commit.)
2. **Attach this gap report + the three review docs to the #765 closing PR description** as evidence of cross-cutting review. (1 PR description update.)
3. **Open follow-up issues** for: #760 missing tests, #759 missing E2E, CompatibilityModeE2E implementation, control-plane docs. (4 issue creations.)
4. **Re-verify #764** by either merging the worktree branch or attaching a "scope changed" rationale.
5. **(Optional) Add `docs/control-plane.md`** as a top-level pointer to the per-cluster evidence.

---

**Generated as part of audit scope.** No source code edits were performed.
