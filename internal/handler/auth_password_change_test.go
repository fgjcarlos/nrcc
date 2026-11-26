package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/store"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
)

// #nosec G101 -- test fixtures, not real credentials.
const (
	pwChangeOld = "password123"
	pwChangeNew = "brand-new-password-456"
)

func changePassword(t *testing.T, h *AuthHandler, claims *model.Claims, targetID string, body map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := requestWithClaims(http.MethodPatch, "/api/auth/users/"+targetID+"/password", raw, claims)
	req.SetPathValue("id", targetID)
	rec := httptest.NewRecorder()
	h.ChangePassword(rec, req)
	return rec
}

func refreshWith(h *AuthHandler, token string) int {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	// #nosec G124 -- test fixture: incoming request cookie.
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: token})
	rec := httptest.NewRecorder()
	h.Refresh(rec, req)
	return rec.Code
}

func responseRefreshCookie(rec *httptest.ResponseRecorder) string {
	for _, c := range rec.Result().Cookies() {
		if c.Name == refreshCookieName {
			return c.Value
		}
	}
	return ""
}

func adminClaims(user *model.CCUser) *model.Claims {
	return &model.Claims{UserID: user.ID, Username: user.Username, Role: user.Role}
}

func addViewer(t *testing.T, authSvc *service.AuthService) *model.CCUser {
	t.Helper()
	hash, _ := authSvc.HashPassword(pwChangeOld)
	viewer := &model.CCUser{
		ID: uuid.New().String(), Username: "viewer", PasswordHash: hash, Role: model.RoleViewer,
		CreatedAt: model.NowISO8601(), UpdatedAt: model.NowISO8601(),
	}
	if err := authSvc.CreateUser(viewer); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	return viewer
}

func TestChangePassword_SelfWithoutCurrentPassword_Returns401AndKeepsHash(t *testing.T) {
	h, authSvc := setupAuthTest(t)
	user := authSvc.GetUserByUsername("admin")
	before := user.PasswordHash
	session, _ := authSvc.CreateRefreshSession(user.ID)

	for name, body := range map[string]map[string]string{
		"missing": {"password": pwChangeNew},
		"wrong":   {"password": pwChangeNew, "currentPassword": "not-the-password"},
	} {
		t.Run(name, func(t *testing.T) {
			rec := changePassword(t, h, adminClaims(user), user.ID, body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("want 401, got %d (%s)", rec.Code, rec.Body.String())
			}
			var resp model.ApiResponse[any]
			_ = json.Unmarshal(rec.Body.Bytes(), &resp)
			if resp.Error == nil || resp.Error.Code != "INVALID_PASSWORD" {
				t.Fatalf("want INVALID_PASSWORD, got %+v", resp)
			}
		})
	}

	if got := authSvc.GetUserByID(user.ID).PasswordHash; got != before {
		t.Fatal("password hash changed after a rejected self-service change")
	}
	if code := refreshWith(h, session); code != http.StatusOK {
		t.Fatalf("rejected change must not revoke sessions; refresh got %d", code)
	}
}

func TestChangePassword_SelfWrongCurrentPassword_FeedsLoginRateLimit(t *testing.T) {
	h, authSvc := setupAuthTest(t)
	user := authSvc.GetUserByUsername("admin")
	limiter := middleware.NewRateLimiter(t.TempDir())
	h.SetRateLimiter(limiter)

	for i := 0; i < 10; i++ {
		_ = changePassword(t, h, adminClaims(user), user.ID, map[string]string{"password": pwChangeNew, "currentPassword": "nope"})
	}
	if blocked, _ := limiter.CheckAuthUser(user.Username, "192.0.2.1"); !blocked { // httptest default peer
		t.Fatal("repeated wrong current passwords must lock the user for this address")
	}
	rec := changePassword(t, h, adminClaims(user), user.ID, map[string]string{"password": pwChangeNew, "currentPassword": pwChangeOld})
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("want 429 while locked, got %d (%s)", rec.Code, rec.Body.String())
	}
	if !authSvc.VerifyPassword(authSvc.GetUserByID(user.ID).PasswordHash, pwChangeOld) {
		t.Fatal("password changed while the user bucket was locked")
	}
}

