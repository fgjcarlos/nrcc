# Exploration — Next SDD Slice Recommendation

> SDD phase: **explore** (sdd-init → spec → design → plan → implement → verify → review → deliver)
> Artifact store: `openspec` (per parent preflight)
> Delivery strategy: `ask-on-risk` · review budget: 400 lines · chain strategy: `deferred`
> Skill resolution: `paths-injected` (work-unit-commits, branch-pr, go-testing, cognitive-doc-design)
> Generated against `/home/composedof2/Dev/Codex/nrcc/nrcc` on the current working tree
> (read-only inspection — no source files were modified).

## 1. Existing capability specs — coverage and gap matrix

| Capability spec | Code coverage today | Spec / config drift | Doc artifacts |
| --- | --- | --- | --- |
| `openspec/specs/dashboard-runtime-metrics/spec.md` | **Implemented** — ring buffer (`internal/service/metrics_buffer.go`), sampler (`internal/service/metrics_sampler_linux.go`), `/api/system/history` + `/api/runtime/history` handlers, `RestartEvent` buffer, restart-count persistence (`internal/service/restart_count_store.go`), panic-recovery middleware (`internal/middleware/recover.go` installed as first `r.Use` in `internal/server/server.go:88`), enriched `/api/health` (`internal/handler/system.go:GetHealth`). | Spec is consistent with `openspec/config.yaml`; the "Existing Behaviour" + "New Behaviour" split (retroactive vs. this change) is intact. | `spec.md` only — **no `design.md`, no `odd/tasks/<feature>.md`**. |
| `openspec/specs/metrics-endpoint/spec.md` | **Implemented but spec is stale.** `/metrics` is registered, the counters/gauges are wired through `internal/metrics/collector.go`, and `internal/metrics/process_collector.go` exposes `nrcc_nodered_running`, `nrcc_nodered_restarts_total`, `nrcc_nodered_uptime_seconds`. | **#671 regression:** spec says `MUST be reachable without authentication` / `MUST NOT appear under any auth-guarded route group`, but `internal/server/server.go:63-70` (`metricsArePublic()`) now gates `/metrics` by default and registers it inside the auth group unless `NRCC_METRICS_PUBLIC=true` (see `CHANGELOG.md` Unreleased and `SECURITY.md` Operator Responsibilities). The spec therefore contradicts the shipped security posture. | `spec.md` only — **no `design.md`, no `odd/tasks/<feature>.md`**. |
| `openspec/specs/user-management-ui/spec.md` | **Implemented.** `frontend/src/features/auth/components/{UsersView,UserTable,UserModal}.tsx` render create / `edit_full` / `edit_password` modal modes; admin gating, loading / empty / error `StateContainer` slots, responsive table-vs-card layout, and the 5 exempt hardcoded strings (lines 119-124) are all in place. | The spec already declares itself a "Delta for User Management UI" (issue #112) and is consistent with `strict_tdd: true`. Two intentional follow-ups are documented (lines 137-139): "Create user — API error" and "Password reset — API error" mutation error paths. | `spec.md` only — **no `design.md`, no `odd/tasks/<feature>.md`**. |
| `openspec/specs/user-role-editing/spec.md` | **Implemented.** `RequireSelfOrAdmin` middleware plus `ErrCannotDemoteLastAdmin` path (`internal/handler/auth.go:UpdateUser`) and the front-end `edit_full` last-admin gate (`frontend/src/features/auth/components/UsersView.tsx`). | Spec matches the code; backend delete guard returns 403 `CANNOT_DELETE_LAST_ADMIN` per spec. | `spec.md` only — **no `design.md`, no `odd/tasks/<feature>.md`**. |
| `openspec/specs/user-update-api/spec.md` | **Implemented.** `PATCH /api/auth/users/{id}` (role only) and `PATCH /api/auth/users/{id}/password` are both wired in `internal/server/server.go` with `RequireAdmin` / `RequireSelfOrAdmin` middleware; 400 / 403 / 404 paths and the role enum guard all exist. | Spec's `updatedAt` note ("the code returns a superset") is acknowledged; otherwise aligned. | `spec.md` only — **no `design.md`, no `odd/tasks/<feature>.md`**. |

### Spec-level gaps visible across the board

- **No `design.md` for any of the five specs.** `openspec/config.yaml:phase_rules.artifacts_per_phase.spec` requires only `spec.md`, but `design.md` is the next mandatory artifact for any new feature work and is missing everywhere. A `spec → design → plan` slice will create the design doc as its first deliverable.
- **No `odd/tasks/<feature>.md`** for any of the five capabilities. This is the organic-driven-development plan artifact; absent for capabilities that have shipped. Closing one of those gaps is the most natural place to anchor a follow-up `odd/tasks/...` file.
- **Capability specs are stored only in `openspec/specs/<capability>/spec.md`.** Legacy predecessor specs are kept under `docs/history/specs/...` with full `proposal.md` / `design.md` / `tasks.md` / `verify-report.md`. New work should NOT regress to that pattern; the active capability folder is `openspec/specs/`.

## 2. Gaps visible from `odd/audits/issue-765/gap-report.md`

Three 🟡 AMBER clusters remain open. Two of them are code-and-tests gaps with clearly named acceptance items.

| Cluster | Theme | Gap | Recommended artifact |
| --- | --- | --- | --- |
| **#759** Reliable access administration | 🟡 AMBER | **G3:** `access-administration E2E` and `MFA lifecycle E2E` not present in `frontend/e2e/`. | Two Playwright specs following the `frontend/e2e/auth.spec.ts` pattern. |
| **#760** Authentication surfaces | 🟡 AMBER | **G2:** `TestLegacyAuthFieldMigration` (Go) and `SecuritySurfaceIsolationE2E` (Playwright) not present anywhere in the codebase. | One Go unit test in `internal/service/auth_surfaces_test.go` + one Playwright spec in `frontend/e2e/security-center.spec.ts`. |
| **#764** Advanced settings escape hatches | 🟡 AMBER | **G1:** Maintainer-level security review attestation not recorded in `git log`. The four acceptance tests are all present on `origin/main`. | Doc-only follow-up comment / audit-log entry. Not a code slice. |

G1 is a doc-only attestation and is outside the SDD implement/verify scope; it cannot be a PR slice by itself. G2 and G3 are the natural code-bearing PR slices.

## 3. Tooling and CI gates the next slice must satisfy

| Tool | Command | Gate |
| --- | --- | --- |
| Go (race) | `go test -race -count=1 -timeout 20m ./...` | required at `implement`, `verify`, `review` |
| `golangci-lint` v2.12.2 | `golangci-lint run --config=.golangci.yml ./...` | repo-wide zero-finding (`.github/workflows/pr.yml:lint`) |
| Vitest | `pnpm --filter frontend test --run` | required at `implement`, `verify` |
| ESLint | `pnpm --filter frontend lint` | required at `verify` |
| TypeScript | `pnpm --filter frontend typecheck` | required at `verify` |
| Playwright | `pnpm --filter frontend test:e2e` (Chromium only) | required for `user_facing_change`, `auth_flow_change`, `new_e2e_capability` |
| OpenAPI lint | `npx @redocly/cli lint docs/openapi.yaml --format stylish` | gate |
| Language policy | `go run ./tools/langscan` | advisory only |
| Branch / commit | `^(feat|fix|chore|docs|style|refactor|perf|test|build|ci|revert)/[a-z0-9._-]+$` / Conventional Commit | required |
| PR | `Closes #N` + exactly one `type:*` label | required |
| **Build constraint** | Linux-only (`docs/adr/0004-linux-only-build.md`); POSIX-only syscall symbols | env constraint |

The test isolation contract in `metrics-endpoint/spec.md` ("custom `prometheus.Registry` per test case, no bleed") is a concrete, table-driven example of how strictly the repo expects tests to be written.

## 4. Recommended next PR slice

### Slice: **Cluster #760 — security surface test closure (chain slice 1)**

- **Capability: `Fix_CLOSURE` of cluster #760.** New spec folder: `openspec/specs/authentication-surfaces-closure/`.
- **Single work unit: `TestLegacyAuthFieldMigration` in `internal/service/auth_surfaces_test.go`.**
  - Table-driven Go unit test covering the legacy NRCC field-name → canonical Node-RED 5.x surface mapping: legacy `adminAuth` / `httpAuth` → canonical `adminAuth` / `httpNodeAuth` / `httpStaticAuth` (mirrors the round-trip already exercised by `TestRenderHTTPAuthCanonicalKeys` in the same file).
  - Behaviour under test: a settings file using the legacy HTTP auth field name is parsed, normalised, and re-rendered with the canonical Node-RED 5 keys without losing credential values.
  - TDD sequence (per `openspec/config.yaml:strict_tdd` and `go-testing` skill): RED → GREEN → REFACTOR. The first commit writes the failing test against the current parser. If the parser lacks the migration, the second commit implements it; otherwise GREEN lands on the first.
- **Acceptance criteria** (extracted from `odd/audits/issue-765/gap-report.md` Gap G2 item 1 and `openspec/config.yaml:review_budget_threshold_lines`):
  1. `internal/service/auth_surfaces_test.go` adds `TestLegacyAuthFieldMigration` with at least four sub-cases (legacy adminAuth single user, legacy adminAuth multi-user, legacy httpAuth → httpNodeAuth, legacy static-auth alias → httpStaticAuth).
  2. `go test -race -count=1 -timeout 20m ./internal/service/...` passes; race detector clean.
  3. `golangci-lint run --config=.golangci.yml ./...` passes with zero findings.
  4. `git diff --stat` reports ≤ 400 authored lines added in the single work-unit commit.
  5. PR body carries `Closes #760` (or the equivalent issue that gap-report references) and a single `type:test` label per `branch-pr` skill rules.
  6. Branch name: `test/issue-760-legacy-auth-field-migration` (matches the `^(feat|fix|chore|docs|style|refactor|perf|test|build|ci|revert)/[a-z0-9._-]+$` regex from `branch-pr`).
- **Estimated authored lines:** ~150-250 (test cases + helper fixtures), well under the 400-line budget.
- **Human-controlled gates touched:** none (no auth-policy change, no destructive op, no publishing).
- **Risk:** low. The code under test already exists; the test is additive and the legacy mapping is a documented invariant in the spec.

### Why this slice (not another)

| Candidate | Why it ranks below |
| --- | --- |
| `access-administration E2E` + `MFA lifecycle E2E` (cluster #759) | Two Playwright specs are tightly coupled to a live MFA / admin-bootstrap stack; expected ~300-350 lines **per** spec plus shared fixtures, which strains the budget if combined and forces `ask-on-risk` if chained. Both are valuable but heavier. |
| The two missing `user-management-ui` mutation-error tests ("Create user — API error" / "Password reset — API error") | Real but small; does not close a roadmap cluster to 🟢 GREEN, so the value-vs-effort ratio is weaker. |
| `metrics-endpoint` spec update for #671 | Doc-only fix; satisfies the language-policy / strict-TDD integrity, but does not generate an implementation commit. Better as a separate `docs/issue-671-metrics-spec-update` slice. |
| `SecuritySurfaceIsolationE2E` Playwright spec (cluster #760, item 2) | Should be the **chained follow-up** to this slice, not the first one — it is larger, requires the dashboard stack fixture, and benefits from `TestLegacyAuthFieldMigration` already landing first. |

### Chain strategy

Per parent preflight, `chain_strategy: deferred`. The recommendation is:

- **Slice 1 (this one):** `TestLegacyAuthFieldMigration` Go unit test, single PR to `main`.
- **Deferred slice 2 (chained PR):** `SecuritySurfaceIsolationE2E` Playwright spec in `frontend/e2e/security-center.spec.ts`, opened once slice 1 is green and cluster #760 is half-closed.

Both are within budget; they protect review focus because they isolate one contract at a time. The orchestrator should `ask-on-risk` again at the slice-1 close before promoting slice 2.

## 5. Blockers / open risks for the recommended slice

1. **No upstream `design.md` exists for any of the 5 capability specs.** The SDD phase ordering requires `spec` → `design` → `plan` → `implement`. If the user prefers to follow strict phase ordering, slice 1 needs a `design.md` and an `odd/tasks/issue-760-security-surface-tests.md` first. The minimal viable path is to author those two artifacts in the same work-unit commit as the test (≈ +60 lines), keeping the slice under budget.
2. **The legacy field mapping may already be partially implemented elsewhere.** `grep` for `LegacyAuth` / `legacy.*adminAuth` finds no matches today; the test will fail RED on the first run if the migration is missing, which is the correct TDD outcome but requires the implementer to know the canonical mapping. The `user-update-api` and `user-role-editing` specs hint at the canonical names already; a 10-minute read of `internal/service/auth_surfaces.go` (the file under test) confirms the table.
3. **`golangci-lint` is repo-wide, not ratcheted** (`openspec/config.yaml:testing.runners.backend_lint.ci_gate`). The slice cannot introduce any new findings; any unused-import / unused-var / line-length / revive / gosec complaint must be fixed before push.
4. **CI fork-PR guard.** `.github/workflows/pr.yml:25-27` skips heavy CI on untrusted forks unless `safe-to-test` is applied. The slice does not need any new CI workflow, so this is informational only.
5. **Linux-only build.** Tests must compile and run on Linux. The slice touches no syscall or `/proc` path, so the constraint is satisfied without further work.
6. **`TestLegacyAuthFieldMigration` may need a small parser fix.** That fix MUST stay inside the same work-unit commit (work-unit commits keep behaviour + tests together per the `work-unit-commits` skill), and the fix itself must be table-driven and redaction-safe (no plaintext credentials in test fixtures). The slice budget should still hold — estimate 60-90 extra lines for the parser fix + tests.

## 6. What this exploration deliberately does NOT cover

- No implementation was attempted; this is exploration only, per `addendum` rule "Produce exploration notes only; do not implement."
- No subagent delegation; the parent owns delegation.
- No changes to OpenSpec config (`openspec/config.yaml`), no changes to `AGENTS.md`, `CONTRIBUTING.md`, `CHANGELOG.md`, `SECURITY.md`, CI workflows, or any test, source, or docs file.
- No new artifacts under `openspec/changes/` other than this `explore.md`.

## 7. Suggested next phase

`spec` for the new capability folder `openspec/specs/authentication-surfaces-closure/spec.md`. The new spec will:

- Reference and inherit the relevant requirements from `openspec/specs/user-update-api/spec.md` and `openspec/specs/user-role-editing/spec.md`.
- Add an explicit `TestLegacyAuthFieldMigration` requirement with EARS-style scenarios.
- Re-state the existing `SecuritySurfaceIsolationE2E` requirement so the chained follow-up has a contract.
- Carry the 🟡 → 🟢 GREEN gap-report closure as a measured outcome.

Then `design` → `plan` (under `odd/tasks/issue-760-security-surface-tests.md`) → `implement` → `verify` → `review` (only if judgment-day is selected) → `deliver` (single PR per `branch-pr`).