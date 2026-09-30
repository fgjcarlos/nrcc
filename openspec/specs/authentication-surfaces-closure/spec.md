# Authentication Surfaces Closure — Specification

## Purpose

Closes the remaining 🟡 AMBER gap inside cluster **#760 (Authentication surfaces)** by
locking the two missing named acceptance tests into the project's contract set. This
spec is a **test-closure capability** — its only deliverables are two named tests
that already appear in `odd/audits/issue-765/gap-report.md` Gap G2 as the remediation
contract for #760:

1. `TestLegacyAuthFieldMigration` — Go unit test in `internal/service/auth_surfaces_test.go`
   covering the legacy NRCC settings field name → canonical Node-RED 5.x surface migration
   (legacy `adminAuth` / `httpAuth` / static-auth alias → canonical `adminAuth` /
   `httpNodeAuth` / `httpStaticAuth`).
2. `SecuritySurfaceIsolationE2E` — Playwright spec in `frontend/e2e/security-center.spec.ts`
   that proves each Node-RED authentication surface (NRCC admin UI, Node-RED editor,
   HTTP node endpoints, static resources) protects **only** its declared surface, with
   separate credentials and no cross-surface access.

The spec inherits the auth-surface invariants already captured in
`openspec/specs/user-update-api/spec.md` (PASSWORD field separation, role enum
discipline) and `openspec/specs/user-role-editing/spec.md` (last-admin protection),
which are the upstream contracts those two tests are expected to honour.

This change is delivered as **slice 1** of a chained follow-up: the Go unit test lands
first to keep the slice under the 400-line review budget; the Playwright e2e is deferred
to a chained slice 2 PR once slice 1 is 🟢 GREEN on `origin/main`. The chained follow-up
is recorded here as a dependency rather than as in-scope work, because the second test
requires the dashboard stack fixture and is heavier than the 400-line budget when paired
with slice 1.

## Requirements

### Requirement: Legacy Auth Field Migration Support

The configuration parser / renderer that produces `settings.js` for the supervised
Node-RED process MUST accept legacy field names (`adminAuth`, `httpAuth`, static-auth
alias) and emit the canonical Node-RED 5.x surface keys (`adminAuth`, `httpNodeAuth`,
`httpStaticAuth`). The migration MUST preserve credential material (username, bcrypt
hash) byte-for-byte; it MUST NOT silently drop the `pass` field, rewrite the hash,
or alter `permissions`. When the parser encounters a legacy `httpAuth` field, it
MUST route the value to `HTTPNodeAuth` (not `HTTPStaticAuth`); when it encounters a
legacy static-auth alias, it MUST route the value to `HTTPStaticAuth` (not
`HTTPNodeAuth`). The round-trip MUST be stable: parse → render → parse yields a
struct equal to the post-parse struct of the original.

#### Scenario: Legacy `httpAuth` migrates to `httpNodeAuth`

- GIVEN a `settings.js` source using the legacy `httpAuth: { user: "nodes", pass: <bcrypt> }` field
- WHEN the parser parses the source
- THEN `HTTPNodeAuth` is populated with the same username and the same bcrypt hash
- AND `HTTPStaticAuth` is `nil`
- AND the rendered output uses the canonical `httpNodeAuth` key (not `httpAuth`)

#### Scenario: Legacy `httpStaticAuth` alias migrates to `httpStaticAuth`

- GIVEN a `settings.js` source using the static-auth alias (`nodeHttpAuth` or `staticAuth`)
- WHEN the parser parses the source
- THEN `HTTPStaticAuth` is populated with the username and bcrypt hash
- AND `HTTPNodeAuth` is `nil`
- AND the rendered output uses the canonical `httpStaticAuth` key

#### Scenario: Legacy `adminAuth` single user round-trip

- GIVEN a `settings.js` source with a legacy single-user `adminAuth` block
- WHEN the parser parses and re-renders the source
- THEN the rendered output contains `adminAuth:` with the original username and bcrypt hash
- AND `AdminAuth.Users` has exactly one entry
- AND the bcrypt hash string is byte-identical to the input

#### Scenario: Legacy `adminAuth` multi-user round-trip

