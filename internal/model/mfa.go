package model

// MfaEnrollment is the per-user TOTP enrollment record.
//
// Secret is stored as a base32-encoded string (no padding) compatible
// with the standard otpauth:// URI scheme. Recovery codes are bcrypt
// hashes — a leak of the JSON store does not let the attacker use
// recovery codes. UsedAt is non-nil once a recovery code is consumed;
// consumed codes are pruned at disable time but kept until then for
// audit traceability.
type MfaEnrollment struct {
	UserID        string         `json:"userId"`
	Secret        string         `json:"secret"`
	EnrolledAt    string         `json:"enrolledAt"`
	UpdatedAt     string         `json:"updatedAt"`
	Pending       bool           `json:"pending,omitempty"` // true between Begin and Confirm
	RecoveryCodes []RecoveryCode `json:"recoveryCodes"`
}

// RecoveryCode is a single-use recovery code hash + consumption marker.
type RecoveryCode struct {
	Hash   string `json:"hash"`
	UsedAt string `json:"usedAt,omitempty"`
}

// MfaStore is the on-disk JSON root for the MFA store.
type MfaStore struct {
	Enrollments []MfaEnrollment `json:"enrollments"`
}

// MfaStatusResponse is the public MFA status payload returned to
// the user (and embedded in /api/auth/me).
type MfaStatusResponse struct {
	Enabled                bool `json:"enabled"`
	RecoveryCodesRemaining int  `json:"recoveryCodesRemaining"`
}

// MfaLoginResponse is the body returned by /api/auth/login when the
// user is enrolled. mfaToken is short-lived (5 min), single use.
type MfaLoginResponse struct {
	MfaRequired bool   `json:"mfaRequired"`
	MfaToken    string `json:"mfaToken"`
}

// MfaEnrollBeginResponse is the body returned by /api/auth/mfa/enroll.
type MfaEnrollBeginResponse struct {
	Secret     string `json:"secret"`
	OtpauthURL string `json:"otpauthUrl"`
}

// MfaEnrollConfirmResponse is the body returned by
// /api/auth/mfa/enroll/confirm on success — recovery codes are
// shown to the user exactly once.
type MfaEnrollConfirmResponse struct {
	RecoveryCodes          []string `json:"recoveryCodes"`
	RecoveryCodesRemaining int      `json:"recoveryCodesRemaining"`
}

// MfaVerifyRequest is the body of /api/auth/mfa/verify.
type MfaVerifyRequest struct {
	MfaToken string `json:"mfaToken"`
	Code     string `json:"code"`
}

// MfaDisableRequest is the body of /api/auth/mfa/disable.
type MfaDisableRequest struct {
	UserID   string `json:"userId,omitempty"` // optional; defaults to caller
	Password string `json:"password"`
}
