# Issue #1055: Credential revocation
https://github.com/fgjcarlos/nrcc/issues/1055
Risk: item 2 (authentication and session invalidation); item 4 (concurrent refresh).

## Specs
S1. An access token issued before a password change returns 401 on any protected route afterwards.
S2. An access token issued before an MFA disable returns 401 afterwards.
S3. If revocation fails, the password change is not applied (or is applied together with revocation), and a test pins this.
S4. A refresh that races a password change cannot leave a valid session issued before the change.
S5. The calling device of a self-service change stays signed in (its new access token and refresh cookie carry the new version).

## Tasks
- T1 — S1–S5 — inline implementation and independent verification — verified — commits: `56d4a54` (backend, tests and API contract), `f0a611d` (frontend token adoption and test)

## Log
L1. Resuelve la issue #1055

L2. Password and MFA each persist their version in the same atomic write as the credential change. A replacement refresh session is staged for a never-used version before that write and becomes valid only at commit. Rotation rechecks the old session inside the ledger update and mints from the same credential snapshot. Expired staged and superseded rows are handled by normal pruning; they are never accepted or counted as active.

L3. Focused handler tests reproduced stale access tokens, missing replacement access tokens and password persistence on a failed session write. Independent verification confirmed S1–S5, including concurrent refresh and failure injection, and checked the subsequent regressions in cookie fixtures and refresh responses. The final focused Go race run and the deleted-user refresh differential passed.

L4. The Go suite passed with `TestSaveConfig_WritesExplicitRuntimeSettingsAndRestarts` excluded after reproducing its Node-RED 5.0.10 vs 5.0.6 failure on `origin/main`. That suite ran before the final focused deleted-user replay refinement. Frontend Vitest: 743 passed, 1 skipped; typecheck passed; Playwright: 59 passed, 3 skipped.

L5. Local delivery slices: base `bc8c61b` → backend `56d4a54` → frontend `f0a611d`. The backend slice is over the ~400-line target; splitting credential versioning from issuance and refresh checks would leave an intermediate authentication contract that does not enforce the complete invariant. No push or PR; the issue still requires maintainer approval before a PR is opened.
