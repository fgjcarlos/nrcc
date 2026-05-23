package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/composedof2/nrcc/internal/audit"
	mw "github.com/composedof2/nrcc/internal/middleware"
	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
	"github.com/google/uuid"
)

const refreshCookieName = "nrcc_refresh"

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authSvc *service.AuthService
	audit   *audit.Service
	limiter *mw.RateLimiter
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// SetAuditService injects the audit logger.
func (h *AuthHandler) SetAuditService(a *audit.Service) { h.audit = a }

// SetRateLimiter injects the rate limiter.
func (h *AuthHandler) SetRateLimiter(rl *mw.RateLimiter) { h.limiter = rl }

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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Username and password are required")
		return
	}

	if err := service.ValidatePassword(req.Password); err != nil {
		model.RespondError(w, http.StatusBadRequest, "WEAK_PASSWORD", err.Error())
		return
	}

	// Check if users already exist (setup only works once)
	users, _ := h.authSvc.GetAllUsers()
	if len(users) > 0 {
		if h.limiter != nil {
			h.limiter.Record("setup-ip:" + mw.ExtractIP(r))
		}
		model.RespondError(w, http.StatusConflict, "ALREADY_CONFIGURED", "System already configured with users")
		return
	}

	// Create first user as admin
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

	// Save user
	if err := h.authSvc.CreateUser(user); err != nil {
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
	h.setRefreshCookie(w, user.ID)

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

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ip := mw.ExtractIP(r)

	if h.limiter != nil {
		if blocked, retry := h.limiter.Check("ip:" + ip); blocked {
			mw.RespondTooManyRequests(w, retry)
			return
		}
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Username and password are required")
		return
	}

	if h.limiter != nil {
		if blocked, retry := h.limiter.Check("user:" + req.Username); blocked {
			mw.RespondTooManyRequests(w, retry)
			return
		}
	}

	// Find user
	user := h.authSvc.GetUserByUsername(req.Username)
	if user == nil {
		if h.limiter != nil {
			h.limiter.Record("ip:" + ip)
			h.limiter.Record("user:" + req.Username)
		}
		h.audit.Log(r, req.Username, "LOGIN", "", "fail", map[string]string{"reason": "unknown_user"})
		model.RespondError(w, http.StatusUnauthorized, "AUTH_FAILED", "Invalid username or password")
		return
	}

	// Verify password
	if !h.authSvc.VerifyPassword(user.PasswordHash, req.Password) {
		if h.limiter != nil {
			h.limiter.Record("ip:" + ip)
			h.limiter.Record("user:" + req.Username)
		}
		h.audit.Log(r, req.Username, "LOGIN", "", "fail", map[string]string{"reason": "bad_password"})
		model.RespondError(w, http.StatusUnauthorized, "AUTH_FAILED", "Invalid username or password")
		return
	}

	// Successful login — reset rate limit counters
	if h.limiter != nil {
		h.limiter.Reset("ip:" + ip)
		h.limiter.Reset("user:" + req.Username)
	}

	// Rehash if stored with lower bcrypt cost
	if service.NeedsRehash(user.PasswordHash) {
		if newHash, err := h.authSvc.HashPassword(req.Password); err == nil {
			user.PasswordHash = newHash
			_ = h.authSvc.UpdateUser(user)
		}
	}

	// Generate access token
	token, err := h.authSvc.GenerateToken(user)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
		return
	}

	// Issue refresh cookie
	h.setRefreshCookie(w, user.ID)

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

	model.RespondJSON(w, http.StatusOK, resp)
}

// Logout handles POST /api/auth/logout - protected endpoint
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// JWT logout is client-side (token invalidation). We just return success.
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
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

	// Check if username already exists
	existing := h.authSvc.GetUserByUsername(req.Username)
	if existing != nil {
		model.RespondError(w, http.StatusConflict, "USERNAME_EXISTS", "Username already exists")
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

	// Check if user exists
	user := h.authSvc.GetUserByID(userID)
	if user == nil {
		model.RespondError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	// Cannot delete last admin
	if user.Role == model.RoleAdmin {
		users, _ := h.authSvc.GetAllUsers()
		adminCount := 0
		for _, u := range users {
			if u.Role == model.RoleAdmin {
				adminCount++
			}
		}
		if adminCount <= 1 {
			model.RespondError(w, http.StatusBadRequest, "CANNOT_DELETE_LAST_ADMIN", "Cannot delete the last admin user")
			return
		}
	}

	if err := h.authSvc.DeleteUser(userID); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "DELETE_ERROR", "Failed to delete user")
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
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

	// Get user
	user := h.authSvc.GetUserByID(userID)
	if user == nil {
		model.RespondError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	// Hash new password
	hash, err := h.authSvc.HashPassword(req.Password)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "HASH_ERROR", "Failed to hash password")
		return
	}

	// Update user
	user.PasswordHash = hash
	user.UpdatedAt = model.NowISO8601()

	if err := h.authSvc.UpdateUser(user); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update user")
		return
	}

	h.audit.Log(r, claims.Username, "PASSWORD_CHANGE", user.Username, "ok", nil)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
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

	// Get user
	user := h.authSvc.GetUserByID(userID)
	if user == nil {
		model.RespondError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	// Last-admin guard: cannot demote the sole admin
	if *req.Role == model.RoleViewer && user.Role == model.RoleAdmin {
		users, _ := h.authSvc.GetAllUsers()
		adminCount := 0
		for _, u := range users {
			if u.Role == model.RoleAdmin {
				adminCount++
			}
		}
		if adminCount <= 1 {
			model.RespondError(w, http.StatusForbidden, "CANNOT_DEMOTE_LAST_ADMIN", "Cannot demote the last admin user")
			return
		}
	}

	// Update user role
	user.Role = *req.Role
	user.UpdatedAt = model.NowISO8601()

	if err := h.authSvc.UpdateUser(user); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update user")
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
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		model.RespondError(w, http.StatusUnauthorized, "NO_REFRESH_TOKEN", "Refresh token missing")
		return
	}

	sess, err := h.authSvc.ValidateRefreshSession(cookie.Value)
	if err != nil {
		clearRefreshCookie(w)
		model.RespondError(w, http.StatusUnauthorized, "INVALID_REFRESH", "Refresh token invalid or expired")
		return
	}

	// Rotate: revoke old, issue new.
	_ = h.authSvc.RevokeRefreshSession(cookie.Value)

	user := h.authSvc.GetUserByID(sess.UserID)
	if user == nil {
		clearRefreshCookie(w)
		model.RespondError(w, http.StatusUnauthorized, "USER_NOT_FOUND", "User no longer exists")
		return
	}

	token, err := h.authSvc.GenerateToken(user)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
		return
	}

	h.setRefreshCookie(w, user.ID)

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

func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, userID string) {
	refreshToken, err := h.authSvc.CreateRefreshSession(userID)
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refreshToken,
		Path:     "/api/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(service.RefreshTokenLifetime / time.Second),
	})
}

func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}
