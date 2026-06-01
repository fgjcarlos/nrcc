# Metrics Endpoint Specification

## Purpose

Defines the Prometheus-compatible metrics scrape endpoint and the instrumentation
points across NRCC operations. This is a new capability with no prior spec.

---

## Requirements

### Requirement: Expose Scrape Endpoint

The system MUST expose a GET /metrics endpoint that returns current metric values
in Prometheus text exposition format (version 0.0.4 or later).

The endpoint MUST be reachable without authentication.
The endpoint MUST be registered before the SPA fallback handler.
The endpoint MUST NOT appear under any auth-guarded route group.

#### Scenario: Unauthenticated scrape succeeds

- GIVEN the server is running
- WHEN a Prometheus scraper calls GET /metrics with no credentials
- THEN the server responds with HTTP 200
- AND the Content-Type header matches `text/plain; version=0.0.4`
- AND the body contains at least the declared metric names

#### Scenario: Endpoint position before SPA fallback

- GIVEN the router is initialised
- WHEN GET /metrics is requested with a path that also matches the SPA `/*` pattern
- THEN the metrics handler takes priority and returns Prometheus output, not the SPA HTML

---

### Requirement: Login Instrumentation

The system MUST increment `nrcc_login_attempts_total` on every login attempt.
The counter MUST carry a `result` label with value `success` or `failure`.

#### Scenario: Successful login

- GIVEN a valid username and password
- WHEN POST /api/auth/login returns HTTP 200
- THEN `nrcc_login_attempts_total{result="success"}` is incremented by 1

#### Scenario: Failed login

- GIVEN an invalid username or password
- WHEN POST /api/auth/login returns HTTP 401
- THEN `nrcc_login_attempts_total{result="failure"}` is incremented by 1

---

### Requirement: Backup Instrumentation

The system MUST increment `nrcc_backup_created_total` each time a backup is
successfully created. The counter MUST carry a `type` label with one of:
`manual`, `auto`, or `pre_restore`.

#### Scenario: Manual backup created

- GIVEN a user triggers a manual backup via POST /api/backups
- WHEN the backup file is written successfully
- THEN `nrcc_backup_created_total{type="manual"}` is incremented by 1

#### Scenario: Scheduler backup created

- GIVEN the backup scheduler fires
- WHEN the backup completes without error
- THEN `nrcc_backup_created_total{type="auto"}` is incremented by 1

#### Scenario: Pre-restore backup created

- GIVEN a restore operation is triggered
- WHEN the system creates a pre-restore safety backup
- THEN `nrcc_backup_created_total{type="pre_restore"}` is incremented by 1

---

### Requirement: Restore Instrumentation

The system MUST increment `nrcc_restore_attempts_total` on every restore attempt.
The counter MUST carry a `result` label with value `success` or `failure`.

#### Scenario: Successful restore

- GIVEN a valid backup archive
- WHEN POST /api/backups/{id}/restore returns HTTP 200
- THEN `nrcc_restore_attempts_total{result="success"}` is incremented by 1

#### Scenario: Failed restore

- GIVEN a corrupt or missing backup archive
- WHEN the restore returns a non-2xx status
- THEN `nrcc_restore_attempts_total{result="failure"}` is incremented by 1

---

### Requirement: Node-RED Process State Instrumentation

The system MUST expose the following gauges reflecting live ProcessManager state
at scrape time:

| Metric | Type | Description |
|--------|------|-------------|
| `nrcc_nodered_running` | Gauge | 1 if Node-RED process is running, 0 otherwise |
| `nrcc_nodered_restarts_total` | Gauge | Durable cumulative auto-restart count for this NRCC installation |
| `nrcc_nodered_uptime_seconds` | Gauge | Seconds since the process last started; 0 when stopped |

When ProcessManager is nil (server started before SetProcessManager is called),
all three gauges MUST report 0 without panicking.

#### Scenario: Node-RED running

- GIVEN Node-RED is running with 2 restarts and started 300 seconds ago
- WHEN GET /metrics is scraped
- THEN `nrcc_nodered_running` is 1
- AND `nrcc_nodered_restarts_total` is 2
- AND `nrcc_nodered_uptime_seconds` is approximately 300 (within ±2 s of elapsed wall time)

#### Scenario: Node-RED stopped

- GIVEN Node-RED is not running
- WHEN GET /metrics is scraped
- THEN `nrcc_nodered_running` is 0
- AND `nrcc_nodered_uptime_seconds` is 0

#### Scenario: ProcessManager not yet wired

- GIVEN the server has started but SetProcessManager has not been called
- WHEN GET /metrics is scraped
- THEN `nrcc_nodered_running` is 0 and no panic occurs

---

### Requirement: Update Instrumentation

The system MUST increment `nrcc_update_attempts_total` on every Node-RED update
attempt. The counter MUST carry a `result` label with value `success` or `failure`.

#### Scenario: Successful update

- GIVEN an update is applied via POST /api/updates/apply
- WHEN the update process completes without error
- THEN `nrcc_update_attempts_total{result="success"}` is incremented by 1

#### Scenario: Failed update

- GIVEN an update attempt fails (network error, checksum mismatch, etc.)
- WHEN the update process returns an error
- THEN `nrcc_update_attempts_total{result="failure"}` is incremented by 1

---

### Requirement: Library Operation Instrumentation

The system MUST increment `nrcc_library_operations_total` on every install or
uninstall operation. The counter MUST carry two labels:

| Label | Values |
|-------|--------|
| `operation` | `install` or `uninstall` |
| `result` | `success` or `failure` |

#### Scenario: Successful library install

- GIVEN a POST /api/libraries/install request completes without error
- WHEN the npm operation returns exit code 0
- THEN `nrcc_library_operations_total{operation="install",result="success"}` is incremented by 1

#### Scenario: Failed library uninstall

- GIVEN a DELETE /api/libraries/{name} request fails
- WHEN the npm operation returns a non-zero exit code
- THEN `nrcc_library_operations_total{operation="uninstall",result="failure"}` is incremented by 1

---

### Requirement: Label Cardinality Constraint

All metric labels MUST use fixed, enumerated values.
Dynamic values (usernames, IP addresses, file paths, backup IDs, library names,
error messages) MUST NOT be used as label values.
No metric MAY have more than two labels.

#### Scenario: No dynamic label values emitted

- GIVEN any instrumented operation completes
- WHEN GET /metrics is scraped
- THEN every label value is one of the statically declared values for that label

---

### Requirement: No Secrets in Metric Output

Metric names, label names, label values, and help strings MUST NOT contain
authentication tokens, passwords, encryption keys, or any other secret material.

#### Scenario: Metrics output inspected for secrets

- GIVEN the system is running with authentication configured
- WHEN GET /metrics is scraped
- THEN the response body contains no JWT tokens, password hashes, or encryption keys

---

### Requirement: Test Isolation via Custom Registry

All metrics MUST be registered on a custom `prometheus.Registry` (not the default
global registry). Tests MUST be able to create a fresh registry per test case to
prevent counter bleed between test runs.

#### Scenario: Counter does not bleed between test cases

- GIVEN two independent test cases each performing one successful login
- WHEN each test uses its own MetricsCollector with a dedicated registry
- THEN each collector reports a count of 1 for `nrcc_login_attempts_total{result="success"}`
- AND the counts are independent of each other
