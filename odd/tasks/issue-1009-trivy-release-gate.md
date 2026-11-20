# Issue 1009 — make Trivy block the release
Branch: `fix/1009-trivy-release-gate`; base: `origin/main` (`15a07e8`); delivery: PR #1031, merged as `1850673`; forecast: ~120 authored lines.
Checks: `python3 -m unittest scripts/release/test_workflow.py`, YAML parse of the touched workflows, langscan. The gate itself is only observable on a real release/dispatch run.

## Specs
S1. "Fail on `CRITICAL,HIGH` when the workflow is called from the release, and record accepted exceptions in `.trivyignore` with a reason. Pull requests can stay advisory."
S2. Acceptance: "A release run fails when the image has an unignored critical or high finding."
S3. Acceptance: "`SECURITY.md` describes the new gate."

## Tasks
- T1 [S1,S2] `security.yml` gains a `fail-on-findings` `workflow_call` input (default false) and a separate CRITICAL,HIGH table scan with `exit-code: "1"` and `.trivyignore`; `release.yml` passes `true`; `.trivyignore` with entry rules; release workflow tests. Route: inline, test-first. Risk: item 5. Commit: 09bba1f.
- T2 [S3] `SECURITY.md` and `docs/release-packaging.md` describe the gate and the exception process. Route: inline. Commit: 21e3251.
- T3 [S1-S3] independent verification (`gentle-ai-verify`, Risk: item 5). Done: S1-S3 met statically; S2 end to end needs a real dispatch run.

## Log
L1. User: "Resuelve la issue #1009"
L2. Evidence: trivy-action v0.36.0 (`ed142fd`) `entrypoint.sh` unsets `TRIVY_SEVERITY` for SARIF unless `limit-severities-for-sarif: true`, so flipping `exit-code` on the SARIF step would fail on LOW/MEDIUM too. Decision: keep the SARIF step advisory and add a separate gate step.
L3. Evidence: open Trivy code-scanning alerts on 2026-11-20 include critical (CVE-2026-123158) and several high findings, so the first release run after this change will fail until they are fixed or triaged into `.trivyignore`. No exceptions were pre-filled: accepting a finding is a maintainer decision.
L4. Evidence T1/T2: RED 5 new tests (3 fail, 2 error) before the workflow change, SARIF-advisory pin passed; GREEN `python3 -m unittest scripts/release/test_workflow.py` 63 OK. actionlint: only the pre-existing SC2034 at `release.yml:272` (same on base). Real Trivy (`aquasec/trivy:latest`, `python:3.6-alpine3.10`, `--severity CRITICAL,HIGH --exit-code 1`): template `.trivyignore` exit 1; all IDs ignored with comment + future `exp:` exit 0; expired `exp:` exit 1. langscan: no finding in touched files.
L5. Verify (gentle-ai-verify): S1-S3 met; trivy on PATH for the second call via setup-trivy `$GITHUB_PATH`, cache shared, table format keeps severity; package/image/release have no `always()`. Advisory fixed: release-packaging wording (PRs are direct triggers, not calls). Accepted advisories: duplicate cache-save warning; pre-existing SARIF reports every severity.
L6. Evidence S2 end to end: release dry run `workflow_dispatch` 44828804742 (`v1.0.0-rc.0`, branch head 46b8ead) concluded `failure`; only `Trivy release gate (CRITICAL,HIGH)` failed (OS packages: 13 = 10 HIGH + 3 CRITICAL, e.g. CVE-2026-123158 openssh, CVE-2026-105437 git; language packages: 11 HIGH, e.g. CVE-2026-111181 body-parser); package, image, release and install smoke were skipped. PR #1031 checks: 20 pass, Trivy advisory there.
L7. Re-verify (gentle-ai-verify) at 0b9e248: S1-S3 met, 63 OK, PR Trivy gate step skipped (advisory). PR #1031 was merged before 0b9e248 was pushed, so this closing document lands in a follow-up PR. Open item for the maintainer: fix or accept the 24 blocking findings before the next release.
