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

T7 | S1,S5,S6 | CI rollback fixture and backend recovery (complete) | Fix default-settings readiness recovery and portable runtime failure injection, retaining rollback assertions and regression coverage. | commit: f8cfc48
T8 | S1,S5,S6 | independent verification (targeted recheck passed) | Verify shared readiness runtime-port semantics and regression risk (item 3). | commit: f8cfc48
T9 | S5,S6 | delivery (slice committed; acceptance green locally) | Record correction evidence, commit bounded slice, update PR #954, and observe CI. | commit: f8cfc48 (code), docs commit (this document)

T10 | S1,S5,S6 | visual-save HTTP500 diagnosis (not reproduced: no cause found) | Reproduce the reported save failure in isolated Docker state, retain logs/traces, and correct only the demonstrated cause. | commit: none (no cause demonstrated)

T11 | S1,S5,S6 | CI lint gate on the pending slice | The pending Go test additions trip gosec (G114, G306 x2) under the CI-pinned golangci-lint v2.14.0. Replace the timeout-less fixture listener with an explicit http.Server and annotate the two executable launcher fixtures. | commit: f8cfc48

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
L10. User (verbatim): "Puedes arreglar eso?" Authorizes fixing the reported
    CI failure. Run 42375267554 on 66ea06b failed at stack.spec.ts:388 because
    default Docker settings omit uiPort; 2 tests passed, 1 failed, 9 did not run.
    T7 is a separate bounded correction slice after 66ea06b, not a rewrite of
    the existing PR or the unavailable native validator result.
L11. User (verbatim): "Si  dale" authorizes extending T7 to
    internal/server/apply_runtime.go and its tests after Luna-6 reproduced
    RUNTIME_RECOVERY_FAILED when restored default settings omit uiPort.
    Preserve fail-closed handling of ambiguous or dynamic settings.
L12. Luna-6 implemented T7 and observed Go RED/GREEN, full Go suite and
    TypeScript pass, and 12 passing CI-equivalent default Docker tests against
    a rebuilt candidate image. FlowFuse was excluded exactly as in CI. Two
    no-retry full runs failed the visual-save scenario; independent verification
    must distinguish candidate causality. Native assess reviewDue=false
    (under_budget); escalation item 3 requires independent verification, now
    running. No native approval is claimed.
L13. Parent spot-check passed: go test ./internal/server -count=1 and
    git diff --check. Independent verifier crashed with ENOSPC before returning
    a verdict. Post-failure root filesystem reports 22G free (95% used), so
    the exact exhausted mount/quota remains unknown. No automatic retry,
    cleanup, commit, or push was performed; publication awaits user disposition.
L14. User authorized environment investigation and verification retry, then
    explicitly authorized deleting only /tmp/nrcc-887-verify.h5dwLI. That
    failed verifier scratch occupied the separate 16G /tmp tmpfs. Read-only
    Go module directories initially prevented deletion; enabling owner writes
    within that exact directory allowed cleanup. /tmp now reports 2% used,
    16G available. Independent verification restarted with no cache copying
    or separate module downloads.
L15. Independent verification reproduced two candidate-caused regressions vs
    66ea06b: computed keys can be misclassified as an omitted uiPort, and
    mandatory /proc child-environment reads reject hardened managed processes
    that worked before. T7 reopened for one bounded correction batch by Luna-6.
    The writer also retracted the claim that visual-save failures were unrelated:
    POST /api/config returned 500 in one run and retained failure artifacts were
    overwritten; causality remains unproven.
L16. Final independent verdict: correction required (S1/S5/S6 unmet).
    A narrow internal/service/process_linux.go runtime-port helper and tests
    in internal/service/process_runtime_test.go are included in T7 to preserve
    launch-port authority without mandatory /proc access; existing environment
    precedence and locking must remain intact. This is part of the same
    authorized backend recovery fix, not a new feature. Only the two confirmed
    blockers receive the bounded correction and targeted recheck.
