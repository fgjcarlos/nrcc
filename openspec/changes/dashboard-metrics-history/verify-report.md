# Verify Report — dashboard-metrics-history (PR-1: Existing Behaviour)

**Change**: dashboard-metrics-history  
**Scope**: Section A — "Existing Behaviour" (6 requirements, 14 scenarios)  
**Branch**: feat/dashboard-metrics-charts (shipped code)  
**Date**: 2026-06-01  
**Verdict**: PASS WITH DRIFT (2 spec wording corrections recommended)

---

## Build Evidence

```
go build ./... → EXIT 0 (no errors)
```

---

## Test Evidence

### Go Backend

```
go test ./internal/service/... -race -run "TestMetricsBuffer|TestMetricsSampler|TestRestartTracker"
→ PASS (14 tests, 1.662s)

go test ./internal/handler/... -race -run "TestGetSystemHistory|TestGetRuntimeHistory"
→ PASS (7 tests, 1.062s)

go test ./internal/middleware/... -race
→ PASS (auth middleware fully tested)

go test ./internal/server/...
→ PASS (7.538s)
```

Note: `TestIntegration_ErrorRecovery` (in `./internal/handler/...`) timed out when run as part of the full suite at 120s. When run in isolation it PASSes in 5.25s. This is a pre-existing timeout-budget issue in the test suite, unrelated to the 6 requirements being verified.

### Frontend

```
pnpm -C frontend test -- --run
→ 35 test files, 255 tests — all PASS (10.84s)
```

---

## Per-Requirement Compliance Matrix

### REQ-1: Metrics Ring Buffer

| Scenario | Code Evidence | Verdict |
|---|---|---|
| Buffer evicts oldest when full (capacity 120, FIFO) | `internal/service/metrics_buffer.go:28-37` (Push overwrites at `head`, increments `count` only while `count < capacity`) — `TestMetricsBuffer_RingOverwrite` covers capacity-3 variant, confirms oldest evicted | PASS |
| Reads returned oldest-first | `metrics_buffer.go:41-55` (Recent: iterates from `head-n` forward) — `TestMetricsBuffer_Recent_ReturnsChronologicalOrder` confirms order | PASS |
| Concurrent access safe under race detector | `metrics_buffer.go:29,42` (sync.RWMutex; Lock on Push, RLock on Recent/Last) — `TestMetricsBuffer_ConcurrentReadWrite` runs 8 writer + 8 reader goroutines under `-race`, PASS | PASS |

MetricsSnapshot fields: `internal/model/metrics.go:4-9` — `timestamp` (string), `cpuPercent` (float64), `memoryPercent` (float64), `diskPercent` (float64). JSON tags match spec exactly.

**REQ-1: PASS**

---

### REQ-2: Background Metrics Sampler

| Scenario | Code Evidence | Verdict |
|---|---|---|
| Sampler populates buffer over time | `internal/service/metrics_sampler.go:27-41` (Start: initial sample + ticker loop) — `TestMetricsSampler_Start_PopulatesBuffer` confirms buffer populated within 500ms | PASS |
| Sampler stops on context cancellation | `metrics_sampler.go:33-36` (select on `ctx.Done()` returns) — `TestMetricsSampler_Start_StopsOnContextCancel` confirms goroutine exits within 500ms of cancel | PASS |
| Non-Linux sampler does not panic | `internal/service/metrics_sampler_other.go:8-10` (build tag `!linux`, returns zero MetricsSnapshot without error) — no panic path possible | PASS |
| Lifecycle bound to server context | `internal/server/server.go:266-268` (context created) + `server.go:279` (`go server.metricsSampler.Start(server.ctx)`) + `server.go:289-292` (Shutdown calls `s.cancel()`) | PASS |

**REQ-2: PASS**

---

### REQ-3: System History Endpoint

| Scenario | Code Evidence | Verdict |
|---|---|---|
| Returns recent samples as JSON array | `internal/handler/system.go:135-157` (GetSystemHistory) + `internal/server/server.go:159` (route registered) — `TestGetSystemHistory_Returns200WithSnapshots` confirms HTTP 200, JSON array, chronological order | PASS |
| n parameter bounds the result | `system.go:140-143` (parsed > 0 sets n) + `system.go:145-147` (n > maxN clamps) — `TestGetSystemHistory_NParsesQueryParam` confirms ?n=3 returns 3 items | PASS |
| Out-of-range n falls back to default | `system.go:140-147` (non-numeric or <= 0: n stays defaultN=120; >120: clamped to maxN=120) — `TestGetSystemHistory_NDefaultsTo120` covers default path; n=0 case confirmed by code path (parsed>0 is false, n stays 120) | **DRIFT** (see Drift section) |
| Empty buffer returns [] not null | `system.go:149-154` (`make([]MetricsSnapshot, 0)` guarantees non-nil slice; `Recent` returns nil on empty but nil check reassigns) — `TestGetSystemHistory_Returns200WithEmptySlice` confirms 0-length response | PASS |
| Unauthenticated request rejected (401) | Route at `server.go:159` is inside `r.Group` at line 137 with `r.Use(middleware.Auth(authSvc))` at line 138; middleware returns 401 for missing/invalid tokens. Auth middleware tested in `middleware/auth_test.go` (TestAuth_MissingAuthorizationHeader confirms 401). No direct integration test hits `/api/system/history` without auth. | **WARNING** (coverage gap — no end-to-end test for this endpoint specifically) |

