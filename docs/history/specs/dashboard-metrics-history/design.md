# Design: Dashboard Runtime Metrics History + Resilience Hardening

> Scope note: Section A of this change (retroactive spec) requires **no design** —
> the code already exists and `sdd-verify` proves it against the spec. This
> document covers only the **new behaviour** (Section B): panic-recovery
> middleware, persisted restart count, and the enriched health endpoint. Hygiene
> (Section C) is mechanical and needs no design.

## Verification basis

All decisions below were checked against the real code this turn:

| Fact | Source | Confidence |
|------|--------|-----------|
| `/api/health` is an **inline closure** returning `{"status":"ok"}` (no `HealthHandler` struct) | server.go:105-109 | ✅ verified |
| Global middleware = SecurityHeaders → CORS → Logger (plain funcs via `r.Use`) | server.go:48-50 | ✅ verified |
| `restartCount` is the **backoff counter**, not a lifetime metric | process.go:400 (give-up), :412 (`1<<restartCount` backoff), :428 (`++`), :222 (reset on user start) | ✅ verified |
| `restartCount` reset is gated by a `resetCounter` flag (user-start resets, auto-restart doesn't) | process.go:221-223 | ✅ verified |
| `SystemHandler` already has `SetProcessManager`, wired at server.go:318 | system.go:31-33, server.go:318 | ✅ verified |
| `model.RuntimeStatus.RestartCount` currently maps to the backoff counter | process.go:317 (inside `Status()`) | ✅ verified |
| Runtime history returns `{events, status}` object | system.go:167-184 | ✅ verified |
| chi router, `model.RespondJSON` helper exists and is the response convention | server.go:106 | ✅ verified |
| Persistence precedent: ratelimit/env/flow_version use JSON files under dataDir | grep `os.WriteFile` internal/ | ✅ verified |

---

## Decision 1 — Panic-recovery middleware

### Context
chi gives us `middleware.Recoverer`, but it writes a bare `500` with chi's own
formatting and prints the stack to stderr. We already standardise responses
through `model.RespondJSON` and log through the project logger. A bespoke
middleware keeps the contract consistent and lets us control exactly what
crosses the wire.

### Decision
Add `internal/middleware/recover.go` exposing `Recoverer(logger) func(http.Handler) http.Handler`.
On a recovered panic it:
1. Logs the panic value **and** the stack (`debug.Stack()`) server-side only.
2. Responds `model.RespondJSON(w, http.StatusInternalServerError, {"error":"internal server error"})`.
3. Guards against a double-write if the handler already started the response
   (check a flag / `http.ErrAbortHandler` is re-panicked, not swallowed).

### Ordering — the critical part
Register it as the **FIRST** `r.Use(...)`, before SecurityHeaders/CORS/Logger.
chi runs middleware in registration order (outermost first), so the recoverer
must wrap the entire stack to catch a panic inside any other middleware. If it
went after Logger, a panic in Logger would escape and drop the connection.

```go
// server.go:48-50 today — Recoverer prepended:
r.Use(middleware.Recoverer)        // ← NEW, must be first (plain func, matches style)
r.Use(middleware.SecurityHeaders)
r.Use(middleware.CORS(corsCfg))
r.Use(middleware.Logger)
```

Note: `SecurityHeaders` and `Logger` are registered as **plain functions**
(`middleware.Logger`, not `middleware.Logger(...)`), so `Recoverer` follows the
same shape — `func Recoverer(next http.Handler) http.Handler`. Logging goes
through the project's `ui` package (already used by process.go), not an injected
logger, to stay consistent with `middleware.Logger`.

### Why not chi's built-in `middleware.Recoverer`
- Leaks stack to the client in some configurations / writes a non-JSON body.
- Inconsistent with our `RespondJSON` envelope.
- We want a single logging path through the project's `ui` logger, not stderr.

### Rejected alternative
Per-route `defer/recover` wrappers — rejected: easy to forget on a new route,
doesn't cover middleware panics, multiplies boilerplate.

---

## Decision 2 — Persisted cumulative restart count (NEW counter, not the backoff one)

### Context — the critical distinction
The existing `pm.restartCount` is **NOT a lifetime metric**. The code proves it:
- `process.go:412` → `delay := pm.restartDelay * (1 << uint(restartCount))` — it
  drives **exponential backoff**.
- `process.go:400` → `if restartCount >= pm.maxRestarts` — it drives the
  **give-up** decision.
- `process.go:221-222` → reset to 0 when `resetCounter` is true (user start).
- `process.go:428` → `pm.restartCount++` on each auto-restart attempt.

So `restartCount` is the **consecutive-failure counter** for the supervision
loop. **Persisting it would be a bug**: after an NRCC restart, the first crash
would compute `restartDelay * 2^N` (a huge delay) and could trip the give-up
threshold immediately. We must NOT touch this field's lifecycle.

The value the user wants in `/api/health` is a **different concept**: the total
number of Node-RED restarts over the install's lifetime. That needs its own
field.

### Decision
Introduce a **separate** durable counter, leaving the backoff counter untouched.

1. New field on `ProcessManager`: `cumulativeRestarts int` (guarded by `mu`).
2. New file `internal/service/restart_count_store.go`:

```go
type restartCountStore struct {
    path string
    mu   sync.Mutex
}
func newRestartCountStore(dataDir string) *restartCountStore
func (s *restartCountStore) Load() int       // 0 on any read/parse error (logged)
func (s *restartCountStore) Save(n int) error
```

- File: `<dataDir>/restart_count.json`, payload `{"cumulativeRestarts": N}`.
- JSON object (not a bare int) so the schema can grow (`lastResetAt`, etc.)
  without a migration.
- Atomic write: write `restart_count.json.tmp` then `os.Rename` — no torn file
  if the process dies mid-write. Precedent: ratelimit/flow_version already
  persist JSON under dataDir.
- `Load()` failure (missing/corrupt/permission) → return 0 + `ui.Warn`, never
  block startup (spec: "degrades gracefully").

### Wiring into ProcessManager (exact sites)
- `NewProcessManager`: build the store, `cumulativeRestarts = store.Load()`.
- At **process.go:428** (the existing `pm.restartCount++` auto-restart site):
  also `pm.cumulativeRestarts++` and `store.Save(pm.cumulativeRestarts)` —
  best-effort; a Save error is logged, not fatal. This is the ONLY increment
  site, so the lifetime count tracks real auto-restarts exactly.
- The `resetCounter` reset at **:222 is left exactly as-is** — it only zeroes
  the backoff counter, never the cumulative one. No risk of clobbering durable
  state.
- Expose `func (pm *ProcessManager) CumulativeRestarts() int` (locked read) for
  the health handler.

### Counter ↔ event-ring relationship
The 50-entry event ring (`restart_tracker.go`) stays in-memory and unchanged —
it's bounded history for the dashboard, not a lifetime counter. The new scalar
is the durable lifetime total. They answer different questions; both stay.

### `RuntimeStatus.RestartCount` — left alone (scoped out)
`Status()` (process.go:317) currently sets `RestartCount: pm.restartCount` (the
backoff counter). Changing that would alter the `/api/runtime/history` payload
contract this same change is spec'ing retroactively — out of scope here. Flagged
as a follow-up: runtime status arguably should expose the cumulative count too,
but that's a separate, contract-affecting decision.

### Concurrency
The auto-restart increment runs from the supervisor goroutine under `pm.mu`
(already held at the increment site). `CumulativeRestarts()` takes `pm.mu` for a
clean read. The store's own `mu` is a leaf lock (file I/O only, never calls back
into ProcessManager) — no lock-ordering hazard.

