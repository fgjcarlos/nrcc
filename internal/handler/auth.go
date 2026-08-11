package handler

import (
	"crypto/subtle"
	"errors"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fgjcarlos/nrcc/internal/audit"
	mw "github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	setupstate "github.com/fgjcarlos/nrcc/internal/setup"
	"github.com/google/uuid"
)

const (
	refreshCookieName  = "nrcc_refresh"
	refreshLockStripes = 64
)

// loginMetricsRecorder is the narrow interface for recording login metrics.
// Using an interface instead of *metrics.MetricsCollector keeps AuthHandler
// testable with simple stubs and avoids a direct dependency on the metrics package.
type loginMetricsRecorder interface {
	RecordLoginAttempt(success bool)
}

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authSvc              *service.AuthService
	mfaSvc               *service.MfaService
	audit                *audit.Service
	limiter              *mw.RateLimiter
	loginMetrics         loginMetricsRecorder
	createRefreshSession func(string) (string, error)
	refreshLocks         [refreshLockStripes]sync.Mutex
	setupMu              sync.Mutex
	setupTokenPath       string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authSvc:              authSvc,
		createRefreshSession: authSvc.CreateRefreshSession,
	}
}

// SetMfaService injects the MFA service so the login handler can
// branch on TOTP enrollment. nil is a valid value (MFA disabled).
func (h *AuthHandler) SetMfaService(m *service.MfaService) { h.mfaSvc = m }

// SetAuditService injects the audit logger.
func (h *AuthHandler) SetAuditService(a *audit.Service) { h.audit = a }

// SetRateLimiter injects the rate limiter.
func (h *AuthHandler) SetRateLimiter(rl *mw.RateLimiter) { h.limiter = rl }

// SetLoginMetrics injects the metrics recorder for login attempts.
func (h *AuthHandler) SetLoginMetrics(m loginMetricsRecorder) { h.loginMetrics = m }

// SetSetupTokenPath configures the one-time token used for authorized recovery.
func (h *AuthHandler) SetSetupTokenPath(path string) { h.setupTokenPath = path }

// SetupRequest represents the setup endpoint request
type SetupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest represents the login endpoint request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse represents the auth response with token and user
type AuthResponse struct {
	Token string             `json:"token"`
	User  model.CCUserPublic `json:"user"`
}

// StatusResponse represents the status endpoint response
type StatusResponse struct {
	Initialized bool `json:"initialized"`
}

// UserListResponse represents the users list response
type UserListResponse struct {
	Users []model.CCUserPublic `json:"users"`
}

// CreateUserRequest represents create user request
type CreateUserRequest struct {
	Username string         `json:"username"`
	Password string         `json:"password"`
	Role     model.UserRole `json:"role"`
}

// PasswordChangeRequest represents password change request
type PasswordChangeRequest struct {
	Password string `json:"password"`
}

// UpdateUserRequest represents update user request (role only)
type UpdateUserRequest struct {
	Role *model.UserRole `json:"role,omitempty"` // pointer: nil means "not provided"
}

