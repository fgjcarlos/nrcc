package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/fgjcarlos/nrcc/internal/audit"
	mw "github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// MfaHandler owns the /api/auth/mfa/* endpoints.
type MfaHandler struct {
	mfaSvc  *service.MfaService
	authSvc *service.AuthService
	audit   *audit.Service
	limiter *mw.RateLimiter
}

// NewMfaHandler wires the MFA handler. The authSvc is the same
// instance used by the rest of the auth subsystem so login-time
// branching (token + refresh cookie) stays consistent.
func NewMfaHandler(mfaSvc *service.MfaService, authSvc *service.AuthService) *MfaHandler {
	return &MfaHandler{mfaSvc: mfaSvc, authSvc: authSvc}
}

// SetAuditService injects the audit logger.
func (h *MfaHandler) SetAuditService(a *audit.Service) { h.audit = a }

// SetRateLimiter injects the rate limiter (shared with the auth
// handler). Used to protect POST /mfa/verify from brute force.
func (h *MfaHandler) SetRateLimiter(rl *mw.RateLimiter) { h.limiter = rl }

// Enroll handles POST /api/auth/mfa/enroll — auth required.
// Begins a pending enrollment and returns the candidate secret +
// otpauth URL. Does not commit the enrollment.
func (h *MfaHandler) Enroll(w http.ResponseWriter, r *http.Request) {
	claims := mw.ClaimsFromContext(r)
	if claims == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	secret, otpauthURL, err := h.mfaSvc.BeginEnrollment(claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMfaAlreadyEnrolled):
			model.RespondError(w, http.StatusConflict, "MFA_ALREADY_ENROLLED", "User is already enrolled in MFA")
		case errors.Is(err, service.ErrMfaUserNotFound):
			model.RespondError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		default:
			model.RespondError(w, http.StatusInternalServerError, "MFA_ENROLL_ERROR", "Failed to start MFA enrollment")
		}
		return
	}

	h.audit.Log(r, claims.Username, "MFA_ENROLL", claims.Username, "ok", map[string]string{"phase": "begin"})
	model.RespondJSON(w, http.StatusOK, model.MfaEnrollBeginResponse{
		Secret:     secret,
		OtpauthURL: otpauthURL,
	})
}

// EnrollConfirm handles POST /api/auth/mfa/enroll/confirm — auth
// required. Verifies the first TOTP code from the user's
// authenticator and commits the enrollment.
func (h *MfaHandler) EnrollConfirm(w http.ResponseWriter, r *http.Request) {
	claims := mw.ClaimsFromContext(r)
	if claims == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Code is required")
		return
	}

	codes, err := h.mfaSvc.ConfirmEnrollment(claims.UserID, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMfaInvalidCode):
			h.audit.Log(r, claims.Username, "MFA_ENROLL", claims.Username, "fail", map[string]string{"phase": "confirm", "reason": "bad_code"})
			model.RespondError(w, http.StatusBadRequest, "INVALID_CODE", "The provided code is not valid")
		case errors.Is(err, service.ErrMfaAlreadyEnrolled):
			model.RespondError(w, http.StatusConflict, "MFA_ALREADY_ENROLLED", "User is already enrolled in MFA")
		case errors.Is(err, service.ErrMfaPendingMissing):
			model.RespondError(w, http.StatusBadRequest, "MFA_NOT_ENROLLING", "No enrollment in progress — call /api/auth/mfa/enroll first")
		default:
			model.RespondError(w, http.StatusInternalServerError, "MFA_ENROLL_ERROR", "Failed to confirm MFA enrollment")
		}
		return
	}

	h.audit.Log(r, claims.Username, "MFA_ENROLL", claims.Username, "ok", map[string]string{"phase": "confirm"})
	model.RespondJSON(w, http.StatusOK, model.MfaEnrollConfirmResponse{
		RecoveryCodes:          codes,
		RecoveryCodesRemaining: len(codes),
	})
}

// Disable handles POST /api/auth/mfa/disable — auth required.
// Body: { password: "..." } (the calling user's password). For
// admins, the optional { userId: "..." } field targets another user.
func (h *MfaHandler) Disable(w http.ResponseWriter, r *http.Request) {
	claims := mw.ClaimsFromContext(r)
	if claims == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var req model.MfaDisableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	if req.Password == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Password is required")
		return
	}

	target := claims.UserID
	if req.UserID != "" && req.UserID != target {
		if claims.Role != model.RoleAdmin {
			model.RespondError(w, http.StatusForbidden, "FORBIDDEN", "Only admins can disable MFA for another user")
			return
		}
		target = req.UserID
	}

	isAdmin := claims.Role == model.RoleAdmin
	err := h.mfaSvc.Disable(claims.UserID, target, req.Password, isAdmin)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMfaInvalidPassword):
			h.audit.Log(r, claims.Username, "MFA_DISABLE", target, "fail", map[string]string{"reason": "bad_password"})
			model.RespondError(w, http.StatusUnauthorized, "INVALID_PASSWORD", "Current password is incorrect")
		case errors.Is(err, service.ErrMfaNotEnrolled):
			model.RespondError(w, http.StatusConflict, "MFA_NOT_ENROLLED", "Target user is not enrolled in MFA")
		case errors.Is(err, service.ErrMfaUserNotFound):
			model.RespondError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		default:
			// includes "forbidden" path from the service
			if strings.HasPrefix(err.Error(), "forbidden") {
				model.RespondError(w, http.StatusForbidden, "FORBIDDEN", "Only admins can disable MFA for another user")
				return
			}
			model.RespondError(w, http.StatusInternalServerError, "MFA_DISABLE_ERROR", "Failed to disable MFA")
		}
		return
	}

	h.audit.Log(r, claims.Username, "MFA_DISABLE", target, "ok", nil)
	model.RespondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// Status handles GET /api/auth/mfa/status — auth required.