func TestChangePassword_SelfWithCurrentPassword_RevokesOldSessionsAndKeepsCaller(t *testing.T) {
	h, authSvc := setupAuthTest(t)
	user := authSvc.GetUserByUsername("admin")
	oldSession, _ := authSvc.CreateRefreshSession(user.ID)
	oldAccess, _ := authSvc.GenerateToken(user)

	rec := changePassword(t, h, adminClaims(user), user.ID, map[string]string{"password": pwChangeNew, "currentPassword": pwChangeOld})
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if !authSvc.VerifyPassword(authSvc.GetUserByID(user.ID).PasswordHash, pwChangeNew) {
		t.Fatal("new password not stored")
	}
	if code := refreshWith(h, oldSession); code != http.StatusUnauthorized {
		t.Fatalf("refresh token issued before the change must be rejected; got %d", code)
	}
	assertCredentialChangeAccess(t, authSvc, oldAccess, rec)
	fresh := responseRefreshCookie(rec)
	if fresh == "" {
		t.Fatal("self-service change must issue a fresh refresh cookie for the calling device")
	}
	if code := refreshWith(h, fresh); code != http.StatusOK {
		t.Fatalf("fresh refresh cookie must work; got %d", code)
	}
}

func TestChangePassword_AdminResetsOtherUser_NoCurrentPasswordAndRevokesTargetOnly(t *testing.T) {
	h, authSvc := setupAuthTest(t)
	admin := authSvc.GetUserByUsername("admin")
	viewer := addViewer(t, authSvc)
	adminSession, _ := authSvc.CreateRefreshSession(admin.ID)
	viewerSession, _ := authSvc.CreateRefreshSession(viewer.ID)

	rec := changePassword(t, h, adminClaims(admin), viewer.ID, map[string]string{"password": pwChangeNew})
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if !authSvc.VerifyPassword(authSvc.GetUserByID(viewer.ID).PasswordHash, pwChangeNew) {
		t.Fatal("new password not stored for target")
	}
	if responseRefreshCookie(rec) != "" {
		t.Fatal("admin reset must not issue a session cookie")
	}
	if code := refreshWith(h, viewerSession); code != http.StatusUnauthorized {
		t.Fatalf("target's pre-change refresh token must be rejected; got %d", code)
	}
	if code := refreshWith(h, adminSession); code != http.StatusOK {
		t.Fatalf("admin's own session must survive resetting another user; got %d", code)
	}
}