- GIVEN a `settings.js` source with a legacy multi-user `adminAuth` block (≥ 2 users, distinct permissions)
- WHEN the parser parses and re-renders the source
- THEN the rendered output contains each user in order
- AND each `permissions` value is preserved
- AND `AdminAuth.Users` length matches the input
- AND a second parse of the rendered output produces a struct equal to the first parse

#### Scenario: Migration never re-renders a non-canonical alias

- GIVEN any legacy field name above is present in the input
- WHEN the renderer produces the output
- THEN the output MUST NOT contain `httpAuth`, `nodeHttpAuth`, or `staticAuth`
- AND the output MUST NOT contain plaintext credentials

### Requirement: Authentication Surface Isolation

The runtime MUST enforce that each Node-RED authentication surface accepts **only**
the credentials explicitly assigned to that surface. NRCC admin credentials MUST
NOT grant access to the Node-RED editor; HTTP-node credentials MUST NOT grant
access to static resources; and conversely each surface MUST NOT accept
credentials bound to a different surface. A request presenting mismatched
credentials to a surface MUST be rejected with HTTP 401. The runtime MUST NOT
allow a single credential set to be configured against two surfaces simultaneously
without an explicit operator override.

#### Scenario: NRCC admin credentials do not grant editor access

- GIVEN an NRCC admin user `admin` exists with password `pw-admin`
- WHEN a login is attempted against the Node-RED editor endpoint with `admin` / `pw-admin`
- THEN the editor login is rejected with HTTP 401
- AND the Node-RED editor session cookie is NOT issued

#### Scenario: HTTP-node credentials do not grant static-resource access

- GIVEN `httpNodeAuth` is configured with `nodes` / `<bcrypt>`
- WHEN a request for a static asset is made presenting `nodes` / `<bcrypt>`
- THEN the static-resource gate rejects the request with HTTP 401
- AND the static asset body is NOT returned

#### Scenario: Static-resource credentials do not grant HTTP-node access

- GIVEN `httpStaticAuth` is configured with `static` / `<bcrypt>`
- WHEN a request to a protected HTTP-node endpoint is made presenting `static` / `<bcrypt>`
- THEN the HTTP-node gate rejects the request with HTTP 401
- AND the protected endpoint body is NOT returned

#### Scenario: A single credential set bound to two surfaces is rejected at apply time

- GIVEN an operator attempts to apply a `settings.js` where the same username / hash appears under both `httpNodeAuth` and `httpStaticAuth`
- WHEN the apply pipeline validates the configuration
- THEN the apply is rejected with a machine-readable error code distinct from generic validation failures
- AND the existing on-disk configuration is NOT overwritten

### Requirement: `TestLegacyAuthFieldMigration` Verifies the Migration Contract

The Go test `TestLegacyAuthFieldMigration` MUST exist in
`internal/service/auth_surfaces_test.go`. It MUST be a table-driven test (per
`openspec/config.yaml` `testing.capabilities.backend` table-driven tests where
useful) with at least four sub-cases covering the four scenarios above
(legacy `httpAuth` → `httpNodeAuth`, legacy static-auth alias → `httpStaticAuth`,
legacy `adminAuth` single user, legacy `adminAuth` multi-user). The test MUST be
self-contained (no network, no temp config files written outside `t.TempDir()`,
no real bcrypt rounds), MUST use the existing `testBcryptHash` constant for
fixture credentials (never plaintext), and MUST be runnable under
`go test -race -count=1 -timeout 20m ./internal/service/...` with the race
detector clean and zero data-race reports.

#### Scenario: The test is RED on a missing migration

- GIVEN the parser does NOT perform the legacy → canonical migration
- WHEN `go test -race -count=1 -timeout 20m ./internal/service/...` runs
- THEN `TestLegacyAuthFieldMigration` fails with a clear diff message naming the missing canonical key
- AND the failure is the strict-TDD RED signal required before the GREEN commit

#### Scenario: The test is GREEN once the migration lands

- GIVEN the parser performs the legacy → canonical migration per the contract
- WHEN `go test -race -count=1 -timeout 20m ./internal/service/...` runs
- THEN `TestLegacyAuthFieldMigration` passes
- AND all existing tests in `internal/service/...` continue to pass
- AND the race detector reports zero races

#### Scenario: `golangci-lint` is clean