func (h *MfaHandler) Status(w http.ResponseWriter, r *http.Request) {
	claims := mw.ClaimsFromContext(r)
	if claims == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	status, err := h.mfaSvc.Status(claims.UserID)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "MFA_STATUS_ERROR", "Failed to read MFA status")
		return
	}
	model.RespondJSON(w, http.StatusOK, status)
}

// Verify handles POST /api/auth/mfa/verify — public (mfaToken
// acts as the proof of first factor). Body: { mfaToken, code }.
// On success, issues a session exactly like /api/auth/login does.
func (h *MfaHandler) Verify(w http.ResponseWriter, r *http.Request) {
	ip := mw.ExtractIP(r)

	// Rate limit by IP even before decoding — we want a single
	// shared bucket with the rest of the auth surface.
	if h.limiter != nil {
		if blocked, retry := h.limiter.Check("mfa-verify-ip:" + ip); blocked {
			mw.RespondTooManyRequests(w, retry)
			return
		}
	}

	var req model.MfaVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	if req.MfaToken == "" || req.Code == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "mfaToken and code are required")
		return
	}

	// Look up the username for rate-limiting per-user and audit.
	username := h.mfaSvc.LookupUsernameByMfaToken(req.MfaToken)
	if username != "" && h.limiter != nil {
		if blocked, retry := h.limiter.Check("mfa-verify-user:" + username); blocked {
			mw.RespondTooManyRequests(w, retry)
			return
		}
	}

	user, method, err := h.mfaSvc.VerifyAndIssueSession(req.MfaToken, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMfaTokenInvalid):
			h.audit.Log(r, username, "MFA_VERIFY", username, "fail", map[string]string{"reason": "bad_token"})
			model.RespondError(w, http.StatusUnauthorized, "MFA_TOKEN_INVALID", "MFA token is invalid")
		case errors.Is(err, service.ErrMfaTokenExpired):
			h.audit.Log(r, username, "MFA_VERIFY", username, "fail", map[string]string{"reason": "expired"})
			model.RespondError(w, http.StatusUnauthorized, "MFA_TOKEN_EXPIRED", "MFA token has expired — please log in again")
		case errors.Is(err, service.ErrMfaTokenUsed):
			h.audit.Log(r, username, "MFA_VERIFY", username, "fail", map[string]string{"reason": "used"})
			model.RespondError(w, http.StatusUnauthorized, "MFA_TOKEN_INVALID", "MFA token has already been used")
		case errors.Is(err, service.ErrMfaInvalidCode):
			if h.limiter != nil {
				if username != "" {
					h.limiter.Record("mfa-verify-user:" + username)
				}
				h.limiter.Record("mfa-verify-ip:" + ip)
			}
			h.audit.Log(r, username, "MFA_VERIFY", username, "fail", map[string]string{"reason": "bad_code"})
			model.RespondError(w, http.StatusUnauthorized, "INVALID_CODE", "The provided code is not valid")
		case errors.Is(err, service.ErrMfaUserNotFound):
			model.RespondError(w, http.StatusUnauthorized, "USER_NOT_FOUND", "User no longer exists")
		default:
			model.RespondError(w, http.StatusInternalServerError, "MFA_VERIFY_ERROR", "Failed to verify MFA code")
		}
		return
	}

	// Reset the rate limit on success.
	if h.limiter != nil {
		h.limiter.Reset("mfa-verify-ip:" + ip)
		if username != "" {
			h.limiter.Reset("mfa-verify-user:" + username)
		}
	}

	// Issue access token + refresh cookie (mirrors AuthHandler.Login).
	token, err := h.authSvc.GenerateToken(user)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
		return
	}
	if err := h.setRefreshCookieFor(w, r, user.ID); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "REFRESH_SESSION_ERROR", "Failed to create refresh session")
		return
	}

	h.audit.Log(r, user.Username, "MFA_VERIFY", user.Username, "ok", map[string]string{"method": method})
	if method == "recovery" {
		h.audit.Log(r, user.Username, "MFA_RECOVERY_USE", user.Username, "ok", nil)
	}

	model.RespondJSON(w, http.StatusOK, AuthResponse{
		Token: token,
		User: model.CCUserPublic{
			ID:        user.ID,
			Username:  user.Username,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	})
}

// setRefreshCookieFor is a thin wrapper that lets the MFA handler
// emit the same shape of refresh cookie the auth handler does,
// without duplicating the cookie definition.
func (h *MfaHandler) setRefreshCookieFor(w http.ResponseWriter, r *http.Request, userID string) error {
	refreshToken, err := h.authSvc.CreateRefreshSession(userID)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refreshToken,
		Path:     "/api/auth",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(service.RefreshTokenLifetime / time.Second),
	})
	return nil
}