func TestChangePassword_ViewerSelfRequiresCurrentPassword(t *testing.T) {
	h, authSvc := setupAuthTest(t)
	viewer := addViewer(t, authSvc)

	rec := changePassword(t, h, adminClaims(viewer), viewer.ID, map[string]string{"password": pwChangeNew})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d (%s)", rec.Code, rec.Body.String())
	}
	rec = changePassword(t, h, adminClaims(viewer), viewer.ID, map[string]string{"password": pwChangeNew, "currentPassword": pwChangeOld})
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestMfaHandler_DisableRevokesTargetSessions(t *testing.T) {
	h, authHandler, mfaSvc, uid := setupMfaHandlerTest(t)
	secret, _, _ := mfaSvc.BeginEnrollment(uid)
	code, _ := totp.GenerateCode(secret, time.Now())
	if _, err := mfaSvc.ConfirmEnrollment(uid, code); err != nil {
		t.Fatalf("ConfirmEnrollment: %v", err)
	}
	session, _ := h.authSvc.CreateRefreshSession(uid)
	oldAccess, _ := h.authSvc.GenerateToken(h.authSvc.GetUserByID(uid))

	body, _ := json.Marshal(map[string]string{"password": pwChangeOld})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/mfa/disable", bytes.NewReader(body))
	req = req.WithContext(authedContext(uid, "admin", model.RoleAdmin))
	rec := httptest.NewRecorder()
	h.Disable(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if got := refreshWith(authHandler, session); got != http.StatusUnauthorized {
		t.Fatalf("refresh token issued before MFA disable must be rejected; got %d", got)
	}
	assertCredentialChangeAccess(t, h.authSvc, oldAccess, rec)
	fresh := responseRefreshCookie(rec)
	if fresh == "" || refreshWith(authHandler, fresh) != http.StatusOK {
		t.Fatal("self-disable must issue a working fresh refresh cookie")
	}
}

func TestMfaHandler_AdminDisablesOtherUser_RevokesTargetOnly(t *testing.T) {
	h, authHandler, mfaSvc, adminID := setupMfaHandlerTest(t)
	viewer := addViewer(t, h.authSvc)
	secret, _, _ := mfaSvc.BeginEnrollment(viewer.ID)
	code, _ := totp.GenerateCode(secret, time.Now())
	if _, err := mfaSvc.ConfirmEnrollment(viewer.ID, code); err != nil {
		t.Fatalf("ConfirmEnrollment: %v", err)
	}
	adminSession, _ := h.authSvc.CreateRefreshSession(adminID)
	viewerSession, _ := h.authSvc.CreateRefreshSession(viewer.ID)

	body, _ := json.Marshal(map[string]string{"password": pwChangeOld, "userId": viewer.ID})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/mfa/disable", bytes.NewReader(body))
	req = req.WithContext(authedContext(adminID, "admin", model.RoleAdmin))
	rec := httptest.NewRecorder()
	h.Disable(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if responseRefreshCookie(rec) != "" {
		t.Fatal("admin disabling another user's MFA must not issue a session cookie")
	}
	if got := refreshWith(authHandler, viewerSession); got != http.StatusUnauthorized {
		t.Fatalf("target's pre-disable refresh token must be rejected; got %d", got)
	}
	if got := refreshWith(authHandler, adminSession); got != http.StatusOK {
		t.Fatalf("admin's own session must survive; got %d", got)
	}
}

func assertCredentialChangeAccess(t *testing.T, svc *service.AuthService, old string, response *httptest.ResponseRecorder) {
	t.Helper()
	var body model.ApiResponse[struct {
		Token string `json:"token"`
	}]
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	protected := middleware.Auth(svc)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, tt := range []struct {
		token string
		want  int
	}{{old, http.StatusUnauthorized}, {body.Data.Token, http.StatusNoContent}} {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tt.token)
		rec := httptest.NewRecorder()
		protected.ServeHTTP(rec, req)
		if rec.Code != tt.want {
			t.Errorf("protected route status = %d, want %d: %s", rec.Code, tt.want, rec.Body)
		}
	}
}

func TestChangePassword_SessionWriteFailureKeepsOldCredentials(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	svc := service.NewAuthService("test-secret", store.NewJSONStore[model.CCUsers](filepath.Join(dir, "users.json")), store.NewJSONStore[model.RefreshSessions](path))
	user := addViewer(t, svc)
	h := NewAuthHandler(svc)
	old, err := svc.CreateRefreshSession(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	access, _ := svc.GenerateToken(user)
	// Fail the session write but keep the existing ledger readable.
	if err := os.Mkdir(path+".tmp", 0700); err != nil {
		t.Fatal(err)
	}
	rec := changePassword(t, h, adminClaims(user), user.ID, map[string]string{"password": pwChangeNew, "currentPassword": pwChangeOld})
	if rec.Code != http.StatusInternalServerError || responseRefreshCookie(rec) != "" {
		t.Fatalf("want 500 without a replacement cookie, got %d: %s", rec.Code, rec.Body)
	}
	hash := svc.GetUserByID(user.ID).PasswordHash
	if !svc.VerifyPassword(hash, pwChangeOld) || svc.VerifyPassword(hash, pwChangeNew) {
		t.Error("password was changed despite failed session write")
	}
	if _, err := svc.VerifyToken(access); err != nil {
		t.Errorf("old access token should still work: %v", err)
	}
	if err := os.Remove(path + ".tmp"); err != nil {
		t.Fatal(err)
	}
	if got := refreshWith(h, old); got != http.StatusOK {
		t.Errorf("old refresh got %d, want 200", got)
	}
}

// Hold the response after rotation commits to model a refresh that started
// before a credential change but reaches its client after the change.
func TestChangePassword_ConcurrentRefreshCannotCarrySessionAcrossChange(t *testing.T) {
	h, svc := setupAuthTest(t)
	user := svc.GetUserByUsername("admin")
	old, err := svc.CreateRefreshSession(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	rotated, resume := make(chan struct{}), make(chan struct{})
	h.rotateRefreshSession = func(token string) (*model.CCUser, string, error) {
		u, next, err := svc.RotateRefreshSession(token)
		if token == old && err == nil {
			close(rotated)
			<-resume
		}
		return u, next, err
	}
	response := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		rec := httptest.NewRecorder()
		req := refreshRequest(old)
		req.RemoteAddr = "192.0.2.2:1234"
		h.Refresh(rec, req)
		response <- rec
	}()
	select {
	case <-rotated:
	case rec := <-response:
		t.Fatalf("refresh failed before the change: %d %s", rec.Code, rec.Body)
	}
	changed := changePassword(t, h, adminClaims(user), user.ID, map[string]string{"password": pwChangeNew, "currentPassword": pwChangeOld})
	close(resume)
	rec := <-response
	if changed.Code != http.StatusOK || rec.Code != http.StatusOK {
		t.Fatalf("want successful change and rotation: change=%d refresh=%d", changed.Code, rec.Code)
	}
	replacement := responseRefreshCookie(rec)
	if replacement == "" {
		t.Fatal("rotation returned no cookie")
	}
	if got := refreshWith(h, replacement); got != http.StatusUnauthorized {
		t.Fatalf("concurrent rotation left a valid session: %d", got)
	}
	var body model.ApiResponse[AuthResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Data.Token == "" {
		t.Fatalf("expected a rotation access token: %v", err)
	}
	assertCredentialChangeAccess(t, svc, body.Data.Token, changed)
}
