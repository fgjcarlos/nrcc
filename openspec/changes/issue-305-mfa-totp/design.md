# Design: Issue #305 — MFA / TOTP second factor

## Technical Approach

Add a new service (`service.MfaService`) backed by a typed JSON store,
expose its operations through a dedicated handler
(`handler.MfaHandler`), and register five HTTP endpoints under
`/api/auth/mfa/*`. The login handler gains a small branch that returns
`mfaRequired: true` for enrolled users. A pure `MfaEnforcer` function
sits next to the service for exposure-gating callers.

## Architecture Decisions

| Decision | Choice | Rejected | Rationale |
|---|---|---|---|
| TOTP library | `github.com/pquerna/otp` | Stdlib HMAC + custom impl | Library handles base32, step window, drift; ~200 LOC off our plate. MIT, widely used, easy to swap later. |
| Secret storage | Plaintext in `data/mfa.json` (mode 0600) | Encrypted at rest | Required for verification; data dir is the existing trust boundary. Same posture as JWT secret. |
| Recovery-code hashing | bcrypt | SHA-256 | Bcrypt is consistent with the rest of the auth flow and gives a constant-time compare. Throughput is fine: a user consumes a recovery code once per lost-device event. |
| Login branching | Two-step (mfaToken then verify) | Single-step with code in login body | Keeps the login request shape stable for the non-enrolled case, and lets the verify endpoint have its own rate-limit and audit events. The mfaToken is in-memory only, 5 min TTL, single use. |
| `mfaToken` store | In-memory `sync.Map` with TTL eviction | New JSON file / Redis | Volume is tiny (only pending logins), 5 min TTL is short, and a restart drops all pending MFA logins which is safe (password was valid seconds ago; the user re-logs in). |
| `MfaEnforcer` API | Pure function on a snapshot | Service method that reads the store | Keeps the gate easy to unit-test and lets the caller (a future exposure handler) decide whether to read with a lock or a snapshot. |
| New file vs. extend `auth.go` | New `handler/mfa.go` + new `service/mfa.go` | Extend `auth.go` | Auth handler is already 700 LOC; new endpoints are 200+ LOC of their own logic. |

## Data Flow

### Enrollment

```
Authed user ──POST /api/auth/mfa/enroll──→ MfaHandler.Enroll
                                            │
                                            └─ MfaService.BeginEnrollment(userID)
                                                  │
                                                  ├─ if existing enrollment → 409
                                                  ├─ generate 160-bit secret (base32)
                                                  ├─ store candidate in pending map
                                                  │
                                            ← { secret, otpauthUrl }
```

`BeginEnrollment` writes the candidate secret to a per-user field on
the existing store row (added but marked `pending`). Persisting the
secret on the first call (not on confirm) lets the user restart their
authenticator app and the same secret still works; it does NOT mark
the user as enrolled.

```
Authed user ──POST /api/auth/mfa/enroll/confirm { code }──→ MfaHandler.EnrollConfirm
                                                              │
                                                              └─ MfaService.ConfirmEnrollment(userID, code)
                                                                    │
                                                                    ├─ verify code against pending secret
                                                                    ├─ on success: persist enrollment, set enrolledAt,
                                                                    │   generate 10 recovery codes, bcrypt-hash each
                                                                    │
                                                              ← { recoveryCodes, recoveryCodesRemaining: 10 }
```

### Login with MFA

```
User ──POST /api/auth/login {username, password}──→ AuthHandler.Login
                                                       │
                                                       └─ password valid?
                                                              │
                                              no ─→ existing 401 path
                                              yes, no enrollment → existing success
                                              yes, enrolled:
                                                       │
                                                       └─ MfaService.IssueMfaToken(userID)
                                                              │
                                                              ├─ random 32-byte token
                                                              ├─ store userID + TTL in pendingMfa map
                                                              │
                                                       ← 200 { mfaRequired: true, mfaToken: "..." }
```

```
User ──POST /api/auth/mfa/verify {mfaToken, code}──→ MfaHandler.Verify
                                                        │
                                                        └─ MfaService.VerifyAndIssueSession(mfaToken, code)
                                                              │
                                                              ├─ validate mfaToken (exists, not expired, not used)
                                                              ├─ if code matches "aaaa-bbbb-cccc" pattern:
                                                              │   MfaService.ConsumeRecoveryCode(userID, code)
                                                              ├─ else: MfaService.VerifyTotp(userID, code)
                                                              ├─ on success: mark token used, issue access token + refresh cookie
                                                              │
                                                        ← 200 AuthResponse (token + user)
```

