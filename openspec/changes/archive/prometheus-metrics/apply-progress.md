# Apply Progress — prometheus-metrics (PR 2)

**Change**: prometheus-metrics
**PR Slice**: PR 2 — Handler Instrumentation + Server Wiring + Integration Test
**Mode**: Strict TDD
**Status**: All tasks complete

## Completed Tasks

- [x] 4.1 `internal/handler/auth.go` — narrow interface `loginMetricsRecorder` + `loginMetrics` field + `SetLoginMetrics` setter + `RecordLoginAttempt` calls with nil guards already present in PR 1
- [x] 4.2 `internal/handler/backup.go` — `backupMetricsRecorder` interface + `backupMetrics` field + `SetBackupMetrics` setter + `RecordBackupCreated(string(backup.Type))` on success + `RecordRestoreAttempt(bool)` on success/failure
- [x] 4.3 `internal/handler/libraries.go` — `libraryMetricsRecorder` interface + `libraryMetrics` field + `SetLibraryMetrics` setter + `RecordLibraryOperation("install", bool)` + `RecordLibraryOperation("uninstall", bool)`
- [x] 4.4 `internal/handler/updates.go` — `updateMetricsRecorder` interface + `updateMetrics` field + `SetUpdateMetrics` setter + `RecordUpdateAttempt(bool)` inside async goroutine after flow settles
- [x] 5.1 `internal/server/server.go` — `metricsCollector *metrics.MetricsCollector` field + `metrics.NewCollector()` in `NewServerWithConfig()` + `GET /metrics` public route + `SetLoginMetrics/SetBackupMetrics/SetLibraryMetrics/SetUpdateMetrics` wiring
- [x] 5.2 `internal/server/server.go` — `SetProcessManager()` calls `s.metricsCollector.SetProcessManager(pm)` with nil guard
- [x] 6.1 `internal/server/metrics_test.go` — integration tests: 200 response, Prometheus format content, nrcc_nodered_running metric, nrcc_login_attempts_total after activity, public endpoint (no auth)

## Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/handler/backup.go` | Modified | Added backupMetricsRecorder interface, field, setter; RecordBackupCreated + RecordRestoreAttempt calls |
| `internal/handler/libraries.go` | Modified | Added libraryMetricsRecorder interface, field, setter; RecordLibraryOperation for install/uninstall |
| `internal/handler/updates.go` | Modified | Added updateMetricsRecorder interface, field, setter; RecordUpdateAttempt in async goroutine |
| `internal/server/server.go` | Modified | Added metricsCollector field, creation, /metrics route, handler wiring, SetProcessManager wiring |
| `internal/handler/backup_metrics_test.go` | Fixed | Fixed newNoopAudit to return real *audit.Service (was returning wrong stub type) |
| `internal/handler/libraries_metrics_test.go` | Created | TDD tests for library metrics (install/uninstall operations) |
| `internal/handler/updates_metrics_test.go` | Created | TDD tests for update metrics (PostApply async flow) |
| `internal/server/metrics_test.go` | Created | Integration tests for /metrics endpoint |

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 4.1 | `handler/auth_metrics_test.go` | Unit | N/A (pre-existing) | Pre-existing | Pre-existing | 3 cases (fail, success, nil guard) | N/A |
| 4.2 | `handler/backup_metrics_test.go` | Unit | Pre-existing (5/5) | Pre-existing (build fail) | GREEN after impl | 4 cases (manual, auto, restore success, restore fail, nil guard) | Refactored to use backup.Type |
| 4.3 | `handler/libraries_metrics_test.go` | Unit | N/A (new file) | Written (build fail) | GREEN after impl | 4 cases (install success, install op label, uninstall op, nil guard) | None needed |
| 4.4 | `handler/updates_metrics_test.go` | Unit | N/A (new file) | Written (build fail) | GREEN after impl | 2 cases (records attempt, correct success flag) | None needed |
| 5.1+5.2 | `server/metrics_test.go` | Integration | N/A (new file) | Written (404 failure) | GREEN after server impl | 5 cases (200, format, nrcc runtime metric, login metric after activity, public) | None needed |

## Test Summary

- **Total tests written**: 17 new tests (5 library, 3 update, 5 server, + fixed 4 existing backup tests)
- **Total tests passing**: All (go test ./... passes)
- **Layers used**: Unit (handler package), Integration (server package)
- **Approval tests** (refactoring): None — no refactoring tasks
- **Pure functions created**: 0 — metrics recording is a side-effect operation

## Deviations from Design

1. **backup.go**: Used `string(backup.Type)` instead of `req.Type` for the metric label — this is better because it uses the normalized type from the service (handles empty type → "manual" automatically). Minor deviation that improves correctness.
2. **updates.go**: Metric is recorded inside the async goroutine after `ApplyUpdateWithBackup` returns, using the final flow state to determine success. This correctly reflects the actual outcome rather than a premature recording.
3. **backup_metrics_test.go**: Fixed `newNoopAudit` to return a real `*audit.Service` — this was a bug in the pre-existing test file (returned `*auditServiceForTest` which doesn't satisfy `*audit.Service`).

## Issues Found

None beyond the audit stub type mismatch in backup_metrics_test.go (fixed).

## Workload / PR Boundary

- Mode: chained PR slice
- Current work unit: PR 2 — Handler Instrumentation + Server Wiring + Integration Test
- Boundary: Starts from Phase 4 (handler narrow interfaces), ends at Phase 6 (integration test)
- Estimated review budget impact: ~115 lines changed in production code + ~200 lines of tests
