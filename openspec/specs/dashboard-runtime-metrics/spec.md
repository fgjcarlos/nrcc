# Dashboard Runtime Metrics — Specification

## Purpose

Defines the runtime-metrics history pipeline that feeds the dashboard charts:
a thread-safe ring buffer, a background sampler, two authenticated history
endpoints, and the restart-event tracker. Sections "Existing Behaviour" are
written **retroactively** to lock in code that already ships in `main` (merged
via #214); sections "New Behaviour" specify the resilience hardening introduced
by this change.

---

## Existing Behaviour (retroactive — code already present)

### Requirement: Metrics Ring Buffer

The system MUST maintain a fixed-capacity, thread-safe ring buffer of
`MetricsSnapshot` samples. When the buffer is full, the oldest sample MUST be
evicted (FIFO). Concurrent reads and writes MUST be safe.

A `MetricsSnapshot` MUST carry: `timestamp` (string), `cpuPercent` (float),
`memoryPercent` (float), `diskPercent` (float).

#### Scenario: Buffer evicts oldest when full

- GIVEN a buffer with capacity 120 holding 120 samples
- WHEN a 121st sample is pushed
- THEN the buffer still holds 120 samples
- AND the oldest sample has been discarded
- AND the newest sample is present

#### Scenario: Reads are returned oldest-first

- GIVEN a buffer holding N samples
- WHEN the most recent M samples are requested
- THEN they are returned in chronological order (oldest first)
- AND no more than M samples are returned

#### Scenario: Concurrent access does not corrupt state

- GIVEN concurrent goroutines pushing and reading samples
- WHEN the operations run under the race detector
- THEN no data race is reported and no panic occurs

---

### Requirement: Background Metrics Sampler

The system MUST run a background goroutine that samples host CPU, memory, and
disk usage at a fixed interval (30 seconds) and pushes each sample into the ring
buffer. The sampler lifecycle MUST be bound to the server context: it starts when
the server starts and stops when the server context is cancelled.

On non-Linux platforms the sampler MUST still run but MAY emit zero-valued
samples (host sampling is Linux-only).

#### Scenario: Sampler populates the buffer over time

- GIVEN the server has started on Linux
- WHEN the sampler interval elapses
- THEN a new `MetricsSnapshot` is appended to the buffer

#### Scenario: Sampler stops on context cancellation

- GIVEN a running sampler bound to the server context
- WHEN the server context is cancelled
- THEN the sampler goroutine returns and stops sampling

#### Scenario: Non-Linux sampler does not panic

- GIVEN the server runs on a non-Linux platform
- WHEN the sampler interval elapses
- THEN a zero-valued snapshot is produced without error

---

### Requirement: System History Endpoint

The system MUST expose `GET /api/system/history` returning a JSON array of
`MetricsSnapshot`. The endpoint MUST require authentication.

The endpoint MUST accept an optional `n` query parameter bounding the number of
samples returned. `n` narrows the result when it parses to a positive integer
(strictly greater than 0); the value is capped at the default of 120, so `n=120`
is accepted and returns at most 120 samples, and any `n > 120` is clamped to 120.
A missing, non-numeric, or non-positive (`≤ 0`) value yields the default of 120.

When the buffer is empty or unset, the endpoint MUST return an empty JSON array
`[]`, never `null`.

#### Scenario: Returns recent samples as JSON array

- GIVEN the buffer holds several samples
- WHEN an authenticated client calls GET /api/system/history
- THEN the response is HTTP 200
- AND the body is a JSON array of MetricsSnapshot objects, oldest first

#### Scenario: n parameter bounds the result

- GIVEN the buffer holds 100 samples
- WHEN an authenticated client calls GET /api/system/history?n=10
- THEN at most 10 samples are returned

#### Scenario: Out-of-range n falls back to default

- GIVEN the buffer holds samples
- WHEN an authenticated client calls GET /api/system/history?n=0 or ?n=500 or ?n=abc
- THEN the default bound of 120 is applied (n=0 and n=abc are ignored; n=500 is clamped to 120 — the same effective result)

#### Scenario: Empty buffer returns empty array

- GIVEN the buffer holds no samples
- WHEN an authenticated client calls GET /api/system/history
- THEN the response is HTTP 200 with body `[]` (not `null`)

#### Scenario: Unauthenticated request is rejected

- GIVEN no valid auth token
- WHEN GET /api/system/history is called
- THEN the response is HTTP 401

---

### Requirement: Restart Event Tracker

The system MUST maintain a thread-safe ring buffer of `RestartEvent` entries
recording Node-RED auto-restarts. A `RestartEvent` MUST carry: `timestamp`,
`exitCode`, `attempt`, `maxAttempts`. Capacity is fixed (50); oldest events are
evicted FIFO.

#### Scenario: Restart events recorded and ordered

- GIVEN Node-RED has auto-restarted twice
- WHEN the restart events are read
- THEN two events are returned, oldest first
- AND each carries exitCode, attempt, and maxAttempts

---

### Requirement: Runtime History Endpoint

The system MUST expose `GET /api/runtime/history` returning a JSON object with
two fields: `events` (an array of `RestartEvent`, oldest first) and `status` (a
`RuntimeStatus` object carrying at least `uptime`, `restartCount`,
`consecutiveFailures`, and `startedAt`). `restartCount` MUST be the durable
cumulative auto-restart count; `consecutiveFailures` MUST be the backoff/give-up
counter since the last user-initiated start. The endpoint MUST require authentication. When the ProcessManager
is unset or has no events, `events` MUST be `[]` (never `null`) and `status` MUST
be the zero-valued RuntimeStatus.

#### Scenario: Returns events and status object

- GIVEN the ProcessManager has recorded restart events
- WHEN an authenticated client calls GET /api/runtime/history
- THEN the response is HTTP 200
- AND the body has an `events` array (oldest first) and a `status` object

#### Scenario: No ProcessManager returns empty events

- GIVEN the ProcessManager has not been wired
- WHEN GET /api/runtime/history is called by an authenticated client
- THEN the response is HTTP 200 with `events` equal to `[]` and a zero-valued `status`
- AND `startedAt` is omitted from the `status` JSON when empty (Go `omitempty`); consumers MUST treat its absence as a zero/empty value

---

### Requirement: Dashboard Chart Rendering

The frontend MUST render CPU, memory, and disk usage as area charts fed by
`GET /api/system/history`, polling at a fixed interval (30 seconds). Each chart
MUST handle loading and empty states without rendering a broken chart. The
frontend `MetricsSnapshot` type MUST match the backend JSON shape exactly.

#### Scenario: Charts render the polled series

- GIVEN the history endpoint returns samples
- WHEN the dashboard mounts
- THEN CPU, memory, and disk area charts render the series
- AND the data refreshes on the poll interval

#### Scenario: Loading state shown before first response

- GIVEN the first history request is in flight
- WHEN the dashboard renders
- THEN a loading placeholder is shown instead of a chart

#### Scenario: Empty history shows empty state

- GIVEN the history endpoint returns an empty array
- WHEN the dashboard renders
- THEN an explicit empty state is shown, not a broken chart

---

## New Behaviour (this change)

### Requirement: Panic Recovery Middleware

The system MUST install panic-recovery middleware as the first global middleware
in the chain, so it wraps all subsequent middleware and every route. When a
handler (or downstream middleware) panics, the middleware MUST recover, log the
panic, respond with HTTP 500, and keep the server process alive to serve
subsequent requests.

The recovery response MUST NOT leak stack traces or secret material to the
client.

#### Scenario: Panicking handler yields 500 and server survives

- GIVEN a route whose handler panics
- WHEN that route is requested
- THEN the response is HTTP 500
- AND a subsequent request to a healthy route succeeds (server still alive)

#### Scenario: Panic is logged without leaking to client

- GIVEN a handler panics with a message
- WHEN the request is handled
- THEN the panic is logged server-side
- AND the client response body contains no stack trace or secret material

---

### Requirement: Persisted Restart Count

The cumulative Node-RED auto-restart count MUST survive a process restart of
NRCC itself. The count MUST be persisted durably (e.g. a small JSON file under
the data directory) and reloaded on startup. A user-initiated start MAY reset the
in-session counter, but the durable cumulative count MUST NOT silently reset to
zero on NRCC restart.

If the persisted count cannot be read, the system MUST treat it as zero, log the
condition, and continue startup without blocking.

#### Scenario: Restart count survives NRCC restart

- GIVEN Node-RED has auto-restarted and the count is persisted
- WHEN the NRCC process is restarted
- THEN the reloaded cumulative restart count reflects the prior restarts (not 0)

#### Scenario: Unreadable count file degrades gracefully

- GIVEN the persisted count file is missing or corrupt
- WHEN NRCC starts
- THEN the count is treated as 0, the condition is logged, and startup proceeds

---

### Requirement: Enriched Health Endpoint

`GET /api/health` MUST remain public and MUST continue to return a `status`
field. It MUST additionally return `uptime` (integer seconds since process
start) and `restartCount` (the persisted cumulative Node-RED auto-restart
count). The new fields MUST be additive and MUST NOT break existing consumers.

When the ProcessManager is not yet wired, `uptime` and `restartCount` MUST report
0 without panicking.

#### Scenario: Health returns status, uptime, and restart count

- GIVEN the server is running and Node-RED started 3600 seconds ago with 2 restarts
- WHEN GET /api/health is called
- THEN the response is HTTP 200
- AND `status` is "ok"
- AND `uptime` is approximately 3600
- AND `restartCount` is 2

#### Scenario: Health before ProcessManager wiring does not panic

- GIVEN the server has started but SetProcessManager has not been called
- WHEN GET /api/health is called
- THEN the response is HTTP 200 with `uptime` 0 and `restartCount` 0, no panic

---

## Requirements Count

| Domain | Added | Modified | Removed | Scenarios |
|--------|-------|----------|---------|-----------|
| dashboard-runtime-metrics | 9 | 0 | 0 | 22 |
| **Total** | **9** | **0** | **0** | **22** |
