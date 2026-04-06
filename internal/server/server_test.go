package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"nrcc/internal/model"
	"nrcc/internal/service"
)

func TestAuthFlowAndProtectedRoutes(t *testing.T) {
	t.Parallel()

	authService, configService := newTestServices(t)
	router := chi.NewRouter()
	testDataDir := t.TempDir()
	registerAPIRoutes(router, nil, authService, configService, service.NewManagedEnvService(testDataDir), service.NewBackupService(testDataDir))

	registerBody := map[string]string{
		"username": "alice",
		"password": "password123",
	}

	registerReq := newJSONRequest(t, http.MethodPost, "/api/auth/register", registerBody)
	registerRec := httptest.NewRecorder()
	router.ServeHTTP(registerRec, registerReq)

	if registerRec.Code != http.StatusOK {
		t.Fatalf("register status = %d, want %d", registerRec.Code, http.StatusOK)
	}

	registerCookie := sessionCookieFromResponse(t, registerRec.Result())
	if !registerCookie.HttpOnly {
		t.Fatal("session cookie HttpOnly = false, want true")
	}
	if registerCookie.Secure {
		t.Fatal("session cookie Secure = true on plain request, want false")
	}
	if registerCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("session cookie SameSite = %v, want %v", registerCookie.SameSite, http.SameSiteStrictMode)
	}
	if registerCookie.Path != "/" {
		t.Fatalf("session cookie Path = %q, want /", registerCookie.Path)
	}
	if registerCookie.MaxAge <= 0 {
		t.Fatalf("session cookie MaxAge = %d, want > 0", registerCookie.MaxAge)
	}

	var registerResp model.APIResponse[authResponse]
	decodeResponse(t, registerRec.Body.Bytes(), &registerResp)
	if !registerResp.Success || registerResp.Data.User == nil || registerResp.Data.CSRFToken == "" {
		t.Fatalf("register response = %+v", registerResp)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.AddCookie(registerCookie)
	meRec := httptest.NewRecorder()
	router.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me status = %d, want %d", meRec.Code, http.StatusOK)
	}

	protectedBody := model.AppConfig{
		HTTPAdminRoot:      "/admin",
		FlowFile:           "flows.json",
		DiagnosticsEnabled: true,
		ProjectsEnabled:    false,
		CredentialSecret:   "very-secret-123",
	}

	noAuthReq := newJSONRequest(t, http.MethodPost, "/api/config/apply", protectedBody)
	noAuthRec := httptest.NewRecorder()
	router.ServeHTTP(noAuthRec, noAuthReq)
	if noAuthRec.Code != http.StatusUnauthorized {
		t.Fatalf("protected POST without auth = %d, want %d", noAuthRec.Code, http.StatusUnauthorized)
	}

	noCSRReq := newJSONRequest(t, http.MethodPost, "/api/config/apply", protectedBody)
	noCSRReq.AddCookie(registerCookie)
	noCSRRec := httptest.NewRecorder()
	router.ServeHTTP(noCSRRec, noCSRReq)
	if noCSRRec.Code != http.StatusForbidden {
		t.Fatalf("protected POST without csrf = %d, want %d", noCSRRec.Code, http.StatusForbidden)
	}

	withCSRReq := newJSONRequest(t, http.MethodPost, "/api/config/apply", protectedBody)
	withCSRReq.AddCookie(registerCookie)
	withCSRReq.Header.Set("X-CSRF-Token", registerResp.Data.CSRFToken)
	withCSRRec := httptest.NewRecorder()
	router.ServeHTTP(withCSRRec, withCSRReq)
	if withCSRRec.Code != http.StatusOK {
		t.Fatalf("protected POST with csrf = %d, want %d body=%s", withCSRRec.Code, http.StatusOK, withCSRRec.Body.String())
	}

	logoutReq := newJSONRequest(t, http.MethodPost, "/api/auth/logout", nil)
	logoutReq.AddCookie(registerCookie)
	logoutReq.Header.Set("X-CSRF-Token", registerResp.Data.CSRFToken)
	logoutRec := httptest.NewRecorder()
	router.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logoutRec.Code, http.StatusOK)
	}

	logoutCookie := sessionCookieFromResponse(t, logoutRec.Result())
	if logoutCookie.MaxAge >= 0 {
		t.Fatalf("logout cookie MaxAge = %d, want negative", logoutCookie.MaxAge)
	}

	reusedMeReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	reusedMeReq.AddCookie(registerCookie)
	reusedMeRec := httptest.NewRecorder()
	router.ServeHTTP(reusedMeRec, reusedMeReq)
	if reusedMeRec.Code != http.StatusUnauthorized {
		t.Fatalf("reused cookie after logout = %d, want %d", reusedMeRec.Code, http.StatusUnauthorized)
	}
}

func TestLoginCookieSecureOnForwardedHTTPS(t *testing.T) {
	t.Parallel()

	authService, configService := newTestServices(t)
	if _, _, err := authService.RegisterInitial("alice", "password123"); err != nil {
		t.Fatalf("RegisterInitial() error = %v", err)
	}

	router := chi.NewRouter()
	testDataDir := t.TempDir()
	registerAPIRoutes(router, nil, authService, configService, service.NewManagedEnvService(testDataDir), service.NewBackupService(testDataDir))

	loginReq := newJSONRequest(t, http.MethodPost, "/api/auth/login", map[string]string{
		"username": "alice",
		"password": "password123",
	})
	loginReq.Header.Set("X-Forwarded-Proto", "https")

	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", loginRec.Code, http.StatusOK)
	}

	loginCookie := sessionCookieFromResponse(t, loginRec.Result())
	if !loginCookie.Secure {
		t.Fatal("session cookie Secure = false on forwarded https request, want true")
	}
}

func newTestServices(t *testing.T) (*service.AuthService, service.ConfigService) {
	t.Helper()

	dataDir := t.TempDir()
	authService, err := service.NewAuthService(dataDir)
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}
	t.Cleanup(func() {
		if err := authService.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	return authService, service.NewConfigService(dataDir)
}

func newJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func sessionCookieFromResponse(t *testing.T, resp *http.Response) *http.Cookie {
	t.Helper()

	for _, cookie := range resp.Cookies() {
		if cookie.Name == service.SessionCookieName {
			return cookie
		}
	}
	t.Fatalf("session cookie %q not found", service.SessionCookieName)
	return nil
}

func decodeResponse[T any](t *testing.T, payload []byte, target *T) {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode response error = %v body=%s", err, strings.TrimSpace(string(payload)))
	}
}