### Why a file under dataDir (not SQLite / settings.js)
- `dataDir` is already the durable-state location; no new dependency.
- A single scalar doesn't justify a DB.
- Keeping it out of `settings.js` avoids coupling NRCC operational state to
  Node-RED's own config file (which process.go rewrites at :150).

---

## Decision 3 — Enriched health endpoint

### Context — the wiring constraint (corrected)
There is **no `HealthHandler` struct**. `/api/health` is an **inline closure**
registered at `server.go:105-109`, inside `NewServerWithConfig`, BEFORE the
`Server` value exists and BEFORE `SetProcessManager` runs (server.go:302). An
inline closure registered at construction therefore cannot read the
ProcessManager (it's nil at that point and the closure has no handle to mutate
later).

### Decision — move health onto `SystemHandler`
Reuse the existing wiring instead of inventing a new handler:
- `SystemHandler` already exists at construction (server.go:58) and already
  receives the ProcessManager later via `SetProcessManager` (server.go:318,
  system.go:31-33). The nil-guard window is exactly what we need.
- Add `func (h *SystemHandler) GetHealth(w, r)` and register
  `r.Get("/api/health", systemHandler.GetHealth)` at server.go:105 (replacing
  the inline closure). Still public — registered outside the auth group.
- Capture NRCC start time in `NewSystemHandler` (`startedAt: time.Now()`), since
  handler construction ≈ NRCC start. Add a `startedAt time.Time` field.

