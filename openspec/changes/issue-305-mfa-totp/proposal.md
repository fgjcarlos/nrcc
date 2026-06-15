# Proposal: Issue #305 — MFA / TOTP second factor

## Intent

ADR 0002 (edge-mode defaults and exposure UX) lists "MFA or an equivalent
second factor" as a prerequisite for any one-click public-exposure control.
This change introduces the missing authentication factor: time-based
one-time passwords (TOTP, RFC 6238) for NRCC admin users, with recovery
codes, and a server-side enforcement hook that future exposure controls
can call. The scope is intentionally backend-first and minimal — no new
public-exposure flow ships here, but every requirement needed to build
one is now in place.

## Scope

### In Scope

- TOTP enrollment for the authenticated user: generate a per-user
  secret, return an `otpauth://` URI plus a one-time preview QR-ready
  payload, and persist the secret only after the user submits a valid
  first code (RFC 6238, 30s step, SHA1, 6 digits).
- TOTP verification on login: when a user has TOTP enrolled, the login
  endpoint requires a code (or a recovery code) in addition to the
  password. Password-only logins for enrolled users fail with
  `MFA_REQUIRED` and the user must re-submit with a code.
- Recovery codes: on enrollment, generate 10 single-use recovery codes.
  Codes are stored as bcrypt hashes alongside the user. A consumed code
  cannot be reused; remaining-valid count is returned on every status
  read.
- A pure `MfaEnforcer.CanExpose(userID, action)` server hook that
  future exposure endpoints (Tailscale Funnel, public URL, etc.) can
  call to gate admin-only operations. Default policy: enrolled
  required for `public` actions, otherwise allowed.
- A `GET /api/auth/me` extension that returns `mfa: { enabled, recoveryCodesRemaining }`
  so the UI can show enrollment state without a separate call.
- Audit log events for `MFA_ENROLL`, `MFA_VERIFY`, `MFA_RECOVERY_USE`,
  `MFA_DISABLE`, each with the acting username and the target.

### Out of Scope

- Frontend enrollment UI (TOTP entry page, QR display): the API surface
  is shipped and documented; the React side is a follow-up issue.
- Push-based second factor (WebAuthn, FIDO2, SMS). Only TOTP per ADR 0002.
- Per-action MFA policy beyond a binary `enrolled or not`. Edge mode
  default is to require enrollment before `public` actions; finer-grained
  policies ship later with the exposure controls.
- Migration of existing users: enrollment is opt-in. A new user that
  completes setup without enrolling is not blocked.

## Capabilities

### New Capabilities

- `mfa-totp`: Server-side TOTP enrollment, login-time verification, and
  recovery codes. A small `MfaEnforcer` exposes a `CanExpose` hook for
  future public-exposure endpoints.

## Approach

1. **Crypto library.** Add `github.com/pquerna/otp` (de-facto Go TOTP
   library, MIT licensed) for secret generation and code verification.
   The standard library's `crypto/hmac` and `crypto/rand` cover the
   remaining primitives.
2. **Persistence.** Store one row per user in a new
   `internal/store/mfa.json` (typed through the existing
   `JSONStore[T]`). Schema:
   ```json
   {
     "enrollments": [
       { "userId": "uuid", "secret": "base32", "enrolledAt": "...",
         "recoveryCodes": [ { "hash": "bcrypt", "usedAt": null } ] }
     ]
   }
   ```
   Secret is kept in plaintext at rest (required for verification, the
   file is mode 0600 and the data dir is the same root as the existing
   user store). Recovery codes are bcrypt-hashed so a leak of the JSON
   file does not let an attacker impersonate a user with a recovery
   code.
3. **Enforcement hook.** `service.MfaEnforcer.CanExpose(userID, action)`
   returns `(allowed bool, reason string)`. For `action == "public"` it
   requires an enrollment row. For `action == "tailnet"` or
   `action == "local"` it always allows. This is the seam exposure
   endpoints call before they accept a configuration change.
4. **Login flow change.** The existing `Login` handler now branches:
   - Password wrong → existing 401.
   - Password correct, no enrollment → existing success path.
   - Password correct, enrollment present → 200 with body
     `{ mfaRequired: true, mfaToken: "short-lived opaque" }` and the
     client must call `POST /api/auth/mfa/verify` with the code or
     recovery code. On success the handler issues the access token and
     refresh cookie as before.
   The `mfaToken` is a random 32-byte value (URL-safe base64), stored
   in memory only, TTL 5 minutes, single use. No new file is written.