## File Changes

| File | Action | Description |
|---|---|---|
| `go.mod`, `go.sum` | Modify | Add `github.com/pquerna/otp` |
| `internal/model/mfa.go` | Create | Types: `MfaEnrollment`, `RecoveryCode`, `MfaStore` |
| `internal/service/mfa.go` | Create | `MfaService`: BeginEnrollment, ConfirmEnrollment, Disable, Status, IssueMfaToken, VerifyAndIssueSession, VerifyTotp, ConsumeRecoveryCode, `MfaEnforcer.CanExpose` |
| `internal/service/mfa_test.go` | Create | Unit tests for TOTP generation, code verification, recovery code consumption, `MfaEnforcer` |
| `internal/handler/mfa.go` | Create | `MfaHandler` with Enroll, EnrollConfirm, Disable, Status, Verify |
| `internal/handler/mfa_test.go` | Create | Endpoint tests for all 5 routes |
| `internal/handler/auth.go` | Modify | Login branches on enrollment; `GetMe` returns `mfa` block |
| `internal/server/server.go` | Modify | Register the 5 MFA routes, wire `MfaService` into `AuthHandler` for the login branch |
| `internal/cmd/root.go` (or similar bootstrap) | Modify | Construct `MfaService` from `dataDir` and pass to handler / server |
| `docs/openapi.yaml` | Modify | Document the 5 new endpoints and the `mfaRequired` login response |

No frontend changes in this PR — the API surface is shipped and the
React side is a follow-up issue. `frontend/src/features/auth/services/authService.ts`
is left untouched; the existing dev test (MSW handlers) does not need
new stubs because no frontend code calls the new endpoints yet.

## Interfaces / Contracts

### `MfaService` (Go)

```go
type MfaService struct { /* … */ }

func NewMfaService(dataDir string, authSvc *AuthService) *MfaService

// Enrollment
func (s *MfaService) BeginEnrollment(userID string) (secret, otpauthURL string, err error)
func (s *MfaService) ConfirmEnrollment(userID, code string) (recoveryCodes []string, err error)
func (s *MfaService) Disable(userID, actingID string, actingIsAdmin bool, password string) error
func (s *MfaService) Status(userID string) (enabled bool, remaining int, err error)

// Login
type PendingMfa struct { UserID string; ExpiresAt time.Time }
func (s *MfaService) IssueMfaToken(userID string) (string, error)
func (s *MfaService) VerifyAndIssueSession(token, code string) (*AuthService, *model.CCUser, string /*method: totp|recovery*/, error)
func (s *MfaService) VerifyTotp(userID, code string) bool
func (s *MfaService) ConsumeRecoveryCode(userID, code string) (ok bool, err error)

// Enforcer
func (s *MfaService) Enforcer() MfaEnforcer
```

### `MfaEnforcer` (pure)

```go
type MfaEnforcer struct { IsEnrolled func(userID string) bool }

func (e MfaEnforcer) CanExpose(userID, action string) (bool, string) {
    if action != "public" { return true, "" }
    if !e.IsEnrolled(userID) {
        return false, "MFA enrollment required for public exposure"
    }
    return true, ""
}
```

### Wire format

All responses use the existing `model.ApiResponse[T]` envelope. New
request/response structs live next to the handler in
`internal/handler/mfa.go`. Recovery codes are formatted as
`xxxx-xxxx-xxxx` (12 hex chars + 2 dashes) and matched case-insensitive
in verify.

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit — service | TOTP code verify (RFC 6238 vectors), recovery-code generate/consume, mfaToken TTL, `MfaEnforcer` table tests | `internal/service/mfa_test.go` |
| Unit — handler | Each of the 5 routes: success, bad payload, no auth, wrong code, rate-limit hit | `internal/handler/mfa_test.go` |
| Unit — auth handler | Login with enrollment returns `mfaRequired`; mfa/verify issues token; used recovery code rejected | `internal/handler/auth_mfa_test.go` |
| Existing regression | `go test ./...` stays green | Make test |

## Migration / Rollout

No data migration. New file `data/mfa.json` is created on first
enrollment. No existing user has an enrollment row, so the login
branch is dormant for current installations until they opt in.

## Open Questions

- Should disabling MFA for another admin require a separate confirmation
  (e.g. re-typing the acting admin's password)? Defer; current proposal
  trusts the admin role.
- WebAuthn as a second factor later? Out of scope but the data model
  should not block it; recovery codes can be shared across factors.
