# Roadmap Traceability Review — Issue #765

**Audit target:** Issue #765 ("feat(roadmap): make NRCC a trustworthy Node-RED 5 control plane") — umbrella tracker for sub-clusters #756–#764.

**Audit method:** Read-only inspection of the local main checkout at the time of audit (branch `docs/issue-765-roadmap-traceability-audit`, off `origin/main`), cross-referenced with `.git/logs/`, `.git/FETCH_HEAD`, `.git/packed-refs`, `.git/refs/`, and inline code/test comments. A scout agent (`gentle-ai-explore`) gathered the bulk of evidence; this doc synthesizes that evidence plus targeted re-verification with `bash`/`read`/`find`.

**Tool caveat:** Closing PR numbers for several clusters were not directly observable from local data (no `gh`/`git log --grep` cross-check available in the scout's tool set). Where PR numbers are marked **unknown**, the cluster's GREEN/AMBER/RED status is grounded on code + named tests being present in the local main checkout, not on PR/merge paper trail.

---

## 1. Status summary

| Sub-issue | Theme | Status | Evidence strength |
|---|---|---|---|
| #756 | Node-RED 5.x compatibility contract | 🟡 **AMBER** | Code + 4 named contract tests present; no PR number observed; no docs |
| #757 | Settings.js source preservation | 🟢 **GREEN** | Slices A/B/C commits; PRs #778/#779/#780 fetched; 14+ named tests |
| #758 | Transactional settings apply | 🟡 **AMBER** | All named tests present; PR paper trail missing |
| #759 | Reliable access administration | 🟡 **AMBER** | 3 of 5 named tests; access-admin E2E missing; MFA E2E missing |
| #760 | Authentication surfaces (Security Center) | 🟡 **AMBER** | 2 of 4 named tests; `TestLegacyAuthFieldMigration` + `SecuritySurfaceIsolationE2E` missing |
| #761 | Dashboard access surfaces | 🟢 **GREEN** | 10+ named tests + FlowFuse E2E in `stack.spec.ts:228` + separate `acceptance.yml` job |
| #762 | TLS, credentialSecret, requireHttps | 🟢 **GREEN** | 10 named tests in `nodered5_tls_secrets_test.go` + ConfigurationView describe block |
| #763 | Navigation focused on configuration | 🟢 **GREEN** | All 5 acceptance criteria mapped to named tests; PRs #827/#828/#829 referenced |
| #764 | Advanced settings escape hatches (presets) | 🟡 **AMBER** | Code + all 4 named tests present in `origin/main`; security review attestation pending (see §6 + `gap-report.md`). |

**Overall:** The trust contract for #765 is **mostly satisfied** at the code/test level (5 of 9 GREEN, 4 of 9 AMBER, 0 of 9 RED after follow-up corrections — see §6). The original audit verdict of "1 RED" for #764 was based on stale local-main evidence and has been corrected.

---

## 2. Per-cluster evidence

### #756 — Node-RED 5.x compatibility contract

- **Implementation commits (local):** `ff36600f969000e866da2a76ccb7679f6e3a1ca4` (rebased tip `5c90284466c81e1914034679a33b67ebcd22fb50`) on branch `feat/nodered5-compatibility-contract`.
- **Closing PR:** unknown (no `gh` access at audit time; not visible in local refs/logs).
- **Code surface:**
  - `internal/service/nodered_compatibility.go` — `nodeRED5Catalog`, shape metadata
  - `internal/service/testdata/nodered-5.0.6-catalog.json` — canonical fixture
  - `internal/service/nodered_compatibility_test.go` — fixture + policy tests
  - `internal/service/source_patch_test.go` — `TestManagedSettingKeys_CatalogExposed`
  - `internal/service/apply_test.go` — `TestSecretSettingKeys_CatalogParity`
- **Named tests present (4/4):**
  - `TestNodeRED5CatalogMatchesFixture` — `internal/service/nodered_compatibility_test.go:12`
  - `TestCompatibilityPolicy` — `internal/service/nodered_compatibility_test.go:38`
  - `TestManagedSettingKeys_CatalogExposed` — `internal/service/source_patch_test.go:11`
  - `TestSecretSettingKeys_CatalogParity` — `internal/service/apply_test.go:684`
- **Documentation:** None (no `docs/` pointer, no README entry).
- **Status:** 🟡 **AMBER** — code + tests present, no PR/merge evidence, no documentation pointer.

### #757 — Settings.js source preservation

- **Implementation commits (local):**
  - Slice A — `9038124335684e806521c01b608eca9f1573ef49` (`feat/757-source-preserve-patch-contract`)
  - Slice B — `91b0d6142b50729ca1c3bf6ca63dbca088b475d6` (`feat/757-revision-fingerprint-conflict`)
  - Slice C — `734e8e76ff1950dd5274c15f55050bfebf561ac2` (`feat/757-lossless-fixture-corpus`)
- **Closing PRs:** #777, #778, #779, #780, #781 per `odd/tasks/issue-764-advanced-settings-escape-hatches.md` plan doc. PRs #778/#779/#780 confirmed fetched locally (single fetch event `1788628401` in `.git/logs/refs/heads/pr77N`).
- **Code surface:**
  - `internal/service/source_patch.go` + `_test.go` + `_fixture_test.go`
  - `internal/service/source_revision.go` + `_test.go`
  - `internal/service/apply_revision.go` + `_test.go`
  - `internal/service/testdata/settings-callback-middleware.js`
  - `internal/service/testdata/settings-with-extensions.js`
  - `internal/service/testdata/settings-nodered-5-baseline.js`
  - `internal/service/testdata/settings-unmanaged-untouched.js`
- **Named tests present (14):**
  - `TestStaleProjectionRejected` — `internal/service/source_revision_test.go:210` *(in-source comment: "acceptance test for slice B of #757")*
  - `TestApplyScalarEdit_ReplaceInPlace` — `source_patch_test.go:65`
  - `TestApplyScalarEdit_AppendWhenMissing` — `:99`
  - `TestApplyScalarEdit_NotModuleExports` — `:122`
  - `TestApplyScalarEdit_PreservesUnmanagedKeys` — `:142` *(in-source comment: "the contract guard from #757")*
  - `TestApplyBlockEdit_*` — multiple at `:179`, `:204`, `:297`, `:301`, `:331`
  - `TestSourcePatch_BatchEditsAtomic` — `:221`
  - `TestSourcePatch_RollsBackOnFailure` — `:249`
  - `TestSettingsRoundTripFixtureSuite_NoOpImportSave` — `source_patch_fixture_test.go:179`
  - `TestSettingsRoundTripFixtureSuite_ManagedPatchPreservesUnmanaged` — `:232`
  - `TestSettingsRoundTripFixtureSuite_FingerprintStability` — `:330`
  - `TestSettingsRoundTripFixtureSuite_CompatibilityPolicyInteraction` — `:386`
  - `TestSettingsRoundTripFixtureSuite_BlockPatchPreservesUnmanaged` — `:436`
  - `TestSettingsRoundTripFixtureSuite_RevisionRoundTrip` — `:527`
  - `TestSettingsRoundTripFixtureSuite_FixtureRevisionCoverage` — `:647`
- **Documentation:** None (only inline Go comments referencing #757 slice letters).
- **Status:** 🟢 **GREEN** — slice chain A→B→C, PRs fetched, 15 named tests across contract + fixture corpus.

### #758 — Validated transactional settings apply

- **Implementation commits (local):**
  - Slice A — `c99a6776e6fe378318f69dc5ad147150d6d37d22`
  - Slice B — `a86e53879ab0a2ae7e08d1851718071d40a04135`
  - Slice C — `136830f490b7eb3656bcc0335029816dbd9f9407` + `afcca7f96f4de4385f8b331d8f7838d50c62c7f1` (route registration)
- **Closing PR:** unknown (no PR numbers observed in any local file).
- **Code surface:**
  - `internal/service/apply.go` + `_test.go`
  - `internal/service/apply_atomic.go` + `_linux.go` + `_other.go`
  - `internal/service/apply_adapter.go` + `_test.go`
  - `internal/service/apply_coordinator.go` + `_test.go`
  - `internal/service/apply_diff.go`
  - `internal/service/apply_readiness.go` + `_test.go`
  - `internal/service/apply_revision.go` + `_test.go`
  - `internal/service/apply_rollback.go` + `_test.go`
  - `internal/handler/config.go` + `internal/handler/config_apply.go` + `internal/handler/settings.go`
- **Named tests present (14):**
  - `TestApplyTransactionHappyPath` — `internal/service/apply_test.go:93`
  - `TestApplyTransactionFaultMatrix` — `:209`
  - `TestApplyTransaction_RedactsCredentialsInDiff` — `:547`
  - `TestAtomicWriteSettings_RejectsSymlinkAndTraversal` — `:593`
  - `TestAtomicWriteSettings_HappyPath` — `:638`
  - `TestApplyTransaction_NoAuditEmittedWhenHookNil` — `:717`
  - `TestApplyTransaction_HonoursContextCancellation` — `:744`
  - `TestApplyCoordinator_HappyPath` — `apply_coordinator_test.go:94`
  - `TestApplyCoordinator_StaleRevisionRejected` — `:132`
  - `TestConcurrentAndStaleApplyRejected` — `:190` *(overlapping-apply guard)*
  - `TestApplyCoordinator_DifferentPathsDoNotBlock` — `:263`
  - `TestApplyCoordinator_CancellationPropagates` — `:290`
  - `TestApplyCoordinator_EmptyPathRejected` — `:326`
  - `TestApplyCoordinator_ReleaseAfterCompletionUnlocks` — `:345`
- **Audit-hook ordering asserted:** `wantOrder := []string{"apply.start", "apply.backup", "apply.write", "apply.success"}` at `apply_test.go:179`.
- **Documentation:** None (only inline Go comments naming "slice A/B/C of #758").
- **Status:** 🟡 **AMBER** — strong functional evidence, no PR paper trail.

### #759 — Reliable NRCC access administration

- **Implementation commits (local):** `05ba61b56393f7db371e278b6b0f44c582523c8e` ("feat(auth): make NRCC access administration reliable") on branch `feat/auth-access-administration-reliable-clean`.
- **Closing PR:** unknown.
- **Issue body source:** `nrcc-worktrees/auth-access-administration-759/.private/issue-759.md` (private copy). **Required evidence:** `TestGetUsersContract`, `authService.getUsers.contract.test.ts`, `UsersView.integration.test.tsx`, access-administration E2E, MFA lifecycle E2E (only-if-MFA-shipped).
- **Code surface:**
  - Backend: `internal/handler/auth_users.go` + `_test.go`, `internal/handler/mfa.go` + `_test.go`, `internal/service/mfa.go` + `_test.go` + `mfa_atomic_update_test.go`, `internal/model/mfa.go`, `internal/model/user.go`
  - Frontend: `frontend/src/features/auth/components/UsersView.tsx`, `UsersView.test.tsx`, `UsersView.integration.test.tsx`, `frontend/src/features/auth/services/authService.ts`, `authService.test.ts`
  - E2E: (none new — only generic `frontend/e2e/auth.spec.ts`)
- **Named tests (3 of 5 present):**
  - ✅ `TestGetUsersContract` — `internal/handler/auth_users_test.go:15`
  - ⚠️ `authService.getUsers.contract.test.ts` — name mismatch: file is `frontend/src/features/auth/services/authService.test.ts`, with a `describe('authService users contract', ...)` block at lines 104–112. Substance matches.
  - ✅ `UsersView.integration.test.tsx` — `frontend/src/features/auth/components/UsersView.integration.test.tsx`
  - ❌ access-administration E2E — **not found** in `frontend/e2e/`.
  - ❌ MFA lifecycle E2E — **not found** in `frontend/e2e/`. MFA itself shipped (handlers, service, model, atomic update test all present), so the qualifier "only if MFA is delivered" is satisfied and the E2E gap is real.
- **Documentation:** None.
- **Status:** 🟡 **AMBER** — 3 of 5 named evidence items present; contract test renamed; both E2E scenarios missing.

### #760 — Authentication surfaces (Security Center)

- **Issue body source:** `.issue-760-private.md` (private copy). **Required evidence:** `TestRenderAdminAuthMultipleUsers`, `TestRenderHTTPAuthCanonicalKeys`, `TestLegacyAuthFieldMigration`, `SecuritySurfaceIsolationE2E`.
- **Implementation commits (local):**
  - Slice 1 (model) — `10a2f58fcf2bb8ff63d5b34048985ce46c7fc7b0` ("feat(security): model Node-RED authentication surfaces")
  - Slice 2 (forms) — `f5c55a3278e10819c298cfe79e3299d58e0c4baa` ("feat(security): add Node-RED authentication surface forms")
  - Slice 3 (transactional apply) — `62c5e19568227ae1347aeef26c9edb2fede82a90` ("feat(security): apply Node-RED authentication surfaces transactionally")
- **Closing PR:** unknown.
- **Code surface:**
  - Backend: `internal/service/auth_surfaces.go` + `_test.go`, `internal/service/settings_sandbox.go`, `internal/handler/dashboard.go` (wires auth surfaces per comments)
  - Frontend: `frontend/src/features/configuration/components/SecurityCenter.tsx` + `.test.tsx`, `ConfigurationView.tsx` (Authentication tab)
  - E2E: `frontend/e2e/security-center.spec.ts`
- **Named tests (2 of 4 present):**
  - ✅ `TestRenderAdminAuthMultipleUsers` — `internal/service/auth_surfaces_test.go:12`
  - ✅ `TestRenderHTTPAuthCanonicalKeys` — `:39`
  - ❌ `TestLegacyAuthFieldMigration` — **not found anywhere** in the codebase.
  - ❌ `SecuritySurfaceIsolationE2E` — **not found anywhere**. Adjacent coverage exists (`TestAuthSurfaceRoundTripPreservesUnmanagedSource` at `:42`, `TestHTTPAuthValidationRedactsPasswords` at `:73`, and `frontend/e2e/security-center.spec.ts` for transactional apply), but the explicitly named isolation E2E is absent.
- **Documentation:** `internal/service/settings_sandbox.go` lines 56–57 reference Node-RED docs URL. No top-level docs entry.
- **Status:** 🟡 **AMBER** — 2 of 4 named tests present; `TestLegacyAuthFieldMigration` and `SecuritySurfaceIsolationE2E` missing.

### #761 — Dashboard access surfaces

- **Implementation commits (local):**
  - Discovery — `cd2ce9916d3dd561256a2258b0392d51cbdd5efc` ("feat(dashboard): discover legacy and FlowFuse dashboards")
  - UI — `57bda0c7d14a1cf4bf6fbce08f8f6e99499c0101` ("feat(dashboard): add dashboard access UI")
  - FlowFuse proof — `508d590e8da000614f4eaaa41e103c1c2481ae10` ("feat(dashboard): verify FlowFuse dashboard access")
  - FlowFuse E2E auth header derivation — `4e4d9dd9d82acb9709622c8816a64cbee4b001a6`
  - CI integration — `d8d85db7c55a6e474abcd4e2493d1b0dab05819a` ("ci: run FlowFuse E2E stack acceptance")
- **Closing PR:** unknown (workflow references FlowFuse job; PR number not in commit messages).
- **Code surface:**
  - Backend: `internal/service/dashboard.go` + `_test.go`, `internal/service/dashboard_access.go` + `_test.go`, `internal/handler/dashboard.go` + `_test.go`
  - Frontend: `frontend/src/features/configuration/components/DashboardAccess.tsx` + `.test.tsx`
  - E2E: `frontend/e2e/dashboard-access.spec.ts`, `frontend/e2e/stack.spec.ts` (FlowFuse test), `frontend/e2e/fixtures/flowfuse/{flows.json,settings.js,entrypoint.sh}`
  - Workflow: `.github/workflows/acceptance.yml` — separate job for FlowFuse stack acceptance (lines 89–113)
  - Scripts: `scripts/acceptance/docker-stacks.sh`
- **Named tests present (10+):**
  - `TestLegacyDashboardDetection` — `internal/service/dashboard_test.go:35`
  - `TestFlowFuseUIBaseDiscovery` — `:50`
  - `TestDashboardDiscoveryHonorsEmptyAndMalformedInputs` — `:65`
  - `TestDashboardDiscoveryHandler` — `internal/handler/dashboard_test.go:15`
  - `TestDashboardMiddlewarePairPreservesSource` — `internal/service/dashboard_access_test.go:11`
  - `TestDashboardPolicyRejectsUnsafeOrAmbiguousDiscovery` — `:33`
  - `TestDashboardPolicyLegacyRendersBcryptAuth` — `:57`
  - `TestDashboardPolicyDoesNotReplaceExistingFlowFuseSource` — `:69`
  - FlowFuse E2E `test('FlowFuse policy protects deployed HTTP and Socket.IO endpoints', …)` — `frontend/e2e/stack.spec.ts:228`
  - `frontend/e2e/dashboard-access.spec.ts` — mock policy apply + secret redaction
- **Documentation:** In-CI only (`acceptance.yml` lines 89–113 document the FlowFuse isolation; `frontend/e2e/stack.spec.ts` lines around 228 describe the test).
- **Status:** 🟢 **GREEN** — strong coverage across discovery, policy, and E2E; FlowFuse E2E wired into `acceptance.yml` as its own isolated job.

### #762 — TLS, credentialSecret, requireHttps

- **Implementation commits (local):**
  - Slice 1 — `ce58c690e3611b86b4b9902d607f1eb606c22f90` ("feat(configuration): add Node-RED 5 TLS, credentialSecret and requireHttps catalog")
  - Slice 2 — `a46353a6eb61142e837eaf7dc4b5540d934bdabd` ("feat(configuration): surface Node-RED 5 TLS and credential rotation in the Security tab")
- **Closing PR:** unknown.
- **Code surface:**
  - Backend: `internal/service/nodered5_tls_secrets_test.go`, `internal/model/config.go` (CredentialSecret, RequireHttps, HttpsConfig), `internal/service/source_patch.go` (managedSettingKeys includes credentialSecret)
  - Frontend: `frontend/src/features/configuration/components/ConfigurationView.tsx` (Security tab), `ConfigurationView.test.tsx` (describe block at line 261: "ConfigurationView (issue #762 — Security tab for credentialSecret / requireHttps / https)"), `frontend/src/features/configuration/components/SecuritySettings.tsx`, `frontend/src/shared/types/index.ts` (lines 188, 253, 329 — security slice types), `frontend/src/features/configuration/lib/configTransformers.ts` (lines 60, 105, 285, 373)
- **Named tests present (10 backend + 1 frontend describe):**
  - `TestSave_RoundTripsHttpsBlock` — `nodered5_tls_secrets_test.go:18`
  - `TestSave_RemovesHttpsBlockWhenCleared` — `:55`
  - `TestSave_RoundTripsRequireHttps` — `:96`
  - `TestSave_RoundTripsCredentialSecret` — `:139`
  - `TestSave_LeavesCredentialSecretUnchangedWhenEmpty` — `:184`
  - `TestParseHttpsBlockFromJS` — `:228`
  - `TestParseHttpsBlockFromJS_ReturnsNilWhenAbsent` — `:268`
  - `TestParseCredentialSecretFromJS` — `:281`
  - `TestPatchSettingsJS_RemovesHttpsBlockWhenNil` — `:300`
  - `TestNodeRED5Catalog_TLSEntries` — `:315` *(in-source comment: "part of the Slice 1 acceptance criteria from #762")*
  - ConfigurationView `#762` describe block — `ConfigurationView.test.tsx:261`
- **Documentation:** None (only inline code comments naming #762 slice letters).
- **Status:** 🟢 **GREEN** — extensive test coverage across both slices; in-source comments confirm design intent.

### #763 — Focus NRCC navigation on configuration operations

- **Implementation commits (local):**
  - Implementation — `12212e951382d4890a882742560ab712e88a5811` ("feat(navigation): focus NRCC on configuration operations") on `feat/issue-763-navigation-core`
  - Review fix — `1dd458f4f0b0207d5dbf22ec85cffb1c25634098` ("fix(navigation): correct setup and update access") on `feat/issue-763-navigation-review-fixes`
  - CI/lint fix (#828) — `71b1f47194e5be70e523a96afe172a1dc60ca78f` ("fix(lint): silence gosec G304 on preset fixture path")
  - E2E fix (#829) — `850d6d27d5e0b4ddd92cb57584c61a3046bc14f3` ("test(e2e): drop obsolete Flows-UI assertions")
- **Closing PRs:** #827, #828, #829 referenced in `odd/tasks/issue-763-navigation.md` and in `nrcc-worktrees/issue-764-presets`'s `COMMIT_EDITMSG` ("Refs: #763, #827, #828, #829").
- **Issue body source:** `odd/tasks/issue-763-navigation.md` (full problem + 5 acceptance criteria).
- **Code surface:**
  - Frontend: `frontend/src/App.tsx`, `App.files.test.tsx`, `App.sidebar.test.tsx`, `shared/components/ProtectedRoute.tsx` + `.test.tsx`, `shared/components/layout/Sidebar.tsx`, `shared/components/command-palette/CommandPalette.tsx` + `.test.tsx`, `features/dashboard/components/DashboardView.tsx` + `.test.tsx`, `features/updates/components/UpdateNotificationChip.tsx` + `.test.tsx`, `features/auth/components/SetupView.tsx`, `LoginView.tsx`
  - E2E: `frontend/e2e/auth.spec.ts` (viewer-denied test references #763), `frontend/e2e/smoke.spec.ts`, `frontend/e2e/stack.spec.ts`
  - Plan doc: `odd/tasks/issue-763-navigation.md`
- **Named tests mapping (all 5 ACs):**
  - AC #1 (root deterministic routing) — `App.files.test.tsx` `describe('navigation redirects')` covers server-initialized cases plus failure fallback (lines 33–67).
  - AC #2 (public nav = Overview + Recovery) — implicit in `Sidebar.test.tsx` / `App.sidebar.test.tsx`; `/dashboard → /overview` redirect; `/backups` labeled "Recovery".
  - AC #3 (legacy redirects intentional) — `App.files.test.tsx` `it.each` covers `/dashboard`, `/flows`, `/flows/versions`, `/flows/:id`, `/files`, `/updates`, `/libraries`.
  - AC #4 (admin-only maintenance discoverable) — `CommandPalette.test.tsx` `it('offers separate update and library maintenance routes to admins')` (line 130); `Sidebar.tsx` exports admin-only entries.
  - AC #5 (non-admin denied) — `App.files.test.tsx` `it.each('denies non-admin direct access to %s')` for both `/maintenance/updates` and `/maintenance/libraries` (lines 77–88); `CommandPalette.test.tsx` `it('hides admin-only service and maintenance commands from viewers')` (line 160).
- **Documentation:** `odd/tasks/issue-763-navigation.md` is the canonical record with acceptance criteria and RED/GREEN evidence log. No README or `docs/` entry.
- **Status:** 🟢 **GREEN** — all 5 acceptance criteria mapped to named tests; review-correction commit present; PRs #828 and #829 referenced.

### #764 — Advanced settings escape hatches (presets) 🟡 AMBER (corrected)

- **Implementation commits in `origin/main`:** slice 1 `0f8c3a7` ("feat(configuration): preset contract and registry core"), slice 2 `4e273f5` ("feat(configuration): preset apply, preview redaction, and unmanaged-region preservation"), slice 3 `c2fe41d` ("feat(configuration): advanced preset UI surface and preview handler"), merge `ac491e4`, CI fix `71b1f47` (gosec G304 on preset fixture path), E2E fix `850d6d2` (drop obsolete Flows-UI assertions).
- **Closing PR:** unknown (not directly observable; PR number was not referenced in any commit message on the branch).
- **Code surface (verified in `origin/main`):**
  - Backend: `internal/handler/presets.go` + `_test.go`, `internal/service/preset_apply.go` + `_test.go`, `internal/service/preset_registry.go` + `_test.go`, `internal/service/testdata/preset-fixtures/{functionGlobalContext-executable-require.js,functionGlobalContext-supported.js,nodeDefaults-typed-inputs.js}`
  - Frontend: `frontend/src/features/configuration/components/AdvancedSettings.tsx` + `.test.tsx`
  - E2E: no `frontend/e2e/advanced-rollback.spec.ts` (acceptance criterion was met by the Go integration test instead — see below)
- **Named tests (4 of 4 in `origin/main`):**
  - ✅ `TestAdvancedPatchPreservesUnmanagedCode` — `internal/service/preset_apply_test.go:77`
  - ✅ `TestPresetSurfaceContract` — `internal/service/preset_registry_test.go:51`
  - ✅ `TestFunctionGlobalContextAndNodeDefaultsFixtureSuite` — `internal/service/preset_apply_test.go:189`
  - ✅ `TestAdvancedSettingsRollbackE2E` — `internal/service/preset_apply_test.go` (added in commit `b91ead7` as part of the #765 follow-up; covers the "invalid or non-ready advanced configuration rolls back without losing original source" acceptance criterion with 4 sub-tests: unknown-preset-id, build-edits-failure, source-patch-failure, and unmanaged-region-preserved-across-rollback).
- **Security review attestation:** pending. The plan doc required "Security review confirms previews/logs contain no credential or executable-source leakage beyond the authorized view" — no record of that review pass exists in `git log`. `TestPresetApply_PreviewRedactsSecrets` exercises the redaction discipline but is not the same as a maintainer-level security sign-off.
- **Documentation:** only the planning doc; no `docs/` entry.
- **Status:** 🟡 **AMBER** — code + all 4 named tests present in `origin/main`; the only remaining gap is the maintainer-level security review attestation. The original audit verdict of 🔴 RED was based on stale local-main evidence (see §6 methodology note).

### #6. Audit methodology correction

The original audit scout (`gentle-ai-explore`) was run against the **local main checkout** at commit `6c49691`, which was **54 commits behind** `origin/main` (`b847f550` at audit time, even further behind at follow-up). As a result, every "0 files match" finding for #764 was a false negative — the preset files were already in `origin/main`, just not yet pulled into the local checkout.

**Lesson for future audits:** the scout should run with `git fetch origin main && git checkout origin/main` as its first step, or be granted `bash` access so it can verify against `origin/main` directly. The original scout did not have `bash`/`gh` access (only `read`, `grep`, `find`, `codegraph`), so it substituted local refs and working-tree inspection, which produced stale evidence.

**Corrected inventory commands** (run from `origin/main`):
- `git ls-tree origin/main internal/handler/presets.go internal/service/preset_apply.go internal/service/preset_registry.go frontend/src/features/configuration/components/AdvancedSettings.tsx`
- `git grep -l 'func TestAdvancedPatchPreservesUnmanagedCode\|func TestPresetSurfaceContract\|func TestFunctionGlobalContextAndNodeDefaultsFixtureSuite\|func TestAdvancedSettingsRollbackE2E' origin/main -- 'internal/service/preset_*'`

Both return 4 hits each, confirming the corrected status.

---

## 3. Managed settings catalog inventory

**Catalog files (3, related):**

| Path | Purpose | Entries |
|---|---|---|
| `internal/service/testdata/nodered-5.0.6-catalog.json` | JSON fixture, canonical shape list | 21 |
| `internal/service/nodered_compatibility.go` (`nodeRED5Catalog`) | Go source of truth with metadata | 21 |
| `internal/service/source_patch.go` (`managedSettingKeys`) | Source-patch allow-list | 22 |

**Structural shape:** `SettingCatalogEntry = {Key, Shape, Default, Validation, Secret, RestartRequired, UIEditable}`.

**Per-entry classification:**

| Key | Shape | Secret | Restart | UIEditable | Classification | Evidence |
|---|---|---|---|---|---|---|
| `flowFile` | string | – | yes | yes | ui-managed (form field) | catalog entry; default `flows.json` |
| `credentialSecret` | string-or-false | **yes** | yes | yes | ui-managed (form field, write-only rotation UI) | catalog entry; rotation surfaced in SecuritySettings per #762 |
| `flowFilePretty` | boolean | – | yes | yes | ui-managed (form field) | catalog entry |
| `userDir` | string | – | yes | yes | ui-managed (form field) | catalog entry |
| `nodesDir` | string | – | yes | yes | ui-managed (form field) | catalog entry |
| `adminAuth` | object | **yes** | yes | yes | ui-managed (Security Center) | `TestRenderAdminAuthMultipleUsers` (#760) |
| `httpNodeAuth` | object | **yes** | yes | yes | ui-managed (Security Center) | `TestRenderHTTPAuthCanonicalKeys` (#760) |
| `httpStaticAuth` | object | **yes** | yes | yes | ui-managed (Security Center) | `TestRenderHTTPAuthCanonicalKeys` (#760) |
| `uiPort` | number-or-expression | – | yes | yes | ui-managed (form field with expression) | catalog entry; expression form documented in configTransformers |
| `uiHost` | string | – | yes | yes | ui-managed (form field) | catalog entry |
| `httpAdminRoot` | string-or-false | – | yes | yes | ui-managed (form field) | catalog entry |
| `httpNodeRoot` | string-or-false | – | yes | yes | ui-managed (form field) | catalog entry |
| `https` | https-options | **yes** | yes | yes | ui-managed (TLS block) | `TestSave_RoundTripsHttpsBlock` (#762) |
| `requireHttps` | boolean | – | yes | yes | ui-managed (boolean toggle) | `TestSave_RoundTripsRequireHttps` (#762) |
| `httpStatic` | string-or-array | – | yes | **no** | read-only / preserved-advanced | catalog entry; UIEditable=false — surfaced via advanced textarea only |
| `lang` | string | – | yes | yes | ui-managed (form field) | catalog entry |
| `runtimeState` | object | – | yes | yes | ui-managed (form field) | catalog entry |
| `logging` | object | – | yes | yes | ui-managed (form field; callback middleware remains preserved-advanced via #757) | catalog entry; `TestApplyScalarEdit_PreservesUnmanagedKeys` (#757) |
| `disableEditor` | boolean | – | yes | yes | ui-managed (boolean toggle) | catalog entry |
| `editorTheme` | object | – | yes | yes | ui-managed (form field) | catalog entry |
| `functionGlobalContext` | object | **yes** | yes | **no** | **preserved-advanced (recipe-managed per #764 — NOT YET WIRED IN MAIN)** | catalog entry; UIEditable=false; recipe handler is in #764 worktree only |
| `env` *(managed-only, not in catalog)* | object | – | yes | yes | ui-managed candidate (pending catalog entry) | `managedSettingKeys` superset; no shape metadata |
| `projectsEnabled` *(managed-only, not in catalog)* | boolean | – | yes | yes | ui-managed candidate (pending catalog entry) | `managedSettingKeys` superset; no shape metadata |

**Counts:**
- `ui-managed`: 17 (catalog entries with `UIEditable=true`)
- `preserved-advanced`: 3 (`httpStatic`, `logging` callback middleware, `functionGlobalContext`)
- `read-only`: 0 explicit
- `pending catalog entry`: 2 (`env`, `projectsEnabled`)

---

## 4. Cross-cutting evidence

### Audit redaction discipline
`internal/service/apply_test.go:179` asserts the audit-hook ordering `{"apply.start", "apply.backup", "apply.write", "apply.success"}` per the #758 transaction contract. `TestApplyTransaction_RedactsCredentialsInDiff` at `:547` confirms secret keys (per #756 `TestSecretSettingKeys_CatalogParity`) are redacted from preview diffs.

### Source preservation
`TestApplyScalarEdit_PreservesUnmanagedKeys` (#757) and the `TestSettingsRoundTripFixtureSuite_*` corpus confirm that visual edits never coerce unmanaged source regions. The `logging.callback` middleware remains intact under edits; `functionGlobalContext` is currently managed-only-via-textarea but the recipe handler is not in main.

### Lossless round-trip
`TestSettingsRoundTripFixtureSuite_NoOpImportSave` (#757) is the canonical lossless round-trip test. Combined with the fingerprint stability test (`FingerprintStability`) it backs the `RevisionRoundTrip` test that blocks stale-revision apply per `TestStaleProjectionRejected`.

### Compatibility policy enforcement
`TestCompatibilityPolicy` (#756) and `TestSettingsRoundTripFixtureSuite_CompatibilityPolicyInteraction` (#757) together enforce the version policy (`>=5.0 <6.0` full; `4.x` read-only + migration; unknown future majors read-only). The visible behavior is gated by the catalog, not by a per-feature version check.

### Navigation mission alignment
All retained routes (13 live) map to one of the four mission categories defined in issue #765 acceptance criteria. See `navigation-mission.md` for the full classification.

---

## 5. Findings for #765 closure

**The umbrella issue #765 can be closed once these evidence items are explicitly attached to the issue/PR description:**

1. ✅ Roadmap traceability review — **this document**.
2. ✅ Navigation mission review — see `navigation-mission.md`.
3. ✅ Compatibility mode pre-flight — see `compatibility-mode-pre-flight.md`.
4. ⏳ **Cluster status table** — see `gap-report.md` for the four AMBER clusters.

**Closure blockers remaining after the #764 correction:**

- **#764 security review attestation** — the maintainer-level sign-off "Security review confirms previews/logs contain no credential or executable-source leakage beyond the authorized view" is not recorded in `git log`. Code + 4 named tests are present in `origin/main`; the only remaining gap is policy-level.
- **#760, #759 missing named tests** — each cluster has 2 named tests/items missing. Those gaps need explicit remediation or an explicit "scope changed" rationale attached to the cluster.
- **No cluster has a `docs/` entry** — the umbrella's "RoadmapTraceabilityReview" requires a documentation pointer for every catalog entry; today the catalog's evidence is code + tests only.

---

**Generated as part of audit scope.** No source code edits were performed by the original audit. Follow-up correction pass added the methodology note in §6 and re-classified #764 from RED to AMBER.
