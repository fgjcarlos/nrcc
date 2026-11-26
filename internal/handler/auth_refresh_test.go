package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/store"
	"github.com/google/uuid"
)

func setupAuthTest(t *testing.T) (*AuthHandler, *service.AuthService) {
	t.Helper()
	dir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](dir + "/users.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](dir + "/sessions.json")
	authSvc := service.NewAuthService("test-secret", userStore, sessionStore)

	hash, _ := authSvc.HashPassword("password123")
	user := &model.CCUser{
		ID:           uuid.New().String(),
		Username:     "admin",
		PasswordHash: hash,
		Role:         model.RoleAdmin,
		CreatedAt:    model.NowISO8601(),
		UpdatedAt:    model.NowISO8601(),
	}
	_ = authSvc.CreateUser(user)

	handler := NewAuthHandler(authSvc)
	handler.SetRateLimiter(middleware.NewRateLimiter(dir))
	return handler, authSvc
}

func TestRefresh_NoCookie(t *testing.T) {
	h, _ := setupAuthTest(t)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	h, _ := setupAuthTest(t)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	// #nosec G124 -- test fixture: incoming request cookie, not the response SetCookie (which is in production code).
	req.AddCookie(&http.Cookie{Name: "nrcc_refresh", Value: "bogus-token"})
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRefresh_ValidFlow(t *testing.T) {
	h, authSvc := setupAuthTest(t)

	user := authSvc.GetUserByUsername("admin")
	refreshToken, err := authSvc.CreateRefreshSession(user.ID)
	if err != nil {
		t.Fatalf("CreateRefreshSession: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	// #nosec G124 -- test fixture: incoming request cookie, not the response SetCookie (which is in production code).
	req.AddCookie(&http.Cookie{Name: "nrcc_refresh", Value: refreshToken})
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var envelope model.ApiResponse[AuthResponse]
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Data.Token == "" {
		t.Fatal("expected non-empty access token")
	}
	if envelope.Data.User.Username != "admin" {
		t.Fatalf("expected username admin, got %s", envelope.Data.User.Username)
	}

	// Old token should be revoked
	_, err = authSvc.ValidateRefreshSession(refreshToken)
	if err == nil {
		t.Fatal("old refresh token should be revoked after rotation")
	}

	// New cookie should be set
	cookies := rec.Result().Cookies()
	var newCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "nrcc_refresh" && c.MaxAge > 0 {
			newCookie = c
		}
	}
	if newCookie == nil {
		t.Fatal("expected new refresh cookie")
		return
	}
	if !newCookie.HttpOnly {
		t.Fatal("refresh cookie must be HttpOnly")
	}
	if newCookie.Path != "/" {
		t.Fatalf("expected cookie path /, got %s", newCookie.Path)
	}
}

func TestLogin_SetsRefreshCookie(t *testing.T) {
	h, _ := setupAuthTest(t)

	body := `{"username":"admin","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "nrcc_refresh" {
			refreshCookie = c
		}
	}
	if refreshCookie == nil {
		t.Fatal("login should set refresh cookie")
		return
	}
	if !refreshCookie.HttpOnly {
		t.Fatal("refresh cookie must be HttpOnly")
	}
}

func TestRefresh_RevokedTokenRejected(t *testing.T) {
	h, authSvc := setupAuthTest(t)

	user := authSvc.GetUserByUsername("admin")
	refreshToken, _ := authSvc.CreateRefreshSession(user.ID)
	_ = authSvc.RevokeRefreshSession(refreshToken)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	// #nosec G124 -- test fixture: incoming request cookie, not the response SetCookie (which is in production code).
	req.AddCookie(&http.Cookie{Name: "nrcc_refresh", Value: refreshToken})
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for revoked token, got %d", rec.Code)
	}
}

func TestLogout_ClearsCookieAndRevokesRefreshSession(t *testing.T) {
	h, authSvc := setupAuthTest(t)
	user := authSvc.GetUserByUsername("admin")
	refreshToken, err := authSvc.CreateRefreshSession(user.ID)
	if err != nil {
		t.Fatalf("CreateRefreshSession: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	// #nosec G124 -- test fixture: incoming request cookie, not the response SetCookie (which is in production code).
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: refreshToken})
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := authSvc.ValidateRefreshSession(refreshToken); err == nil {
		t.Fatal("logout should revoke the refresh session")
	}

	var cleared *http.Cookie
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == refreshCookieName {
			cleared = cookie
			break
		}
	}
	if cleared == nil {
		t.Fatal("logout should clear the refresh cookie")
		return
	}
	if cleared.MaxAge >= 0 || cleared.Value != "" {
		t.Fatalf("expected expired empty refresh cookie, got value=%q maxAge=%d", cleared.Value, cleared.MaxAge)
	}
}

func TestLogin_RefreshSessionCreateFailureReturnsServerError(t *testing.T) {
	dir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](dir + "/users.json")
	// Use the temp directory itself as the session store path so writes fail
	// deterministically when CreateRefreshSession tries to create <dir>.tmp.
	sessionStore := store.NewJSONStore[model.RefreshSessions](dir)
	authSvc := service.NewAuthService("test-secret", userStore, sessionStore)

	hash, err := authSvc.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	user := &model.CCUser{
		ID:           uuid.New().String(),
		Username:     "admin",
		PasswordHash: hash,
		Role:         model.RoleAdmin,
		CreatedAt:    model.NowISO8601(),
		UpdatedAt:    model.NowISO8601(),
	}
	if err := authSvc.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	NewAuthHandler(authSvc).Login(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == refreshCookieName {
			t.Fatalf("login must not set refresh cookie when session creation fails: %#v", cookie)
		}
	}
}

func TestRefresh_DeletedUserPreservesUnauthorizedResponse(t *testing.T) {
	h, svc := setupAuthTest(t)
	user := svc.GetUserByUsername("admin")
	token, err := svc.CreateRefreshSession(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteUser(user.ID); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.Refresh(rec, refreshRequest(token))
	var resp model.ApiResponse[any]
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusUnauthorized || resp.Error == nil || resp.Error.Code != "USER_NOT_FOUND" || resp.Error.Message != "User no longer exists" {
		t.Fatalf("unexpected response %d: %s", rec.Code, rec.Body)
	}
	found := false
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == refreshCookieName && cookie.Path == "/" && cookie.MaxAge == -1 && cookie.Value == "" {
			found = true
		}
	}
	if !found {
		t.Fatal("refresh cookie was not cleared")
	}
	if blocked, _ := h.limiter.Check("refresh-ip:192.0.2.1"); blocked {
		t.Fatal("deleted user should not trigger a rate limit")
	}
	for i := 0; i < 6; i++ {
		retry := httptest.NewRecorder()
		h.Refresh(retry, refreshRequest(token))
		if retry.Code != http.StatusUnauthorized || !strings.Contains(retry.Body.String(), `"code":"INVALID_REFRESH"`) {
			t.Fatalf("attempt %d: expected 401 INVALID_REFRESH, got %d %s", i, retry.Code, retry.Body)
		}
	}
	retry := httptest.NewRecorder()
	h.Refresh(retry, refreshRequest(token))
	if retry.Code != http.StatusTooManyRequests || retry.Header().Get("Retry-After") == "" {
		t.Fatalf("expected replay rate limiting, got %d %s", retry.Code, retry.Body)
	}

}