- GIVEN the new test file is added
- WHEN `golangci-lint run --config=.golangci.yml ./...` runs
- THEN the result has zero findings (the lint gate is repo-wide, not ratcheted, per `openspec/config.yaml` `testing.runners.backend_lint.ci_gate`)

### Requirement: `SecuritySurfaceIsolationE2E` Verifies the Isolation Contract

The Playwright spec `SecuritySurfaceIsolationE2E` MUST exist in
`frontend/e2e/security-center.spec.ts`. It MUST exercise the four scenarios
above (admin ≠ editor, HTTP-node ≠ static-resource, static-resource ≠ HTTP-node,
cross-surface apply-time rejection) against the running stack
(`pnpm --filter frontend test:e2e:stack`, Chromium only per
`openspec/config.yaml` `testing.runners.frontend_e2e.install_browsers`). The spec
MUST follow the existing `frontend/e2e/auth.spec.ts` fixture pattern, MUST use
distinct bcrypt-hashed fixture credentials per surface (never plaintext), MUST
assert HTTP 401 explicitly (not merely "no success"), and MUST NOT depend on
timing-sensitive assertions that flake under `-race`.

#### Scenario: Cross-surface credentials are rejected with 401

- GIVEN the stack is running with NRCC admin, HTTP-node, and static-resource surfaces each configured with distinct bcrypt-hashed credentials
- WHEN `SecuritySurfaceIsolationE2E` runs against `pnpm --filter frontend test:e2e:stack`
- THEN every cross-surface login attempt is rejected with HTTP 401
- AND no protected session cookie is issued for the wrong surface

#### Scenario: Apply-time cross-surface binding is rejected

- GIVEN the operator UI is open and an apply attempt would bind the same credentials to two surfaces
- WHEN the apply is submitted
- THEN the apply UI surfaces a machine-readable error code
- AND the on-disk configuration is unchanged

#### Scenario: The spec runs as part of the e2e suite

- GIVEN `frontend/e2e/security-center.spec.ts` exists with `SecuritySurfaceIsolationE2E`
- WHEN `pnpm --filter frontend test:e2e` runs (Chromium only)
- THEN the spec is discovered and executed
- AND no other e2e spec regresses because of shared fixture state

### Requirement: Strict TDD Ordering

Per `openspec/config.yaml` `strict_tdd.requires_failing_test_first: true` and the
RED → GREEN → REFACTOR discipline in the same block, the slice 1 work-unit commit
MUST land the failing `TestLegacyAuthFieldMigration` against the current parser
first; the parser fix (if required) MUST land in a separate, immediately following
work-unit commit; any REFACTOR pass MUST NOT move behaviour and MUST NOT add or
remove test cases. If the parser already implements the migration, the RED commit
is replaced by a single GREEN commit that adds the test — no parser change is
required. Slice 2 (the Playwright spec) is **not** part of this slice and MUST NOT
be added to the slice 1 work-unit commit under any circumstance.

#### Scenario: RED commit precedes GREEN commit

- GIVEN the parser does NOT yet implement the migration
- WHEN slice 1 lands
- THEN `git log` shows a RED commit (failing test only, no parser change) immediately followed by a GREEN commit (parser change making the test pass)
- AND no GREEN commit ever precedes a RED commit for the same test

#### Scenario: Slice 2 is deferred

- GIVEN slice 1 closes cluster #760 to 🟢 GREEN on the Go half
- WHEN the slice 1 PR is merged
- THEN `frontend/e2e/security-center.spec.ts` is NOT touched in the slice 1 commits
- AND a follow-up issue tracks `SecuritySurfaceIsolationE2E` as slice 2

### Requirement: Cluster #760 Closure Evidence

The slice MUST produce evidence that cluster #760's 🟡 AMBER status in
`odd/audits/issue-765/gap-report.md` Gap G2 is partially closed (slice 1 covers
the Go half; slice 2 will close the Playwright half). The PR body MUST reference
Gap G2 and MUST list the exact test names added. The PR MUST close against the
issue that gap-report references for #760.

#### Scenario: PR body references Gap G2 and the new test

- GIVEN the slice 1 PR is opened
- WHEN the PR body is reviewed
- THEN it contains a `Closes #N` (or equivalent) reference that resolves to the #760 cluster issue
- AND the body explicitly names `TestLegacyAuthFieldMigration`
- AND the body explicitly names `SecuritySurfaceIsolationE2E` as a **deferred** follow-up (not as slice 1 scope)

