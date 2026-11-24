package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
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
	if blocked, _ := limiter.Check(middleware.AuthUserKey(user.Username)); !blocked {
		t.Fatal("repeated wrong current passwords must lock the per-user auth bucket")
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
