package service

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/store"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

const (
	// MfaIssuer is the issuer label baked into otpauth:// URLs and
	// shown in the user's authenticator app.
	MfaIssuer = "NRCC"

	// MfaAccountPrefix is prepended to the username when building
	// the otpauth account-name. The full account name is e.g.
	// "user:admin".
	MfaAccountPrefix = "user:"

	// MfaRecoveryCodeCount is the number of recovery codes generated
	// on a successful enrollment.
	MfaRecoveryCodeCount = 10

	// MfaRecoveryCodeLength is the number of hex characters per
	// group in a recovery code. Three groups of 4 hex chars = 12 hex
	// chars = 48 bits of entropy per code.
	MfaRecoveryCodeLength = 4

	// MfaTokenTTL is how long a mfaToken issued by the login flow
	// remains valid.
	MfaTokenTTL = 5 * time.Minute

	// MfaOtpDigits is the standard 6-digit code length.
	MfaOtpDigits = 6

	// MfaOtpPeriod is the standard 30-second TOTP step.
	MfaOtpPeriod = 30
)

// recoveryCodePattern matches a recovery code in xxxx-xxxx-xxxx shape.
// Used to disambiguate recovery codes from 6-digit TOTP codes on the
// verify path before reaching for bcrypt.
var recoveryCodePattern = regexp.MustCompile(`^[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}$`)

// Sentinel errors returned by MfaService. Handlers translate these
// to HTTP status codes; callers should test via errors.Is.
var (
	ErrMfaAlreadyEnrolled   = errors.New("mfa: user already enrolled")
	ErrMfaNotEnrolled       = errors.New("mfa: user not enrolled")
	ErrMfaInvalidCode       = errors.New("mfa: invalid code")
	ErrMfaTokenInvalid      = errors.New("mfa: token invalid")
	ErrMfaTokenExpired      = errors.New("mfa: token expired")
	ErrMfaTokenUsed         = errors.New("mfa: token already used")
	ErrMfaUserNotFound      = errors.New("mfa: user not found")
	ErrMfaInvalidPassword   = errors.New("mfa: invalid password")
	ErrMfaPendingMissing    = errors.New("mfa: no enrollment in progress")
	ErrMfaRecoveryExhausted = errors.New("mfa: no unused recovery codes")
)

// MfaService owns the on-disk MFA state plus the in-memory mfaToken
// ledger. It is the single integration point for TOTP enrollment,
// verification, recovery codes, and the MfaEnforcer seam.
type MfaService struct {
	authSvc *AuthService
	store   *store.JSONStore[model.MfaStore]

	mu            sync.Mutex
	pendingTokens map[string]pendingMfa
}

// pendingMfa tracks a single mfaToken between login and verify.
type pendingMfa struct {
	userID    string
	username  string
	expiresAt time.Time
	used      bool
}

// NewMfaService wires the JSON store at dataDir/mfa.json.
func NewMfaService(dataDir string, authSvc *AuthService) *MfaService {
	path := dataDir + string('/') + "mfa.json"
	return &MfaService{
		authSvc:       authSvc,
		store:         store.NewJSONStore[model.MfaStore](path),
		pendingTokens: make(map[string]pendingMfa),
	}
}

// BeginEnrollment generates a candidate TOTP secret for the user and
// stores it in a pending (uncommitted) enrollment row. The secret is
// returned along with the otpauth:// URL the user can paste into any
// RFC 6238 authenticator.
//
// If the user already has an active enrollment this returns
// ErrMfaAlreadyEnrolled. To re-enroll, disable the existing one first.
func (s *MfaService) BeginEnrollment(userID string) (secret, otpauthURL string, err error) {
	if s.authSvc.GetUserByID(userID) == nil {
		return "", "", ErrMfaUserNotFound
	}

	now := model.NowISO8601()
	s.mu.Lock()
	defer s.mu.Unlock()

	data, _ := s.store.Read()
	if hasActiveEnrollment(data, userID) {
		return "", "", ErrMfaAlreadyEnrolled
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      MfaIssuer,
		AccountName: accountNameFor(userID),
		Period:      MfaOtpPeriod,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", "", fmt.Errorf("generate totp key: %w", err)
	}

	// Upsert a pending row.
	replaced := false
	for i := range data.Enrollments {
		if data.Enrollments[i].UserID == userID {
			data.Enrollments[i] = model.MfaEnrollment{
				UserID:    userID,
				Secret:    key.Secret(),
				Pending:   true,
				UpdatedAt: now,
			}
			replaced = true
			break
		}
	}
	if !replaced {
		data.Enrollments = append(data.Enrollments, model.MfaEnrollment{
			UserID:    userID,
			Secret:    key.Secret(),
			Pending:   true,
			UpdatedAt: now,
		})
	}
	if err := s.store.Write(data); err != nil {
		return "", "", fmt.Errorf("persist pending enrollment: %w", err)
	}

	return key.Secret(), key.URL(), nil
}