// Setup handles POST /api/auth/setup - initial admin user creation
// Only works when no users exist
func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	if h.limiter != nil {
		ip := mw.ExtractIP(r)
		if blocked, retry := h.limiter.Check("setup-ip:" + ip); blocked {
			mw.RespondTooManyRequests(w, retry)
			return
		}
	}

	var req SetupRequest
	if !DecodeJSON(w, r, &req) {
		return
	}
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))

	// Validate input
	if req.Username == "" || req.Password == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Username and password are required")
		return
	}

	if err := service.ValidatePassword(req.Password); err != nil {
		model.RespondError(w, http.StatusBadRequest, "WEAK_PASSWORD", err.Error())
		return
	}

	h.setupMu.Lock()
	defer h.setupMu.Unlock()

	users, err := h.authSvc.GetAllUsers()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		model.RespondError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to inspect user configuration")
		return
	}
	configured := len(users) > 0
	var recoveryToken setupstate.SetupToken
	if configured {
		if h.setupTokenPath == "" {
			h.respondAlreadyConfigured(w, r)
			return
		}
		recoveryToken, err = setupstate.ReadTokenFile(h.setupTokenPath)
		if err != nil || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Setup-Reset-Token")), []byte(recoveryToken.Raw)) != 1 {
			h.respondAlreadyConfigured(w, r)
			return
		}
		if err := setupstate.ConsumeTokenFile(h.setupTokenPath); err != nil {
			model.RespondError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to claim setup recovery token")
			return
		}
	}

	// Create the requested administrator.
	hash, err := h.authSvc.HashPassword(req.Password)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "HASH_ERROR", "Failed to hash password")
		return
	}

	now := model.NowISO8601()
	user := &model.CCUser{
		ID:           uuid.New().String(),
		Username:     req.Username,
		PasswordHash: hash,
		Role:         model.RoleAdmin,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	createUser := h.authSvc.BootstrapFirstAdmin
	if configured {
		createUser = h.authSvc.CreateUser
	}
	if err := createUser(user); err != nil {
		if configured {
			_ = setupstate.WriteTokenFile(h.setupTokenPath, recoveryToken)
		}
		if errors.Is(err, service.ErrAlreadyConfigured) {
			h.respondAlreadyConfigured(w, r)
			return
		}
		model.RespondError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to create user")
		return
	}

	// Generate access token
	token, err := h.authSvc.GenerateToken(user)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
		return
	}

	// Issue refresh cookie
	if err := h.setRefreshCookie(w, r, user.ID); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "REFRESH_SESSION_ERROR", "Failed to create refresh session")
		return
	}

	resp := AuthResponse{
		Token: token,
		User: model.CCUserPublic{
			ID:        user.ID,
			Username:  user.Username,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}

	h.audit.Log(r, req.Username, "SYSTEM_SETUP", "", "ok", nil)
	model.RespondJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) respondAlreadyConfigured(w http.ResponseWriter, r *http.Request) {
	if h.limiter != nil {
		h.limiter.Record("setup-ip:" + mw.ExtractIP(r))
	}
	model.RespondError(w, http.StatusConflict, "ALREADY_CONFIGURED", "System already configured with users")
}

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ip := mw.ExtractIP(r)

	if h.limiter != nil {
		if blocked, retry := h.limiter.Check(mw.AuthIPKey(ip)); blocked {
			mw.RespondTooManyRequests(w, retry)
			return
		}
	}

	var req LoginRequest
	if !DecodeJSON(w, r, &req) {
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Username and password are required")
		return
	}

	if h.limiter != nil {
		if blocked, retry := h.limiter.Check(mw.AuthUserKey(req.Username)); blocked {
			mw.RespondTooManyRequests(w, retry)
			return
		}
	}

	// Find user
	user := h.authSvc.GetUserByUsername(req.Username)
	if user == nil {
		if h.limiter != nil {
			h.limiter.Record(mw.AuthIPKey(ip))
			h.limiter.Record(mw.AuthUserKey(req.Username))
		}
		if h.loginMetrics != nil {
			h.loginMetrics.RecordLoginAttempt(false)
		}
		h.audit.Log(r, req.Username, "LOGIN", "", "fail", map[string]string{"reason": "unknown_user"})
		model.RespondError(w, http.StatusUnauthorized, "AUTH_FAILED", "Invalid username or password")
		return
	}

	// Verify password
	if !h.authSvc.VerifyPassword(user.PasswordHash, req.Password) {
		if h.limiter != nil {
			h.limiter.Record(mw.AuthIPKey(ip))
			h.limiter.Record(mw.AuthUserKey(req.Username))
		}
		if h.loginMetrics != nil {
			h.loginMetrics.RecordLoginAttempt(false)
		}
		h.audit.Log(r, req.Username, "LOGIN", "", "fail", map[string]string{"reason": "bad_password"})
		model.RespondError(w, http.StatusUnauthorized, "AUTH_FAILED", "Invalid username or password")
		return
	}

	// Successful login — reset rate limit counters
	if h.limiter != nil {
		h.limiter.Reset(mw.AuthIPKey(ip))
		h.limiter.Reset(mw.AuthUserKey(req.Username))
	}

	// Rehash if stored with lower bcrypt cost
	if service.NeedsRehash(user.PasswordHash) {
		if newHash, err := h.authSvc.HashPassword(req.Password); err == nil {
			now := model.NowISO8601()
			if err := h.authSvc.UpdateUser(user.ID, func(current *model.CCUser) error {
				current.PasswordHash = newHash
				current.UpdatedAt = now
				return nil
			}); err == nil {
				user.PasswordHash = newHash
				user.UpdatedAt = now
			}
		}
	}

	// MFA branch: if the user is enrolled in TOTP, do NOT issue a
	// session yet — return mfaRequired + a short-lived mfaToken
	// and let the client call /api/auth/mfa/verify.
	if h.mfaSvc != nil && h.mfaSvc.IsEnrolled(user.ID) {
		mfaToken, err := h.mfaSvc.IssueMfaToken(user.ID)
		if err != nil {
			model.RespondError(w, http.StatusInternalServerError, "MFA_TOKEN_ERROR", "Failed to issue MFA challenge")
			return
		}
		if h.loginMetrics != nil {
			h.loginMetrics.RecordLoginAttempt(true)
		}
		h.audit.Log(r, req.Username, "LOGIN", "", "ok", map[string]string{"stage": "password", "mfa": "required"})
		model.RespondJSON(w, http.StatusOK, model.MfaLoginResponse{
			MfaRequired: true,
			MfaToken:    mfaToken,
		})
		return
	}

	// Generate access token
	token, err := h.authSvc.GenerateToken(user)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
		return
	}

	// Issue refresh cookie
	if err := h.setRefreshCookie(w, r, user.ID); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "REFRESH_SESSION_ERROR", "Failed to create refresh session")
		return
	}

	resp := AuthResponse{
		Token: token,
		User: model.CCUserPublic{
			ID:        user.ID,
			Username:  user.Username,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}

	if h.loginMetrics != nil {
		h.loginMetrics.RecordLoginAttempt(true)
	}
	h.audit.Log(r, req.Username, "LOGIN", "", "ok", nil)
	model.RespondJSON(w, http.StatusOK, resp)
}

