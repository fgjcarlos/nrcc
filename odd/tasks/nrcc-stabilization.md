# NRCC stabilization

## Objective, scope, and authorization

Stabilize the existing all-in-one image before UI redesign: normalize Node-RED
version detection, preserve sessions on reload, and expose truthful update
capabilities/feedback. Implementation authorized after the repository audit.

- Tracker branch: `fix/nrcc-stabilization` (baseline; do not merge automatically).
- Current child: `fix/nrcc-stabilization-version`, based on the tracker.
- Baseline: `a3e7382d8a1cb814cebcc5881fda75edabfbff42`.
- User explicitly authorized one local commit per verified unit with tests/docs.
  Push, pull requests, merge, and publishing remain **not authorized**.
- Preserve untracked `.codegraph/`; exclude production containers/data/secrets.
- Out of scope: UI redesign, expanded backups, settings apply/rollback wiring,
  backend authentication-policy changes, Docker socket integration, auto-upgrades.

## Evidence, routing, and delivery

The canonical image emits multiline `Node-RED v5.0.7` output; prior parsing gates
configuration read-only and reports update version unknown. Login then reload
reproduced session loss; parallel refresh rotation is the likely client cause.
Image-managed updates are refused, but the UI offers them and hides the reason.

Baseline: 84 frontend files, 575 tests passed, two skipped; frontend build and
14 Go test-bearing packages passed. Use declared Corepack pnpm 11.12.0 (global
11.13.0 failed before scripts), genuine frontend dist for Go embed, and Go caches
outside source snapshots. Service tests take about 274 seconds.

Route each unit through one delegated writer (mandatory multi-file trigger),
with observed caller-level RED, minimum GREEN, and focused refactor checks.
After writer returns, follow native ASSESS for independent verification; runtime
checks remain applicable. RDD is off; do not enable it. No concurrent writers.

Forecast: 550–800 authored diff lines including tracking, generated schema
excluded; advisory only, not a cap. Keep behavior/tests/docs together.
Delivery strategy: `auto-chain`; user chose `feature-branch-chain`. Local branch
order: tracker → version → session → updates. First future PR targets tracker,
later slices target the preceding child. This plan does not authorize PR creation.
Each unit needs its own authorized Conventional Commit; uncommitted or
partially verified work stays unchecked. Actual S1 diff: 158 additions and 12
deletions (170 authored lines), including 87 new-file lines; committed count zero.

## S1 — Normalize managed Node-RED versions

- [ ] Complete verification and authorized work-unit commit.
- Status: **in progress — verified; preparing authorized local commit**.
- Acceptance: consistent bare version for native/Docker/update callers; Node-RED
  5 editable through unchanged compatibility policy; invalid or Node.js-only
  output fails closed; prerelease/build metadata retained; generic tools unchanged.
- Allowed surfaces: `internal/service/version.go`, `version_test.go`, `host.go`,
  `host_test.go`, `update.go`, `update_test.go` (all under `internal/service/`),
  and `CHANGELOG.md`.
- Implemented: shared strict `nodeREDVersionFromOutput`; all three callers use it.
- RED observed: host retained full CLI output; update version was undetectable.
- GREEN: `go test -count=1 ./internal/service -run
  'NodeRED|NodeRed|Version|Compatibility|HostService'` passed; same focused run
  with `-race` passed; `git diff --check` passed (writer evidence).
- Independent assessment: **unassessable** because explicit untracked declaration
  was unavailable; returned high-risk plan requires independent verification.
  Do not stage files or start review to bypass this assessment outcome.
- Independent checks passed: focused Go (54.108s), focused race (57.387s), full
  service suite (210.968s), and diff check. No candidate-caused regression found.
- Canonical default-target Docker rebuild passed. Live API: `5.0.7`, adapter
  `nodered-5`, editable/writable true; all five observed UI fields enabled.
  NRCC and Node-RED probes returned 200. No settings saved or rollback tested.
- Evidence: `/tmp/nrcc-audit-DRYCSyHs/s1-checks-5FIyUb/evidence/`, source hashes
  matched the working tree. Both new S1 containers stopped; volumes/images kept.
  An intermediate-stage build was diagnosed and not used for acceptance.
- The local tracking file matches `.gitignore` pattern `nrcc-*`; explicitly
  include this intended feature document if its work-unit commit is authorized.
- Worker: `musesdzt-e-e4hi`; verifier: `musf5ava-f-wvt7`.
- Rollback boundary: helper/callers/tests/release note only.
- Commit: authorized; prepare `fix(runtime): normalize Node-RED version detection`.
  Do not start S2 until this verified unit is committed.

## S2 — Coalesce refresh during session rehydration

- [ ] Complete behavior, checks, and authorized work-unit commit.
- Status: **pending**.
- Acceptance: bootstrap/interceptor share in-flight refresh; concurrent mounts
  issue one request, sequential attempts work, failures release coordination,
  reload retains authentication. Preserve logout/bootstrap gates/backend rotation.
- Planned surfaces: actual shared API/auth hook and corresponding tests under
  `frontend/src/shared/lib/` and `frontend/src/features/auth/hooks/`, plus
  `CHANGELOG.md`. Resolve exact files before dispatch, no unused re-export.
- Test-first: concurrent callers/mounts, sequential calls, error retry, stale
  session. Checks: focused/full Vitest and canonical frontend build in a snapshot
  synchronized from working sources; real browser login/reload/request count.
- Rollback boundary: client refresh coordination/tests/release note only.
- Evidence/assessment/commit: pending.

## S3 — Expose truthful update capabilities

- [ ] Complete behavior, checks, and authorized work-unit commit.
- Status: **pending**.
- Acceptance: server-derived image/external/unknown capabilities fail safely;
  no impossible Update Now action; unknown versions do not imply an upgrade;
  actionable image replacement guidance and distinct async acceptance/completion;
  preserve supported npm-managed path and backend guards.
- Design: reuse status endpoint, not new endpoint/client strategy guesses. Check
  cache migration semantics and actual locale/model paths before implementation.
- Planned surfaces: update model/service/handler tests, update client/view/tests,
  English/Spanish update locales, `docs/openapi.yaml`, generated
  `frontend/src/shared/api/schema.ts`, and `CHANGELOG.md`; narrow before dispatch.
- Test-first: unsupported/supported/absent fields, unknown version, cached status,
  direct-apply guard and UI feedback. Checks: focused Go/Vitest, schema generation,
  locale checks, build and final full suites; real image status/UI without npm
  upgrade, both processes healthy. Regenerate generated code, never edit by hand.
- Rollback boundary: status/feedback contract, consumers/tests/schema/docs only.
- Evidence/assessment/commit: pending.

## Recovery and next step

Feature document and full Engram mirror: `odd/nrcc-stabilization/tasks`.
Audit artifacts: `/tmp/nrcc-audit-DRYCSyHs/`; baseline labeled container
`nrcc-audit-nrcc-audit-drycsyhs` is stopped; image/volumes/evidence retained.
Only test credentials in existing mode-0600 files may be used; never print them.
Build/test from current working source, including intended new parser files,
not stale HEAD. No production access. New verifier owns separate labeled runtime
resources and must stop only its own container after checks.

Next: commit verified S1 on the version child branch, record its identity, then
create the session child from that boundary and start S2 with regression tests.
No further commit/delivery permission prompt is needed within the selected scope.
Full settings persistence/rollback and complete backup recovery remain unverified
and outside this stabilization phase.
