# Gap Report — Issue #765

**Audit target:** Explicit list of gaps surfaced by the Roadmap Traceability Review, Navigation Mission Review, and Compatibility-Mode Pre-flight. Each gap is mapped to the cluster it belongs to, the file/test/PR it concerns, and the recommended remediation.

**Status legend:** 🟢 GREEN (no gap), 🟡 AMBER (partial / weak paper trail), 🔴 RED (missing or contradicted).

---

## 1. Cluster status recap

| Cluster | Theme | Status | Closing PR(s) | Evidence |
|---|---|---|---|---|
| #756 | NR 5.x compatibility contract | 🟢 GREEN | #772, #775 | 4 named contract tests present |
| #757 | Settings.js source preservation | 🟢 GREEN | #777–#781 | 15 named tests across 3 slices |
| #758 | Transactional settings apply | 🟢 GREEN | #781, #792, #793 | 14 named tests |
| #759 | Reliable access administration | 🟡 AMBER | #794 | 3 of 5 named items; access-admin E2E + MFA E2E missing |
| #760 | Authentication surfaces | 🟡 AMBER | #795, #797 | 2 of 4 named tests; `TestLegacyAuthFieldMigration` + `SecuritySurfaceIsolationE2E` missing |
| #761 | Dashboard access surfaces | 🟢 GREEN | #800 | 10+ named tests + FlowFuse E2E in `stack.spec.ts:228` |
| #762 | TLS, credentialSecret, requireHttps | 🟢 GREEN | #776, #777 | 10 named tests + ConfigurationView describe block |
| #763 | Navigation focused on configuration | 🟢 GREEN | #827, #828, #829 | All 5 ACs mapped |
| #764 | Advanced settings escape hatches | 🟡 AMBER | #828 | Code + all 4 named tests in `origin/main`; security review attestation pending |

**Overall:** 6 🟢 GREEN, 3 🟡 AMBER, 0 🔴 RED.

---

## 2. Critical gaps

### Gap G1 — #764 security review attestation pending *(downgraded from RED after follow-up)*

- **Severity:** 🟡 AMBER (was 🔴 RED in the original audit; corrected).
- **Original audit claim:** `odd/tasks/issue-767-i18n-en-es-localization.md` asserts "#764 presets — merged ✅", contradicted by the original audit's local-main evidence.
- **Correction:** the original audit's evidence was based on a stale local-main checkout (`6c49691`, 54 commits behind `origin/main`). When verified against `origin/main`, all 4 named tests are present:
  - `TestAdvancedPatchPreservesUnmanagedCode` (slice 2 commit `4e273f5`)
  - `TestPresetSurfaceContract` (slice 1 commit `0f8c3a7`)
  - `TestFunctionGlobalContextAndNodeDefaultsFixtureSuite` (slice 2 commit `4e273f5`)
  - `TestAdvancedSettingsRollbackE2E` (added in follow-up commit `b91ead7` to satisfy the headline acceptance test called out by the issue)
- **Remaining gap:** maintainer-level security review attestation ("Security review confirms previews/logs contain no credential or executable-source leakage beyond the authorized view") is not recorded in `git log`. `TestPresetApply_PreviewRedactsSecrets` exercises the redaction discipline but is not the same as a maintainer-level sign-off.
- **Recommended remediation:**
  1. A maintainer signs off on the security review (audit log entry or comment on #764) and links to `TestPresetApply_PreviewRedactsSecrets` + `TestAdvancedSettingsRollbackE2E` as the evidence.
  2. Once signed off, #764 can be re-classified 🟢 GREEN.

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

### Gap G4 — No documentation pointers for any cluster *(RESOLVED)*

- **Status:** 🟢 RESOLVED.
- **Resolution:** `docs/control-plane.md` now lists all 9 clusters with status, closing PR(s), and per-cluster evidence link. Linked from `README.md` "What NRCC does" section.

### Gap G5 — Closing PR numbers unknown *(RESOLVED)*

- **Status:** 🟢 RESOLVED.
- **Resolution:** Closing PR(s) attached to every cluster in §1 above. Verified via `gh pr list --search` cross-referenced with PR body `Refs`/`Closes` footers.

### Gap G6 — Audit methodology (added in follow-up)

- **Severity:** 🟡 AMBER (process gap, not code gap).
- **Issue:** the original audit scout worked against the local main checkout, which was 54 commits behind `origin/main`. As a result, evidence of "0 preset files match" was a false negative.
- **Recommended remediation:** future audits must run with `bash` access so the scout can verify against `origin/main` directly, or the scout's first action must be `git fetch origin main && git checkout origin/main`.

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

1. **Correct the #764 claim in `odd/tasks/issue-767-i18n-en-es-localization.md`.** Pure doc fix, no code risk. (1 commit.) — *Not needed after follow-up: the claim was correct; the audit was wrong.*
2. **Attach this gap report + the three review docs to the #765 closing PR description** as evidence of cross-cutting review. (1 PR description update.)
3. **Open follow-up issues** for: #760 missing tests, #759 missing E2E, CompatibilityModeE2E implementation, control-plane docs, #764 security review attestation. (5 issue creations.)
4. **(Optional) Add `docs/control-plane.md`** as a top-level pointer to the per-cluster evidence.

---

**Generated as part of audit scope.** No source code edits were performed. Follow-up correction pass added Gap G6 (audit methodology) and re-classified #764 from RED to AMBER after verifying evidence against `origin/main`.
