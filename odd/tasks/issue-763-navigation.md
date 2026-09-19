# Issue 763 — Focus NRCC navigation on configuration operations

## Objective
Refocus the frontend navigation on setup, sign-in, overview, backup recovery, and administrator maintenance operations without removing backend capabilities.

## Problem / Why
Legacy public navigation exposed Flows, Files, Updates, and Libraries even though routine NRCC use should focus on configuration and recovery. Older URLs must continue to resolve intentionally, while maintenance operations remain distinct and restricted to server-issued administrators.

## Scope
- Root route selection for setup, login, and Overview.
- User-facing labels: Dashboard → Overview; Backups → Recovery.
- Retire public navigation entries for Flows, Files, Updates, and Libraries.
- Preserve legacy redirects and distinct admin-only maintenance routes for Updates and Libraries.
- Add focused route/navigation/authorization tests.

## Constraints
- Preserved newer #759–#761 behavior.
- Did not delete backend capabilities or invent a maintenance capability; routes use the server-issued `admin` role.
- Original dirty checkout remained untouched.
- One cohesive implementation work unit; 336 authored changed lines, within the ≈400-line advisory target.

## Authorized scope
Frontend routing, sidebar and command-palette navigation, and associated focused tests in this worktree only.

## Delivery strategy
`ask-on-risk`

## TDD
- Mode: enabled (from AGENTS.md)
- Runner: `cd frontend && npm test -- --run ...`
- Observed sequence: RED → GREEN → REFACTOR

## Task checklist
- [x] **T1** Adapt navigation and route authorization to the current main baseline, with focused tests and verification.

## Acceptance criteria
1. `/` deterministically routes to setup, login, or Overview. ✓
2. Public navigation displays Overview and Recovery, but not Flows, Files, Updates, or Libraries. ✓
3. Legacy routes redirect intentionally without removing backend capabilities. ✓
4. `/maintenance/updates` and `/maintenance/libraries` remain distinct, admin-only, and discoverable to administrators through sidebar and command palette. ✓
5. Non-admin direct access is denied for both maintenance routes. ✓

## Checks
- RED: focused Vitest command failed on the pre-change implementation (19 failing tests); first run also exposed the worktree's missing `vitest` binary before a read-only dependency symlink was added, then the focused test run exercised the intended failures.
- GREEN: `cd frontend && npm test -- --run src/App.files.test.tsx src/App.sidebar.test.tsx src/shared/components/command-palette/CommandPalette.test.tsx src/shared/components/ProtectedRoute.test.tsx src/features/updates/components/UpdateNotificationChip.test.tsx src/features/dashboard/components/DashboardView.test.tsx` — 6 files, 38 tests passed.
- `cd frontend && npm run typecheck` — passed.
- `cd frontend && npm run lint` — passed with 2 pre-existing `react-refresh/only-export-components` warnings in `src/features/backups/components/CronBuilder.tsx`.
- `cd frontend && npm test -- --run` — 59 files, 396 tests passed.
- `git diff --check` — passed.

## Progress / evidence
- Implementation commit: `12212e951382d4890a882742560ab712e88a5811` (`feat(navigation): focus NRCC on configuration operations`).
- Rollback boundary: revert the implementation commit to restore pre-Issue-763 route names, navigation entries, and unguarded updates/libraries routes without affecting unrelated backend capabilities.
- No runtime harness is applicable: this is covered by frontend route/component tests.

## Next step
Commit this ODD bookkeeping update as `chore(odd): record issue 763 completion`; do not push, open a PR, or mutate GitHub.
