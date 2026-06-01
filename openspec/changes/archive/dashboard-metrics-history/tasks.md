# Tasks: Dashboard Runtime Metrics History + Resilience Hardening

> Strict TDD is active. For every behavioural task: write the failing test
> FIRST (RED), then implement to GREEN, then refactor. `go test ./... -race`
> must stay green. Tasks are grouped by the three chained PRs from the design's
> Sequencing section.
>
> Hard ordering: this whole change lands **after** `prometheus-metrics` is
> committed/archived — both edit `server.go`'s router block.

---

## PR-1 — Retroactive spec + verify (no production code)

Goal: lock in the already-shipped Section A behaviour and prove the current code
satisfies it. Zero runtime change.

- [ ] 1.1 Confirm `specs/dashboard-runtime-metrics/spec.md` is complete (9 reqs / 22 scenarios) — done in SPEC phase
- [ ] 1.2 Run `sdd-verify` against the spec; map each "Existing Behaviour" scenario to real code:
  - [ ] Metrics Ring Buffer → `metrics_buffer.go`
  - [ ] Background Sampler → `metrics_sampler*.go` + `server.go:62-63`
  - [ ] System History Endpoint → `system.go:133-157`
  - [ ] Restart Event Tracker → `restart_tracker.go`
  - [ ] Runtime History Endpoint (object `{events,status}`) → `system.go:167-184`
  - [ ] Dashboard Chart Rendering → frontend
- [ ] 1.3 Record any spec↔code drift found by verify; if drift is in the spec, fix the spec (code is the source of truth for Section A)
- [ ] 1.4 PR-1 contains ONLY `openspec/` files — assert `git diff --stat` touches no `.go`/`.ts`/`.tsx`

**Estimated changed lines:** ~0 production, spec already written. PR is docs-only.

---

## PR-2 — Resilience hardening (new behaviour, TDD)

### 2A. Panic-recovery middleware

- [x] 2A.1 (RED) `internal/middleware/recover_test.go`: panicking handler → HTTP 500
- [x] 2A.2 (RED) test: after a panic, a subsequent request to a healthy route succeeds (server alive)
- [x] 2A.3 (RED) test: response body contains no stack trace / no file paths / no `goroutine`
- [x] 2A.4 (RED) test: panic raised inside a *later* middleware is still caught (ordering)
- [x] 2A.5 (GREEN) `internal/middleware/recover.go` — `func Recoverer(next http.Handler) http.Handler`; recover, `ui` log (value + `debug.Stack()` server-side only), `model.RespondJSON(w, 500, {"error":"internal server error"})`
- [x] 2A.6 Re-panic `http.ErrAbortHandler` instead of swallowing it (chi convention)
- [x] 2A.7 Wire as the **FIRST** `r.Use` in `server.go` (before SecurityHeaders at :48)
- [x] 2A.8 (GREEN) all 2A tests pass; `-race` clean

### 2B. Persisted cumulative restart count

- [x] 2B.1 (RED) `internal/service/restart_count_store_test.go`: `Save(n)` then `Load()` round-trips in a temp dir
- [x] 2B.2 (RED) test: missing file → `Load()` returns 0
- [x] 2B.3 (RED) test: corrupt/garbage file → `Load()` returns 0 (and logs), no error to caller
- [x] 2B.4 (RED) test: after `Save`, no `restart_count.json.tmp` is left behind (atomic rename)
- [x] 2B.5 (GREEN) `internal/service/restart_count_store.go`: `restartCountStore{path, mu}`, `newRestartCountStore(dataDir)`, `Load() int`, `Save(n int) error`; file `<dataDir>/restart_count.json`, payload `{"cumulativeRestarts": N}`, write `.tmp` + `os.Rename`
- [x] 2B.6 (RED) `process_test.go`: a new `ProcessManager` over a dataDir with a saved count loads it into `CumulativeRestarts()`
- [x] 2B.7 (RED) test: a user-initiated start (`resetCounter=true`) does NOT change `CumulativeRestarts()` (only the backoff counter resets)
- [x] 2B.8 (RED) test: an auto-restart increments `CumulativeRestarts()` and persists it (covered by store round-trip + PM wiring tests)
- [x] 2B.9 (GREEN) add `cumulativeRestarts int` field to `ProcessManager` (guarded by `mu`); build store in `NewProcessManager`; `cumulativeRestarts = store.Load()`
- [x] 2B.10 (GREEN) at `process.go` auto-restart site (the existing `pm.restartCount++`): also `pm.cumulativeRestarts++` + `store.Save(...)` best-effort (log on error, non-fatal)
- [x] 2B.11 (GREEN) leave the `resetCounter` reset UNCHANGED (must not touch cumulative)
- [x] 2B.12 (GREEN) add `func (pm *ProcessManager) CumulativeRestarts() int` (locked read)
- [x] 2B.13 (GREEN) all 2B tests pass; `-race` clean