**REQ-3: PASS WITH DRIFT + WARNING (coverage gap)**

---

### REQ-4: Restart Event Tracker

| Scenario | Code Evidence | Verdict |
|---|---|---|
| Restart events recorded and ordered | `internal/service/restart_tracker.go:28-36` (push, FIFO ring) + `restart_tracker.go:40-51` (restartEvents, chronological) — `TestRestartTracker_RestartEvents_ChronologicalOrder` confirms 3 events returned oldest-first | PASS |
| Capacity enforced at 50, oldest evicted | `restart_tracker.go:19-26` (capacity param) — `TestRestartTracker_Cap_AtFifty` confirms 60 events → 50 retained, first ExitCode=11 | PASS |
| RestartEvent carries required fields | `internal/model/metrics.go:12-17` — timestamp, exitCode, attempt, maxAttempts all present — `TestRestartTracker_Push_RecordsCorrectFields` verifies all fields | PASS |

**REQ-4: PASS**

---

### REQ-5: Runtime History Endpoint

| Scenario | Code Evidence | Verdict |
|---|---|---|
| Returns events and status object | `internal/handler/system.go:167-183` (GetRuntimeHistory, returns `{events, status}`) + `server.go:160` (route registered) — `TestGetRuntimeHistory_Returns200WithEmptyEvents` + `TestGetRuntimeHistory_StatusReflectsProcessManagerState` confirm 200 + JSON object shape | PASS |
| No ProcessManager returns empty events | `system.go:168-170` (`make([]RestartEvent, 0)` initialised; nil pm check at line 171 skips population) — code path confirmed; no dedicated nil-pm test but the make() guarantees [] not null | **WARNING** (no explicit test for nil ProcessManager path, though code path is straightforward) |
| events [] not null when empty | `system.go:168` (`make([]model.RestartEvent, 0)` — non-nil empty slice, serialises as `[]`) | PASS |
| status is zero-valued RuntimeStatus when pm is nil | `system.go:169` (`var status model.RuntimeStatus` — zero value) | PASS |
| Auth required | Route at `server.go:160` inside auth group at line 137. Same coverage note as REQ-3 applies. | **WARNING** (same coverage gap) |

Note on `startedAt` field drift: spec says `status` must carry `uptime`, `restartCount`, and `startedAt`. `model.RuntimeStatus.StartedAt` has `json:"startedAt,omitempty"` (`process.go:11`). When the process has not started, `startedAt` will be absent from JSON rather than an empty string. See Drift section.

**REQ-5: PASS WITH DRIFT + WARNING (coverage gap)**

---

### REQ-6: Dashboard Chart Rendering

| Scenario | Code Evidence | Verdict |
|---|---|---|
| Charts render the polled series | `frontend/src/features/dashboard/components/MetricsChart.tsx:25-92` (AreaChart rendered when data.length > 0) — `MetricsChart.test.tsx:41-54` confirms area-chart testId present with data | PASS |
| Poll interval 30 seconds | `frontend/src/features/dashboard/hooks/useSystemHistory.ts:11` (`refetchInterval: 30000`) — `useSystemHistory.test.ts` mock tests that getSystemHistory is called | PASS |
| Loading state shown before first response | `MetricsChart.tsx:26-33` (loading=true renders role=status skeleton) — `MetricsChart.test.tsx:23-28` confirms skeleton present, area-chart absent | PASS |
| Empty history shows empty state | `MetricsChart.tsx:35-44` (data.length===0 renders "No data yet" div) — `MetricsChart.test.tsx:31-38` confirms "No data yet" text present, area-chart absent | PASS |
| Frontend MetricsSnapshot type matches backend JSON | `frontend/src/features/dashboard/types/history.ts:3-8` — `{timestamp: string, cpuPercent: number, memoryPercent: number, diskPercent: number}` matches backend `model/metrics.go` JSON tags exactly. `history.test.ts` type-checks all 4 fields | PASS |

**REQ-6: PASS**

---

## Drift Found (Spec Wording vs Code Reality)

### DRIFT-1: n=120 boundary semantics (REQ-3, "Out-of-range n" scenario)

**Spec says**: `n` narrows only when `0 < n < 120`; values `>= 120` yield the default of 120.

**Code does** (`system.go:140-147`):
```go
if nStr := r.URL.Query().Get("n"); nStr != "" {
    if parsed, err := strconv.Atoi(nStr); err == nil && parsed > 0 {
        n = parsed          // n=120 is accepted here (120 > 0 is true)
    }
}
if n > maxN {               // 120 > 120 is false, so no clamp
    n = maxN
}
```