// ConfirmEnrollment verifies a 6-digit code against the candidate
// secret and, on success, commits the enrollment, generates recovery
// codes, and returns them (single display).
func (s *MfaService) ConfirmEnrollment(userID, code string) ([]string, error) {
	data, err := s.store.Read()
	if err != nil {
		// No MFA file at all means no enrollment in progress.
		return nil, ErrMfaPendingMissing
	}

	row := findEnrollment(data, userID)
	if row == nil || row.Secret == "" {
		return nil, ErrMfaPendingMissing
	}
	if !row.Pending {
		return nil, ErrMfaAlreadyEnrolled
	}
	if !validCodeFormat(code) {
		return nil, ErrMfaInvalidCode
	}
	if !totp.Validate(code, row.Secret) {
		return nil, ErrMfaInvalidCode
	}

	codes, hashes, err := generateRecoveryCodes()
	if err != nil {
		return nil, fmt.Errorf("generate recovery codes: %w", err)
	}

	now := model.NowISO8601()
	for i := range data.Enrollments {
		if data.Enrollments[i].UserID == userID {
			data.Enrollments[i].Pending = false
			data.Enrollments[i].EnrolledAt = now
			data.Enrollments[i].UpdatedAt = now
			data.Enrollments[i].RecoveryCodes = hashes
			break
		}
	}
	if err := s.store.Write(data); err != nil {
		return nil, fmt.Errorf("commit enrollment: %w", err)
	}

	return codes, nil
}