L17. Luna-6 completed the bounded code correction: computed keys reject and
    managed readiness uses the last successful ProcessManager launch port,
    preserving environment precedence without /proc access. Full Go tests and
    rebuilt Docker focused rollback passed. Full acceptance was not rerun;
    visual-save HTTP500 remains unresolved. Writer status is partial. Targeted
    independent recheck and parent focused regression spot-check are running.
    Native assess remains reviewDue=false; no new native review was started.
L18. Independent targeted recheck passed both original blockers: computed
    keys reject even with explicit launch authority; a hardened managed child
    remains ready despite confirmed /proc environment denial. Environment
    precedence and launch snapshot probes passed. Parent focused server/service
    regression tests and diff check passed. T8 is complete for the bounded
    recheck only, not a full acceptance approval. T7/S6 and publication remain
    blocked by unresolved visual-save HTTP500 and missing full acceptance proof.
    No additional verification sweep, commit, or push was performed.
L19. User (verbatim): "Si autorizo" authorizes the focused Luna-6
    investigation of visual-save HTTP500 with retained logs and traces.
    Any fix must address a reproduced cause without weakened assertions,
    extra retries, or masking a backend error.
L20. Review of PR #954 found no merge conflict: the branch is level with
    origin/main (0 commits behind, 8 ahead) and GitHub reports MERGEABLE.
    The BLOCKED state comes from the failing "Docker stack acceptance (#434)"
    check (run 42375267554) plus the missing approving review, not from
    textual conflicts. The failing assertion was stack.spec.ts:388 —
    "the live settings source carries a uiPort to replace" — because the
    default container settings.js carries no uiPort, so the replace was a
    no-op. The uncommitted slice already replaces that fixture with a
    uiHost-based failure injection.
L21. Full real-stack acceptance reproduced green against the pending slice
    built into the release image: `docker compose build` + `up --wait`, then
    the CI-equivalent Playwright stack config (FlowFuse case excluded exactly
    as in CI) reported 12 passed, including "a runtime failure rolls back to
    the previous live configuration" (previously failing) and the
    issue-881 visual edit scenario (the reported HTTP500). The visual-save
    scenario was additionally re-run three times in isolation: 3 passed, so
    the HTTP500 was not reproduced and no cause was demonstrated. T10 is
    therefore closed as not-reproduced; no masking fix was applied.
L22. CI-pinned golangci-lint v2.14.0 (verified against the release
    checksum) reported 3 gosec findings in the pending slice — G114 on the
    fixture listener and G306 on the two executable launcher fixtures —
    which would fail the "Lint (golangci-lint)" required check. The
    committed baseline lints clean, so all three are introduced by this
    slice. Fixed in T11: the fixture now uses an explicit http.Server with
    ReadHeaderTimeout, and the two launcher fixtures carry #nosec G306
    annotations matching the existing repo precedent.
L23. Post-fix evidence: `go build ./...` clean; `go vet` clean; golangci-lint
    v2.14.0 reports 0 issues; `go test ./internal/server/... ./internal/service/...`
    passes; `cd frontend && npm test -- --run` passes 86 files / 587 tests;
    `npx tsc --noEmit` clean; eslint clean on the modified e2e spec;
    `git diff --check` clean. The visual-save scenario was re-run three
    times in isolation with no failure. No commit or push was performed;
    publication still awaits the user's disposition.
L24. User (verbatim): "Lo primero, el del slice" authorizes committing the
    pending slice first. The parent re-ran the gates independently before
    committing: `go build ./...` and `go vet ./...` clean; `go test
    ./internal/server/... ./internal/service/... -count=1` passes
    (internal/server 47s, internal/service 374s); golangci-lint v2.14.0 —
    the CI-pinned version, release tarball verified against the published
    checksum, built with go1.27.0 — reports 0 issues; `npx tsc --noEmit`
    and eslint on `frontend/e2e/stack.spec.ts` clean; `npm test -- --run`
    passes 86 files / 587 tests (2 skipped); `git diff --check` clean.
    Slice committed as f8cfc48 (5 files, 296 insertions, 37 deletions) and
    the branch is pushed to PR #954 for a fresh acceptance run.
