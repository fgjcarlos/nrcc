# Change Proposal — issue-760-security-surface-tests

## Why

Cluster #760 (`feat(security): separate and manage Node-RED authentication surfaces`) is
🟡 AMBER in `odd/audits/issue-765/gap-report.md` because two of four named acceptance
tests are missing in the codebase:

1. `TestLegacyAuthFieldMigration` (Go unit test).
2. `SecuritySurfaceIsolationE2E` (Playwright spec).

This change closes item 1 (slice 1 of Gap G2) inside a single Go-only work-unit
commit/PR. The Playwright spec is deferred to a chained follow-up slice (slice 2).

The chosen PR boundary is **single PR, single work-unit commit, ≤ 400 authored
lines** so the slice fits the `ask-on-risk` delivery strategy with the canonical
400-line review budget.

## What

Add `TestLegacyAuthFieldMigration` to `internal/service/auth_surfaces_test.go`
covering the legacy NRCC field-name → canonical Node-RED 5.x surface migration
(`adminAuth` legacy shape → canonical, `httpAuth` → `httpNodeAuth`, static-auth
alias → `httpStaticAuth`). Fixtures are table-driven and use only the existing
`testBcryptHash` constant — no plaintext credentials.

If the existing parser does not migrate legacy aliases implicitly, extend
`extractAuthenticationSurfaces` (or an adjacent helper) inside
`internal/service/settings_sandbox.go` to perform that migration, preserving
unmanaged source and existing redaction discipline.

## Out of scope (slice 1)

- The `SecuritySurfaceIsolationE2E` Playwright spec.
- Any handler, middleware, persistence, or UI change.
- The five existing capability specs under `openspec/specs/`.
- `odd/audits/issue-765/gap-report.md` rewrites.

## Delivery

- Branch: `test/issue-760-legacy-auth-field-migration` based on `origin/main`.
- Worktree: `nrcc-worktrees/test-issue-760-legacy-auth-field-migration` (already
  the project convention).
- Strict TDD: RED → GREEN → REFACTOR (single GREEN permitted with documented
  rationale when the parser already implements the migration).
- PR body must include `Closes #760` and explicitly defer
  `SecuritySurfaceIsolationE2E` to slice 2.

## References

- Issue #760 body (acceptance tests list).
- `odd/audits/issue-765/{gap-report,roadmap-traceability}.md`.
- `openspec/config.yaml` (`strict_tdd`, `ask-on-risk`, 400-line review budget).
- `openspec/specs/authentication-surfaces-closure/{spec.md,design.md}`.
- `odd/tasks/issue-760-security-surface-tests.md` (this change's task list).
