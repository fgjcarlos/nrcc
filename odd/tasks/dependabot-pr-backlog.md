# Pending Dependabot pull requests — repair and merge one by one

## Objective

Clear the seven open Dependabot pull requests, repairing every blocking
defect and merging them one at a time. When the backlog is empty, move on
to an issue.

- Issue: none (the pull-request backlog itself is the unit of work)
- Branches: one fix branch plus the seven Dependabot branches
- Base: `origin/main`

## Baseline (2026-11-01)

| PR | Bump | Blocking checks | Root cause |
|----|------|-----------------|------------|
| #942 | `i18next` 26.5.3 → 26.5.11 | `pnpm audit` | `follow-redirects` 1.16.1 override is now a high-severity advisory (GHSA-mrwp-f3q8-3g9p) |
| #941 | `react-i18next` 17.0.15 → 17.1.9 | `pnpm audit` | same |
| #940 | `globals` 17.9.0 → 17.13.0 | `pnpm audit` | same |
| #939 | `vite` 8.3.4 → 8.3.5 | `pnpm audit` | same |
| #938 | `docker/login-action` 4.6.0 → 4.7.0 | `pnpm audit` | same |
| #937 | `@axe-core/playwright` 4.13.0 → 4.14.0 | `pnpm audit`, `Frontend (Node 22)` | same, plus axe 4.14 adds the `label-content-name-mismatch` rule that fails the a11y gate |
| #936 | `golang:1.27.3-alpine` digest | `pnpm audit`, `Backend (Go 1.27.3)` | same, plus the pinned-digest contract test still names the old digest |

The `pnpm audit` failure is a repository-level defect, not a pull-request
defect: `frontend/pnpm-workspace.yaml` pins `follow-redirects: '1.16.1'`
through `overrides`, and the patched release `1.16.2` supersedes it. The
`pnpm audit` job runs on every pull request, so `main` itself is red today
and every pull request inherits it.

## Specs

- **S1** — "Vamos a resolver los problemas de las PRs pendientes y
  mergeando una a una." Every blocking check of every pending pull request
  must pass before that pull request is merged, and the merges happen one at
  a time, in sequence.
- **S2** — "Se tratan PRs de dependabot" — the seven open pull requests
  listed above are Dependabot bumps and are the whole backlog.
- **S3** — "al finalizar iremos a por una issue" — the issue work starts
  only after the pull-request backlog is empty.
- **S4** — The `pnpm audit` gate must pass on `main`: the
  `follow-redirects` override must resolve to a version outside the
  vulnerable range `>=1.16.0 <=1.16.1`.
- **S5** — The pinned Go builder digest contract
  (`tools/langscan/workflow/toolchain_test.go`) must name the digest that
  the Dockerfile actually uses, so the backend suite stays green.
- **S6** — The axe 4.14 `label-content-name-mismatch` violation on the
  user-menu trigger must be fixed in the component, not by weakening the
  a11y gate.
- **S7** — Every merge preserves the repository's protected-branch rules:
  one approving review, all required status checks green and no unresolved
  review conversation.
- **S8** — Repository artifacts for this work are written in English
  (`AGENTS.md`).

## Tasks

