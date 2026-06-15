# Tasks: Issue #305 — MFA / TOTP second factor

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~750 LOC (mostly new files) |
| 400-line budget risk | Medium — split if frontend work is added in a follow-up |
| Chained PRs recommended | No (single backend PR keeps review atomic) |
| Delivery strategy | single PR |

### Breakdown

| Group | Tasks | Estimated LOC | Focus |
|-------|-------|--------------|-------|
| 1. Spec | 3 | ~300 (markdown) | proposal, design, spec |
| 2. Backend types & service | 5 | ~280 | `model/mfa.go`, `service/mfa.go`, `service/mfa_test.go` |
| 3. Backend handlers | 3 | ~220 | `handler/mfa.go`, `handler/mfa_test.go`, `handler/auth_mfa_test.go` |
| 4. Wiring & docs | 4 | ~80 | `server.go` routes, `cmd/root.go` bootstrap, `auth.go` login branch + `me` field, `openapi.yaml` |
| **Total** | **15** | **~880** | |

The single-PR threshold (400 LOC) refers to code+tests in a single
change; this change adds roughly 580 LOC of code+tests plus 300 LOC
of markdown. If reviewers prefer, the spec docs can ship first as a
`docs:` PR and the code as a follow-up.

## Implementation Order

1. **Spec docs** (already drafted). Allows reviewers to anchor on the
   contract before code.
2. **Add `pquerna/otp` dependency.** `go get github.com/pquerna/otp`.
3. **`model/mfa.go`** — types and store schema.
4. **`service/mfa.go`** — `MfaService` and `MfaEnforcer` with full
   unit tests.
5. **`handler/mfa.go`** — 5 endpoints with handler-level tests.
6. **`handler/auth.go`** — Login branching, `me` extension, audit
   events. Tests in `handler/auth_mfa_test.go`.
7. **`server/server.go`** + bootstrap — register routes, construct
   `MfaService`.
8. **`docs/openapi.yaml`** — document the new endpoints and the
   `mfaRequired` login response shape.

## Detailed Tasks

### Spec

- [x] Draft `proposal.md` (intent, scope, capabilities, approach, API, risks, rollback, success)
- [x] Draft `design.md` (architecture decisions, data flow, file changes, interfaces, testing)
- [x] Draft `specs/mfa-totp/spec.md` (requirements + scenarios)

### Dependency

- [ ] `go get github.com/pquerna/otp` and `go mod tidy`
- [ ] Confirm `go.sum` includes the new module and `go mod verify` passes

### Backend types & service

- [ ] Create `internal/model/mfa.go` with `MfaEnrollment`, `RecoveryCode`, `MfaStore`
- [ ] Create `internal/service/mfa.go`:
  - `NewMfaService(dataDir string, authSvc *AuthService) *MfaService`
  - `BeginEnrollment(userID) (secret, otpauthURL string, err error)`
  - `ConfirmEnrollment(userID, code string) (recoveryCodes []string, err error)`
  - `Disable(userID, actingID string, actingIsAdmin bool, password string) error`
  - `Status(userID string) (enabled bool, remaining int, err error)`
  - `IssueMfaToken(userID string) (string, error)` — 5 min TTL, single use
  - `VerifyAndIssueSession(token, code string) (*AuthService, *model.CCUser, string, error)`
  - `VerifyTotp(userID, code string) bool`
  - `ConsumeRecoveryCode(userID, code string) (bool, error)`
  - `Enforcer() MfaEnforcer` returning a pure function over an in-memory snapshot
- [ ] `internal/service/mfa_test.go`:
  - TOTP code verify (RFC 6238 vector)
  - Recovery code generate produces 10 unique 12-hex codes formatted as `xxxx-xxxx-xxxx`
  - `ConsumeRecoveryCode` removes from valid set
  - Reusing a consumed code returns false
  - mfaToken expires after TTL
  - mfaToken is single-use
  - `MfaEnforcer.CanExpose("public")` returns false for non-enrolled
  - `MfaEnforcer.CanExpose("public")` returns true for enrolled
  - `MfaEnforcer.CanExpose("local")` always true
  - `MfaEnforcer.CanExpose("tailnet")` always true

### Backend handlers

- [ ] Create `internal/handler/mfa.go` with:
  - `MfaHandler.Enroll` — `POST /api/auth/mfa/enroll`
  - `MfaHandler.EnrollConfirm` — `POST /api/auth/mfa/enroll/confirm`
  - `MfaHandler.Disable` — `POST /api/auth/mfa/disable`
  - `MfaHandler.Status` — `GET /api/auth/mfa/status`
  - `MfaHandler.Verify` — `POST /api/auth/mfa/verify` (public, mfaToken in body)
- [ ] `internal/handler/mfa_test.go`:
  - Enroll returns secret + otpauthURL when unauthed → 401
  - Enroll returns 409 when already enrolled
  - EnrollConfirm with wrong code → 400 INVALID_CODE
  - Disable with wrong password → 401
  - Status returns `{enabled: false, recoveryCodesRemaining: 0}` for new user
  - Verify with invalid mfaToken → 401
- [ ] `internal/handler/auth_mfa_test.go`:
  - Login for enrolled user returns `mfaRequired: true`, no token
  - mfa/verify with valid TOTP issues token + cookie
  - mfa/verify with valid recovery code issues token + cookie
  - mfa/verify with used recovery code → 401 INVALID_CODE
  - Login for non-enrolled user is unchanged

### Wiring & docs

- [ ] `internal/server/server.go`:
  - Register `POST /api/auth/mfa/enroll`, `POST /api/auth/mfa/enroll/confirm`, `POST /api/auth/mfa/disable`, `GET /api/auth/mfa/status` (auth required)
  - Register `POST /api/auth/mfa/verify` (public)
  - Pass `MfaService` to `AuthHandler` for the login branch
- [ ] `cmd/root.go` (or server bootstrap): construct `MfaService` from `dataDir` and wire it
- [ ] `internal/handler/auth.go`:
  - `Login` branches on enrollment; returns `mfaRequired: true` when applicable
  - `GetMe` response includes `mfa: { enabled, recoveryCodesRemaining }`
  - Audit events for `MFA_ENROLL`, `MFA_VERIFY`, `MFA_RECOVERY_USE`, `MFA_DISABLE`
- [ ] `docs/openapi.yaml`:
  - Add paths for the 5 MFA endpoints
  - Update `/api/auth/login` response to document the `mfaRequired` branch
  - Add `mfa` block to `/api/auth/me` response

## Verification

- `go test ./...` from the repo root — must pass.
- `go vet ./...` — must be clean.
- `go build ./...` — must succeed (validates `cmd` bootstrap wiring).
- `make test` if the frontend is touched in a follow-up; not needed
  for this backend-only PR.

## Rollback

1. Revert the PR. All routes are additive; nothing else in `/api/auth`
   changes shape for users that are not enrolled.
2. If a stale `data/mfa.json` is left behind, deleting it has no
   effect on any other record (no foreign keys).
