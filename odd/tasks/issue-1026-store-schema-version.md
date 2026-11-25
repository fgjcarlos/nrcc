# Issue #1026 — JSON store schema version

Base: `origin/main`. Risk: item 1 (persistent file format).

## Specs

- S1: Add a `schemaVersion` field to each file.
  - `config.json`, `cc-users.json` and `backup_config.json`
- S2: A file with a higher version stops startup with an explanatory error and is left untouched.
- S3: The README status note and `docs/operations/docker-stack.md` state the compatibility promise.
  - Document the policy: data written by 1.x is readable by any later 1.y.

## Tasks

- T1 [done] S1–S3 — Inline test-first implementation and independent verification — `9c34f9e`.

## Log

- L1: Resuelvela issue #1026
- L2: GitHub issue #1026 (verbatim):

### Pre-flight checks

- [x] Searched open and closed issues on 2026-11-20; no open issue covers this finding.
- [x] This issue needs maintainer triage (`status:approved`) before a PR is opened.

### Problem description

`config.json`, `cc-users.json` and `backup_config.json` carry no schema version; only the backup manifest has one. The README says there is "no promised compatibility between minor versions". A stable release needs the opposite statement and a way to keep it.

Suspected impact, not reproduced: an older binary reading a newer file drops the fields it does not know on its next write, for example after a rollback to a previous image.

**Audit:** v1.0 readiness audit, 2026-11-20, finding 18. **Priority:** P1, closes before `v1.0.0`. **Milestone:** D, Data durability. **Evidence:** single read; reproduce before fixing.

Evidence: `internal/store/json_store.go`; the only version field is at `internal/service/backup.go:1218`; README status note.

### Proposed solution

Add a `schemaVersion` field to each file. Refuse to start on a version newer than the binary understands, with a clear message. Document the policy: data written by 1.x is readable by any later 1.y.

### Affected area

Backend (Go services / API)

### Acceptance criteria

- [ ] A file with a higher version stops startup with an explanatory error and is left untouched.
- [ ] The README status note and `docs/operations/docker-stack.md` state the compatibility promise.

### Relevant source

- [internal/store/json_store.go](https://github.com/fgjcarlos/nrcc/blob/80ad47c411bd042b1a36d3a8acb47d42210aab03/internal/store/json_store.go)
- [internal/service/backup.go:1218](https://github.com/fgjcarlos/nrcc/blob/80ad47c411bd042b1a36d3a8acb47d42210aab03/internal/service/backup.go#L1218)
- [README.md](https://github.com/fgjcarlos/nrcc/blob/80ad47c411bd042b1a36d3a8acb47d42210aab03/README.md), [docs/operations/docker-stack.md](https://github.com/fgjcarlos/nrcc/blob/80ad47c411bd042b1a36d3a8acb47d42210aab03/docs/operations/docker-stack.md)

### Environment and baseline

- Static audit of `main` at [80ad47c](https://github.com/fgjcarlos/nrcc/commit/80ad47c411bd042b1a36d3a8acb47d42210aab03).
- No local build or test run during the audit; CI on `main` was green on 2026-11-20.

- L3: RED: baseline starts with each future-version file and creates additional data-dir files; unversioned writes omit the marker; restore publishes future-version payloads. Focused checks GREEN for startup (exit, exact stderr, byte preservation), legacy writes, and restore rejection.
- L4: Independent verifier: S1–S3 met. Unversioned store differential: identical stdout, stderr, and exit. Build and changed-file language scan passed.
- L5: The default full suite failed only at `TestSaveConfig_WritesExplicitRuntimeSettingsAndRestarts`. The baseline fails identically because the host Docker Node-RED is 5.0.10 (fixture expects 5.0.6). The same test passed with Docker isolated. The complete `go test ./...` run also passed with `DOCKER_HOST` pointed at a nonexistent socket in a temporary directory.
