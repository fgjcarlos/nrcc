# Proposal: Dashboard Runtime Metrics History + Resilience Hardening

## Intent

Two things already shipped to `main` (merged via #214) with **no
OpenSpec contract behind them**: a runtime-metrics history pipeline (ring buffer +
background sampler + two history endpoints) and the recharts dashboard cards that
consume it. This change writes the spec **retroactively** to lock in the behaviour
that already exists, then closes the resilience and hygiene gaps that surfaced
during analysis. Nothing here invents new product surface — it documents what was
built and hardens the edges that were left undefined.

## Scope

### In Scope

**A. Document existing behaviour (retroactive spec, code already present):**
- `MetricsBuffer` — 120-entry thread-safe ring buffer, FIFO eviction
- `MetricsSampler` — 30s background goroutine, Linux sampling + non-Linux zero stub, lifecycle bound to server context
- `GET /api/system/history` — auth-guarded, `?n=` bounded sample count, JSON array of `MetricsSnapshot`
- `GET /api/runtime/history` — auth-guarded restart-event history
- `RestartTracker` — 50-entry in-memory restart event ring
- Frontend: `MetricsChart` area chart + CPU/Memory/Disk cards + `useSystemHistory` 30s poll, with loading/empty states

**B. Resilience corrections (new behaviour):**
- Panic-recovery middleware: recover from any handler panic, log it, return HTTP 500, keep the server alive — wired globally before the route tree
- Restart count persistence: survive process restarts (currently resets to 0 every start)
- `/api/health` enrichment: surface `uptime` and `restartCount` alongside `status`

**C. Repo hygiene:**
- Remove `frontend/package-lock.json` (violates pnpm-only policy)
- Remove committed binaries `nrcc`, `nrcc-final`, `nrcc-test` from the repo root
- Add `.gitignore` rules to prevent both from returning

### Out of Scope

- Per-resource history filtering (`?resource=cpu`) — frontend does not need it today; defer
- Persisting the full metrics history buffer to disk (in-memory is acceptable for a dashboard sparkline)
- Alerting / threshold notifications on metrics
- Windows/macOS real sampling (the zero stub stays; only documented, not implemented)
- Migrating restart history (the event ring) to disk — only the **count** persists

## Capabilities

### New Capabilities
- `dashboard-runtime-metrics`: the sampler → buffer → history-endpoint → chart pipeline and the runtime restart history, specified end to end
- `runtime-resilience`: panic recovery, persisted restart count, enriched health probe

### Modified Capabilities
- None (existing `metrics-endpoint` Prometheus capability is untouched)

## Approach

1. **Retroactive spec first.** Write `specs/dashboard-runtime-metrics/spec.md` describing the buffer, sampler, both history endpoints, and the chart contract as they behave today. `sdd-verify` then proves the existing code already satisfies it — zero code change for section A.
2. **Panic recovery** as standard chi middleware (`middleware.Recoverer` pattern): `r.Use` it as the **first** global middleware so it wraps everything, including the existing SecurityHeaders/CORS/Logger chain. Log via the existing logger, respond `model.RespondJSON(w, 500, ...)`.
3. **Restart count persistence**: persist a small JSON counter file under `dataDir` (same pattern as audit/rate-limiter state). Load on `ProcessManager` init, increment + write on each auto-restart. The event ring stays in-memory; only the cumulative count is durable.
4. **Health enrichment**: `/api/health` reads uptime from process start time and restart count from the persisted counter, with nil-guards so it never panics before `SetProcessManager`.
5. **Hygiene** as an isolated, no-logic PR: `git rm --cached` the binaries, delete the lockfile, extend `.gitignore`.

## API Contract

### GET /api/health (modified — additive)

**Response (200 OK):**
```json
{
  "status": "ok",
  "uptime": 3600,
  "restartCount": 2
}
```
- `uptime` — integer seconds since process start
- `restartCount` — cumulative auto-restarts, persisted across process restarts
- Backward compatible: `status` field unchanged; new fields are additive

### GET /api/system/history (documented — no change)
- Auth required; `?n=<1..120>` optional (default 120); returns `MetricsSnapshot[]`

### GET /api/runtime/history (documented — no change)
- Auth required; returns `RestartEvent[]` (oldest first)

## Affected Areas

| Path | Impact | Description |
|------|--------|-------------|
| `internal/service/metrics_buffer.go` | Documented | Already implemented; covered by retroactive spec |
| `internal/service/metrics_sampler*.go` | Documented | Already implemented; covered by retroactive spec |
| `internal/service/restart_tracker.go` | Documented | Already implemented; covered by retroactive spec |
| `internal/handler/system.go` | Documented | history handlers already implemented |
| `frontend/src/features/dashboard/**` | Documented | charts/hooks already implemented |
| `internal/middleware/recover.go` | New | Panic-recovery middleware |
| `internal/server/server.go` | Modified | Wire recoverer first; enrich `/api/health` |
| `internal/service/process.go` | Modified | Persist + load restart count |
| `internal/service/restart_count_store.go` | New | Small JSON counter persistence under dataDir |
| `.gitignore` | Modified | Ignore binaries + npm lockfile |
| `frontend/package-lock.json` | Removed | pnpm-only policy |
| `nrcc`, `nrcc-final`, `nrcc-test` | Removed | Build artifacts must not live in git |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Retroactive spec drifts from actual code behaviour | Medium | `sdd-verify` runs against real code; write scenarios from observed behaviour, not assumptions |
| Recoverer placed after other middleware → panics in them escape | Medium | Register recoverer as the FIRST `r.Use`; integration test asserts a panicking handler yields 500 not a dropped connection |
| Restart-count file corrupt/unreadable | Low | Treat read error as count=0 and log; never block startup |
| Removing tracked binaries breaks someone's local path | Low | `git rm --cached` keeps the local file; only untracks |
| Hygiene PR mixed with logic → noisy review | Low | Ship hygiene as its own isolated PR |

## Rollback Plan

1. **Resilience revert**: remove `recover.go` + its `r.Use`; revert health handler to `{"status":"ok"}`; delete `restart_count_store.go` and its calls → restart count reverts to in-memory.
2. **Spec revert**: delete `specs/dashboard-runtime-metrics/` — no runtime impact.
3. **Hygiene revert**: `git revert` the hygiene commit; binaries/lockfile return to tracking.

## Dependencies

- `prometheus-metrics` is already merged to `main` (code via #209/#210) and its OpenSpec artifact is being archived in PR #237; the feared `server.go` wiring conflict was **moot** because `/metrics` is already in `main`. PR-2 (resilience) edits `server.go` on top of it cleanly.
- chi router already in place; `model.RespondJSON` helper already exists.
- pnpm-only policy and "binaries don't belong in git" are pre-existing project constraints.

## Success Criteria

- [ ] `specs/dashboard-runtime-metrics/spec.md` exists and `sdd-verify` confirms current code satisfies section A
- [ ] A handler that panics returns HTTP 500 and the server stays up (integration test)
- [ ] Restart count is non-zero after a real restart (persistence test)
- [ ] `GET /api/health` returns `uptime` and `restartCount`; old `status` field intact
- [ ] `frontend/package-lock.json` gone; `pnpm-lock.yaml` is the only lockfile
- [ ] `nrcc`/`nrcc-final`/`nrcc-test` untracked and `.gitignore`d
- [ ] `go build ./...` and `go test ./...` green; frontend tests green

## Notes

- **Delivery**: chained/stacked PRs to keep each ≤400 lines — (1) retroactive spec + verify, (2) resilience, (3) hygiene.
- **Engram unavailable this session** → artifacts are OpenSpec file-based, consistent with existing `prometheus-metrics` and `issue-112-users-admin` changes.
