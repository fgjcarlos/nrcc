# Issue #1075 — Node-RED runtime logs
Source: https://github.com/fgjcarlos/nrcc/issues/1075
Base: origin/main. Raw runtime logs are restricted to administrators (owner decision).

## Specs

### S1 — Backend
- Tee the managed Node-RED output into a bounded in-memory ring buffer (keeping the current container-log forwarding). Parse timestamp, level and message from Node-RED's console format; unparsed lines keep `level: unknown`.
- Authenticated `GET /api/logs` with filters (levels, text search, time range, limit/cursor) and an SSE `GET /api/logs/stream` for live tail. Decide the viewer-role access explicitly (see #1022).
- Bound memory (entries and bytes). Respect `EDGE_MODE` with a smaller buffer (ADR 0002).

### S2 — Frontend
- Virtualized table: time, level badge, source, message, expandable detail.
- Filters: level multi-select, free text, time range. Quick filters: "errors last 15 min", "warnings", "since last restart", "live tail on/off".
- Charts: events per level over time (stacked) and top error messages. Charts are selectable: clicking or brushing a bar or range filters the table. Pause and resume the live stream.
- Empty, loading and error states. i18n for all copy (en/es).

### S3 — Configured level
- Show the console level in settings.js and whether it is in effect (applied plus restarted) or pending. Never show a fabricated default: if the value is missing or unreadable, say so.
- Change the level through the existing apply pipeline, with a clear "requires restart" hint.

### S4 — Logging settings fixes
- Send explicit `false` for `metrics` and `audit` so they can be disabled. Add a regression test for the round trip.
- Remove the `|| 'info'` fallback.

### S5 — Generated API
The stale `/api/logs`, `/api/logs/stream` and `LogEntry` entries in the generated `frontend/src/shared/api/schema.ts:1561-1610` should be resolved either way (regenerate or reuse).

### S6 — Acceptance
- [ ] Runtime logs are visible in the NRCC UI for authenticated users, with level, text and time filters plus quick filters.
  - [ ] At least one chart (events per level over time) is selectable and filters the table.
  - [ ] Live tail via SSE works and can be paused.
  - [ ] The log buffer is bounded and covered by tests (parse, eviction, filters).
  - [ ] The logging settings show the real configured level or an explicit "not set/unreadable" state, never a fabricated `info`.
  - [ ] `metrics` and `audit` can be turned off and the change persists in settings.js (regression test).

## Tasks

- T1 [done] S1, S5, S6 (backend): capture, retention, authenticated history and live API; writer; commits 445b91e, d55e17a, 09e5fcd, 3802aaa.
- T2 [done] S3, S4, S6 (settings): configured/applied state and round-trip fixes; writer; commits 68b4caa, eb40db4, 7df8d14.
- T3 [done] S2, S6 (viewer): admin UI, selectable charts and pause/resume; writer; commits 69ac019, ae1e02f, 7134ca5.
- T4 [done] S1–S6: independent risk verification and public checks; verifier; verified candidate 7134ca5.

## Log

- L1: Resuelve la issue #1075
- L2: Solo administradores (recomendado)
- L3: Issue is `status:needs-review`; local implementation only, no PR.
- L4: Initial capture/history/SSE and settings changes passed focused tests. Code inspection found sparse filtered history could return an empty page with no continuation, and restart erased all logs. Both must be corrected before T1 is done. T2 still needs runtime-observed applied versus pending status; no fabricated signal. Browser SSE requires bearer authorization.

- L5: Focused Go checks pass for restart retention, parse/eviction/filtering, history/SSE and the exhaustive role check. LogsView chart and pause/resume tests, sidebar/shell tests, typecheck, and i18n check pass. Independent and full-suite checks remain open.
- L6: The initial UI passes its focused checks, but inspection found 'since last restart' used the oldest row instead of the restart boundary, generic filters were limited to the initial sample, and history/stream updates could race. Reopened T3; the UI correction can run independently of T2.
- L7: Applied/pending/unknown and explicit false checks passed. UI correction passed 19 focused tests. Integration inspection showed log timestamps are not a reliable run identifier (Node-RED omits subsecond precision; no launch synthesized a changing timestamp). Fix the launch boundary before checking off T1 and T3.
- L8: Targeted RED reproduced missing current-run logs with coarse timestamps and a false initial start time. Go focused GREEN verified launch tagging and stable unstarted state. API types regenerated. Final UI, Go, browser, and build checks launched (jobs 1–4).
- L9: Independent verifier confirmed admin-only 401/403/200 behavior, restart retention, output parity, Go race check, and schema sync. Reopened T2: missing-level placeholder is selectable but silently ignored, and an implicit audit=false overwrites configured true. Reopened T3: reconnect can miss a restart while disconnected; failed history falsely shows an empty result. Frontend build/typecheck/i18n passed; full unit suite 755 passed, 1 skipped (before these fixes).
- L10: Applied all independent findings: reconnect compares the launch manifest, failed history does not claim no results, the missing-level placeholder is disabled, and the audit switches preserve absent/true values unless explicitly changed. Latest focused frontend: 98 passed, typecheck passed; targeted Chromium logs: 2 passed; Go logs/audit checks passed; schema and i18n checks passed. Previous full browser suite: 59 passed, 3 skipped.
- L11: Original full Go suite failed only at TestSaveConfig_WritesExplicitRuntimeSettingsAndRestarts (expected its Node-RED 5.0.6 stub; discovered local Docker Node-RED 5.0.10). The same test passed when Docker discovery was isolated via a temp socket. Final whole Go suite with that isolation (job-5) and final frontend suite/build (job-6) are running. No issue/PR changes.
- L12: Final frontend verification passed: 94 files, 761 tests passed, 1 skipped; production build exit 0. Waiting for job-5 only.
- L13: Final `go test ./...` passed with Docker discovery isolated (job-5; exit 0). The initial unisolated failure is recorded in L11. Verification includes Chromium against mocked network plus independent authz and streaming against the Go router and a managed fixture. No live deployment was tested. All four tasks are complete locally.
- L14: Maintainer approved the issue and selected the feature-tracker branch chain. Integrate the child PRs back-to-front or otherwise ensure the tracker contains every reviewed slice before requesting review against `main`. The tracker remains draft until integration and CI are complete.
- L15: CI on the published chain exposed defects the full-stack verification could not see: slice 01 failed golangci-lint (QF1011 on the cursor bound), slice 03 referenced `model.RuntimeConsoleLoggingStatus` that only slice 04 defined, and slice 05 carried an unquoted comma in an OpenAPI flow mapping. Each fix now lives in its originating slice. Every slice was checked on its own with build, vet, golangci-lint, Redocly, and service tests before force-pushing with leases. The merge order is back-to-front with `--admin` after green CI, then the tracker into main once its CI is green on the integrated tree.