// Disable removes the enrollment for targetUserID. An admin can
// disable any user; a non-admin can only disable themselves.
//
// Password is the calling user's current password. This re-uses the
// bcrypt hash on the caller (NOT the target) so an admin cannot
// silently nuke another admin's MFA without re-authenticating.
func (s *MfaService) Disable(actingID, targetUserID, password string, actingIsAdmin bool) error {
	actor := s.authSvc.GetUserByID(actingID)
	if actor == nil {
		return ErrMfaUserNotFound
	}
	if !s.authSvc.VerifyPassword(actor.PasswordHash, password) {
		return ErrMfaInvalidPassword
	}

	if actingID != targetUserID && !actingIsAdmin {
		return fmt.Errorf("forbidden: only admin can disable another user's MFA")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Read()
	if err != nil {
		return fmt.Errorf("read mfa store: %w", err)
	}
	if findEnrollment(data, targetUserID) == nil {
		return ErrMfaNotEnrolled
	}

	kept := data.Enrollments[:0]
	for _, e := range data.Enrollments {
		if e.UserID != targetUserID {
			kept = append(kept, e)
		}
	}
	data.Enrollments = kept
	return s.store.Write(data)
}

// Status reports enrollment state for a user. enabled is true only
// for committed (non-pending) enrollments. recoveryCodesRemaining
// counts only unused recovery codes.
func (s *MfaService) Status(userID string) (model.MfaStatusResponse, error) {
	data, err := s.store.Read()
	if err != nil {
		return model.MfaStatusResponse{}, nil // no file → not enrolled
	}
	row := findEnrollment(data, userID)
	if row == nil || row.Pending {
		return model.MfaStatusResponse{Enabled: false, RecoveryCodesRemaining: 0}, nil
	}
	remaining := 0
	for _, c := range row.RecoveryCodes {
		if c.UsedAt == "" {
			remaining++
		}
	}
	return model.MfaStatusResponse{Enabled: true, RecoveryCodesRemaining: remaining}, nil
}

// IsEnrolled reports whether the user has a committed TOTP enrollment.
// Cheap to call on the login hot path.
func (s *MfaService) IsEnrolled(userID string) bool {
	data, err := s.store.Read()
	if err != nil {
		return false
	}
	row := findEnrollment(data, userID)
	return row != nil && !row.Pending
}

// IssueMfaToken creates a single-use, 5-minute token that the client
// must present to /api/auth/mfa/verify with a TOTP code.
func (s *MfaService) IssueMfaToken(userID string) (string, error) {
	user := s.authSvc.GetUserByID(userID)
	if user == nil {
		return "", ErrMfaUserNotFound
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate mfa token: %w", err)
	}
	token := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingTokens[token] = pendingMfa{
		userID:    userID,
		username:  user.Username,
		expiresAt: time.Now().Add(MfaTokenTTL),
	}
	return token, nil
}

// VerifyAndIssueSession consumes an mfaToken, verifies the supplied
// code (TOTP or recovery), and on success returns the resolved user
// and the method used ("totp" or "recovery"). The caller is
// responsible for issuing the access token and refresh cookie.
//
// The token is single-use: regardless of success or failure, it is
// removed from the pending map after this call so a leaked code from
// the user cannot be retried.
func (s *MfaService) VerifyAndIssueSession(token, code string) (*model.CCUser, string, error) {
	code = strings.TrimSpace(code)

	s.mu.Lock()
	p, ok := s.pendingTokens[token]
	if !ok {
		s.mu.Unlock()
		return nil, "", ErrMfaTokenInvalid
	}
	if p.used {
		s.mu.Unlock()
		return nil, "", ErrMfaTokenUsed
	}
	if time.Now().After(p.expiresAt) {
		delete(s.pendingTokens, token)
		s.mu.Unlock()
		return nil, "", ErrMfaTokenExpired
	}
	// Mark used up-front so a slow / replayed request cannot
	// race a second request through.
	delete(s.pendingTokens, token)
	s.mu.Unlock()

	user := s.authSvc.GetUserByID(p.userID)
	if user == nil {
		return nil, "", ErrMfaUserNotFound
	}

	method, err := s.verifyCodeForUser(p.userID, code)
	if err != nil {
		return nil, "", err
	}
	return user, method, nil
}

// Enforcer returns a pure MfaEnforcer that reads the enrollment table
// through a snapshot at call time. The snapshot keeps the enforcer
// trivially unit-testable and lets future exposure handlers pass
// their own snapshot if they already hold a lock.
func (s *MfaService) Enforcer() MfaEnforcer {
	return MfaEnforcer{IsEnrolled: s.IsEnrolled}
}

// verifyCodeForUser accepts a 6-digit TOTP code OR a recovery code
// formatted xxxx-xxxx-xxxx (case-insensitive). Returns the method
// that succeeded.
func (s *MfaService) verifyCodeForUser(userID, code string) (string, error) {
	if recoveryCodePattern.MatchString(code) {
		ok, err := s.ConsumeRecoveryCode(userID, code)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", ErrMfaInvalidCode
		}
		return "recovery", nil
	}

	if !validCodeFormat(code) {
		return "", ErrMfaInvalidCode
	}
	if !s.VerifyTotp(userID, code) {
		return "", ErrMfaInvalidCode
	}
	return "totp", nil
}

// VerifyTotp returns true if the 6-digit code is valid for the
// user's enrolled TOTP secret. Uses the standard 30s step with
// ±1 step window for clock drift tolerance.
func (s *MfaService) VerifyTotp(userID, code string) bool {
	data, err := s.store.Read()
	if err != nil {
		return false
	}
	row := findEnrollment(data, userID)
	if row == nil || row.Pending {
		return false
	}
	return totp.Validate(code, row.Secret)
}

// ConsumeRecoveryCode marks a recovery code as used. Returns true if
// a previously-unused matching code was found and consumed.
func (s *MfaService) ConsumeRecoveryCode(userID, code string) (bool, error) {
	normalized := normalizeRecoveryCode(code)

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Read()
	if err != nil {
		return false, fmt.Errorf("read mfa store: %w", err)
	}
	idx := -1
	for i := range data.Enrollments {
		if data.Enrollments[i].UserID == userID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return false, ErrMfaNotEnrolled
	}

	row := &data.Enrollments[idx]
	now := model.NowISO8601()
	for i, c := range row.RecoveryCodes {
		if c.UsedAt != "" {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(c.Hash), []byte(normalized)) == nil {
			row.RecoveryCodes[i].UsedAt = now
			if err := s.store.Write(data); err != nil {
				return false, fmt.Errorf("persist recovery code use: %w", err)
			}
			return true, nil
		}
	}
	return false, nil
}

