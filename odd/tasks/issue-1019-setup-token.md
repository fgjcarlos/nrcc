# Issues #1019 and #1023: Setup token

Base: `origin/main`; feature branch: `fix/1019-require-setup-token`.

## Specs

- S1: “The first admin can only be created by someone who can read the setup token from the host.” “Setup without the token returns 401 on a fresh data directory.”
- S2: “Require the same token for the first setup. Print it once in the startup log and document `docker compose logs nrcc` as the place to find it.”
- S3: “The acceptance suite reads the token from the container log and completes setup.”
- S4: “`docs/setup-recovery.md` and the README describe the step.”
- S5: “Fix together with #1023 (setup token stays valid after first setup).” “The setup token is single-use, and recovery needs a deliberate operator action on the host.” “After first setup the token file is absent.” “A setup request carrying the old token is rejected.”

## Tasks

- T1 — S1–S2, S5 — Backend token lifecycle and startup — worker + independent verification — done, `cbe0cc7`. Risk: item 2 (authentication).
- T2 — S1–S2 — First-run UI and API client — worker + independent verification — done, `4ca2596`. Risk: item 2 (authentication).
- T3 — S2–S5 — Container/Playwright acceptance and operator documentation — worker + independent verification — done, `09de796`. Risk: item 5 (acceptance workflow).

## Log

- L1: Resuelve la issue #1019
- L2: Both linked issues are `status:approved`; #1019 explicitly asks to fix #1023 together. Started on current `origin/main` (`e048a82`).
- L3: Backend negative checks failed before the fix; afterward targeted Go checks and setup race tests passed. First-run token is consumed before account mutation. The recovery runbook provisions a fresh token deliberately on the host; pre-fix configured installations must revoke any leftover token.
- L4: Frontend regression initially failed on missing field/header. Frontend unit suite: 739 passed, 1 skipped; fixture Playwright smoke: 8 passed.
- L5: Isolated live Docker acceptance: all 8 scenarios passed; real-stack Playwright: 13 default and 1 FlowFuse passed. Static shell, workflow YAML, Compose config, OpenAPI generation, TypeScript, and disposable recovery-file checks passed. GitHub-hosted CI was not run.
- L6: Full `go test ./...` exited 1 on `TestSaveConfig_WritesExplicitRuntimeSettingsAndRestarts`; independent verifier reproduced the same Node-RED capability failure on baseline `e048a82`. No change to that unrelated path.
- L7: Review slice boundary after T2 (`4ca2596`, 300 authored changed lines since base); T3 is a separate 174-line work unit. No push or PR.