## Acceptance Scenarios

The slice 1 PR is acceptable when **all** of the following hold:

- [ ] `internal/service/auth_surfaces_test.go` contains `TestLegacyAuthFieldMigration` as a table-driven test with at least four sub-cases.
- [ ] `go test -race -count=1 -timeout 20m ./internal/service/...` passes with the race detector clean.
- [ ] `golangci-lint run --config=.golangci.yml ./...` reports zero findings.
- [ ] `git diff --stat` shows ≤ 400 authored lines added across the slice 1 commits.
- [ ] The slice 1 commits follow the RED → GREEN → REFACTOR strict-TDD discipline (or single GREEN commit if the parser already implements the migration).
- [ ] The slice 1 PR body carries `Closes #N` for the #760 cluster issue plus a `type:test` label per the `branch-pr` skill, and explicitly defers `SecuritySurfaceIsolationE2E` to a chained follow-up.
- [ ] No plaintext credentials appear in any test fixture; only the existing `testBcryptHash` constant (or a parallel hash constant) is used.
- [ ] No existing test in `internal/service/...` regresses because of the parser fix.
- [ ] `odd/audits/issue-765/gap-report.md` is **not** rewritten by this slice; only Gap G2 row is referenced from the PR description.

## Out-of-scope

The following are explicitly **not** delivered by this capability:

