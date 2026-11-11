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
T9 | S5,S6 | delivery (slice committed; hosted CI green) | Record correction evidence, commit bounded slice, update PR #954, and observe CI. | commit: f8cfc48 (code), docs commit (this document)

T10 | S1,S5,S6 | visual-save HTTP500 diagnosis (not reproduced: no cause found) | Reproduce the reported save failure in isolated Docker state, retain logs/traces, and correct only the demonstrated cause. | commit: none (no cause demonstrated)

T11 | S1,S5,S6 | CI lint gate on the pending slice | The pending Go test additions trip gosec (G114, G306 x2) under the CI-pinned golangci-lint v2.14.0. Replace the timeout-less fixture listener with an explicit http.Server and annotate the two executable launcher fixtures. | commit: f8cfc48

T12 | S2 | restart helper | `restartNodeRed` in stack.spec.ts requires pid>0, status running, changed pid and a live probe (same contract as the dashboard restart test); `applyEditorAuth` inherits the running check. | commit: abce08e
T13 | S1 | source preservation | Visual-save scenario asserts the raw settings.js outside the edited key survives byte for byte (marker/unmanaged content before and after). | commits: abce08e, d24e722 (client timeout fix, L30-L31)
T14 | S1 | cancellation | Real-stack scenario: a valid changed candidate reaches the confirmation dialog and is cancelled with the mouse and with the keyboard; no apply request, no write, no restart. The HTTP-abort test is renamed to what it proves. | commit: abce08e
T15 | S4 | Go constructor injection | Every empty-cache `NewUpdateService` call site in update_test.go and internal/handler/updates_test.go gets its version-check dependencies (seeded cache or fixture PATH) before construction. | commit: 339e3a7
T16 | S5 | CI evidence retention | pr.yml and acceptance.yml upload Playwright reports and traces (`if: always()`), with distinct folders for the default and FlowFuse stack runs. | commit: 067c91b
T17 | S5,S6 | evidence documentation | Document the acceptance scenarios, which suites are mocked fixtures and which are real-runtime evidence, the skipped cases and the environment limitations. | commit: d0b02a2
T18 | S6 | required gate | After the slice is green, add "Docker stack acceptance (#434)" to the required status checks of main (user decision L29). | done 2026-11-11 (repository setting, L36)

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
L25. Hosted CI on the pushed head is green: run 42895843928
    ("Docker stack acceptance (#434)") passed in 5m44s, the gate that failed
    at 66ea06b, and run 42895843896 passed "Backend (Go 1.27.3)",
    "Frontend (Node 22)", "Lint (golangci-lint)", "OpenAPI Spec" and
    "Release policy". All 20 reported checks pass, none failing or skipped;
    GitHub reports the PR as MERGEABLE with mergeStateStatus BLOCKED only
    because reviewDecision is REVIEW_REQUIRED.
    Native review: the parent started the ordinary review for this target
    (lineage review-e10d8c755d5bc2f1, tier medium, lenses review-risk,
    review-resilience and review-reliability, 13 changed paths, 1179
    original changed lines, correction budget 200). The provider's collect
    route rejects every offered binding without mutation or authority burn
    — the reviewer group returns "Provider collectBinding collides with a
    different registered route" and the single-slot capture returns
    "capture-route-registration-rejected / selector-mismatch" — while
    STATUS keeps re-offering the same three materialize bindings. The
    native review outcome for this exact candidate is therefore recorded as
    unavailable (explicit); no native approval is claimed. The
    risk-gated plan applies: writer self-verification (L24) stands and no
    separate verifier runs.
L26. The final documentation commit changed the candidate again, so the
    parent started a second native review for it (lineage
    review-f771b1ed9ddb2a0b, tier medium, same three lenses, 1199 original
    changed lines, correction budget 200). The provider's collect route
    rejects it the same way, and this time the rejection is independent of
    any transcription: the exact provider-owned collect input, read
    verbatim from the provider's own status, is refused by the single-slot
    route as "missing or stale for current STATUS", and the reviewer-group
    route refuses every array shape the host can emit. No authority was
    burned, no reviewer ran, and no native approval is claimed; the native
    review outcome for this candidate is recorded as unavailable. The
    reviewer-group route also requires the three bindings as one flat
    array, which this host's argument emission cannot produce, so the
    native review cannot be advanced from this session. Further native
    attempts for this feature need an explicit human request after the
    provider route is repaired; the hosted CI and the parent's gate re-run
    (L24) remain the verification evidence for the slice.
