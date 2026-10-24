# Issue #878 — Prevent JSON commits on stale revision rejection (P1)

## Objective

Issue #878 (`fix(config): prevent JSON commits on stale revision rejection (P1)`)
reports that a structured apply commits `config.json` before the
`expectedRevision` precondition is enforced, so a rejected request still
changes the structured state while leaving `settings.js` untouched.

- Issue: <https://github.com/fgjcarlos/nrcc/issues/878>
- Branch: `fix/878-stale-revision-json`
- Base: `origin/main` = `eff5918` (merge of #921, `fix/877-redact-config-secrets`)
- Worktree: `/home/composedof2/Dev/Codex/nrcc-878` (isolated: the primary
  checkout holds unrelated uncommitted work for #875)

## Problem and why it matters

The audited baseline (`58d922ab`, finding R04) had two defects on the
same transaction:

1. **The structured commit ran before the precondition.** The apply
   coordinator committed `config.json` and only then compared the
   caller's `expectedRevision` with the live `settings.js` fingerprint.
   A stale request therefore produced HTTP 409 while leaving a durable
   change in the JSON store — the two representations disagreed.
2. **The visual save never carried a revision at all.** `POST /api/config`
   generated a *fresh server-side expectation* (`live.Revision`) and sent
   it into the transaction, so the precondition compared the live revision
   against itself and always passed. Two browser sessions editing the same
   deployment silently overwrote each other: the second save won, and the
   first operator's edit was gone without any conflict signal.

Both defects break the issue's expected behaviour: *"A stale request has
no durable effects and cannot overwrite current or independently
maintained state."*

## What the baseline already had (and what it did not)

The #876 delivery (runtime-coordinated apply) re-ordered the
coordinator, so on `origin/main` the revision precondition already runs
under the single-flight guard **before** `CommitStructured`, and every
later stage failure restores the structured snapshot. The audited
JSON-commit defect is therefore already closed on the current baseline —
verified by the issue's own reproduction test, which passes on
`origin/main` before any change in this slice.

What was still open is the visual save: `POST /api/config` had no way to
carry the revision the operator read, and the frontend never sent one.
That is the gap this slice closes.

## Authorized policy and constraints

- The precondition is **optional on the wire**. A client that never
  learned a revision keeps the historical behaviour (the server compares
  the live revision against itself). The operator UI always sends the
  revision it read.
- The precondition is enforced **before** any durable effect. The
  coordinator's guard covers the structured commit and the file write;
  the legacy path (no apply coordinator) enforces it in the handler.
- Both representations must be unchanged after **every** rejected
  transaction: conflict, validation failure, backup failure, write
  failure.
- Concurrent environment edits (the env-var slice of the same JSON
  store) must survive a stale whole-config snapshot.
- Repository language rule: every artifact in this feature is written in
  English (see `AGENTS.md`).
- No merge, no issue-label change, no issue closure: those stay human
  decisions.

## Tasks

- [x] **T1 — Baseline audit (RED).** Reproduced the issue's test verbatim
      on `origin/main` (it passes: the JSON-commit half is closed by
      #876) and reproduced the remaining defect with a scratch probe:
      a stale `POST /api/config` returned HTTP 200 and overwrote the
      newer configuration (`uiPort 7777 → 9999`).
- [x] **T2 — Backend precondition on the visual save.** `POST /api/config`
      now decodes the optional `expectedRevision` field (two-pass decode,
      so the typed `NodeRedConfig` decode cannot swallow it) and passes
      it into the apply transaction as the precondition. The legacy path
      (handler without an apply coordinator) enforces the same
      precondition through `checkExpectedRevision`.
- [x] **T3 — Frontend wiring.** The visual save, the raw editor and the
      curated-preset apply all send the revision of the document the
      operator opened. A successful save invalidates the raw-settings
      query so the next save carries the fresh revision.
- [x] **T4 — Regression coverage.** Two-session scenario
      (`internal/handler/config_revision_test.go`), the store/file
      invariants on every rejected transaction and the concurrent
      environment-edit invariant
      (`internal/handler/config_rejected_transaction_test.go`), the
      issue's release-audit reproduction, and the frontend wiring
      assertions.
- [x] **T5 — Documentation.** OpenAPI documents the new body field and
      the 409 envelope on `POST /api/config`; the apply-pipeline handbook
      documents the revision precondition across the save surfaces.
- [x] **T6 — Verification.** Go build/vet/test, golangci-lint, the
      frontend lint/typecheck/unit suite, the OpenAPI lint and the
      language scanner.

## Acceptance criteria

- [x] Check the precondition under the transaction guard before
      persisting JSON or files.
- [x] Keep both representations unchanged on conflict, validation,
      backup and write failure.
- [x] Ensure concurrent environment edits are not overwritten by a stale
      whole-config snapshot.
- [x] Wire the visual save to the revision it actually read rather than
      generating a fresh server-side expectation.
- [x] Test two browser sessions and store/file invariants on every
      rejected transaction.

## Verification evidence

**RED (before the fix).** The scratch probe on `origin/main`:

```
$ go test ./internal/handler -run TestScratch_SaveConfigStaleRevision -count=1 -v
    session A revision: 509adac7d4d44927c74c62f6f7e185a4615481e6e56980d03a135e96a4510688
    session B status: 200
    session A status: 200 body={"success":true,"data":{"uiPort":9999,...}}
    config.json uiPort after: 9999
```

The stale session A save was accepted and overwrote session B's
configuration. The issue's own reproduction
(`TestReleaseAuditStaleApplyMustNotCommitJSON`) already passed on
`origin/main`, which confirmed that the JSON-commit half of the finding
was closed by #876 and that the remaining gap was the visual save.

**GREEN (after the fix).**

```
$ go test ./internal/handler -run "TestConfigHandler_SaveConfig_|TestReleaseAuditStaleApplyMustNotCommitJSON" -count=1 -v
--- PASS: TestConfigHandler_SaveConfig_StaleRevisionRejectedKeepsBothRepresentations
--- PASS: TestConfigHandler_SaveConfig_AcceptedRevisionCommits
--- PASS: TestConfigHandler_SaveConfig_RejectedTransactionKeepsBothRepresentations
    --- PASS: .../stale_revision        (409 SETTINGS_REVISION_CONFLICT)
    --- PASS: .../validation_failure    (400 VALIDATION_ERROR)
    --- PASS: .../backup_failure        (500 SETTINGS_BACKUP_FAILED)
    --- PASS: .../write_failure         (500 SETTINGS_WRITE_ERROR)
--- PASS: TestConfigHandler_SaveConfig_LegacyPathHonoursExpectedRevision
--- PASS: TestConfigHandler_SaveConfig_ConcurrentEnvEditsSurviveStaleSnapshot
--- PASS: TestReleaseAuditStaleApplyMustNotCommitJSON
ok  	github.com/fgjcarlos/nrcc/internal/handler	40.596s
```

Repository-wide gates:

```
$ go build ./... && go vet ./...
$ go test ./...            # every package ok
$ golangci-lint run ./...  # v2.14.0 (the version CI pins) → 0 issues
$ go run ./tools/langscan . # → []
$ cd frontend && pnpm typecheck && pnpm lint && pnpm test --run
  # tsc --noEmit clean; eslint 0 errors (1 pre-existing warning in
  # FormFields.tsx); 85 test files, 586 passed | 2 skipped
$ npx --yes @redocly/cli lint docs/openapi.yaml --format stylish
  # valid; 9 warnings — the same count as the main baseline
```

Work units (one reviewable unit each, all inside the 400-line budget):

1. `fix(config): enforce the client revision precondition on the visual save`
2. `fix(frontend): send the revision the operator read with every save`
3. `test(config): cover the stale-revision rejection of issue #878`
4. `test(config): pin the store and file invariants of a rejected save`
5. `test(frontend): assert every save carries the revision it read`
6. `docs(odd): track the issue #878 delivery`

## Known boundaries

- The precondition is optional on the wire. A client that never learned
  a revision keeps the historical "no expectation" behaviour; only the
  operator UI sends one today.
- The structured commit re-derives two host-owned metadata fields
  (`settingsPath`, `settingsSource`) on every save. A rejected
  transaction on a store that never recorded them leaves those two
  derived fields behind; they are idempotent, host-owned and returned by
  every read, and the restore deliberately skips them because they are
  not operator state.
- The raw editor and the curated-preset apply now carry the revision of
  the document they were opened against. A preset preview computed
  against an older document is therefore rejected with 409 instead of
  being applied to newer source — the operator reloads and re-previews.
- The revision precondition compares the `settings.js` fingerprint. A
  change that only touches the JSON store (for example an env-var edit)
  does not advance the revision, which matches the transaction boundary:
  the env slice is preserved by the structured commit itself.

## Next step

The pull request is open and green: <https://github.com/fgjcarlos/nrcc/pull/922>
(`fix/878-stale-revision-json` → `main`, label `type:bug`, `Closes #878`).

CI status on the pull request — every job green:

| Job | Result |
|-----|--------|
| Backend (Go 1.27.3) | pass |
| Frontend (Node 22) | pass |
| Lint (golangci-lint) | pass |
| Docker stack acceptance (#434) | pass |
| OpenAPI Spec | pass |
| Language policy scan | pass |
| Hardcoded copy guard | pass |
| i18n catalog coverage | pass |
| Pages content + link integrity | pass |
| Trivy container scan / Trivy / CodeQL / govulncheck / pnpm audit | pass |

`main` requires one approving review before the merge, so the remaining
human decisions are: review the pull request and merge it. The issue
stays open until the merge lands `Closes #878`.