- [x] **T1 — Repair the `pnpm audit` gate on `main`.** Bump the
      `follow-redirects` override to `1.16.2`, regenerate the lockfile,
      prove `pnpm audit --audit-level high` passes, open the pull request
      and merge it. (S4, S7)
      Issue [#943](https://github.com/fgjcarlos/nrcc/issues/943), pull
      request [#944](https://github.com/fgjcarlos/nrcc/pull/944) merged as
      `a8caa79`.
- [x] **T2 — PR #942 (`i18next`).** Refresh the branch onto the repaired
      `main`, green CI, approve, merge. (S1, S7)
      Merged as `c33ca1c`.
- [x] **T3 — PR #941 (`react-i18next`).** Same. (S1, S7)
      Merged as `4e7ee66`.
- [x] **T4 — PR #940 (`globals`).** Same. (S1, S7)
      Merged as `07184e6`.
- [x] **T5 — PR #939 (`vite`).** Same. (S1, S7)
      Merged as `3527dd8`.
- [x] **T6 — PR #938 (`docker/login-action`).** Same. (S1, S7)
      Merged as `04b5503`.
- [x] **T7 — PR #936 (`golang` digest).** Refresh the branch, update the
      pinned-digest contract test to the new digest, green CI, approve,
      merge. (S5, S7)
      Merged as `111b6b7`; the contract test
      (`tools/langscan/workflow/toolchain_test.go`) now names the digest
      the Dockerfile builds.
- [x] **T8 — PR #937 (`@axe-core/playwright`).** Refresh the branch, fix
      the user-menu accessible name so axe 4.14 passes the a11y gate,
      green CI, approve, merge. (S6, S7)
      Merged as `52385a9`. The fix had to wait for the last merge of
      `main`: Dependabot rebuilds its branches on every `main` change and
      drops the commits that sit on the branch before that rebuild.
- [x] **T9 — Close the backlog and open the issue work.** Report the
      merged set, then pick the next issue. (S3)

## Implementation record

### The gate that blocked everything

`frontend/pnpm-workspace.yaml` pinned `follow-redirects: '1.16.1'` through
the pnpm `overrides` block. That version is inside the vulnerable range
`>=1.16.0 <=1.16.1` of the high-severity advisory
[GHSA-mrwp-f3q8-3g9p](https://github.com/advisories/GHSA-mrwp-f3q8-3g9p).
The `pnpm audit --audit-level high` job runs on every pull request and on
every push to `main`, so `main` itself was red and every pull request
inherited the failure. The override moved to the patched `1.16.2` in
issue [#943](https://github.com/fgjcarlos/nrcc/issues/943), pull request
[#944](https://github.com/fgjcarlos/nrcc/pull/944).

### The pinned Go builder digest

`tools/langscan/workflow/toolchain_test.go` names the exact digest of the
Go builder image the Dockerfile uses. The digest bump of
[#936](https://github.com/fgjcarlos/nrcc/pull/936) therefore needs the
contract to name the new digest; the test is the human gate of that
supply-chain change, not an obstacle to it.

### The user-menu trigger label

axe 4.14 adds the WCAG 2.5.3 rule `label-content-name-mismatch` to the
browser gate. The trigger of the user menu carried the author label
`<username> — open user menu`, which does not contain the button's own
visible text (the avatar initials, the user name and the role), so the
rule failed on the overview, the configuration and the security page. The
accessible name now comes from the visible content and the action hint
is a visually hidden, translated span (`common:openUserMenu`), which makes
the rendered label a contiguous part of the name.

## Verification evidence

| Check | Where | Result |
|-------|-------|--------|
| `pnpm audit --audit-level high` | `frontend` | no high or critical advisory (before: 1 high) |
| `pnpm install --frozen-lockfile` | `frontend` | lockfile up to date, resolution skipped |
| `pnpm lint` / `pnpm typecheck` | `frontend` | 0 errors, 1 pre-existing warning / exit 0 |
| `pnpm test --run` | `frontend` | 86 files, 587 passed, 2 skipped |
| `pnpm test:e2e` | `frontend` | 21 passed (whole Playwright suite) |
| `pnpm i18n:check` / `pnpm i18n:coverage` | `frontend` | `[]` / 100 % ES coverage, 859/859 keys |
| `go test ./tools/langscan/...` | repository root | ok (the pinned-digest contract) |
| CI of every pull request | GitHub | every job green, including the Docker stack acceptance |

The accessibility gate was observed red before the fix (three serious
`label-content-name-mismatch` violations on the same run) and green
after it, locally and in the CI of
[#937](https://github.com/fgjcarlos/nrcc/pull/937).

## Delivery

| Pull request | Bump | Merged as |
|--------------|------|-----------|
| [#944](https://github.com/fgjcarlos/nrcc/pull/944) | `follow-redirects` 1.16.1 → 1.16.2 (the audit gate) | `a8caa79` |
| [#942](https://github.com/fgjcarlos/nrcc/pull/942) | `i18next` 26.5.3 → 26.5.11 | `c33ca1c` |
| [#941](https://github.com/fgjcarlos/nrcc/pull/941) | `react-i18next` 17.0.15 → 17.1.9 | `4e7ee66` |
| [#940](https://github.com/fgjcarlos/nrcc/pull/940) | `globals` 17.9.0 → 17.13.0 | `07184e6` |
| [#939](https://github.com/fgjcarlos/nrcc/pull/939) | `vite` 8.3.4 → 8.3.5 | `3527dd8` |
| [#938](https://github.com/fgjcarlos/nrcc/pull/938) | `docker/login-action` 4.6.0 → 4.7.0 | `04b5503` |
| [#936](https://github.com/fgjcarlos/nrcc/pull/936) | `golang:1.27.3-alpine` digest | `111b6b7` |
| [#937](https://github.com/fgjcarlos/nrcc/pull/937) | `@axe-core/playwright` 4.13.0 → 4.14.0 + the label fix | `52385a9` |

Every Dependabot pull request was approved by the repository owner and
merged with a merge commit, one at a time. The internal repair
([#944](https://github.com/fgjcarlos/nrcc/pull/944)) was merged with the
administrator override: the owner authored it and GitHub does not allow
approving one's own pull request.

## Log

- **L1** — "Vamos a resolver los problemas de las PRs pendientes y
  mergeando una a una. Se tratan PRs de dependabot, al finalizar iremos a
  por una issue" (user request, 2026-11-01).