// GetStatus handles GET /api/auth/status - public endpoint
func (h *AuthHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	users, _ := h.authSvc.GetAllUsers()
	resp := StatusResponse{
		Initialized: len(users) > 0,
	}

	model.RespondJSON(w, http.StatusOK, resp)
}

// GetMe handles GET /api/auth/me - protected endpoint
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims := mw.ClaimsFromContext(r)
	if claims == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not found in context")
		return
	}

	user := h.authSvc.GetUserByID(claims.UserID)
	if user == nil {
		model.RespondError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	resp := model.CCUserPublic{
		ID:        user.ID,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	// Embed MFA status so the UI does not need a second round-trip
	// on page load. If MFA is disabled (mfaSvc == nil) we report
	// { enabled: false, recoveryCodesRemaining: 0 }.
	mfa := model.MfaStatusResponse{Enabled: false, RecoveryCodesRemaining: 0}
	if h.mfaSvc != nil {
		if s, err := h.mfaSvc.Status(user.ID); err == nil {
			mfa = s
		}
	}
	model.RespondJSON(w, http.StatusOK, struct {
		model.CCUserPublic
		Mfa model.MfaStatusResponse `json:"mfa"`
	}{resp, mfa})
}

// Logout handles POST /api/auth/logout - protected endpoint
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(refreshCookieName); err == nil && cookie.Value != "" {
		if err := h.authSvc.RevokeRefreshSession(cookie.Value); err != nil {
			model.RespondError(w, http.StatusInternalServerError, "LOGOUT_ERROR", "Failed to revoke refresh session")
			return
		}
	}
	clearRefreshCookie(w, r)
	model.RespondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GetUsers handles GET /api/auth/users - protected, admin only
func (h *AuthHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	claims := mw.ClaimsFromContext(r)
	if claims == nil || claims.Role != model.RoleAdmin {
		model.RespondError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	users, err := h.authSvc.GetAllUsers()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "FETCH_ERROR", "Failed to fetch users")
		return
	}

	// Convert to public representation (without password hashes)
	publicUsers := make([]model.CCUserPublic, len(users))
	for i, u := range users {
		publicUsers[i] = model.CCUserPublic{
			ID:        u.ID,
			Username:  u.Username,
			Role:      u.Role,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
		}
	}

	resp := UserListResponse{Users: publicUsers}
	model.RespondJSON(w, http.StatusOK, resp)
}

