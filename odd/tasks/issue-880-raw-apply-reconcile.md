# Issue #880 — Reconcile the structured state after a raw settings apply (P1)

## Objective

Issue #880 (`fix(config): reconcile structured state after raw settings apply (P1)`)
reports that a successful raw apply updates `settings.js` but leaves
`config.json` authoritative for subsequent structured reads, so the UI
displays stale configuration.

- Issue: <https://github.com/fgjcarlos/nrcc/issues/880>
- Branch: `fix/880-raw-apply-reconcile`
- Base: `origin/main` = `606ed36` (merge of #923, `fix/879-security-partial-patch`)
- Worktree: `/home/composedof2/Dev/Codex/nrcc-880` (isolated: the primary
  checkout holds unrelated uncommitted work for #875)

## Problem and why it matters

`settings.js` and `config.json` are two representations of the same
Node-RED configuration:

- `settings.js` is the source Node-RED actually reads.
- `config.json` is NRCC's structured mirror. `ConfigService.Get` reads the
  mirror first and only falls back to the source when the mirror does not
  exist.

The apply pipeline writes the source but never refreshes the mirror on the
raw surfaces:

- `POST /api/config/apply/raw` (slice C of #758) drives the apply
  coordinator, which writes `settings.js` and stops there.
- `POST /api/settings/raw` drives the same coordinator in production
  (`internal/server/server.go` wires `SetApplyCoordinator` on both
  handlers), so the legacy `writeSettingsFile(content, true)` mirror sync
  is dead code in a real deployment.

Consequences on the current baseline:

1. A structured read (`GET /api/config`) keeps reporting the pre-apply
   values, so the operator sees the old configuration after a successful
   raw edit.
2. The next visual save renders `settings.js` from the stale mirror and
   silently reverts the raw edit — the operator's raw change disappears.
3. The legacy mirror sync (when it runs at all) writes the parsed source
   wholesale, which drops the NRCC-owned state the mirror also holds:
   `envVars` (with their encryption state), `functionLibraries` and
   `runtimeState`.

The dashboard policy apply (`internal/service/dashboard_access.go`) has
the same defect: it patches `httpNodeAuth` into the source through the
apply service and leaves the mirror reporting the previous value.

## Authoritative state (the contract this slice implements)

- **`settings.js` is authoritative** for every value Node-RED reads. A
  raw apply therefore defines the current configuration.
- **`config.json` is a derived mirror** of the values NRCC's structured
  model can represent, plus the NRCC-owned state the mirror holds
  (`envVars` and their encryption metadata, `functionLibraries`,
  `runtimeState`, the derived `settingsPath` / `settingsSource`).
- After a successful raw apply the mirror is **re-derived from the
  applied bytes** in the same apply transaction, so a structured read
  reports what the running Node-RED uses.
- Values the source does **not** declare keep their mirrored value: the
  source declares a value, it does not reset the ones it is silent about.
- Operator-owned ("unmanaged") source blocks — anything outside
  `managedSettingKeys` — are never rewritten by the reconciliation. They
  live in `settings.js` only and the source-preserving patchers keep
  them verbatim.

## Authorized policy and constraints

- The reconciliation runs **inside the apply transaction**, after the
  atomic write and before the success event, so a failed transaction
  leaves the source and the mirror consistent with each other.
- A failed reconciliation restores the pre-write source bytes, so a
  rejected transaction has no durable effect on either representation.
- The structured apply path is untouched: it commits the mirror before
  the write, exactly as #876/#878 established.
- The reconciliation never takes NRCC-owned state from the source.
  `envVars` keep their encryption state, type, description and source;
  a source value only refreshes the *value* of a key NRCC already
  manages.
- The source is inspected with the existing goja sandbox. Host APIs stay
  blocked; a source that cannot be evaluated (dynamic settings) falls
  back to the regex parser and keeps the mirrored values for the blocks
  the sandbox could not read. Inspection never executes candidate source
  with Node and never grants a host API.
- Repository language rule: every artifact in this feature is written in
  English (see `AGENTS.md`).
- No merge, no issue-label change, no issue closure: those stay human
  decisions.

## Tasks

- [x] **T1 — Baseline audit (RED).** Reproduce the issue's own
      reproduction test on `origin/main` and record the failure.
- [x] **T2 — Source-authoritative reconciliation.** New service API that
      re-derives the mirror from the applied source: declared managed
      values win, undeclared values are preserved, NRCC-owned state is
      preserved, unsupported source blocks stay in the source.
- [x] **T3 — Apply pipeline integration.** The reconciliation runs in the
      apply transaction with rollback consistency: a failed
      reconciliation restores the pre-write source bytes, and a failed
      runtime stage restores the mirror.
- [x] **T4 — Legacy raw save.** The coordinator-less raw save uses the
      same reconciliation instead of writing the parsed source wholesale.
- [x] **T5 — Regression coverage.** The issue's reproduction test, the
      raw → structured read → unrelated visual edit scenario, the
      rollback invariants, and the NRCC-owned state preservation.
- [x] **T6 — Frontend.** A successful raw save refreshes the structured
      configuration query and the revision, so the UI stops showing the
      pre-apply configuration.
- [x] **T7 — Documentation.** The apply-pipeline handbook documents the
      new stage; this task file records the evidence.
- [x] **T8 — Verification.** Go build/vet/test, golangci-lint, the
      frontend lint/typecheck/unit suite, the frontend build, the OpenAPI
      lint and the language scanner.

## Implementation record

### Service layer

| File | Change |
|------|--------|
| `internal/service/config_reconcile.go` (new) | `ReconcileRawSettings` re-derives the mirror from the source; `RestoreMirror` rolls it back; `reconcileConfigWithSource` merges declared values; `mergeEnvVars` adopts source values while keeping the NRCC-owned metadata. |
| `internal/service/settings_export.go` (new) | `exportSettingsObject` evaluates the source in the isolated goja sandbox and returns the exported top-level entries (exact declared-key set, `editorTheme`, `env`). |
| `internal/service/apply.go` | New `reconcile` stage between the atomic write and the success event; `ApplyResult.Mirror` carries the snapshot; `restoreSourceBytes` undoes a failed reconciliation. |
| `internal/service/apply_runtime.go` | The runtime rollback restores the mirror together with the source. |
| `internal/service/config.go` | The coordinator-less raw save reconciles instead of writing the parsed source wholesale. |
| `internal/store/json_store.go` | `Delete` so a reconciliation that created the mirror can be undone. |
| `internal/handler/config.go` | `reconcile` failures map to `VALIDATION_ERROR` (400, `ErrMirrorRejected`) or `SETTINGS_RECONCILE_ERROR` (500). |

### Frontend

| File | Change |
|------|--------|
| `frontend/src/features/configuration/hooks/useConfigurationActions.ts` | A successful raw save invalidates `queryKeys.config.root` in addition to the raw document and the bootstrap status, so the structured configuration is refreshed. |

### Documentation

| File | Change |
|------|--------|
| `docs/handbook/configuration/apply-pipeline.md` | Five stages / nine steps; the new "Step 7 — Reconcile the structured mirror" section; the failure-mode table; the audit transcript. |

## Acceptance criteria

- [x] A successful raw apply makes the structured read report the applied
      values. — `TestApplyRaw_ReconcilesStructuredState`
- [x] The next visual save does not revert the raw change. —
      `TestApplyRaw_UnrelatedVisualEditKeepsRawChange`
- [x] Unsupported source blocks and NRCC-owned environment/state metadata
      survive the reconciliation. —
      `TestReconcileRawSettings_PreservesNRCCOwnedState`,
      `TestReconcileRawSettings_LeavesSourceUntouched`,
      `TestReconcileRawSettings_AdoptsSourceEnvValues`
- [x] A rejected transaction leaves both representations unchanged. —
      `TestApplyRaw_RejectedReconciliationLeavesBothRepresentationsUnchanged`,
      `TestRuntimeRawApplyRestoresMirrorOnReadinessFailure`
- [x] The frontend refreshes the configuration query and the revision
      after a raw change. — `useConfigurationActions.test.ts`

## Verification evidence

Baseline (RED), on `origin/main` = `606ed36`, before the change:

```
--- FAIL: TestApplyRaw_ReconcilesStructuredState (3.34s)
    config_reconcile_test.go:58: expected the structured read to report uiPort 7777, got 1880
--- FAIL: TestApplyRaw_UnrelatedVisualEditKeepsRawChange (5.32s)
    config_reconcile_test.go:98: the visual save reverted the raw change:
        module.exports = {
          uiPort: 1880,
          httpAdminRoot: "/nrcc",
          httpNodeRoot: "/",
        };
```

After the change:

| Check | Command | Result |
|-------|---------|--------|
| Backend build | `go build ./...` | pass |
| Backend vet | `go vet ./internal/...` | pass |
| Backend tests | `go test ./...` | pass (see below) |
| Backend lint | `golangci-lint run ./internal/...` | `0 issues.` |
| Frontend lint | `pnpm lint` | pass |
| Frontend typecheck | `pnpm typecheck` | pass |
| Frontend tests | `pnpm test --run` | 86 files, 590 passed, 2 skipped |
| Frontend build | `pnpm build` | pass |
| OpenAPI lint | `npx @redocly/cli lint docs/openapi.yaml` | valid (9 pre-existing warnings) |
| Language policy | `go run ./tools/langscan .` | `[]` |

RED → GREEN was observed for the frontend as well: removing the
`queryKeys.config.root` invalidation fails
`useConfigurationActions.test.ts` ("expected config.root to be
invalidated").

The only test that had to be adapted is the pre-existing
`TestApplyTransactionHappyPath`, whose documented audit order grew by the
new `apply.reconcile` event.

## Known boundaries

- A source the sandbox cannot evaluate (dynamic settings that call
  `require` / `process`) is rejected by the apply validation stage before
  the write, exactly as on the baseline. The reconciliation's fallback
  (the regex key scan) therefore only matters for the coordinator-less
  legacy path.
- The reconciliation adopts the environment *values* the source declares
  and keeps the NRCC-owned metadata (type, description, encryption state,
  origin). An encrypted key keeps NRCC's stored ciphertext: the source
  never renders an encrypted value, so a plaintext there is not NRCC's to
  adopt.
- `runtimeState` and `functionLibraries` stay NRCC-owned: the structured
  model never renders them into the source, so the source never speaks
  for them.
- The `logging` block is rendered from a fixed template by NRCC, so the
  mirror keeps its value; the source's `logging` block was never read
  back on the baseline either.
- A source that declares a configuration the mirror cannot hold (an
  out-of-range port, an `adminAuth` without users) is rejected with
  `VALIDATION_ERROR`; the pre-write source bytes are restored, so the
  transaction has no durable effect.

## Next step

The pull request is open: <https://github.com/fgjcarlos/nrcc/pull/924>

CI status on the pull request — every job green:

| Workflow | Result |
|----------|--------|
| PR Validation (backend, frontend, OpenAPI, golangci-lint) | success |
| Acceptance (Docker stacks) | success |
| Language policy (advisory) | success, 0 findings |
| i18n checks | success |
| Security (CodeQL, Trivy, govulncheck, pnpm audit, GitGuardian) | success |
| Pages freshness checker | success |

The remaining human decisions are: review the pull request, merge it, and
close the issue. The issue label was not changed on the maintainer's
behalf.
