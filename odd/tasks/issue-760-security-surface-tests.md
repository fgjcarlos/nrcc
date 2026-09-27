# Issue #760 — Legacy Auth Field Migration (slice 1, Go half)

## Feature header

| Field | Value |
|---|---|
| Feature name | `legacy-auth-field-migration` (slice 1 of cluster #760) |
| Capability folder | `openspec/specs/authentication-surfaces-closure/` |
| Related issue | Cluster **#760** "Authentication surfaces (Security Center)" (🟡 AMBER) |
| Related audit refs | `odd/audits/issue-765/gap-report.md` Gap G2; `odd/audits/issue-765/roadmap-traceability.md` #760 evidence (2 of 4 named tests present) |
| Spec source | `openspec/specs/authentication-surfaces-closure/spec.md` (6 requirements, 21 scenarios) |
| Design source | `openspec/specs/authentication-surfaces-closure/design.md` (migration map + parser surface notes) |
| Closed precedence (read-only context) | PRs **#795**, **#796**, **#797** (cluster #760 prior slices; do not modify or amend their history) |
| Strict TDD posture | `enabled`, `requires_failing_test_first: true`, `red_green_refactor: true` |
| Delivery strategy | `ask-on-risk` (per `openspec/config.yaml` `delivery.strategy` and `phase_rules.ask_on_risk_triggers`) |
| Review budget | 400 authored lines (`additions + deletions`) per `delivery.review_budget_lines` |
| Branch prefix | `test/issue-760-legacy-auth-field-migration` (matches `branch-pr` regex `^test\/[a-z0-9._-]+$`) |
| PR label (single, mandatory) | `type:test` (per `branch-pr` type-to-label mapping for `test(...)` commits) |
| Work-unit identity | `T1 RED` (failing test) → `T2 GREEN-parser-fix` (conditional, only if RED demonstrates missing migration) → `T3 REFACTOR` (optional) → `T4 EVIDENCE` (commit + push) → `T5 DEFER-SLICE-2` (follow-up issue) |
| Linked PR body contract | `Closes #N` resolving to the **#760 cluster issue**; names `TestLegacyAuthFieldMigration` as slice 1 scope; explicitly defers `SecuritySurfaceIsolationE2E` to chained slice 2 |

> The artifact store for the plan phase in this project is `odd/tasks/<feature>.md` per `openspec/config.yaml` `phase_rules.artifacts_per_phase.plan`. This file is the canonical slice-1 plan artifact.

## Pre-conditions

1. **Sync origin/main before opening the worktree.** Run `git fetch origin main` and verify the tip is what the audit reports; the slice 1 PR must base against `origin/main`, not a stale local checkout (gap-report.md §6 / roadmap-traceability.md §6: local-main was 54 commits behind in the prior audit cycle).
2. **Open a fresh worktree on a dedicated branch.** From a clean `origin/main`, create `test/issue-760-legacy-auth-field-migration`. Do **not** stack slice 1 on top of an existing feature branch — slice 1 is the Go-half closure PR for the #760 cluster and must be mergeable independently.
3. **Linux host required.** `internal/service/auth_surfaces.go` does not exist as a standalone file in this repo; the live parser surface used by the slice is `internal/service/settings_sandbox.go` (sandbox-backed `ParseAuthenticationSurfacesViaSandbox` + `ParseAdminAuthViaSandbox`) and `internal/service/config.go` (`renderSettingsJS`, `patchSettingsJS`, `parseConfigFromContent`). Both paths use POSIX-only assumptions (file replacement, sandbox interruption). Build and test on Linux — see `docs/adr/0004-linux-only-build.md` and `openspec/config.yaml` `testing.capabilities.backend.linux-only_build`.
4. **Authoring posture.** Author the test in English only (`conventions.english_only: true`). Do not modify `odd/audits/issue-765/gap-report.md` — it is a historical audit document and is exempt from rewrite per `conventions.exempt_paths` spirit.

## Work-unit task list

Each task below is a single work-unit commit with a Conventional Commit message (≤ 50 char summary, imperative), the listed edit surface, the verification command and expected result, and the strict-TDD posture for that commit.

### T1 — RED: add `TestLegacyAuthFieldMigration` (failing test only)

- **Work-unit commit message:** `test(auth): cover legacy field migration contract`
- **Posture:** RED-first. No parser change. No production code touched.
- **Allowed edit surfaces:**
  - `internal/service/auth_surfaces_test.go` (append `TestLegacyAuthFieldMigration` and any helper fixtures inline in the file)
- **Test shape (mandatory):**
  - Table-driven via `t.Run(tt.name, ...)` per `openspec/config.yaml` `testing.capabilities.backend.table_driven_tests`.
  - **At least four sub-cases** matching the spec scenarios:
    1. Legacy single-user `adminAuth` round-trip — username, `testBcryptHash`, permissions preserved; `AdminAuth.Users` length = 1.
    2. Legacy multi-user `adminAuth` round-trip — user order, count, each `Permissions` preserved; a second parse of the rendered output yields a struct equal to the first parse.
    3. Legacy `httpAuth` → canonical `httpNodeAuth` — `HTTPNodeAuth` populated with same username and bcrypt hash; `HTTPStaticAuth` is `nil`; rendered output uses `httpNodeAuth` only.
    4. Static-auth alias (`nodeHttpAuth` or `staticAuth`) → canonical `httpStaticAuth` — `HTTPStaticAuth` populated; `HTTPNodeAuth` is `nil`; rendered output uses `httpStaticAuth` only.
  - Assertions also cover the forbidden-alias absence rule from the spec: rendered output MUST NOT contain `httpAuth`, `nodeHttpAuth`, or `staticAuth` and MUST NOT contain plaintext credentials.
  - Fixtures use `testBcryptHash` constant only; never plaintext.
  - Self-contained: no network, no temp config files written outside `t.TempDir()` if any I/O is needed, no real bcrypt rounds.
  - Reuses the existing `parseConfigFromContent`, `renderSettingsJS`, and `patchSettingsJS` helpers from `internal/service/config.go`; no new public helpers.
- **Verification (must FAIL on pre-change parser):**
  - `go test -race -count=1 -timeout 20m ./internal/service/...` — expected: `TestLegacyAuthFieldMigration` fails with a clear diff message naming the missing canonical destination (e.g. `HTTPNodeAuth` nil or `httpAuth` token present in rendered output). All other tests in `./internal/service/...` continue to pass.
  - `golangci-lint run --config=.golangci.yml ./...` — expected: zero findings (the new test file MUST NOT introduce lint debt).
- **Rollback boundary:** revert this single commit; no production code is touched, so rollback is a pure test deletion.
- **Strict-TDD posture:** RED-only. Do not modify `internal/service/config.go`, `internal/service/settings_sandbox.go`, or any production code in this commit.

### T2 — GREEN-parser-fix (CONDITIONAL: skip if T1 already passes)

- **Work-unit commit message:** `fix(auth): migrate legacy auth fields to canonical`
- **Posture:** GREEN-when-parser-correct. Required **only if** T1 RED demonstrates a missing migration. If T1 already passes against the current parser (e.g. legacy aliases are already routed to canonical surfaces via the existing `extractAuthenticationSurfaces` path), record that observation in the PR body and **skip this commit entirely** — a single GREEN test-only commit is then the slice's only code change. The strict-TDD workflow explicitly allows this in `openspec/specs/authentication-surfaces-closure/spec.md` "Strict TDD Ordering".
- **Allowed edit surfaces (when required):**
  - `internal/service/settings_sandbox.go` — add a focused `NormalizeLegacyAuthFields(*model.AuthenticationSurfaces)` (or in-place extension of `extractAuthenticationSurfaces`) that routes legacy keys per the design's migration map:
    | Legacy field | Canonical destination |
    |---|---|
    | `adminAuth` legacy single-user shape | `AdminAuth.Users` (1 entry) |
    | `adminAuth` legacy multi-user shape | `AdminAuth.Users` (preserve order, count, `Permissions`) |
    | `httpAuth` | `HTTPNodeAuth` (NOT `HTTPStaticAuth`) |
    | `nodeHttpAuth`, `staticAuth` | `HTTPStaticAuth` (NOT `HTTPNodeAuth`) |
    | Unrecognised legacy auth keys | Drop without retention in `Extra`, logs, or round-trip output |
  - Per design: canonical destination wins when both canonical and alias are present; the alias is discarded without exposing its value.
  - Errors MUST NOT interpolate usernames, hashes, or raw settings text (the existing `TestHTTPAuthValidationRedactsPasswords` redaction discipline applies).
- **Forbidden edit surfaces:** `internal/service/config.go` (renderer-only string substitution is rejected by the design — render cannot correct the structured config used by validation, apply, and re-parse); handler or middleware code; settings-persistence paths.
- **Verification:**
  - `go test -race -count=1 -timeout 20m ./internal/service/...` — expected: `TestLegacyAuthFieldMigration` passes; all pre-existing tests continue to pass; race detector clean.
  - `golangci-lint run --config=.golangci.yml ./...` — expected: zero findings. If the parser fix introduces any lint debt, narrow the change; the repo-wide zero-finding gate is not ratcheted.
  - `go vet ./...` and `go build ./...` — expected: clean.
- **Rollback boundary:** revert this commit to restore the pre-fix parser behaviour; T1 test still fails because the test and parser revert together.
- **Strict-TDD posture:** GREEN-when-parser-correct. Behaviour change must be minimal; credential bytes, user counts, and `Permissions` values MUST be preserved on round-trip.

### T3 — REFACTOR (OPTIONAL)

- **Work-unit commit message:** `refactor(auth): tighten migration test layout`
- **Posture:** Optional. Only commit if a real readability or naming improvement exists. **No behaviour change, no test-case addition or removal.** The strict-TDD contract forbids behaviour drift in the refactor pass.
- **Allowed edit surfaces:**
  - `internal/service/auth_surfaces_test.go` (table layout, naming, helper extraction)
  - `internal/service/settings_sandbox.go` (local naming, comment wording only) **only if** T2 landed
- **Verification:**
  - `go test -race -count=1 -timeout 20m ./internal/service/...` — expected: identical pass/fail matrix vs T2's GREEN run.
  - `golangci-lint run --config=.golangci.yml ./...` — expected: zero findings.
- **Rollback boundary:** revert this commit; tests and parser behaviour are unchanged.
- **Strict-TDD posture:** Optional REFACTOR. If there is nothing genuinely worth tightening, do not commit; an empty REFACTOR is a wasted work unit.

### T4 — EVIDENCE: prepare and open the slice 1 PR

- **Work-unit commit message:** N/A (this task produces a PR, not a commit).
- **Posture:** Operate only after T1 and (conditionally) T2 are pushed. No source code edits.
- **Allowed edit surfaces:**
  - `git` operations on the `test/issue-760-legacy-auth-field-migration` branch.
  - GitHub PR body composition via `gh pr create`.
- **Pre-PR checklist (mandatory):**
  - `git diff --stat origin/main...test/issue-760-legacy-auth-field-migration` shows **≤ 400 authored lines** (additions + deletions across the slice 1 commits). If the diff exceeds 400 lines, **stop** and `ask-on-risk`; do not invent a chain strategy or a `size:exception`.
  - `git log origin/main..test/issue-760-legacy-auth-field-migration --oneline` shows the RED-first sequence (RED commit immediately followed by GREEN commit, or single GREEN commit when the parser already implements the migration).
  - All commits use Conventional Commit format matching the `branch-pr` regex.
  - No commit message includes a `Co-Authored-By` trailer.
- **PR body content (mandatory):**
  - `Closes #N` resolving to the **#760 cluster issue** (the GitHub issue that gap-report.md §2 Gap G2 references).
  - References `odd/audits/issue-765/gap-report.md` Gap G2 in the body.
  - Names `TestLegacyAuthFieldMigration` as slice 1 scope and lists its sub-cases.
  - Explicitly states that `SecuritySurfaceIsolationE2E` is **deferred to chained slice 2** (not slice 1 scope) and references the slice 2 follow-up issue.
  - `type:test` label applied (single, per `branch-pr` rules).
  - Summary bullets (1–3) and a changes table per `branch-pr` skill format.
- **Verification:** PR is open with the linked issue in `status:approved` state; the `type:test` label is the only `type:*` label; the `Check Issue Reference`, `Check Issue Has status:approved`, and `Check PR Has type:* Label` GitHub Actions jobs pass.

### T5 — DEFER-SLICE-2: open the follow-up issue for `SecuritySurfaceIsolationE2E`

- **Work-unit commit message:** N/A (this task produces an issue, not a commit).
- **Posture:** Operate immediately after T4 PR is opened (or merged — whichever the orchestrator chooses). No source code edits.
- **Allowed edit surfaces:**
  - GitHub issue creation via `gh issue create`.
- **Issue body content (mandatory):**
  - Title naming `SecuritySurfaceIsolationE2E` and chained slice 2 of cluster #760.
  - Cross-reference to the slice 1 PR and `odd/audits/issue-765/gap-report.md` Gap G2.
  - Scope explicitly: Playwright spec in `frontend/e2e/security-center.spec.ts` covering the four isolation scenarios (admin ≠ editor, HTTP-node ≠ static-resource, static-resource ≠ HTTP-node, cross-surface apply-time rejection).
  - Explicit dependency on slice 1 being 🟢 GREEN on `origin/main`.
  - Reaffirms the 400-line review budget per slice.
- **Verification:** follow-up issue exists with `status:approved` label (or `status:proposed` if the maintainer hasn't approved yet — track approval before opening the slice 2 PR).

## Acceptance checklist

Mirrors `openspec/specs/authentication-surfaces-closure/spec.md` "Acceptance Scenarios":

- [ ] `internal/service/auth_surfaces_test.go` contains `TestLegacyAuthFieldMigration` as a table-driven test with **at least four** named `t.Run` sub-cases.
- [ ] `go test -race -count=1 -timeout 20m ./internal/service/...` passes with the race detector clean (zero races).
- [ ] `golangci-lint run --config=.golangci.yml ./...` reports **zero findings**.
- [ ] `git diff --stat origin/main...test/issue-760-legacy-auth-field-migration` shows **≤ 400 authored lines** across the slice 1 commits.
- [ ] Slice 1 commits follow RED → GREEN → REFACTOR strict-TDD discipline, **or** a single GREEN commit when the parser already implements the migration (rationale recorded in PR body).
- [ ] The slice 1 PR body carries `Closes #N` for the #760 cluster issue plus exactly one `type:test` label per the `branch-pr` skill, and explicitly defers `SecuritySurfaceIsolationE2E` to chained slice 2.
- [ ] No plaintext credentials appear in any test fixture; only `testBcryptHash` (or a parallel bcrypt-hash constant) is used.
- [ ] No existing test in `internal/service/...` regresses because of the parser fix (verify the full `./internal/service/...` suite passes).
- [ ] `odd/audits/issue-765/gap-report.md` is **not** rewritten by this slice; only Gap G2 row is referenced from the PR description.

## Risks

| Risk | Severity | Mitigation |
|---|---|---|
| Linux-only build (`internal/service/settings_sandbox.go` POSIX assumptions + `internal/service/config.go` file replacement) | 🟡 Medium | Run all verification commands on Linux. Document the constraint in the PR body. Do not attempt cross-platform CI matrix. |
| Repo-wide zero-finding `golangci-lint` gate (`openspec/config.yaml` `testing.runners.backend_lint.ci_gate`) | 🟡 Low | Run `golangci-lint run --config=.golangci.yml ./...` locally before push. Fix only findings introduced by this slice; do not chase pre-existing debt. |
| Fork-PR CI gate (`safe-to-test` requirement) | 🟡 Low | Push from a fork that satisfies the repo's `safe-to-test` policy; if running as an outside contributor, confirm with the maintainer. Do not modify any CI workflow in this slice. |
| Strict-TDD RED may be impossible if the parser already implements the migration | 🟡 Medium | The strict-TDD workflow accepts a single GREEN commit when the parser already satisfies the new test. Document the observed pre-change behaviour in the PR body (`spec.md` "Strict TDD Ordering" allows this). Do not manufacture a failing commit. |
| Parser fix grows beyond the 60–90 line estimate (`openspec/changes/issue-760-security-surface-tests/explore.md` §5 item 6) and strains the 400-line budget | 🟡 Medium | `ask-on-risk` if the actual diff exceeds ~90 lines of production change combined with the test file. Do not invent a chain strategy; honour the parent's `ask-on-risk` delivery setting. |
| Plaintext credentials accidentally land in a fixture | 🟠 High | Use `testBcryptHash` only; never write plaintext. `golangci-lint gosec` and a manual diff review are the catch-nets. Spec mandates bcrypt-only fixtures. |
| Slice 2 (`SecuritySurfaceIsolationE2E`) is silently pulled into slice 1 by an implementer | 🟡 Medium | Slice 1 MUST NOT touch `frontend/e2e/security-center.spec.ts`. The orchestrator rejects any PR that adds Playwright content to slice 1. T5 opens the explicit deferral issue. |
| `extractAuthenticationSurfaces` change inadvertently alters credential bytes, user counts, or `Permissions` values | 🟠 High | The spec contract forbids byte-rewriting and permission drift. Any regression indicates the fix is too aggressive; narrow the change to canonical-routing only. |
| `delivery_strategy: ask-on-risk` bypassed by auto-mode chaining | 🟡 Low | `phase_rules.skip_conditions.human_gates_always_pause: true`; the orchestrator honours `ask-on-risk` triggers (`changes_touch_auth_or_secrets`, `delivery_total_lines_exceeds_review_budget`) at every phase boundary. |
| Local checkout falls behind `origin/main` (methodology gap observed in `gap-report.md` §6) | 🟡 Low | First action is `git fetch origin main`; base slice 1 on `origin/main`, not on the previous slice's branch tip. |

## Out of scope

- **`SecuritySurfaceIsolationE2E` Playwright spec.** Deferred to chained slice 2 (see T5). The file `frontend/e2e/security-center.spec.ts` MUST NOT be authored or modified by this slice. Slice 2 also MUST NOT be bundled into the slice 1 PR to fit or exceed the 400-line budget.
- **Modifications to the five existing capability specs.** `openspec/specs/dashboard-runtime-metrics/`, `openspec/specs/metrics-endpoint/`, `openspec/specs/user-management-ui/`, `openspec/specs/user-update-api/`, and `openspec/specs/user-role-editing/` are read-only inputs to this capability. Drift in those specs (e.g. the `metrics-endpoint` #671 stale text flagged in `openspec/changes/issue-760-security-surface-tests/explore.md`) is **not** fixed here.
- **`openspec/specs/authentication-surfaces-closure/design.md` rewrites.** The design was authored in an earlier SDD phase; slice 1 consumes it, it does not amend it.
- **Behavioural changes to authentication policy.** This is a test-closure capability, not a feature. The parser fix, if required, is the **minimal correctness change** to make the legacy migration observable; no new auth surface is introduced, no surface-binding policy is tightened beyond what already exists in `internal/service/settings_sandbox.go` and `internal/service/config.go`.
- **Handler, middleware, or settings-persistence changes** beyond what the parser's legacy-field migration strictly requires. No handler rewrites; no middleware edits; no persistence-layer changes.
- **Source-code edits to `odd/audits/issue-765/gap-report.md` or `odd/audits/issue-765/roadmap-traceability.md`.** Both are historical audit documents; slice 1 only references Gap G2 from the PR description.
- **`CompatibilityModeE2E`** and other compatibility-mode follow-ups. Tracked separately in `odd/audits/issue-765/gap-report.md` §3.
- **Maintainer-level security review attestation for cluster #764** (Gap G1). Tracked separately.
- **Any new CI workflow, branch-protection change, or `openspec/config.yaml` rewrite.** Configuration is the parent preflight's authority; slice 1 only consumes it.

## Verification commands

Run from the repository root, on Linux, after `git fetch origin main`:

```bash
# Focused Go service suite with race detector — required at implement and verify phases
go test -race -count=1 -timeout 20m ./internal/service/...

# Repo-wide lint gate — required at verify phase; must report zero findings
golangci-lint run --config=.golangci.yml ./...

# Static analysis — required at verify phase
go vet ./...

# Build smoke — required at verify phase
go build ./...

# Review budget check — required before opening the PR
git diff --stat origin/main...test/issue-760-legacy-auth-field-migration

# Commit-shape check — required before opening the PR
git log origin/main..test/issue-760-legacy-auth-field-migration --oneline
```

Expected outcomes for a clean slice 1 run:

| Command | RED (T1) | GREEN (T2) | Post-PR (T4) |
|---|---|---|---|
| `go test -race -count=1 -timeout 20m ./internal/service/...` | `TestLegacyAuthFieldMigration` fails; other tests pass; zero races | `TestLegacyAuthFieldMigration` passes; all tests pass; zero races | All tests pass; zero races |
| `golangci-lint run --config=.golangci.yml ./...` | Zero findings | Zero findings | Zero findings |
| `go vet ./...` | Clean | Clean | Clean |
| `go build ./...` | Clean | Clean | Clean |
| `git diff --stat origin/main...HEAD` | Test-file delta only (≤ ~120 lines expected) | Test + parser-fix delta (≤ 400 authored lines total) | Same as T2; total ≤ 400 |

If any command reports a finding that is **pre-existing** (unrelated to slice 1), do not attempt to fix it in this slice — file a separate follow-up and stay inside the scope boundary.

## Notes for the implementer

- The existing parser comment at `internal/service/settings_sandbox.go:55` reads: "It recognizes only adminAuth, httpNodeAuth, and httpStaticAuth; legacy aliases are never migrated implicitly." That comment is the canonical pre-condition for the RED commit — the slice 1 test should fail until the migration is added.
- `testBcryptHash` is defined in `internal/service/auth_surfaces_test.go:11`. Use it directly; do not re-declare or modify it.
- The existing tests `TestRenderAdminAuthMultipleUsers`, `TestRenderHTTPAuthCanonicalKeys`, `TestAuthSurfaceRoundTripPreservesUnmanagedSource`, and `TestHTTPAuthValidationRedactsPasswords` are the regression baseline. They MUST continue to pass after slice 1.
- The PR body MUST follow the `branch-pr` skill format exactly: linked issue with `Closes #N`, exactly one `type:*` label, summary bullets, changes table, test plan, contributor checklist.
- After merge, pause and `ask-on-risk` again before promoting slice 2 to its own PR — the chain strategy is `deferred` and the orchestrator must re-confirm before opening the slice 2 worktree.