// PruneExpiredTokens removes pending mfaTokens that have passed
// their TTL. Safe to call from a background ticker.
func (s *MfaService) PruneExpiredTokens() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.pendingTokens {
		if now.After(v.expiresAt) {
			delete(s.pendingTokens, k)
		}
	}
}

// LookupUsernameByMfaToken returns the username associated with a
// pending mfaToken, or "" if the token is unknown / expired / used.
// Used by the verify handler to enrich audit events and rate-limit
// per-user without leaking the user-id through error responses.
func (s *MfaService) LookupUsernameByMfaToken(token string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pendingTokens[token]
	if !ok {
		return ""
	}
	if p.used || time.Now().After(p.expiresAt) {
		return ""
	}
	return p.username
}

// AuthService exposes the underlying auth service for tests that
// need to seed extra users (e.g. a viewer to test admin-only
// branches). Production code does not use this.
func (s *MfaService) AuthService() *AuthService { return s.authSvc }

// MfaEnforcer is a pure, side-effect-free gate for exposure
// controls. Construct via MfaService.Enforcer; tests can construct
// one with a fake IsEnrolled.
type MfaEnforcer struct {
	IsEnrolled func(userID string) bool
}

// CanExpose returns whether the user is allowed to perform an
// exposure action. Actions outside "public" are always allowed;
// "public" requires a committed TOTP enrollment.
//
// The reason string is non-empty exactly when allowed is false and
// is safe to surface to the operator (it is generic on purpose —
// it does not leak the user state, only the policy).
func (e MfaEnforcer) CanExpose(userID, action string) (bool, string) {
	switch action {
	case "public":
		if e.IsEnrolled == nil || !e.IsEnrolled(userID) {
			return false, "MFA enrollment required for public exposure"
		}
		return true, ""
	case "tailnet", "local", "":
		return true, ""
	default:
		return false, "unknown exposure action"
	}
}

// ─── helpers ───────────────────────────────────────────────────────

func hasActiveEnrollment(data model.MfaStore, userID string) bool {
	row := findEnrollment(data, userID)
	return row != nil
}

func findEnrollment(data model.MfaStore, userID string) *model.MfaEnrollment {
	for i := range data.Enrollments {
		if data.Enrollments[i].UserID == userID {
			return &data.Enrollments[i]
		}
	}
	return nil
}

func accountNameFor(userID string) string {
	return MfaAccountPrefix + userID
}

func validCodeFormat(code string) bool {
	if len(code) != MfaOtpDigits {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// generateRecoveryCodes returns (display, hashes) — display is what
// the user sees; hashes is what we persist.
func generateRecoveryCodes() (display []string, hashes []model.RecoveryCode, err error) {
	display = make([]string, 0, MfaRecoveryCodeCount)
	hashes = make([]model.RecoveryCode, 0, MfaRecoveryCodeCount)
	seen := make(map[string]struct{}, MfaRecoveryCodeCount)
	for len(display) < MfaRecoveryCodeCount {
		buf := make([]byte, MfaRecoveryCodeLength*3/2+1) // enough for 3 groups of 4 hex
		if _, err := rand.Read(buf); err != nil {
			return nil, nil, fmt.Errorf("rand: %w", err)
		}
		hexStr := hex.EncodeToString(buf)
		// Three groups of MfaRecoveryCodeLength hex chars.
		code := strings.ToLower(
			hexStr[0:MfaRecoveryCodeLength] + "-" +
				hexStr[MfaRecoveryCodeLength:MfaRecoveryCodeLength*2] + "-" +
				hexStr[MfaRecoveryCodeLength*2:MfaRecoveryCodeLength*3],
		)
		if _, dup := seen[code]; dup {
			continue
		}
		seen[code] = struct{}{}

		hash, err := bcrypt.GenerateFromPassword([]byte(code), BcryptCost)
		if err != nil {
			return nil, nil, fmt.Errorf("bcrypt: %w", err)
		}
		display = append(display, code)
		hashes = append(hashes, model.RecoveryCode{Hash: string(hash)})
	}
	return display, hashes, nil
}

// normalizeRecoveryCode lower-cases and strips surrounding whitespace
// for a fair comparison. Dashes are kept because the format is part
// of the contract.
func normalizeRecoveryCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}
