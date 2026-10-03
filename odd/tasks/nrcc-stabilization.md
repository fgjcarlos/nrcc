# NRCC stabilization

## Objective, scope, and authorization

Stabilize the existing all-in-one image before UI redesign: normalize Node-RED
version detection, preserve sessions on reload, and expose truthful update
capabilities/feedback. Implementation authorized after the repository audit.

- Tracker branch: `fix/nrcc-stabilization` (baseline; do not merge automatically).
- Current child: `fix/nrcc-stabilization-session`, based on the version child.
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
partially verified work stays unchecked. S1 committed: 285 additions and 12
deletions (297 authored lines), including 170 behavior/test/doc lines and 127
tracking lines. Running committed count: 297; generated files excluded.

## S1 — Normalize managed Node-RED versions

- [x] Complete verification and authorized work-unit commit.
- Status: **done — verified and committed on the version child**.
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
- Commit: `353049834b691e2f748a927824726e489ab69aa4`
  (`fix(runtime): normalize Node-RED version detection`).
- Slice: `fix/nrcc-stabilization-version` against tracker baseline; 297 lines.

## S2 — Coalesce refresh during session rehydration

- [ ] Complete behavior, checks, and authorized work-unit commit.
- Status: **in progress — fully verified; local commit pending**.
- Acceptance: bootstrap/interceptor share in-flight refresh; concurrent mounts
  issue one request, sequential attempts work, failures release coordination,
  reload retains authentication. Preserve logout/bootstrap gates/backend rotation.
- Allowed surfaces:
  - `frontend/src/shared/lib/api.ts`
  - `frontend/src/shared/lib/api.test.ts`
  - `frontend/src/shared/lib/index.ts` (re-export only if used)
  - `frontend/src/features/auth/hooks/useAuth.ts`
  - `frontend/src/features/auth/hooks/useAuth.test.ts`
  - `CHANGELOG.md`
- Foreground checks in a synchronized temporary source snapshot: Corepack pnpm
  focused Vitest for both test files, full frontend tests, frontend build, and
  repository diff check. No generated files or dependency installs in worktree.
- Test-first: concurrent callers/mounts, sequential calls, error retry, stale
  session. Checks: focused/full Vitest and canonical frontend build in a snapshot
  synchronized from working sources; real browser login/reload/request count.
- Implemented: bootstrap and API 401 recovery share `refreshAuthToken`.
- RED: caller regression observed three refresh requests instead of one.
- GREEN: 84 frontend files, 577 tests passed, two skipped; build and diff check
  passed. Independent focused run: two files, ten tests passed; full suite and
  build passed again. CSS minification warnings match the baseline.
  The declared pnpm test invocation ran the full suite rather than filtering.
- Candidate: 97 additions and 17 deletions (114 authored lines), excluding
  parent tracking changes; snapshot `/tmp/nrcc-audit-DRYCSyHs/s2-writer-bNHSCz`.
- Assessment: unassessable due untracked declaration; high-path independent
  automated checks, three authenticated reloads, and corrected cold logout passed.
- Previous verifier finished **partial**: wrong-cwd pnpm commands created root
  `pnpm-lock.yaml`, root `node_modules/`, and `frontend/node_modules/`.
  No Docker/browser checks ran. Direct filesystem evidence corrected the scout's
  absent-directory claims; all three artifacts had matching incident timestamps.
- Incident recovered: three generated paths reversibly moved to
  `/tmp/nrcc-audit-DRYCSyHs/s2-quarantine-iNnCh4/repository`, preserving layout.
  Nothing deleted; tracked diff hash identical before/after; `.codegraph/`
  untouched. Original generated paths absent; snapshot source hashes match.
- Independent evidence: `/tmp/nrcc-audit-DRYCSyHs/s2-checks-QhhbDg`.
- Worker: `musj8xqf-g-e8zr`; verifier: `musjiiq4-h-sm37` (both finished).
- Read-only incident diagnosis: `muskaari-j-ulto`, finished.
- First runtime verifier: `muski9ld-k-rf5n`, finished partial after building from
  the repository instead of the verified snapshot. No container/browser started;
  image `nrcc-s2-acceptance:s2-runtime-FAsdzt` retained, not acceptance evidence.
- Resumed verifier: `musknust-l-0sxh`, finished. Correct absolute snapshot build;
  three real reloads each made one successful refresh, retained authenticated
  `/configuration`, and had no 401/429. Node-RED `5.0.7`, editable/writable true,
  NRCC and Node-RED HTTP 200; no settings saved. Own container stopped.
- Runtime evidence: `/tmp/nrcc-audit-DRYCSyHs/s2-runtime-5VPTzl`; image
  `nrcc-s2-acceptance:s2-runtime-5VPTzl` retained. Working sources unchanged.
- Later logout follow-up reached `/login` but included two refresh 429s and
  protected update API 401s outside the measured reload batches. Causality and
  cookie-clear acceptance need clarification before closing this unit.
- Read-only challenge `musl473r-m-3vhr` finished: refresh limiter counts failed
  attempts (six per 15 minutes), persists per IP, resets on success. Backend is
  unchanged, but incomplete traffic history does not prove 429 causality.
- Cold verifier `musl6l6d-n-206k` stopped pending exact command authorization;
  no runtime mutation. Resumed `musld72k-o-kqol` completed a fresh cold cycle:
  setup 201, reload refresh 200, logout 200, protected route `/login`, cookie
  absent afterward, no resurrection or 429. Own container stopped.
- Cold evidence: `/tmp/nrcc-audit-DRYCSyHs/s2-logout-musl6l6d`.
  Immediate cookie-clear result is inconclusive: harness did not await completed
  logout before sampling cookies and used incomplete response-header access.
  Its refresh assertion also counted expected anonymous 401s outside reload.
- Corrected verifier `muslm80n-p-9mde` passed: one reload refresh 200; logout 200,
  clear-header true, cookie absent after response completion, protected `/login`,
  no resurrection or 429. Anonymous 401s are expected, not failed reloads.
  Playwright `headers()` omits cookie headers; `headerValues()` and response
  completion resolved the earlier measurement gap without product changes.
- Final evidence: `/tmp/nrcc-audit-DRYCSyHs/s2-logout-confirm-musld72k`.
  Both health probes 200, source hashes unchanged, accidental paths absent;
  own container exited 0, volumes/images/evidence retained.
- Rollback boundary: client refresh coordination/tests/release note only.
- Commit: pending verification; authorized locally, parent owns the commit.

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

Next: create the authorized local S2 commit, then the updates child and bounded
S3 implementation. Corrected cold verification passed; no product logout changes
were required. No dependency commands run in the worktree.
No further commit/delivery permission prompt is needed within the selected scope.
Full settings persistence/rollback and complete backup recovery remain unverified
and outside this stabilization phase.
