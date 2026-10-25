# Security-surface partial patch — issue #879

## Objective and baseline

`POST /api/config/apply` must accept a **partial patch** for the Node-RED
authentication surfaces. The security boundary cards (admin auth, HTTP node
basic auth, HTTP static basic auth) send only the surface they own, but the
backend validated the payload as a complete replacement: `port`, `uiPort`,
`httpAdminRoot` and `httpNodeRoot` arrived as zero/empty and the apply was
rejected with `400 VALIDATION_ERROR: port must be between 1 and 65535`.
The mocked Playwright spec reported success, so only a real API call exposed
the defect.

Issue: https://github.com/fgjcarlos/nrcc/issues/879
Pull request: https://github.com/fgjcarlos/nrcc/pull/923
Branch: `fix/879-security-partial-patch`
Base: `origin/main`. The branch was rebased onto `7593c9d` (the merge of
#922, which landed while the slice was open) so the pull request carries
only this slice. `main` was not rewritten.

## Authorized policy and constraints

- The payload of `POST /api/config/apply` is a presence-aware **patch**:
  - a field absent from the payload keeps its live value;
  - a field present with a JSON object merges key by key (so replacing the
    `users` of `adminAuth` keeps the auth type and the session expiry);
  - a field present with any other value, `null` included, replaces the live
    value (`adminAuth: null` disables the admin surface);
  - arrays always replace (the `users` list is owned by the patch).
- `port` and `uiPort` stay in sync: the renderer writes the editor port as
  `uiPort` and the typed model derives `port` from it, so a patch that moves
  `uiPort` moves `port` with it unless the patch carries `port` explicitly.
- Secrets keep their stored value when the payload carries an empty string
  or a redaction marker (`[redacted]`, `[REDACTED]`, `********`); the literal
  `false` still disables credential encryption.
- The merge must happen **inside** the apply coordinator's single-flight and
  revision guards and inside the JSON store update lock.
- `POST /api/config` keeps its full-replacement semantics: the configuration
  editor posts the whole configuration, and changing that contract would
  silently change how the editor clears a surface.
- Companion fix (required by the live acceptance): the renderer appended a
  property before the closing brace without comma-terminating the property
  that preceded it. The Docker image ships a `settings.js` whose last
  property (`externalModules: {}`) carries no trailing comma, so every
  structured apply against the canonical deployment rendered invalid
  JavaScript and failed the candidate parse. The defect predates this
  issue — the full-replacement path renders the same invalid file — but it
  blocks the security patch on the canonical deployment.
- No new endpoint, no frontend redesign, no manual edits to generated files,
  no merge. The maintainer merges the PR.
- No plaintext credential in any committed fixture; the acceptance harness
  keeps using the existing `BOOTSTRAP_PASS` / acceptance fixture conventions.

## Delivery and routing

One work-unit per commit, Conventional Commits, each under the 400-line
review budget. Backend semantics and their regressions land first (they make
the real API accept the patch), then the live acceptance scenario, then the
documentation.

- T1 — Backend patch semantics (model patch type, service merge, commit,
  coordinator, handler) + Go regressions.
- T2 — Live acceptance: real API partial apply against the running stack plus
  a live Node-RED authentication probe.
- T3 — Documentation (operator handbook + task file) and delivery.

## Tasks

- [x] T1 — Backend partial-patch semantics. Status: verified.
  - `model.ConfigPatch` keeps the raw JSON of every field so the merge can
    tell "absent" from "explicitly zero" and "explicitly null".
  - `service.MergePatch` merges the patch onto the live configuration.
  - `service.ConfigService.CommitStructuredPatch` merges under the store lock
    and publishes the result through the existing commit pipeline.
  - The apply coordinator previews the merged candidate so the runtime
    adapter resolves the right port, and commits the patch after the
    revision precondition.
  - `POST /api/config/apply` decodes the body into a patch.
  - Regressions: the issue reproduction (admin-auth-only patch applies),
    independent surfaces, explicit disable, unchanged secrets, and the
    merge unit table.
  - RED evidence: with the implementation stashed,
    `go test ./internal/handler -run TestReleaseAuditSecuritySurfacePatchMustApply -count=1`
    fails with `HTTP 400 VALIDATION_ERROR: port must be between 1 and 65535`
    — the defect the issue reports. GREEN with the implementation.
- [x] T2 — Live acceptance in the existing Docker harness. Status: verified.
  - `scripts/acceptance/docker-stacks.sh` scenario 6 applies an admin-auth-only
    patch through the real API on the running stack, asserts the unrelated
    settings survived, and asserts the live Node-RED enforces the patched
    credentials (token exchange 200, wrong password 403, editor API 401
    without a token and 200 with one).
  - The scenario skips itself in `DRY_RUN=1` mode like the other scenarios.
- [x] T3 — Documentation and delivery. Status: verified.
  - Operator handbook documents the patch contract of the apply endpoint.
  - Task file mirrored with the evidence.
  - PR opened against `main`; CI awaited; the maintainer merges.

## Acceptance criteria

- An admin-auth-only patch returns HTTP 200 through the real apply API and
  keeps every unrelated setting.
- `adminAuth`, `httpNodeAuth` and `httpStaticAuth` can be saved independently.
- An absent field is preserved, an explicit `null` disables the surface, and
  an unchanged secret keeps the stored value.
- A live Node-RED instance enforces the credentials the patch installed.

## Evidence

- Baseline (before the change): `go build ./...` OK;
  `go test ./internal/... -count=1` — all packages ok
  (`handler` 210.9s, `service` 341.3s, `server` 41.4s).
- RED: the issue reproduction fails on the baseline with
  `VALIDATION_ERROR: port must be between 1 and 65535`.
- GREEN: `go test ./internal/handler -run "TestReleaseAuditSecuritySurfacePatchMustApply|TestApplyStructuredHandler" -count=1` — ok;
  `go test ./internal/service -run "TestMergePatch|TestCommitStructuredPatch|TestSave_" -count=1` — ok.
- Companion renderer fix: `go test ./internal/service -run TestSave_RendersValidJavaScriptWhenLastPropertyHasNoComma -count=1` — ok.
- Full suite: `go test ./... -count=1` — see the PR test plan.
- Live acceptance (real Docker stacks, local run):
  `bash scripts/acceptance/docker-stacks.sh` — ALL SCENARIOS PASSED,
  including scenario 6 (patch applied through the real API, live Node-RED
  token exchange 200, wrong password 403, editor API 401/200).
- `shellcheck scripts/acceptance/docker-stacks.sh` — clean.
- `go run ./tools/langscan .` — 0 findings.
- `go run ./scripts/cite_check ./docs` — no broken internal links.
- `DRY_RUN=1 bash scripts/acceptance/docker-stacks.sh` — all scenarios pass.
- Pull request #923: all 18 checks green —
  `Backend (Go 1.27.3)`, `Frontend (Node 22)`, `Lint (golangci-lint)`,
  `OpenAPI Spec`, `Docker stack acceptance (#434)`, `Security`
  (`govulncheck`, `pnpm audit`, `Trivy container scan`, `GitGuardian`),
  `Language policy scan`, `i18n catalog coverage`,
  `Hardcoded copy guard`, `Pages content + link integrity`, `CodeQL`
  and the three CodeQL analyses.
- The pull request is mergeable and waits for the maintainer's review and
  merge.

## Next step

The maintainer reviews and merges pull request #923.