// CreateUser handles POST /api/auth/users - protected, admin only
func (h *AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	claims := mw.ClaimsFromContext(r)
	if claims == nil || claims.Role != model.RoleAdmin {
		model.RespondError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	var req CreateUserRequest
	if !DecodeJSON(w, r, &req) {
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Username and password are required")
		return
	}

	if req.Role != model.RoleAdmin && req.Role != model.RoleViewer {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Role must be 'admin' or 'viewer'")
		return
	}

	if err := service.ValidatePassword(req.Password); err != nil {
		model.RespondError(w, http.StatusBadRequest, "WEAK_PASSWORD", err.Error())
		return
	}

	// Hash password
	hash, err := h.authSvc.HashPassword(req.Password)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "HASH_ERROR", "Failed to hash password")
		return
	}

	now := model.NowISO8601()
	newUser := &model.CCUser{
		ID:           uuid.New().String(),
		Username:     req.Username,
		PasswordHash: hash,
		Role:         req.Role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.authSvc.CreateUser(newUser); err != nil {
		if errors.Is(err, service.ErrUsernameExists) {
			model.RespondError(w, http.StatusConflict, "USERNAME_EXISTS", "Username already exists")
			return
		}
		model.RespondError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to create user")
		return
	}

	resp := model.CCUserPublic{
		ID:        newUser.ID,
		Username:  newUser.Username,
		Role:      newUser.Role,
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
	}

	h.audit.Log(r, claims.Username, "USER_CREATE", req.Username, "ok", map[string]string{"role": string(req.Role)})
	model.RespondJSON(w, http.StatusCreated, resp)
}

// DeleteUser handles DELETE /api/auth/users/:id - protected, admin only
func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	claims := mw.ClaimsFromContext(r)
	if claims == nil || claims.Role != model.RoleAdmin {
		model.RespondError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	userID := r.PathValue("id")
	if userID == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "User ID is required")
		return
	}

	// Cannot delete self
	if userID == claims.UserID {
		model.RespondError(w, http.StatusBadRequest, "CANNOT_DELETE_SELF", "Cannot delete your own user")
		return
	}

	user, err := h.authSvc.DeleteUserWithPolicy(userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			model.RespondError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		case errors.Is(err, service.ErrCannotDeleteLastAdmin):
			model.RespondError(w, http.StatusForbidden, "CANNOT_DELETE_LAST_ADMIN", "Cannot delete the last admin user")
		default:
			model.RespondError(w, http.StatusInternalServerError, "DELETE_ERROR", "Failed to delete user")
		}
		return
	}

	h.audit.Log(r, claims.Username, "USER_DELETE", user.Username, "ok", nil)
	model.RespondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ChangePassword handles PATCH /api/auth/users/:id/password - protected, admin or self
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := mw.ClaimsFromContext(r)
	if claims == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	userID := r.PathValue("id")
	if userID == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "User ID is required")
		return
	}

	// Only admin or self can change password
	if claims.Role != model.RoleAdmin && claims.UserID != userID {
		model.RespondError(w, http.StatusForbidden, "FORBIDDEN", "You can only change your own password")
		return
	}

	var req PasswordChangeRequest
	if !DecodeJSON(w, r, &req) {
		return
	}

	if req.Password == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Password is required")
		return
	}

	if err := service.ValidatePassword(req.Password); err != nil {
		model.RespondError(w, http.StatusBadRequest, "WEAK_PASSWORD", err.Error())
		return
	}

	// Hash new password
	hash, err := h.authSvc.HashPassword(req.Password)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "HASH_ERROR", "Failed to hash password")
		return
	}

	updatedAt := model.NowISO8601()
	username := ""
	if err := h.authSvc.UpdateUser(userID, func(user *model.CCUser) error {
		user.PasswordHash = hash
		user.UpdatedAt = updatedAt
		username = user.Username
		return nil
	}); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			model.RespondError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}
		model.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update user")
		return
	}

	h.audit.Log(r, claims.Username, "PASSWORD_CHANGE", username, "ok", nil)
	model.RespondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// UpdateUser handles PATCH /api/auth/users/:id - protected, admin only
func (h *AuthHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	claims := mw.ClaimsFromContext(r)
	if claims == nil || claims.Role != model.RoleAdmin {
		model.RespondError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	userID := r.PathValue("id")
	if userID == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "User ID is required")
		return
	}

	var req UpdateUserRequest
	if !DecodeJSON(w, r, &req) {
		return
	}

	// Validate: at least one field must be provided
	if req.Role == nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "At least one field (role) must be provided")
		return
	}

	// Validate role value
	if *req.Role != model.RoleAdmin && *req.Role != model.RoleViewer {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Role must be 'admin' or 'viewer'")
		return
	}

	user, err := h.authSvc.UpdateUserRole(userID, *req.Role, model.NowISO8601())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			model.RespondError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		case errors.Is(err, service.ErrCannotDemoteLastAdmin):
			model.RespondError(w, http.StatusForbidden, "CANNOT_DEMOTE_LAST_ADMIN", "Cannot demote the last admin user")
		default:
			model.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update user")
		}
		return
	}

	h.audit.Log(r, claims.Username, "USER_UPDATE", user.Username, "ok", map[string]string{"role": string(user.Role)})

	// Return updated user
	resp := model.CCUserPublic{
		ID:        user.ID,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	model.RespondJSON(w, http.StatusOK, resp)
}

