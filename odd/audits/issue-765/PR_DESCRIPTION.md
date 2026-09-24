# PR Description — Issue #765 Roadmap Traceability Audit

> **Suggested text to paste into the PR body when opening the audit PR.** Adjust as needed.

## What

A read-only cross-cutting audit of the 9 closed sub-clusters (#756–#764) of umbrella issue #765 ("feat(roadmap): make NRCC a trustworthy Node-RED 5 control plane"). No source code edits — only documentation deliverables under `odd/audits/issue-765/`.

## Why

Issue #765 explicitly demands cross-cutting evidence that each closed sub-cluster actually satisfies its named acceptance tests — not just an "implementation claim" closed. The umbrella's own acceptance criteria name four deliverables (`RoadmapTraceabilityReview`, `NavigationMissionReview`, `CompatibilityModeE2E`, named tests per cluster). Until those deliverables exist, the umbrella cannot close honestly.

## Deliverables (under `odd/audits/issue-765/`)

| File | Purpose |
|---|---|
| `roadmap-traceability.md` | Per-cluster evidence table + managed-settings-catalog inventory with per-entry classification (ui-managed / preserved-advanced / read-only / pending-catalog-entry). |
| `navigation-mission.md` | Per-route mission classification (configuration / access-management / recovery / capability-gated-maintenance) for every retained live route in `frontend/src/App.tsx`. |
| `compatibility-mode-pre-flight.md` | What's in place for `CompatibilityModeE2E` (backend guards + catalog-driven UI guard + unit tests) and what remains to build the actual E2E. |
| `gap-report.md` | Explicit list of gaps with severity, evidence, and remediation. Includes the contradicting "#764 merged ✅" claim. |
| `PR_DESCRIPTION.md` | This file. |

## Cluster status

| Cluster | Theme | Status | Evidence |
|---|---|---|---|
| #756 | NR 5.x compatibility contract | 🟡 AMBER | 4 named contract tests present; no PR number; no docs |
| #757 | Settings.js source preservation | 🟢 GREEN | 15 named tests across 3 slices; PRs #777–#781 |
| #758 | Transactional settings apply | 🟡 AMBER | 14 named tests; no PR number |
| #759 | Reliable access administration | 🟡 AMBER | 3 of 5 named items; access-admin E2E + MFA E2E missing |
| #760 | Authentication surfaces | 🟡 AMBER | 2 of 4 named tests; `TestLegacyAuthFieldMigration` + `SecuritySurfaceIsolationE2E` missing |
| #761 | Dashboard access surfaces | 🟢 GREEN | 10+ named tests + FlowFuse E2E in `stack.spec.ts:228` |
| #762 | TLS, credentialSecret, requireHttps | 🟢 GREEN | 10 named tests + ConfigurationView describe block |
| #763 | Navigation focused on configuration | 🟢 GREEN | All 5 ACs mapped; PRs #827/#828/#829 |
| #764 | Advanced settings escape hatches | 🟡 AMBER *(corrected from RED after follow-up)* | Code + all 4 named tests in `origin/main`; security review attestation pending |

**Overall (corrected):** 5 GREEN, 4 AMBER, 0 RED.

## Critical findings

1. **#764 audit verdict was wrong (RED → AMBER after follow-up).** The original audit's local-main checkout was 54 commits behind `origin/main`, so every "0 preset files match" finding was a false negative. Re-verified: all 4 named tests are present in `origin/main`, including `TestAdvancedSettingsRollbackE2E` added in follow-up commit `b91ead7`. The original audit's contradiction with the `#764 presets — merged ✅` claim in `odd/tasks/issue-767-i18n-en-es-localization.md` was the audit's mistake, not the doc's. The only remaining gap for #764 is maintainer-level security review attestation.

2. **No cluster has a `docs/` entry.** Every cluster's evidence is code + tests only. The umbrella's "RoadmapTraceabilityReview" needs documentation pointers.

3. **Closing PR numbers are unknown for 5 clusters** (#756, #758, #759, #760, #762). The audit scout lacked `gh`/`bash` access; verification needs to happen in a follow-up step before #765 closes.

4. **Audit methodology lesson:** future audits must run with `bash` access so the scout can verify against `origin/main` directly, or the scout's first action must be `git fetch origin main && git checkout origin/main`. The original scout substituted local refs and working-tree inspection, which produced stale evidence.

## Navigation review

Every retained live route (12 components + 1 router) maps to one of the four mission categories. **No orphan pages.** See `navigation-mission.md` for the full matrix.

## Compatibility mode pre-flight

Backend `apply` coordinator short-circuits on `migration`/`read-only` modes. Catalog-driven `managedSettingKeys` enforces UI disable. Frontend disabled-state assertion for non-`full` modes is partial. **The actual `CompatibilityModeE2E` Playwright run is missing** (5 items in `compatibility-mode-pre-flight.md` §4).

## Recommended follow-up

1. **(resolved)** Correct the "#764 presets — merged ✅" claim — *the claim was correct; the audit was wrong.*
2. **(resolved)** Verify #764 status — *verified: all 4 named tests in `origin/main` after follow-up commit `b91ead7`.*
3. Open follow-up issues for: #764 security review attestation, #760 missing tests, #759 missing E2E, `CompatibilityModeE2E` Playwright, `docs/control-plane.md`.
4. Re-run the audit with `gh`/`bash` access to fill in closing PR numbers.

## Audit method

- Read-only inspection of the local main checkout (branch `docs/issue-765-roadmap-traceability-audit`, off `origin/main`).
- Cross-referenced with `.git/logs/`, `.git/FETCH_HEAD`, `.git/packed-refs`, `.git/refs/`, and inline code/test comments.
- Bulk evidence gathered by a `gentle-ai-explore` scout subagent; this PR synthesizes that evidence plus targeted re-verification.
- **Tool caveat:** the scout lacked `gh`/`bash` access, so closing PR numbers and `git log --grep` cross-checks were not first-class. Where PR numbers are marked "unknown" in the deliverables, the cluster's status is grounded on code + named tests being present in the local main checkout.

## Checklist

- [x] Read-only audit; no source code edits.
- [x] All deliverables under `odd/audits/issue-765/`.
- [x] Each cluster's evidence cites concrete SHAs, file paths, test names.
- [x] No claims asserted that file-system evidence contradicts.
- [ ] Closing PR numbers filled in for 5 clusters — **needs maintainer follow-up**.

---

**Generated as part of audit scope.** No source code edits were performed.
