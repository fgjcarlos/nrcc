# Issue #882 — Render the logging values submitted by the operator (P1)

## Objective

Issue #882 (`fix(configuration): render the logging values submitted by the operator (P1)`)
reports that the logging controls report a successful persistence while the
renderer writes constant `level: 'info', metrics: false, audit: false` values
into `settings.js` instead of the configuration the operator submitted.

- Issue: <https://github.com/fgjcarlos/nrcc/issues/882>
- Branch: `fix/882-logging-render`
- Base: `origin/main` = `377d11e`

## Problem and why it matters

`renderLoggingBlock()` takes no configuration argument and returns a constant
block, so every structured apply writes the same logging values:

- `internal/service/config.go` — `renderLoggingBlock()` returns
  `"  logging: {\n    console: { level: 'info', metrics: false, audit: false },\n  },"`.
- `internal/service/config.go` — `generateSettingsJS()` writes the same
  constant block when the file is generated from scratch.
- `internal/service/config.go` — `patchSettingsJS()` replaces the whole
  `logging` block with that constant on every save.
- `internal/service/config.go` — `parseConfigFromContent()` never reads the
  `logging` block back, so `cfg.Logging` is always empty when the
  configuration is parsed from `settings.js`.

Consequences on the current baseline:

1. `POST /api/config/apply` with `logging.console.level = "debug"` and
   `logging.console.metrics = true` returns `200 OK` and writes
   `level: 'info'` — the operator's change never reaches Node-RED.
2. The structured mirror never learns the live logging values, so a read
   after a raw apply cannot report them either.
3. The block replacement drops every other logging entry the operator
   authored (a custom handler with its own `handler` function), which is
   the opposite of the "preserve unmanaged source" contract the rest of the
   renderer follows.

## Node-RED contract (verified against node-red 5.0.4)

`@node-red/util/lib/log.js` → `init(settings)`:

- `settings.logging` is an object of `handlerName -> handler config`.
- A handler becomes live when the name is `console` **or** the config
  exposes a `handler` function. Any other name without a `handler`
  function is ignored by Node-RED.
- The supported per-handler properties are `level`, `metrics` and `audit`.
- The levels are `off`, `fatal`, `error`, `warn`, `info`, `debug`, `trace`.

`internal` is an NRCC concept (Node-RED ships only `console` by default).
It still round-trips through the NRCC model and API because that is the
vocabulary the frontend publishes.

## Authorized policy and constraints

- The renderer writes **the values the operator submitted**, not a
  constant. The `level`, `metrics` and `audit` properties of the handlers
  NRCC manages (`console`, `internal`) are the ones it owns.
- Everything outside the edited capability is preserved byte for byte:
  another logging entry, a custom `handler` function, a property of a
  managed handler the operator did not submit, and every comment outside
  the block.
- The mirror learns the logging values from the source: the `logging`
  block is parsed back out of `settings.js` like every other managed
  block, so a read reports what Node-RED actually uses.