// Refresh handles POST /api/auth/refresh — public endpoint.
// Reads the httpOnly refresh cookie, validates the session, rotates
// the refresh token, and returns a new short-lived access token.
// Rate limiting mitigates brute-force attempts and amplification; it does not
// prevent replay of a stolen bearer token. Proactive compromise response uses
// /api/auth/logout-everywhere or AuthService.RevokeUserSessions. Refresh-token
// comparison is not constant-time; that is accepted here for random 256-bit tokens.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ip := mw.ExtractIP(r)
	refreshLock := &h.refreshLocks[refreshLockIndex(ip)]
	refreshLock.Lock()
	defer refreshLock.Unlock()

	// Keep Check and a possible Record/Reset in one transaction. RateLimiter's
	// individual methods are thread-safe, but a split Check/Record sequence is
	// otherwise vulnerable to concurrent requests overshooting the threshold.
	key := "refresh-ip:" + ip
	if h.limiter != nil {
		if blocked, retryAfter := h.limiter.Check(key); blocked {
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
	}

	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		if h.limiter != nil {
			h.limiter.Record(key)
		}
		model.RespondError(w, http.StatusUnauthorized, "NO_REFRESH_TOKEN", "Refresh token missing")
		return
	}

	sess, err := h.authSvc.ValidateRefreshSession(cookie.Value)
	if err != nil {
		if h.limiter != nil {
			h.limiter.Record(key)
		}
		clearRefreshCookie(w, r)
		model.RespondError(w, http.StatusUnauthorized, "INVALID_REFRESH", "Refresh token invalid or expired")
		return
	}

	// Rotate: revoke old, issue new.
	if err := h.authSvc.RevokeRefreshSession(cookie.Value); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "REFRESH_REVOKE_ERROR", "Failed to revoke refresh session")
		return
	}

	user := h.authSvc.GetUserByID(sess.UserID)
	if user == nil {
		clearRefreshCookie(w, r)
		model.RespondError(w, http.StatusUnauthorized, "USER_NOT_FOUND", "User no longer exists")
		return
	}

	token, err := h.authSvc.GenerateToken(user)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
		return
	}

	if err := h.setRefreshCookie(w, r, user.ID); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "REFRESH_SESSION_ERROR", "Failed to create refresh session")
		return
	}

	response := AuthResponse{
		Token: token,
		User: model.CCUserPublic{
			ID:        user.ID,
			Username:  user.Username,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}
	if h.limiter != nil {
		h.limiter.Reset(key)
	}
	model.RespondJSON(w, http.StatusOK, response)
}

func refreshLockIndex(ip string) uint32 {
	const (
		fnvOffset32 = uint32(2166136261)
		fnvPrime32  = uint32(16777619)
	)

	hash := fnvOffset32
	for i := 0; i < len(ip); i++ {
		hash ^= uint32(ip[i])
		hash *= fnvPrime32
	}
	return hash % refreshLockStripes
}

// isSecureRequest reports whether the cookie Secure flag should be set.
// nrcc serves plain HTTP; TLS is terminated upstream by the Portless proxy,
// which forwards X-Forwarded-Proto: https. A direct TLS connection (r.TLS != nil)
// is also treated as secure for local development with TLS.
func isSecureRequest(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, r *http.Request, userID string) error {
	refreshToken, err := h.createRefreshSession(userID)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:  refreshCookieName,
		Value: refreshToken,
		// Path is "/" so the cookie rides every request, not just
		// /api/auth/*. Required when the backend is behind a reverse
		// proxy (e.g. Tailscale Serve, Portless) that may rewrite the
		// Host header upstream — a narrower Path combined with a
		// rewritten Host made the cookie's effective host mismatch
		// the browser's origin and dropped the cookie on F5.
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(service.RefreshTokenLifetime / time.Second),
	})
	return nil
}

func clearRefreshCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}
