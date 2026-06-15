# Spec: MFA / TOTP second factor

## Purpose

Provides a second authentication factor (RFC 6238 TOTP) for NRCC
admin users, with single-use recovery codes and a server-side
enforcement hook that future public-exposure endpoints can call.
A prerequisite per ADR 0002.

## Requirements

### Requirement: TOTP enrollment

A user MUST be able to enroll a TOTP authenticator. The enrollment
flow has two requests:

1. `POST /api/auth/mfa/enroll` (auth required) returns a candidate
   secret and an `otpauth://` URL. The secret is not yet committed.
2. `POST /api/auth/mfa/enroll/confirm` (auth required) with a 6-digit
   code from the user's authenticator. The server MUST verify the code
   against the candidate secret using the standard TOTP parameters
   (HMAC-SHA1, 6 digits, 30-second step, ±1 step window). On success
   the enrollment is committed and ten recovery codes are returned
   exactly once.

#### Scenario: Successful enrollment

- GIVEN an authenticated user with no enrollment
- WHEN they POST `/api/auth/mfa/enroll` and then POST
  `/api/auth/mfa/enroll/confirm` with a valid 6-digit code
- THEN the second response contains 10 recovery codes and the
  enrollment is committed

#### Scenario: Confirm rejected with invalid code

- GIVEN an authenticated user mid-enrollment
- WHEN they POST `/api/auth/mfa/enroll/confirm` with code `"000000"`
- THEN the server returns 400 with code `INVALID_CODE` and the
  enrollment is not committed

#### Scenario: Enroll blocked when already enrolled

- GIVEN a user that already has an active enrollment
- WHEN they POST `/api/auth/mfa/enroll`
- THEN the server returns 409 with code `MFA_ALREADY_ENROLLED`

### Requirement: Login requires TOTP when enrolled

When a user with an active TOTP enrollment authenticates with
username + password, the login endpoint MUST return 200 with body
`{ mfaRequired: true, mfaToken: "<opaque>" }` and MUST NOT issue an
access token. The client then calls `POST /api/auth/mfa/verify` with
the mfaToken and either a 6-digit TOTP code or a recovery code. On
success the server issues the access token and refresh cookie exactly
as the non-MFA login path does.

#### Scenario: Login without enrollment is unchanged

- GIVEN a user with no enrollment
- WHEN they POST `/api/auth/login` with valid credentials
- THEN the server returns 200 with an access token and refresh cookie

#### Scenario: Login with enrollment requires second step

- GIVEN a user with an active enrollment
- WHEN they POST `/api/auth/login` with valid credentials
- THEN the server returns 200 with `mfaRequired: true` and a
  short-lived opaque mfaToken, and issues no access token

#### Scenario: mfaToken expires

- GIVEN a user holds a mfaToken more than 5 minutes old
- WHEN they POST `/api/auth/mfa/verify`
- THEN the server returns 401 with code `MFA_TOKEN_EXPIRED`

### Requirement: Recovery codes

On successful enrollment the server generates 10 single-use recovery
codes formatted `xxxx-xxxx-xxxx` (12 hex chars, lower-case). Codes are
stored as bcrypt hashes. A consumed code MUST NOT be reusable.
`GET /api/auth/mfa/status` MUST return the count of remaining unused
codes.

#### Scenario: Recovery code consumed successfully

- GIVEN a user with an active enrollment and 10 unused recovery codes
- WHEN they submit a valid recovery code to `mfa/verify`
- THEN the server issues a session, marks the code used, and
  `GET /mfa/status` reports 9 remaining

#### Scenario: Reusing a consumed code is rejected

- GIVEN a recovery code that has already been used
- WHEN it is submitted again to `mfa/verify`
- THEN the server returns 401 with code `INVALID_CODE`

### Requirement: Disabling MFA

A user MUST be able to disable their own MFA by submitting
`POST /api/auth/mfa/disable` with their current password. An admin
MUST be able to disable MFA for any user (still requires the admin's
own current password). The endpoint MUST return 409 if the target
user is not enrolled.

#### Scenario: Self-disable with correct password

- GIVEN an authenticated user with an active enrollment
- WHEN they POST `/api/auth/mfa/disable` with the correct password
- THEN the server returns 200 and the enrollment row is deleted

#### Scenario: Disable blocked with wrong password

- GIVEN an authenticated user
- WHEN they POST `/api/auth/mfa/disable` with the wrong password
- THEN the server returns 401 with code `INVALID_PASSWORD`

#### Scenario: Non-admin cannot disable another user

- GIVEN a non-admin authenticated user
- WHEN they POST `/api/auth/mfa/disable` for a different user id
- THEN the server returns 403 with code `FORBIDDEN`

### Requirement: Rate-limiting on the verify path

`POST /api/auth/mfa/verify` MUST share the existing per-IP and
per-username rate limits that protect `/api/auth/login`, using the
same in-memory limiter and key naming (`mfa-verify-ip:<ip>` and
`mfa-verify-user:<username>`). A blocked request MUST return 429.

### Requirement: MfaEnforcer.CanExpose

The server MUST expose a pure function
`MfaEnforcer.CanExpose(userID, action string) (allowed bool, reason string)`
where `action` is one of `"local"`, `"tailnet"`, `"public"`. The
enforcer MUST return `allowed=false` for `action == "public"` when
the user has no enrollment, and `allowed=true` for `"local"` and
`"tailnet"` regardless of enrollment. The enforcer MUST be usable
from any future exposure-control handler without I/O on the caller
side (the caller supplies the enrollment snapshot).

#### Scenario: Public exposure blocked without enrollment

- GIVEN a user with no enrollment
- WHEN a caller asks `CanExpose(userID, "public")`
- THEN the enforcer returns `allowed=false, reason="MFA enrollment required for public exposure"`

#### Scenario: Local exposure always allowed

- GIVEN any user (enrolled or not)
- WHEN a caller asks `CanExpose(userID, "local")`
- THEN the enforcer returns `allowed=true`

### Requirement: Audit logging

The server MUST emit an audit log event for each of:

- `MFA_ENROLL` (user, target=self, result=ok|fail)
- `MFA_VERIFY` (user, target=self, result=ok|fail, meta.method=totp|recovery)
- `MFA_RECOVERY_USE` (user, target=self, result=ok|fail)
- `MFA_DISABLE` (actor, target=user, result=ok|fail)

The event MUST be written via the existing `audit.Service` so it
appears in `data/audit/audit.jsonl` alongside the other auth events.

### Requirement: Status is visible to the user

`GET /api/auth/mfa/status` (auth required) MUST return
`{ enabled: bool, recoveryCodesRemaining: int }`. A user with no
enrollment MUST get `enabled: false, recoveryCodesRemaining: 0`.
`GET /api/auth/me` MUST include an `mfa: { enabled, recoveryCodesRemaining }`
block in its response so the UI does not need a second round-trip on
page load.