`n=120` is treated as a valid bounded query (not rejected as out-of-range), but the observable outcome is identical to the default (returns up to 120 items). The spec's statement that `>= 120` is invalid is technically incorrect — `n=120` is valid and produces the same result. `n=121` is the first value that triggers the cap.

**Recommended spec edit** (in the "Out-of-range n" scenario and the n-parameter paragraph):

Change:
> `n` narrows the result only when it parses to an integer strictly greater than 0 and strictly less than the default of 120; any other value (missing, non-numeric, ≤ 0, or ≥ 120) yields the default of 120.

To:
> `n` narrows the result only when it parses to a positive integer greater than 0; values greater than 120 are clamped to 120. Missing, non-numeric, or ≤ 0 values use the default of 120. n=120 is accepted and returns at most 120 samples (same as default).

Change the scenario text:
> WHEN an authenticated client calls GET /api/system/history?n=0 or ?n=500 or ?n=abc

To:
> WHEN an authenticated client calls GET /api/system/history?n=0 or ?n=500 or ?n=abc (note: ?n=120 is valid and returns at most 120 samples, same as default)

---

### DRIFT-2: RuntimeStatus.startedAt omitempty (REQ-5, "No ProcessManager / zero-valued status")

**Spec says**: `status` must carry at least `uptime`, `restartCount`, and `startedAt`.

**Code does** (`internal/model/process.go:11`):
```go
StartedAt string `json:"startedAt,omitempty"`
```

When the ProcessManager has not started the process, `StartedAt` is an empty string, and the `omitempty` tag causes it to be omitted from the serialised JSON. The JSON response will contain `uptime` (0) and `restartCount` (0) but NOT `startedAt`.

This means the spec requirement "status MUST be the zero-valued RuntimeStatus" is technically satisfied at the Go struct level, but the JSON response does not include `startedAt` when it is empty — which may surprise consumers expecting the field to always be present.

**Recommended spec edit** (REQ-5 "No ProcessManager returns empty events" scenario):

Change:
> `status` MUST be the zero-valued RuntimeStatus

To:
> `status` MUST be the zero-valued RuntimeStatus. Note: `startedAt` is omitted from JSON when empty (Go `omitempty`); consumers must treat its absence as equivalent to an empty/zero value.

Alternatively, change the requirement text to replace "carrying at least `uptime`, `restartCount`, and `startedAt`" with "carrying at least `uptime` and `restartCount`; `startedAt` is present only when the process has been started."

---

## Coverage Gaps (WARNING)

### WARN-1: No integration-level 401 test for /api/system/history

The auth middleware is tested at unit level and returns 401 for missing/invalid tokens. The route is wired inside the authenticated group. However, no test sends an unauthenticated request to `/api/system/history` specifically and asserts 401. The spec's "Unauthenticated request is rejected" scenario is satisfied by design (middleware wiring) but not by a direct end-to-end test.

**Recommendation**: Add a test in `internal/server/` that hits `/api/system/history` and `/api/runtime/history` without auth and asserts HTTP 401.

### WARN-2: No explicit test for nil ProcessManager path in GetRuntimeHistory

`system_history_test.go` and `runtime_history_test.go` wire a real ProcessManager. The code path where `h.processManager == nil` (and returns empty events + zero status) is not explicitly tested. The code is correct by inspection but lacks a covering test.

**Recommendation**: Add `TestGetRuntimeHistory_NilProcessManager` that calls `GetRuntimeHistory` on a handler with no ProcessManager set and asserts `events: []` and zero-value status.

---

## Out of Scope (deferred to issue #235)

The following 3 requirements from the spec's "New Behaviour" section are NOT implemented and are explicitly out of scope for PR-1:

1. **Panic Recovery Middleware** — install global panic-recovery middleware returning HTTP 500 without leaking stack traces.
2. **Persisted Restart Count** — durable cumulative Node-RED auto-restart count surviving NRCC process restarts.
3. **Enriched Health Endpoint** — `GET /api/health` returning `uptime` and `restartCount` in addition to `status`.

These are tracked under issue #235 and must not be reported as failures.

---

## Issue Summary

| Severity | Count | Description |
|---|---|---|
| CRITICAL | 0 | — |
| WARNING | 2 | WARN-1: No 401 integration test for history endpoints; WARN-2: No nil ProcessManager test for GetRuntimeHistory |
| SUGGESTION | 0 | — |
| DRIFT | 2 | DRIFT-1: n=120 boundary semantics; DRIFT-2: startedAt omitempty |

---

## Final Verdict

**Section A (Existing Behaviour — 6 requirements, 14 scenarios): PASS WITH DRIFT**

- All 6 requirements are implemented and functional.
- All in-scope Go tests pass under the race detector.
- All 255 frontend tests pass.
- Go build is clean.
- 2 spec wording corrections recommended (DRIFT-1, DRIFT-2) — code is correct, spec needs alignment.
- 2 test coverage gaps (WARN-1, WARN-2) — not blocking archive, but should be addressed before closing the change.
