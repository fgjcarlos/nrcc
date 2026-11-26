package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/store"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
)

func setupMfaHandlerTest(t *testing.T) (*MfaHandler, *AuthHandler, *service.MfaService, string) {
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

	mfaSvc := service.NewMfaService(dir, authSvc)
	auditSvc, err := audit.NewService(dir)
	if err != nil {
		t.Fatalf("audit.NewService: %v", err)
	}
	t.Cleanup(func() { _ = auditSvc.Close() })

	authHandler := NewAuthHandler(authSvc)
	authHandler.SetAuditService(auditSvc)
	authHandler.SetMfaService(mfaSvc)

	mfaHandler := NewMfaHandler(mfaSvc, authSvc)
	mfaHandler.SetAuditService(auditSvc)

	return mfaHandler, authHandler, mfaSvc, user.ID
}

func authedContext(uid, username string, role model.UserRole) context.Context {
	return context.WithValue(context.Background(), middleware.CtxKeyUser, &model.Claims{
		UserID:   uid,
		Username: username,
		Role:     role,
	})
}

func TestMfaHandler_EnrollRequiresAuth(t *testing.T) {
	h, _, _, _ := setupMfaHandlerTest(t)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/mfa/enroll", nil)
	rec := httptest.NewRecorder()
	h.Enroll(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestMfaHandler_EnrollReturnsSecretAndOtpauthURL(t *testing.T) {
	h, _, _, uid := setupMfaHandlerTest(t)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/mfa/enroll", nil)
	req = req.WithContext(authedContext(uid, "admin", model.RoleAdmin))
	rec := httptest.NewRecorder()
	h.Enroll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var env model.ApiResponse[model.MfaEnrollBeginResponse]
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Data.Secret == "" {
		t.Errorf("empty secret")
	}
	if !strings.HasPrefix(env.Data.OtpauthURL, "otpauth://totp/") {
		t.Errorf("otpauth URL has unexpected shape: %q", env.Data.OtpauthURL)
	}
}

func TestMfaHandler_EnrollCanRestartUntilConfirmed(t *testing.T) {
	h, _, _, uid := setupMfaHandlerTest(t)
	ctx := authedContext(uid, "admin", model.RoleAdmin)
	enroll := func(want int) model.ApiResponse[model.MfaEnrollBeginResponse] {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/mfa/enroll", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.Enroll(rec, req)
		if rec.Code != want {
			t.Fatalf("enroll: want %d, got %d (%s)", want, rec.Code, rec.Body.String())
		}
		var env model.ApiResponse[model.MfaEnrollBeginResponse]
		if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
			t.Fatal(err)
		}
		return env
	}
	first := enroll(http.StatusOK).Data
	req := httptest.NewRequest(http.MethodGet, "/api/auth/mfa/status", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	h.Status(rec, req)
	var status model.ApiResponse[model.MfaStatusResponse]
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil || rec.Code != http.StatusOK || status.Data.Enabled || status.Data.RecoveryCodesRemaining != 0 {
		t.Fatalf("pending status: HTTP %d, %+v, decode error %v", rec.Code, status.Data, err)
	}
	second := enroll(http.StatusOK).Data
	if second.Secret == "" || second.Secret == first.Secret || !strings.HasPrefix(second.OtpauthURL, "otpauth://totp/") {
		t.Fatal("retry did not return a fresh candidate")
	}
	code, err := totp.GenerateCode(second.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{"code": code})
	req = httptest.NewRequest(http.MethodPost, "/api/auth/mfa/enroll/confirm", bytes.NewReader(body)).WithContext(ctx)
	rec = httptest.NewRecorder()
	h.EnrollConfirm(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("confirm retry: HTTP %d (%s)", rec.Code, rec.Body.String())
	}
	enroll(http.StatusConflict)
}

func TestMfaHandler_EnrollConfirmWrongCodeReturns400(t *testing.T) {
	h, _, _, uid := setupMfaHandlerTest(t)

	// Begin
	req := httptest.NewRequest(http.MethodPost, "/api/auth/mfa/enroll", nil)
	req = req.WithContext(authedContext(uid, "admin", model.RoleAdmin))
	rec := httptest.NewRecorder()
	h.Enroll(rec, req)

	body := bytes.NewBufferString(`{"code":"000000"}`)
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/mfa/enroll/confirm", body)
	req2 = req2.WithContext(authedContext(uid, "admin", model.RoleAdmin))
	rec2 := httptest.NewRecorder()
	h.EnrollConfirm(rec2, req2)
	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d (%s)", rec2.Code, rec2.Body.String())
	}
}

func TestMfaHandler_EnrollConfirmHappyPath(t *testing.T) {
	h, _, mfaSvc, uid := setupMfaHandlerTest(t)

	secret, _, err := mfaSvc.BeginEnrollment(uid)
	if err != nil {
		t.Fatalf("BeginEnrollment: %v", err)
	}
	code, _ := totp.GenerateCode(secret, time.Now())
	body, _ := json.Marshal(map[string]string{"code": code})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/mfa/enroll/confirm", bytes.NewReader(body))
	req = req.WithContext(authedContext(uid, "admin", model.RoleAdmin))
	rec := httptest.NewRecorder()
	h.EnrollConfirm(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var env model.ApiResponse[model.MfaEnrollConfirmResponse]
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(env.Data.RecoveryCodes) != service.MfaRecoveryCodeCount {
		t.Errorf("recovery codes count = %d, want %d", len(env.Data.RecoveryCodes), service.MfaRecoveryCodeCount)
	}
}

func TestMfaHandler_DisableRejectsWrongPassword(t *testing.T) {
	h, _, _, uid := setupMfaHandlerTest(t)

	body, _ := json.Marshal(map[string]string{"password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/mfa/disable", bytes.NewReader(body))
	req = req.WithContext(authedContext(uid, "admin", model.RoleAdmin))
	rec := httptest.NewRecorder()
	h.Disable(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestMfaHandler_DisableHappyPath(t *testing.T) {
	h, _, mfaSvc, uid := setupMfaHandlerTest(t)
	secret, _, _ := mfaSvc.BeginEnrollment(uid)
	totpCode, _ := totp.GenerateCode(secret, time.Now())
	if _, err := mfaSvc.ConfirmEnrollment(uid, totpCode); err != nil {
		t.Fatalf("ConfirmEnrollment: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/mfa/disable", bytes.NewReader(body))
	req = req.WithContext(authedContext(uid, "admin", model.RoleAdmin))
	rec := httptest.NewRecorder()
	h.Disable(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestMfaHandler_StatusReturnsDisabledForNewUser(t *testing.T) {
	h, _, _, uid := setupMfaHandlerTest(t)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/mfa/status", nil)
	req = req.WithContext(authedContext(uid, "admin", model.RoleAdmin))
	rec := httptest.NewRecorder()
	h.Status(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	var env model.ApiResponse[model.MfaStatusResponse]
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Data.Enabled {
		t.Errorf("enabled = true, want false for new user")
	}
	if env.Data.RecoveryCodesRemaining != 0 {
		t.Errorf("remaining = %d, want 0", env.Data.RecoveryCodesRemaining)
	}
}

func TestMfaHandler_VerifyRejectsInvalidToken(t *testing.T) {
	h, _, _, _ := setupMfaHandlerTest(t)
	body, _ := json.Marshal(model.MfaVerifyRequest{MfaToken: "nope", Code: "123456"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/mfa/verify", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Verify(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestMfaHandler_VerifyHappyPath(t *testing.T) {
	h, authHandler, mfaSvc, uid := setupMfaHandlerTest(t)
	secret, _, _ := mfaSvc.BeginEnrollment(uid)
	totpCode, _ := totp.GenerateCode(secret, time.Now())
	if _, err := mfaSvc.ConfirmEnrollment(uid, totpCode); err != nil {
		t.Fatalf("ConfirmEnrollment: %v", err)
	}
	tok, _ := mfaSvc.IssueMfaToken(uid)

	fresh, _ := totp.GenerateCode(secret, time.Now())
	body, _ := json.Marshal(model.MfaVerifyRequest{MfaToken: tok, Code: fresh})
	req := httptest.NewRequest(http.MethodPost, "https://nrcc.example/api/auth/mfa/verify", bytes.NewReader(body))
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	jar.SetCookies(req.URL, []*http.Cookie{{Name: refreshCookieName, Value: "old", Path: "/api/auth", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode}})
	for _, cookie := range jar.Cookies(req.URL) {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.Verify(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	jar.SetCookies(req.URL, rec.Result().Cookies())
	assertRefreshCookieLifecycle(t, authHandler, jar)
}

func TestRefresh_MigratesLegacyMFACookie(t *testing.T) {
	h, svc := setupAuthTest(t)
	legacy, err := svc.CreateRefreshSession(svc.GetUserByUsername("admin").ID)
	if err != nil {
		t.Fatal(err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "https://nrcc.example/api/auth/refresh", nil)
	jar.SetCookies(req.URL, []*http.Cookie{{Name: refreshCookieName, Value: legacy, Path: "/api/auth", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode}})
	assertRefreshCookieLifecycle(t, h, jar)
}

func assertRefreshCookieLifecycle(t *testing.T, authHandler *AuthHandler, jar http.CookieJar) {
	t.Helper()
	var req *http.Request
	for i, step := range []struct {
		path    string
		handler http.HandlerFunc
		want    int
	}{
		{"/api/auth/refresh", authHandler.Refresh, http.StatusOK},
		{"/api/auth/refresh", authHandler.Refresh, http.StatusOK},
		{"/api/auth/logout", authHandler.Logout, http.StatusOK},
		{"/api/auth/refresh", authHandler.Refresh, http.StatusUnauthorized},
	} {
		req = httptest.NewRequest(http.MethodPost, "https://nrcc.example"+step.path, nil)
		for _, cookie := range jar.Cookies(req.URL) {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		step.handler(rec, req)
		if rec.Code != step.want {
			t.Fatalf("step %d %s: want HTTP %d, got %d (%s)", i+1, step.path, step.want, rec.Code, rec.Body.String())
		}
		jar.SetCookies(req.URL, rec.Result().Cookies())
	}
	if remaining := jar.Cookies(req.URL); len(remaining) != 0 {
		t.Fatalf("logout left %d cookies", len(remaining))
	}
}

func TestMfaHandler_VerifyRecoveryCodeHappyPath(t *testing.T) {
	h, _, mfaSvc, uid := setupMfaHandlerTest(t)
	secret, _, _ := mfaSvc.BeginEnrollment(uid)
	totpCode, _ := totp.GenerateCode(secret, time.Now())
	codes, err := mfaSvc.ConfirmEnrollment(uid, totpCode)
	if err != nil {
		t.Fatalf("ConfirmEnrollment: %v", err)
	}
	tok, _ := mfaSvc.IssueMfaToken(uid)

	body, _ := json.Marshal(model.MfaVerifyRequest{MfaToken: tok, Code: codes[0]})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/mfa/verify", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Verify(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestAuthHandler_LoginWithEnrollmentReturnsMfaRequired(t *testing.T) {
	_, authHandler, mfaSvc, uid := setupMfaHandlerTest(t)

	// Enroll the user.
	secret, _, _ := mfaSvc.BeginEnrollment(uid)
	totpCode, _ := totp.GenerateCode(secret, time.Now())
	if _, err := mfaSvc.ConfirmEnrollment(uid, totpCode); err != nil {
		t.Fatalf("ConfirmEnrollment: %v", err)
	}

	// #nosec G117 -- test fixture; marshaling is asserted by the handler.
	body, _ := json.Marshal(LoginRequest{Username: "admin", Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	authHandler.Login(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var env model.ApiResponse[model.MfaLoginResponse]
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !env.Data.MfaRequired {
		t.Errorf("mfaRequired = false, want true")
	}
	if env.Data.MfaToken == "" {
		t.Errorf("mfaToken is empty")
	}
	// No refresh cookie should be set on the password-only step.
	if got := rec.Header().Get("Set-Cookie"); strings.Contains(got, "nrcc_refresh=") {
		t.Errorf("refresh cookie should not be set before MFA verify, got %q", got)
	}
}

func TestAuthHandler_LoginWithoutEnrollmentIsUnchanged(t *testing.T) {
	_, authHandler, _, _ := setupMfaHandlerTest(t)
	// #nosec G117 -- test fixture; marshaling is asserted by the handler.
	body, _ := json.Marshal(LoginRequest{Username: "admin", Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	authHandler.Login(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var env model.ApiResponse[map[string]any]
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// The shape should NOT have mfaRequired at top level.
	if _, hasMfa := env.Data["mfaRequired"]; hasMfa {
		t.Errorf("non-enrolled login should not advertise mfaRequired, got %v", env.Data)
	}
	if got := rec.Header().Get("Set-Cookie"); !strings.Contains(got, "nrcc_refresh=") {
		t.Errorf("refresh cookie should be set on the non-MFA path, got %q", got)
	}
}
