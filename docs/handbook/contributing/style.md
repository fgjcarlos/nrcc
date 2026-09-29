# Contributor style guide

> **Stub.** Landed as a skeleton in W1. Detailed style guidance
> arrives in W5.

## Conventions

- English source files only (#768).
- Conventional Commits, one work-unit per change.
- ≤400 authored lines per commit on the feature branch.
- TDD where the test harness is available; otherwise add a `*_test.go`
  or `*.test.mjs` in the same commit.
- Forbidden tokens: `sp-*`, `Beta ·`, `hardening phase`, `no
  production guarantees`, `API keys may change`, `Vitest suites on
  every PR` (the freshness gate from #770 enforces this).

The detailed parser/renderer contract lands in W5 as
[`parser-renderer-contract.md`](parser-renderer-contract.md);
the testing/fixtures guide lands as
[`testing-fixtures.md`](testing-fixtures.md).