5. **Routes.** New endpoints (all under `/api/auth`):
   - `POST /api/auth/mfa/enroll` (auth required) → returns
     `{ secret, otpauthUrl }`. Does NOT persist; persists only after
     verify.
   - `POST /api/auth/mfa/enroll/confirm` (auth required) with
     `{ code }` → verifies against the candidate secret, persists
     enrollment + recovery codes, returns `{ recoveryCodes: [...] }`
     (single display).
   - `POST /api/auth/mfa/disable` (auth required, self or admin) →
     requires current password, deletes the enrollment row.
   - `GET /api/auth/mfa/status` (auth required) →
     `{ enabled, recoveryCodesRemaining }`.
   - `POST /api/auth/mfa/verify` (public, mfaToken in body) →
     issues access token + refresh cookie on success.

## API Contract

### `POST /api/auth/mfa/enroll`

Auth required. Returns the candidate secret. **Not yet enabled.**

```json
{ "secret": "JBSWY3DPEHPK3PXP", "otpauthUrl": "otpauth://totp/NRCC:admin?secret=JBSWY3DPEHPK3PXP&issuer=NRCC" }
```

### `POST /api/auth/mfa/enroll/confirm`

Auth required. Body: `{ "code": "123456" }`. On success returns
recovery codes (shown once) and flips enrollment to enabled.

```json
{ "recoveryCodes": ["a1b2-...","..."], "recoveryCodesRemaining": 10 }
```

Errors: 400 (`INVALID_CODE`, `MFA_NOT_ENROLLING`), 401 (no auth).

### `POST /api/auth/mfa/disable`

Auth required. Body: `{ "password": "..." }`. Admin can disable any
user; non-admin can disable self only. Returns `{ "success": true }`.
Errors: 400 (missing password), 401, 403 (non-admin disabling another
user), 404, 409 (not enrolled).

### `GET /api/auth/mfa/status`

Auth required. Returns:

```json
{ "enabled": true, "recoveryCodesRemaining": 7 }
```

### `POST /api/auth/mfa/verify`

Public. Body: `{ "mfaToken": "...", "code": "123456" }` — code is
either a 6-digit TOTP or a recovery code (`aaaa-bbbb-cccc` shape).

On success: 200 with `AuthResponse` (token + user) and refresh cookie
set, same as login.

Errors: 400 (`INVALID_REQUEST`, `INVALID_CODE`), 401
(`MFA_TOKEN_INVALID`, `MFA_TOKEN_EXPIRED`), 429 (rate limited).

### `service.MfaEnforcer.CanExpose(userID, action string) (bool, string)`

Pure function (no I/O). Caller passes an in-memory snapshot of the
enrollment table. Returns false for `action == "public"` when the user
has no enrollment.

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| TOTP secret leaks via file read | Low | Secret stored only after first successful verify; data dir is 0700; secret and recovery-code hashes are not logged. |
| Brute-force 6-digit code on `/mfa/verify` | Medium | Reuse the existing `RateLimiter` on the mfa-verify route with the same `ip:` and `user:` keys as login. Tests cover the limiter wiring. |
| User loses both device and recovery codes | Low | Out of scope for this change; documented in `SECURITY.md` as a "re-install" path that wipes the enrollment row. Follow-up issue. |
| Login API contract breaks existing clients | Medium | When the user has no enrollment, the response is unchanged. Only enrolled users see the `mfaRequired` branch. The frontend is updated in a follow-up; existing setup is not affected. |
| `pquerna/otp` is unmaintained | Low | Library is small, audited, used widely; if it becomes unmaintained we swap it for a 30-line stdlib implementation. Pinning the version. |

## Rollback Plan

1. Revert the PR. All new routes are additive — the rest of `/api/auth`
   is untouched.
2. Users with an enrollment row retain the row on disk; deleting the
   JSON file (`data/mfa.json`) clears it. No user record references
   the row, so no foreign key is violated.

## Success Criteria

- [ ] Authenticated user can enroll TOTP and sees recovery codes once.
- [ ] Authenticated user can disable TOTP with current password.
- [ ] Enrolled user logging in receives `mfaRequired: true` and must
      call `mfa/verify` with a TOTP code or recovery code.
- [ ] Used recovery code cannot be reused.
- [ ] Brute-force attempts on `mfa/verify` are rate-limited the same
      way login is.
- [ ] `MfaEnforcer.CanExpose` returns false for a non-enrolled admin
      on `action == "public"`.
- [ ] Audit log records `MFA_ENROLL`, `MFA_VERIFY`, `MFA_RECOVERY_USE`,
      `MFA_DISABLE` events.
- [ ] OpenAPI spec lists the new endpoints with request/response shapes.
- [ ] Go test suite passes (`go test ./...`).
- [ ] `go vet` clean.

## Dependencies

- `github.com/pquerna/otp` (new, MIT).
- Existing `service.AuthService`, `store.JSONStore[T]`, `audit.Service`,
  `middleware.RateLimiter`, and `internal/middleware.Auth`.

## Related

- ADR 0002 (exposure UX prerequisites).
- Blocks the public-exposure half of the EDGE_MODE issue.