- **Behavioural changes to authentication policy.** This is a test-closure capability, not a feature. The parser fix, if required, is the minimal correctness change to make the legacy migration observable; no new auth surface is introduced, no surface-binding policy is tightened beyond what already exists in `internal/service/auth_surfaces.go`.
- **`SecuritySurfaceIsolationE2E` Playwright spec implementation.** Deferred to chained slice 2. The spec name is referenced here as a dependency and as the closure target, but the file `frontend/e2e/security-center.spec.ts` is **not** authored by slice 1.
- **Modifications to the five existing capability specs.** `dashboard-runtime-metrics`, `metrics-endpoint`, `user-management-ui`, `user-update-api`, and `user-role-editing` are read-only inputs to this capability. Their drift (e.g., the `metrics-endpoint` #671 stale text flagged in `openspec/changes/issue-760-security-surface-tests/explore.md`) is **not** fixed by this slice.
- **`design.md` and `odd/tasks/issue-760-security-surface-tests.md`.** Both belong to later SDD phases (`design`, `plan`). They MUST NOT be created in this spec phase.
- **Source-code behaviour changes outside the parser migration path.** No handler changes, no middleware changes, no settings-persistence changes beyond what the legacy-field migration strictly requires.
- **`CompatibilityModeE2E` and other compatibility-mode follow-ups.** Tracked separately in `odd/audits/issue-765/gap-report.md` §3; out of scope for #760 closure.
- **Maintainer-level security review attestation for cluster #764.** Tracked separately as Gap G1; out of scope here.
- **Any new CI workflow, branch-protection change, or `openspec/config.yaml` rewrite.** Configuration is the parent preflight's authority; slice 1 only consumes it.

## Dependencies

### Upstream contracts (inherited invariants)

- **`openspec/specs/user-update-api/spec.md`** — establishes the password/role
  endpoint separation that the migration contract MUST respect (password changes
  go through `PATCH /api/auth/users/{id}/password`, never the role endpoint). The
  `TestLegacyAuthFieldMigration` parser contract MUST NOT collapse `adminAuth`
  credentials into a shape that would violate the password-field-separation rule.
- **`openspec/specs/user-role-editing/spec.md`** — establishes the last-admin
  invariant (`AdminAuth.Users` must always contain at least one user with
  `permissions: "*"` after a successful render). The migration MUST NOT drop
  the last-admin user, MUST NOT rewrite `permissions`, and MUST preserve the
  user count on round-trip.

### Tooling / CI gates (per `openspec/config.yaml`)

- `go test -race -count=1 -timeout 20m ./internal/service/...` — required at `implement`, `verify`.
- `golangci-lint run --config=.golangci.yml ./...` — repo-wide zero-finding gate at `verify`.
- `strict_tdd.requires_failing_test_first: true` — RED before GREEN ordering is mandatory.
- `delivery.review_budget_lines: 400` — slice 1 must remain under budget; slice 2 must be its own PR.
- `conventions.commits: conventional_commits` — slice 1 commits MUST use `test(...)` or `fix(...)` types per the `branch-pr` Conventional Commit regex.
- Linux-only build constraint (`docs/adr/0004-linux-only-build.md`) — slice 1 must compile and run on Linux.

### Deferred follow-up (chained slice 2)

- `SecuritySurfaceIsolationE2E` Playwright spec — opens as a chained PR after slice 1 is merged and 🟢 GREEN on `origin/main`. The orchestrator MUST `ask-on-risk` again at the slice-1 close before promoting slice 2. Slice 2 is **not** a dependency that blocks slice 1; slice 1 closes the Go half of Gap G2 independently.

### Inputs consumed by this spec

- `openspec/changes/issue-760-security-surface-tests/explore.md` — exploration notes from the previous SDD phase; the only authoritative source for the slice 1 plan and slice 2 deferral.
- `odd/audits/issue-765/gap-report.md` Gap G2 — the audit contract that names `TestLegacyAuthFieldMigration` and `SecuritySurfaceIsolationE2E` as the missing items.

## Risks

| Risk | Severity | Mitigation |
| --- | --- | --- |
| Parser already implements the migration, making the RED commit impossible. | 🟡 Medium | The strict-TDD workflow accepts a single GREEN commit if the parser already passes the new test. The RED → GREEN discipline is preserved by recording the test author's intent in the commit message and by treating the test addition as the GREEN signal. |
| Parser fix grows beyond ~60-90 lines and strains the 400-line budget when combined with the test. | 🟡 Medium | Per `openspec/changes/issue-760-security-surface-tests/explore.md` §5 item 6, the parser fix is estimated at 60-90 lines. If the actual diff exceeds 90 lines, the orchestrator MUST `ask-on-risk` (delivery strategy is `ask-on-risk`) before opening the PR; do not invent a chain strategy or an exception. |
| `golangci-lint` repo-wide gate rejects the new test file (unused-import, line-length, revive, gosec). | 🟡 Low | Run `golangci-lint run --config=.golangci.yml ./...` locally before push; the slice 1 commits MUST NOT introduce any new finding. |
| Existing tests in `internal/service/...` regress because the parser fix changes the round-trip shape. | 🟡 Medium | The contract explicitly preserves credential bytes and user counts. Any regression indicates the fix is too aggressive and MUST be narrowed. |
| The follow-up slice 2 is silently pulled into slice 1 by an implementer, blowing the 400-line budget. | 🟡 Medium | The spec's Out-of-scope section and Slice-2 deferral in Dependencies block this explicitly. The orchestrator MUST reject any PR that touches `frontend/e2e/security-center.spec.ts` in slice 1. |
| The legacy field mapping table is misread (e.g., legacy `httpAuth` mapped to `httpStaticAuth` instead of `httpNodeAuth`), baking a contract bug into the test. | 🟠 High | The contract scenarios above pin each legacy field to a single canonical target. The implementer MUST verify the canonical mapping against `internal/service/auth_surfaces.go` and the existing `TestRenderHTTPAuthCanonicalKeys` test before writing the table-driven sub-cases. |
| Plaintext credentials accidentally appear in a fixture. | 🟠 High | The spec mandates the existing `testBcryptHash` constant or a parallel hash constant; plaintext strings MUST NOT appear. `golangci-lint` `gosec` and a manual diff review are the catch-nets. |
| `frontend/e2e/security-center.spec.ts` already exists with partial content from another branch; the deferred slice 2 may collide. | 🟡 Low | Slice 1 MUST NOT touch the file; the orchestrator MUST verify the file does not exist on `origin/main` before slice 2 opens. |
| Strict-TDD workflow silently skipped (GREEN without RED) when the parser already implements the migration. | 🟡 Low | The "strict TDD ordering" requirement explicitly allows a single GREEN commit in this case; the PR description MUST document why the RED commit was skipped. |
| `delivery_strategy: ask-on-risk` is bypassed by auto-mode chaining. | 🟡 Low | `phase_rules.skip_conditions.human_gates_always_pause: true` is set; the orchestrator MUST honour `ask-on-risk` triggers (`delivery_total_lines_exceeds_review_budget`, `changes_touch_auth_or_secrets`) at every phase boundary. |