L27. User (verbatim): "Resuelve la issue #887". PR #954 (edc996d8) and
    PR #897 merged with "Refs #887", so the issue stayed open.
L28. Read-only audit of main 2fb76c3 (explorer task mwc4u9qc-1-m3lh) against
    the six acceptance criteria: criterion 3 satisfied; gaps remain in
    criterion 1 (no byte-level source preservation check; the abort test
    submits invalid content, so it passes even when the abort does nothing),
    criterion 2 (`restartNodeRed`, stack.spec.ts:88, checks no pid, status or
    pid change), criterion 4 (empty-cache `NewUpdateService` calls run a real
    version check before mocks), criterion 5 (no CI job uploads Playwright
    reports/traces; mocked vs real-runtime evidence undocumented) and
    criterion 6 (no documented acceptance scenario list; the branch
    protection of main requires only Frontend, Lint, govulncheck, pnpm audit
    and Backend, not "Docker stack acceptance (#434)").
L29. User decisions: (1) "Hacerlo requerido en main (Recomendado)" — add
    "Docker stack acceptance (#434)" to the required checks of main;
    (2) "Cancelar en el diálogo (Recomendado)" — cancellation means the user
    cancels the confirmation dialog; scenario covers mouse and keyboard, no
    request, no write, no restart. No backend abort semantics are added.
L30. Real-stack run after T12-T14: 12 passed; the visual-save scenario failed
    because the "Configuration saved successfully" toast never appeared.
    Diagnosis (worker mwc5i1ri-3-fck6) found a product defect, independent of
    the preservation probes (reproduced without them through the API). The UI
    sends `POST /api/config` with the shared 10 s axios default
    (frontend/src/shared/lib/api.ts:13). The backend write + restart +
    readiness took 12-14.7 s on this stack (readiness budget 30 s,
    internal/service/apply_readiness.go:45-53). The browser aborts at 10 s; the
    apply runs on the request context (internal/handler/config.go:191), so it
    rolls back and logs status=500. This explains the unreproduced HTTP 500
    of L15/L21: the save passes or fails depending on whether it finishes
    under 10 s. With probes and no client timeout the save returned 200 and
    every probe survived byte for byte.
L31. Parent decision: fix the client side only — a per-request timeout that
    covers the backend budget for the config/settings apply calls, with unit
    tests (precedent: L7). No backend abort semantics are added (L29).
