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
- Slice 1/core: 336 authored changed lines; slice 2/review correction: 86 authored changed lines versus `feat/issue-763-navigation-core`, within the <400-line review budget.

## Authorized scope
Frontend routing, sidebar and command-palette navigation, and associated focused tests in this worktree only.

## Delivery strategy
`ask-on-risk`

## Chain boundaries
- Strategy: `feature-branch-chain`
- Tracker/integration branch: `feat/issue-763-navigation` at `origin/main`
- Slice 1/current parent: `feat/issue-763-navigation-core`
- Slice 2/current branch: `feat/issue-763-navigation-review-fixes`

## TDD
- Mode: enabled (from AGENTS.md)
- Runner: `cd frontend && npm test -- --run ...`
- Observed sequence: RED → GREEN → REFACTOR

## Task checklist
- [x] **T1** Adapt navigation and route authorization to the current main baseline, with focused tests and verification.
- [x] **T2** Correct the root setup decision to use `authService.getStatus().initialized` and hide update actions from viewers.

## Review correction defects
1. Root routing used the settled `useAuth().isInitialized` state, which cannot model a fresh server's setup requirement. **Fixed:** root routing now awaits `authService.getStatus().initialized`, uses cancellation-safe cleanup, and falls back to login when status retrieval fails.
2. Viewers could see an update notification that led to an admin-only maintenance route. **Fixed:** the actionable notification is gated by the existing server-issued `admin` role.

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
- T2 RED: focused root/update tests failed on the pre-correction behavior (4 failing tests).
- T2 GREEN: `cd frontend && npm test -- --run src/App.files.test.tsx src/features/updates/components/UpdateNotificationChip.test.tsx` — 2 files, 24 tests passed.
- T2 `cd frontend && npm run typecheck` — passed.
- T2 `cd frontend && npm run lint` — passed with the same 2 pre-existing `react-refresh/only-export-components` warnings in `src/features/backups/components/CronBuilder.tsx`.
- T2 `cd frontend && npm test -- --run` — 59 files, 398 tests passed.
- Parent re-review of the bounded T2 diff found no remaining setup-routing, authorization, redirect, async-cleanup, or test-validity defect; delegated Sol re-review was unavailable because that agent exhausted its usage allowance.

## Progress / evidence
- Implementation commit: `12212e951382d4890a882742560ab712e88a5811` (`feat(navigation): focus NRCC on configuration operations`).
- Rollback boundary: revert the implementation commit to restore pre-Issue-763 route names, navigation entries, and unguarded updates/libraries routes without affecting unrelated backend capabilities.
- No runtime harness is applicable: this is covered by frontend route/component tests.
- Review-correction commit: `1dd458f4f0b0207d5dbf22ec85cffb1c25634098` (`fix(navigation): correct setup and update access`).
- T2 rollback boundary: revert `1dd458f` to restore the prior root/setup selection and update-notification visibility without affecting the core slice.
- RDD remains unavailable because `gentle-ai` could not be resolved in this worktree; no lifecycle command was run.

## Next step
RDD remains unavailable because `gentle-ai` cannot currently be resolved. Prepare the feature-branch chain only if separately authorized.
