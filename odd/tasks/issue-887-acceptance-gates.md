# Issue 887 — Prove configuration safety against the real stack and repair unreliable gates

Header: Real-stack Playwright acceptance must cover the central configuration
transaction and all gates must be reliable. Source: issue #887 (P1, approved).
Read everything below until `## Log` before editing.

## Specs

S1. Add real-stack visual save, source preservation, partial security patches,
    invalid syntax/semantics, stale no-write, runtime rollback and cancellation
    scenarios. (Issue acceptance criterion 1, verbatim scope: "visual save,
    source preservation, partial security patches, invalid syntax/semantics,
    stale no-write, runtime rollback and cancellation scenarios".)
S2. "Verify PID > 0, running status, changed PID and a successful live probe."
    The current restart callback `frontend/e2e/stack.spec.ts:284` returns 0 on
    unavailable/stopped runtime and passes `not.toBe(before pid)` — it must not
    pass on a stopped runtime or unrelated process.
S3. "Parse computed reduced-motion durations correctly; test mouse and keyboard
    restore confirmation without masking interaction defects." (Fix the
    incorrect destructuring of split capture output in the reduced-motion test,
    `frontend/e2e/a11y.spec.ts`.)
S4. "Use a dedicated subprocess fixture/PID and inject version-check
    dependencies before service initialization; wait for polling shutdown
    before fixture cleanup." Applies to
    `internal/service/packagemanager_context_test.go` (sleep-as-npm/global
    pgrep fixture) and `internal/service/update_test.go`
    (constructor-before-mock external version check, TempDir cleanup race).
S5. "Retain traces/reports and distinguish mocks, actual runtime evidence,
    skipped cases and environment limitations."
S6. "Make the exact candidate's required checks and documented acceptance
    scenarios pass before publication."

## Tasks

T1 | S1 | frontend e2e real-stack config transaction | stack.spec.ts scenarios: visual save, source preservation, partial security patches, invalid syntax/semantics rejection, stale no-write, runtime rollback, cancellation. | commit: 8b5e0bf
T2 | S2 | frontend e2e restart gate | Strengthen restart PID polling helper: pid>0, status 'running', pid changed, successful live probe. | commit: 8b5e0bf
T3 | S3 | frontend e2e a11y | Fix reduced-motion duration parsing; add mouse and keyboard restore confirmation. | commit: 985f13f
T4 | S4 | Go test fixtures | Dedicated subprocess fixture/PID (no global pgrep), version-check deps injected before service init, wait polling shutdown before cleanup, fix TempDir race. | commit: 35baf8a
T5 | S5,S6 | gates & evidence | Run `cd frontend && npm test -- --run`, `npm run test:e2e` (or document env limitations with skips annotated), `go test ./...`; retain Playwright traces/reports; record results. | commits: bc59b4a, 052f0ec
T6 | S5,S6 | review correction workflow | Fix R4-001/R4-002, verify the correction, and finish provider-issued targeted validation. | commit: 052f0ec; validation blocked by stale provider binding

## Log

L1. User (verbatim): "Crea un plan para resolver la issue #887, luego delega en
    gpt-luna-6 la implementacion"
L2. Issue #887 open, labels bug/frontend/status:approved/area: devops. Baseline
    audit 2026-10-14 finding R13, commit 58d922ab. Local Chromium blocked by
    environment socket restrictions and Docker unavailable at audit time
    (environment limitation, not product failure — must be annotated, not
    masked).
L3. Root cause of the Node-RED `/auth/token` "getSessions undefined" failure:
    the locally launched stack did not honour the CI environment contract
    (`NODE_RED_USER_DIR` / `NODE_RED_SETTINGS` from docker-compose.e2e.yml).
    NRCC then wrote `adminAuth` into `~/.node-red/settings.js` while the
    running Node-RED loaded `data/settings.js`, so the applied auth never
    reached the runtime. Not a product defect. With the CI contract applied,
    `POST /api/config/apply` + `POST /api/runtime/restart` yields HTTP 401 on
    `/flows` and a valid OAuth token from `/auth/token` (client_id
    `node-red-editor`, client_secret, grant_type=password).
L4. Orchestration incident (parent): two luna-6 tasks ran concurrently against
    the same stack, so one task's cleanup removed the `adminAuth` the other
    task had just applied. The stale task was cancelled; one writer only from
    that point on.
L5. FlowFuse coverage requires the CI compose route (port 31880, Docker
    unavailable locally): it must skip locally with an explicit environment
    message and stay active in CI.
L6. Verification (parent, independent re-run against the live local stack —
    NRCC :3001 and Node-RED :1880 with the CI environment contract):
    `npm run test:e2e:stack` exit 0 — 12 passed, 1 skipped (FlowFuse local
    Compose route unavailable); `npm test -- --run` exit 0 — 86 files, 587
    passed, 2 skipped; `npm run test:e2e` exit 0 — 23 passed; `npx tsc
    --noEmit` exit 0; `go test ./...` exit 0; `git diff --check` exit 0.
L7. Product defect found and fixed by the S3 work: the confirmation dialog
    sat below its own `z-50` backdrop, so a real pointer click never reached
    Confirm. Fixed in frontend/src/shared/components/ConfirmationDialog.tsx
    (`z-[60]`); the mouse test now uses a normal click.
L8. Delivery: four work-unit commits on branch fix/887-acceptance-gates
    (35baf8a S4, 985f13f S3, bc59b4a S5, 8b5e0bf S1+S2), rebased on
    origin/main (d82849e). Total 666 authored changed lines; the repo's
    merged history already contains larger single PRs (#952: 2865 lines),
    so the change ships as one PR for issue #887 rather than a chain.
L9. Review correction (parent): R4-001 and R4-002 fixed in 052f0ec. The
    stack suite passed (12 runnable tests, 1 FlowFuse environment skip),
    frontend unit tests passed (86 files, 587 tests), TypeScript check passed,
    and lint passed with one pre-existing warning. Simulated CI with an
    unavailable stack failed as required. Pushed 052f0ec to PR #954; GitHub
    reports the branch head at that commit. Provider targeted validation has
    not run: gentle_review_capture rejected the exact current binding multiple
    times as missing or stale without mutation. A subsequent bound status
    returned unrelated/empty candidate scope and no collect binding, so native
    review cannot be advanced safely from this session.