L32. Evidence and stop point (2026-11-11). Committed on branch
    test/issue-887-real-stack-acceptance (not pushed, no PR yet): d24e722
    (client timeout fix), abce08e (T12-T14 stack.spec.ts), 067c91b (T16 CI
    artifacts), d0b02a2 (T17 docs). Checks observed: real-stack suite with the
    rebuilt image, FlowFuse excluded as in CI, 13 passed / 0 failed / 0
    skipped (the visual save POST /api/config returned 200 in 11985 ms);
    `pnpm test -- --run` 86 files, 592 passed, 2 skipped; `tsc --noEmit`
    clean; `pnpm lint` 0 errors (1 pre-existing warning in FormFields.tsx);
    actionlint v1.7.12 clean on pr.yml and acceptance.yml; langscan no
    findings. The FlowFuse case was not run locally.
    Slice boundary: these four commits hold 475 authored lines. As in L8,
    the feature ships as one PR with one commit per work unit.
    User (verbatim): "tenemos que cerar por hoy". T15 worker
    mwc54850-2-txav was cancelled mid-run on the user's stop. Its edits stay
    UNCOMMITTED and UNVERIFIED in the worktree: internal/service/update.go,
    update_test.go, update_history_test.go, update_shutdown_test.go,
    internal/handler/updates_test.go, updates_integration_test.go (+237/-61).
    The user approved extending the surfaces to update_history_test.go,
    update_shutdown_test.go and updates_integration_test.go (option A). The
    worker reported RED: the host npm/node-red were invoked by
    empty-cache constructions.
    Resume with: review that diff and confirm the RED regression test exists.
    Then run `go test ./internal/service/ ./internal/handler/ -count=1` (if
    the thread budget runs out, use `-p 1 -parallel 2`), `go vet`, `gofmt -l`
    and golangci-lint v2.14.0, and commit T15. Push, open the PR with
    "Closes #887" and observe CI, including "Docker stack acceptance (#434)".
    Once it is green, run T18 (add that check to the required checks of
    main).
    Also pending outside this feature: stash@{0} holds the L19 log entry of
    the stale local branch fix/issue-884-release-packaging (#884 landed via
    PR #952); its disposition awaits the user.
L33. Missing proof at the stop: the close declared "Risk: item 3" (the new
    CONFIG_APPLY_TIMEOUT_MS applies to configService.updateConfig,
    configService.applyConfig — also used by securityCenterHelpers.ts:44 —
    and settingsService.saveRaw, which several screens share). No independent
    gentle-ai-verify ran, because the user stopped for the day. Resume by
    launching gentle-ai-verify on d24e722 with that Risk line, before the PR
    is published.
L34. User (verbatim): "continua". Independent verification
    (gentle-ai-verify mwc8mtgx-1-odz1) of the Risk item 3 change d24e722:
    PASS. Every caller of updateConfig, applyConfig and saveRaw keeps its
    pending and disabled state until the response arrives; mutations are not
    retried; the 401 refresh resends with the same timeout; no UI timer
    assumes 10 s; vitest on configuration and security: 19 files, 127 passed,
    2 skipped; tsc clean. Advisory, pre-existing: the presets confirm dialog
    (AdvancedSettings.tsx:203) has no pending guard; the backend answers a
    second save with APPLY_IN_FLIGHT. T15 diff verdict: no defect, and no
    assertion removed (counts of t.Fatal and t.Error equal or higher). It was
    unverified in the verifier sandbox (GOPROXY=off). The parent then ran:
    `gofmt -l` on the 6 changed files is clean; `go vet ./internal/service/
    ./internal/handler/` passes; `go test ./internal/handler/ -count=1`
    ok (452 s); `go test ./internal/service/ -count=1 -timeout 30m
    -parallel 4` ok (615 s). A first run with `-p 1 -parallel 2` hit the
    600 s default timeout, which is an environment limit, not a test failure
    (CI uses -timeout 20m). golangci-lint was not run locally; the required
    "Lint (golangci-lint)" check covers it. T15 committed as 339e3a7.
L35. PR #962 opened ("Closes #887"). Native review lineage
    review-4407a171fa3fbeb1 (tier high; resilience, readability and
    reliability lenses): approved, acknowledgement completed (authority
    burned). It left 10 non-blocking advisory findings (R2-001..R2-006,
    R3-001..R3-004) for later work. Hosted CI on #962: 17 checks pass.
    "Frontend (Node 22)" and "Docker stack acceptance (#434)" failed twice
    in the runner's `playwright install --with-deps` step ("Temporary failure
    resolving 'security.ubuntu.com'"), a GitHub runner DNS outage
    independent of the change. "govulncheck" fails on main too
    (GO-2026-6859, net/http, fixed in go1.27.4). User decision: "PR aparte
    con el bump (Recomendado)". PR #963 (d630622, chore/go-1.27.4-govulncheck)
    bumps the toolchain. Independent verification passed, and govulncheck
    passes on #963. Its Frontend and Docker acceptance checks fail on the
    same runner DNS outage. Next: once the runners resolve again, rerun the
    failed jobs, merge #963 (user decision), rebase #962, rerun CI, then T18.
L36. User (verbatim): "opcion 2 y el stash incluyelo se sube". PR #963 was
    merged with --admin (8de386d4); GitHub refuses self-approval, so --admin
    also bypassed the review requirement. #962 was updated with main by a
    merge (f173eba, no force push). Hosted CI on #962: 20 of 20 checks pass,
    including "Docker stack acceptance (#434)" and govulncheck. T18 done: the
    required status checks of main are now Frontend (Node 22), Lint
    (golangci-lint), govulncheck, pnpm audit, Docker stack acceptance (#434)
    and Backend (Go 1.27.4). The bump renamed the Backend job, so the stale
    required context "Backend (Go 1.27.3)" was replaced; otherwise no PR
    could merge. #962 is MERGEABLE except for REVIEW_REQUIRED. The #884
    stash was committed as 1781dc5 and the branch
    fix/issue-884-release-packaging was pushed.
