# Acceptance evidence

> **A green suite proves only what it exercised.** Read every result with
> its evidence class: a mocked browser test proves UI behaviour, never that
> Node-RED accepted, loaded or survived a configuration.

This page lists the test suites, what each one can prove, the acceptance
scenarios of the configuration transaction, and how skipped cases and
environment limitations are reported. It was introduced by issue
[#887](https://github.com/fgjcarlos/nrcc/issues/887).

## Evidence classes

| Class | Suite | Runs against | Proves | Never proves |
|-------|-------|--------------|--------|--------------|
| Mocked fixtures | `frontend/e2e/*.spec.ts` except `stack.spec.ts`, Playwright project `chromium` (`frontend/playwright.config.ts`) | Vite dev server with every API call answered by `installApiMocks` (`frontend/e2e/helpers.ts`) | Rendering, navigation, accessibility, interaction wiring | Any backend or runtime behaviour |
| Real runtime | `frontend/e2e/stack.spec.ts`, Playwright project `chromium-real-stack` (`frontend/playwright.stack.config.ts`) | The release image started with `docker compose up` (NRCC on `:3001`, Node-RED on `:1880`) | The configuration transaction, the live `settings.js` source and the managed Node-RED process | Platforms or deployments the compose stack does not model |
| Docker stacks | `scripts/acceptance/docker-stacks.sh` | Several isolated compose projects | Volumes, permissions, upgrades and backups of the deployed stack | Browser behaviour |
| Unit and fixture | `go test ./...`, `cd frontend && npm test -- --run` | Isolated stores, files and test-owned subprocesses | Handler, service and component rules | Live Node-RED acceptance |

Go fixtures never touch host state. A process test uses a dedicated subprocess
that publishes its own PID, and update-service tests inject their version-check
dependencies before the service is constructed, so no test calls the host
`npm` or `node-red`.

## Configuration transaction scenarios (real runtime)

All of them live in `frontend/e2e/stack.spec.ts` and run serially against one
stack.

| Scenario | Test title |
|----------|------------|
| Visual save and source preservation | `unrelated visual edit keeps every admin account, permission and session expiry` |
| Partial security patch | `partial security patch changes only HTTP-node authentication settings` |
| Invalid syntax | `invalid raw settings are rejected without restarting healthy Node-RED` |
| Invalid semantics | `semantically invalid raw settings are rejected without restarting Node-RED` |
| Stale revision, no write | `stale raw settings revision is rejected without changing live source` |
| Runtime rollback | `a runtime failure rolls back to the previous live configuration` |
| Cancellation | `cancelled configuration review sends no write and leaves source and runtime unchanged` |
| Restart | `dashboard restart changes the managed Node-RED process PID` |

The visual-save scenario seeds unmanaged source (a header comment, a line
comment, a block comment and an unknown key with a function value) and asserts
that it survives the save byte for byte.

Cancellation means the operator dismissing the review panel, with the mouse
and with the keyboard. The test also asserts that no write request was sent.
The test `client-aborted invalid raw submission leaves source and runtime
unchanged` only proves that a client abort leaves no partial write. It does
not prove that the server honours the abort of a valid candidate.

### Restart contract

Every restart in the suite goes through `expectRestartedRuntime`. It passes
only when all four of these hold together:

- the runtime status is `running`
- the PID is greater than 0
- the PID differs from the PID before the restart
- a live `/flows` probe answers 200, or 401 when `adminAuth` is enabled and
  the probe carries no token

A stopped runtime, an unavailable status (PID 0), the previous process or an
unreachable listener can never satisfy it.

## Skipped cases and environment limitations

- **No stack outside CI.** When the stack does not answer, `stack.spec.ts`
  skips with the reason `Real-stack acceptance unavailable: <cause>`. In CI
  the same condition fails the suite (`Real-stack acceptance requires a live
  stack`), so a missing stack can never report success.
- **FlowFuse.** `FlowFuse policy protects deployed HTTP and Socket.IO
  endpoints` needs the isolated `docker-compose.e2e.yml` stack on port 31880.
  Locally it skips with an explicit reason. CI runs it in its own step.
- **Environment problems are not product failures.** When a local run cannot
  start Chromium or Docker, or exhausts a sandbox process or thread budget,
  report the check as not run and name the limitation. Never report it as
  passed, and never claim the product failed.

## Retained evidence

Both Playwright configurations keep traces of failed tests
(`trace: 'retain-on-failure'`) and write an HTML report. CI uploads them as
workflow artifacts, kept for 14 days:

| Artifact | Workflow | Contents |
|----------|----------|----------|
| `playwright-fixture-e2e-node-<version>` | `pr.yml`, job Frontend | Mocked-fixture report and traces |
| `playwright-stack-acceptance-default` | `acceptance.yml` | Real-runtime report and traces, default stack |
| `playwright-stack-acceptance-flowfuse` | `acceptance.yml` | Real-runtime report and traces, FlowFuse stack |

## Gates before publication

`Docker stack acceptance (#434)` is a required status check on `main`, next to
the Frontend, Backend, Lint, govulncheck and pnpm audit checks. A change cannot
merge into `main` unless the real-runtime scenarios above pass on its exact
head.

## Running the real-runtime suite locally

```sh
NRCC_ENCRYPTION_KEY=acceptance-fixture-encryption-key-not-a-secret docker compose build
NRCC_ENCRYPTION_KEY=acceptance-fixture-encryption-key-not-a-secret docker compose up -d --wait
cd frontend
E2E_BASE_URL=http://127.0.0.1:3001 E2E_NODE_RED_URL=http://127.0.0.1:1880 \
  pnpm exec playwright test --config playwright.stack.config.ts
```

The fixture key is a well-known test value for a loopback-only stack. Never
reuse it in a real deployment. Tear the stack down with
`docker compose down -v --remove-orphans`.