### 2C. Enriched health endpoint

- [x] 2C.1 (RED) `get_health_test.go`: `GET /api/health` → 200 with `status:"ok"`, integer `uptime`, integer `restartCount`
- [x] 2C.2 (RED) test: with `processManager == nil`, health returns `uptime` 0 / `restartCount` 0, no panic
- [x] 2C.3 (RED) test: `restartCount` reflects `CumulativeRestarts()`, not the backoff counter
- [x] 2C.4 (GREEN) add `startedAt time.Time` to `SystemHandler`; set `time.Now()` in `NewSystemHandler`
- [x] 2C.5 (GREEN) add `func (h *SystemHandler) GetHealth(w, r)` — uptime from `startedAt`, restarts from `CumulativeRestarts()` behind nil-guard, additive JSON keys
- [x] 2C.6 (GREEN) replace the inline closure at `server.go` with `r.Get("/api/health", systemHandler.GetHealth)` (still public, outside auth group)
- [x] 2C.7 (GREEN) all 2C tests pass; `-race` clean

### 2D. PR-2 close-out

- [x] 2D.1 `go build ./...` green
- [x] 2D.2 `go test ./... -race` green (all new tests; pre-existing slow integration tests time out independently of our changes)
- [x] 2D.3 Verify `/api/health` consumer (frontend/health-check client) does not reject unknown fields — confirmed safe (see design open question #1 finding)
- [x] 2D.4 PR description notes the backoff-vs-cumulative distinction (see commit messages)

**Estimated changed lines:** ~280–360 (2 new files + tests + 3 wiring edits).

---

## PR-3 — Repo hygiene (isolated, no logic)

- [ ] 3.1 `git rm --cached nrcc nrcc-final nrcc-test` (untrack binaries; keep local files)
- [ ] 3.2 `git rm frontend/package-lock.json` (pnpm-only policy)
- [ ] 3.3 Add `.gitignore` rules: `/nrcc`, `/nrcc-final`, `/nrcc-test` (and any build-output pattern), `frontend/package-lock.json`, `**/package-lock.json`
- [ ] 3.4 Confirm `pnpm-lock.yaml` remains the only lockfile (`git ls-files | grep lock`)
- [ ] 3.5 `git diff --stat` shows only deletions + `.gitignore` — no source changes

**Estimated changed lines:** ~10 (`.gitignore`) + deletions.

---

## Review Workload Forecast

| PR | Scope | Est. changed lines | 400-line budget risk |
|----|-------|--------------------|----------------------|
| PR-1 | spec + verify (docs only) | ~0 production | None |
| PR-2 | resilience (recover + count + health) | ~280–360 | **Medium** (near budget; tests are bulk) |
| PR-3 | hygiene | ~10 + deletions | None |

- **Chained PRs recommended:** **Yes** — three slices, each independently
  reviewable; PR-2 depends on PR-1's spec landing, PR-3 is orthogonal.
- **400-line budget risk:** **Medium** for PR-2 only. If it exceeds 400 with
  tests, split 2A (middleware) from 2B+2C (process/health) into a stacked pair.
- **Decision needed before apply:** **Yes** — confirm delivery strategy
  (chained vs `size:exception`) before `sdd-apply`, per the orchestrator's
  Review Workload Guard.

---

## Definition of Done (whole change)

- [ ] `sdd-verify` confirms current code satisfies Section A of the spec
- [ ] Panicking handler → HTTP 500, server stays up (integration test green)
- [ ] Cumulative restart count survives an NRCC restart (persistence test green)
- [ ] Backoff counter behaviour (`process.go:400/412`) is unchanged — no regression
- [ ] `GET /api/health` returns `status` + `uptime` + `restartCount`; `status` intact
- [ ] `frontend/package-lock.json` gone; `pnpm-lock.yaml` is the only lockfile
- [ ] `nrcc`/`nrcc-final`/`nrcc-test` untracked and `.gitignore`d
- [ ] `go build ./...` and `go test ./... -race` green; frontend tests green
