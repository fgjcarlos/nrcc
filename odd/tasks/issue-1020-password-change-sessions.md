# Issue #1020: require the current password and revoke sessions on password change

Branch `fix/1020-password-change-sessions` from `origin/main` (de8934d). Security fix (high-risk item 2).

## Specs

- S1: "A self-service password change needs the current password" — "A self-service change without the current password returns 401."
- S2: "every password change ends the user's existing sessions" — "A refresh token issued before the change is rejected afterwards."
- S3: "call `RevokeUserSessions` after any password change or MFA disable."
- S4: "An admin resetting another user's password keeps the current behaviour, plus the revocation."
- S5: "`internal/handler/auth.go:690`: comment refers to a route that does not exist."

## Tasks

- T1 [S1-S5] inline: backend handler changes + Go tests, OpenAPI, generated schema, frontend profile form (current password field) and self-row routing in the users table. Commits: 3098d76 (backend, OpenAPI, schema, changelog), 14a7cd2 (frontend).
- T2 [S1-S5] verify: independent `gentle-ai-verify` (Risk item 2): all S# pass. Commit: n/a.
- T3 [S1, S3] inline: verify findings — API client no longer refresh-retries a 401 `INVALID_PASSWORD` (double lockout count); admin-disables-other-user MFA test. Commit: 14a7cd2 / 3098d76.

## Log

- L1: "Resuelve la issue #1020"
- L2: Decision: a successful self-service change revokes every session, then issues a fresh refresh cookie for the calling device so the user is not logged out of the browser that made the change. Admin resets of another user only revoke. Failed current-password checks feed the existing per-user login rate-limit bucket (`mw.AuthUserKey`) to stop brute force through this endpoint.
- L3: User chose to put the new copy keys in `common.json` because the Gentle safety policy blocks the `auth` locale catalog by filename.
- L4: Evidence: `go test ./internal/handler/ -run 'Auth|ChangePassword|Mfa|Refresh|Login|Setup'` ok; frontend `npm test -- --run` 92 files / 743 passed, `tsc --noEmit` 0, eslint clean; full `go test ./...` only fails `TestSaveConfig_WritesExplicitRuntimeSettingsAndRestarts`, which also fails on origin/main (pre-existing). Accepted residuals: password persisted before revocation (500 SESSION_REVOKE_ERROR if revoke fails), JWT access tokens live until expiry, tiny Refresh/revoke race.
- L5: Slice boundary: 501 authored lines in two commits (backend 342, frontend 159); the backend commit exceeds the 400 budget mostly through tests.