```go
// system.go
func (h *SystemHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
    uptime := int(time.Since(h.startedAt).Seconds())
    restarts := 0
    if h.processManager != nil {
        restarts = h.processManager.CumulativeRestarts() // ← the DURABLE counter
    }
    model.RespondJSON(w, http.StatusOK, map[string]interface{}{
        "status":       "ok",
        "uptime":       uptime,
        "restartCount": restarts,
    })
}
```

### Which counter
`restartCount` here reads `CumulativeRestarts()` (Decision 2), NOT the backoff
`restartCount`. This is the lifetime total a user expects from a health probe.

### Backward compatibility
`status` and its value are unchanged; `uptime`/`restartCount` are additive keys.
Consumers reading only `status` are unaffected.

### Nil-guard rationale
If `SetProcessManager` hasn't run, `h.processManager == nil` → `restartCount`
reports 0 without panicking. Matches spec scenario "Health before ProcessManager
wiring does not panic".

### `uptime` semantics — explicit choice
We report **NRCC process** uptime (seconds since `SystemHandler` construction),
not Node-RED uptime. Rationale: `/api/health` is the *server's* liveness probe;
Node-RED's own uptime is already in the runtime status object. Documented, not
accidental — the spec scenario phrases it as seconds since process start.

---

## Testing strategy (Strict TDD — tests first)

| Behaviour | Test | Level |
|-----------|------|-------|
| Recoverer: panic → 500, server survives | httptest server + panicking route, then GET healthy route | integration |
| Recoverer: no stack leak | assert body has no `goroutine`/file-path substrings | unit |
| Recoverer ordering | panic inside a later middleware still caught | integration |
| Count store: round-trip | Save(n) → Load() == n in a temp dir | unit |
| Count store: corrupt file → 0 + log | seed garbage, assert Load()==0 | unit |
| Count store: atomic write | no `.tmp` left after Save; Load() valid | unit |
| Count survives restart | Save, construct a new PM on same dataDir, assert count | unit |
| User start doesn't zero cumulative | start → cumulative unchanged | unit |
| Health: status+uptime+restartCount | httptest GET /api/health | integration |
| Health: nil PM → 0/0, no panic | handler with pm=nil | unit |

All new tests RED first, then implement to green. `go test ./... -race` must pass
(the buffer/tracker concurrency scenarios already rely on `-race`).

---

## Sequencing & delivery

1. **PR-1 (spec + verify)**: `specs/dashboard-runtime-metrics/spec.md` + a
   verify pass proving Section A. No production code.
2. **PR-2 (resilience)**: `recover.go`, `restart_count_store.go`, ProcessManager
   wiring, health enrichment + tests. ~250–350 lines.
3. **PR-3 (hygiene)**: remove lockfile + binaries, `.gitignore`. Isolated.

Hard ordering: land **after** `prometheus-metrics` is committed/archived —
both edit `server.go`'s router setup and would otherwise conflict on the
middleware/route block.

## Resolved during design (no longer open)

- ✅ Reset site: process.go:222, gated by `resetCounter`, touches ONLY the
  backoff counter. The new `cumulativeRestarts` is a separate field — the reset
  never reaches it. No "split" of the existing field needed.
- ✅ `dataDir` is a field on `ProcessManager` (process.go:31), already used at
  :150. The store path is `filepath.Join(pm.dataDir, "restart_count.json")`.
- ✅ No `HealthHandler` — health moves onto `SystemHandler` (Decision 3).

## Open questions for verify

1. Does any current consumer parse `/api/health` strictly (reject unknown JSON
   fields)? Expected no — but `sdd-verify` should grep the frontend/health-check
   client for the consumer shape before trusting "additive = safe".
2. Should `RuntimeStatus.RestartCount` (process.go:317) switch from the backoff
   counter to the cumulative one for consistency? Deferred — it changes the
   `/api/runtime/history` contract, so it's a separate decision, not this change.