- The structured apply path stays presence-aware (issue #879): a property
  the payload omits keeps its live value.
- Repository language rule: every artifact in this feature is written in
  English (see `AGENTS.md`).
- No merge, no issue-label change, no issue closure: those stay human
  decisions.

## Tasks

- [x] **T1 — Baseline audit (RED).** Run the issue's reproduction test on
      `origin/main` and record the failure.
- [x] **T2 — Logging model.** Normalise the operator-submitted logging
      value into the handler/property model and render it.
- [x] **T3 — Source-preserving patch.** Apply the submitted values to the
      existing `logging` block without dropping the entries and the
      properties outside the edited capability.
- [x] **T4 — Parse.** Read the `logging` block back out of `settings.js`
      into the structured model.
- [x] **T5 — Regression coverage.** The issue's reproduction test plus
      file/API round-trip tests for levels, metrics and the applicable
      handlers, and the preservation guarantees.
- [x] **T6 — Live validation.** Apply a representative logging change
      against a running Node-RED and observe the effective level.
- [x] **T7 — Documentation.** The setting catalog describes the rendered
      logging block.
- [x] **T8 — Verification.** Go build/vet/test, golangci-lint, the
      frontend lint/typecheck/unit suite, the language scanner.

## Implementation record

### Backend

| File | Change |
|------|--------|
| `internal/service/logging.go` (new) | The logging model: normalisation, rendering, parsing and the source-preserving patch. |
| `internal/service/logging_scan.go` (new) | The depth-aware scanner that reads the entries of a JavaScript object literal, so a nested `handler` function is never mistaken for a sibling. |
| `internal/service/config.go` | `renderLoggingBlock` renders the operator-submitted values; `patchSettingsJS` patches the existing block in place; `parseConfigFromContent` reads the block back into the model; `appendPropertyBeforeClosingBrace` accepts a source that closes the object literal with a semicolon. |
| `internal/service/config_reconcile.go` | `logging` joins the reconcilable source keys, so a raw apply adopts the logging values the source declares. |
| `internal/handler/config_apply_test.go` | The issue's reproduction test plus the API round-trip test. |
| `internal/service/logging_test.go` | The file round-trip tests, the preservation guarantees and the JavaScript validity check. |
| `internal/service/logging_scan_test.go` | The scanner tests. |

### Frontend

| File | Change |
|------|--------|
| `frontend/src/shared/types/index.ts` | `LoggingLevel` carries the `off` level Node-RED supports. |
| `frontend/src/features/configuration/components/FormFields.tsx` | The level list offers `off`. |

### Acceptance

| File | Change |
|------|--------|
| `scripts/acceptance/docker-stacks.sh` | Scenario 7 applies a logging-only patch through the real apply API and proves the live Node-RED runs with the submitted level. |

### Documentation

| File | Change |
|------|--------|
| `docs/handbook/configuration/setting-catalog.md` | The `logging` section describes the rendered block, the supported levels and the `internal` handler. |

## Acceptance criteria

- [x] A structured apply with `logging.console.level = "debug"` and
      `metrics = true` writes those values into `settings.js`. —
      `TestReleaseAuditStructuredApplyMustHonorLogging`,
      `TestPatchSettingsJS_RendersSubmittedLoggingValues`
- [x] The structured read reports the logging values of the source. —
      `TestParseConfigFromContent_ReadsLoggingBlock`,
      `TestApplyStructuredHandler_LoggingRoundTrip`
- [x] A custom logging entry and a property the operator did not submit
      survive a save. — `TestPatchSettingsJS_PreservesCustomLoggingHandler`,
      `TestPatchSettingsJS_PreservesUnsubmittedLoggingProperty`,
      `TestPatchLoggingSettings_Idempotent`
- [x] A representative logging change takes effect in a live Node-RED. —
      acceptance scenario 7

## Verification evidence

Baseline (RED), on `origin/main` = `377d11e`, before the change:

```
--- FAIL: TestReleaseAuditStructuredApplyMustHonorLogging (4.03s)
    config_apply_test.go:482: successful apply rendered logging.console.level=info instead of debug
```

After the change:

| Check | Command | Result |
|-------|---------|--------|
| Backend build | `go build ./...` | pass |
| Backend vet | `go vet ./internal/...` | pass |
| Backend tests | `go test ./internal/...` | pass (one pre-existing environmental failure, see below) |
| Backend lint | `golangci-lint run --config=.golangci.yml ./...` (v2.14.0) | `0 issues.` |
| Frontend lint | `corepack pnpm lint` | pass (1 pre-existing warning) |
| Frontend typecheck | `corepack pnpm typecheck` | pass |
| Frontend tests | `corepack pnpm test --run` | 86 files, 590 passed, 2 skipped |
| Language policy | `go run ./tools/langscan .` | `[]` |
| Acceptance | `bash scripts/acceptance/docker-stacks.sh` | 7 scenarios, ALL SCENARIOS PASSED |

The pre-existing failure is
`TestNpmInstallKillsChildOnCancel` (`internal/service`): it fails the same
way on a pristine `origin/main` worktree, because the `pgrep` in the
sandbox matches unrelated processes. It is unrelated to this change and
does not reproduce on a clean run.

### The defect the CI found

The first CI run failed in the browser acceptance step, not in the
acceptance scenario: the configuration form submits the internal handler
alongside the console one, so the renderer inserted an entry after the
last entry of the logging block. The runtime image ships a `settings.js`
whose last logging entry carries no trailing comma, and the inserted entry
was not separated from it: the rendered file was not valid JavaScript, the
apply was rejected and the save never reached the file.

The fix separates the inserted entry from the last one when the last one
carries no comma. The regression test renders the file and evaluates it
with the JavaScript engine of the sandbox, so a `settings.js` that does
not parse fails the test.

The browser acceptance suite passes on the fixed tree:

```
✓  unknown API routes fail as JSON instead of returning the SPA
✓  invalid raw settings are rejected without restarting healthy Node-RED
✓  dashboard restart changes the managed Node-RED process PID
✓  dashboard labels container-scoped resource metrics returned by the API
✓  environment action reaches a live Node-RED flow after its restart
✓  environment page imports a Node-RED global entry automatically on route entry
✓  unrelated visual edit keeps every admin account, permission and session expiry

7 passed
```

The live Node-RED evidence (acceptance scenario 7):

```
[PASS] the live runtime logs at the info level before the change contains 'nrcc-acceptance-probe'
[PASS] logging patch response contains '"success":true'
[PASS] the response reports the submitted level contains '"level":"off"'
[PASS] rendered settings carry the submitted level contains 'level: 'off''
[PASS] rendered settings kept the live UI port contains 'uiPort: 1880'
[PASS] the live runtime honours the submitted log level
[PASS] restore patch response contains '"success":true'
[PASS] the live runtime logs again after the level was restored contains 'nrcc-acceptance-probe'
[PASS] scenario 7: PASS (the submitted logging configuration reached the live runtime)
```

The probe runs the Node-RED logging library that ships with the runtime
against the live `settings.js` the running instance was started with. The
runtime's own stdout is drained by the supervisor, so the container log
cannot show it. The scenario also proves the apply restarted the runtime:
the Node-RED process runs under a new pid after the change.

## Known boundaries

- `internal` is not a Node-RED built-in handler: Node-RED ignores it
  unless the entry carries a `handler` function. It is rendered and
  parsed because it is the NRCC UI's vocabulary, and it round-trips
  through the configuration model and the API.
- The `audit` property is part of the Node-RED per-handler contract but is
  not exposed by the NRCC UI. It is rendered and parsed when the model or
  the source carries it, and preserved otherwise.
- A logging entry whose value is not an object literal (for example
  `console: null`) cannot carry properties. It is replaced by the
  rendered entry when the operator submits values for that handler.
- The block always carries a console handler. Node-RED registers the
  entries named `console` or the ones exposing a handler function, so a
  block without a console entry would leave the runtime completely
  silent.
- An operator-authored logging block survives a save that configures no
  logging: NRCC does not rewrite a block it was not asked to change.
- A level is not validated against the Node-RED vocabulary: an unknown
  level is written as submitted and Node-RED falls back to `info`, exactly
  as it does for a hand-written settings.js. The value is escaped, so it
  can never inject JavaScript into settings.js.

