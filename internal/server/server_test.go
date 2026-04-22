package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	backupSvc := service.NewBackupService(testDataDir)
	registerAPIRoutes(
		router,
		nil,
		authService,
		configService,
		service.NewManagedEnvService(testDataDir),
		backupSvc,
		service.NewLibraryService(testDataDir),
		service.NewFlowService(testDataDir),
		service.NewUpdateService(testDataDir, &backupSvc),
		service.NewOperationLock(),
		nil,
		nil,
	)

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
	backupSvc := service.NewBackupService(testDataDir)
	registerAPIRoutes(
		router,
		nil,
		authService,
		configService,
		service.NewManagedEnvService(testDataDir),
		backupSvc,
		service.NewLibraryService(testDataDir),
		service.NewFlowService(testDataDir),
		service.NewUpdateService(testDataDir, &backupSvc),
		service.NewOperationLock(),
		nil,
		nil,
	)

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

func TestOperationStatusAndRestoreConflict(t *testing.T) {
	t.Parallel()

	authService, configService := newTestServices(t)
	testDataDir := t.TempDir()
	lock := service.NewOperationLock()
	release, err := lock.Acquire("installing", "lodash")
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer release()

	backupSvc := service.NewBackupService(testDataDir)
	router := chi.NewRouter()
	registerAPIRoutes(
		router,
		nil,
		authService,
		configService,
		service.NewManagedEnvService(testDataDir),
		backupSvc,
		service.NewLibraryService(testDataDir),
		service.NewFlowService(testDataDir),
		service.NewUpdateService(testDataDir, &backupSvc),
		lock,
		nil,
		nil,
	)

	if _, _, err := authService.RegisterInitial("alice", "password123"); err != nil {
		t.Fatalf("RegisterInitial() error = %v", err)
	}

	loginReq := newJSONRequest(t, http.MethodPost, "/api/auth/login", map[string]string{
		"username": "alice",
		"password": "password123",
	})
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", loginRec.Code, http.StatusOK)
	}
	cookie := sessionCookieFromResponse(t, loginRec.Result())

	var loginResp model.APIResponse[authResponse]
	decodeResponse(t, loginRec.Body.Bytes(), &loginResp)

	statusReq := httptest.NewRequest(http.MethodGet, "/api/operations/status", nil)
	statusReq.AddCookie(cookie)
	statusRec := httptest.NewRecorder()
	router.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("operations status = %d, want %d", statusRec.Code, http.StatusOK)
	}

	var statusResp model.APIResponse[model.OperationStatus]
	decodeResponse(t, statusRec.Body.Bytes(), &statusResp)
	if !statusResp.Success || !statusResp.Data.Busy || statusResp.Data.Type != "installing" {
		t.Fatalf("operations status payload = %+v", statusResp)
	}

	restoreReq := newJSONRequest(t, http.MethodPost, "/api/backups/backup-id/restore", nil)
	restoreReq.AddCookie(cookie)
	restoreReq.Header.Set("X-CSRF-Token", loginResp.Data.CSRFToken)
	restoreRec := httptest.NewRecorder()
	router.ServeHTTP(restoreRec, restoreReq)
	if restoreRec.Code != http.StatusConflict {
		t.Fatalf("restore while locked = %d, want %d body=%s", restoreRec.Code, http.StatusConflict, restoreRec.Body.String())
	}
}

func TestFlowsEndpoints(t *testing.T) {
	t.Parallel()

	authService, configService := newTestServices(t)
	testDataDir := t.TempDir()
	if err := service.NewConfigService(testDataDir, nil).SaveFullConfig(model.DefaultFullAppConfig()); err != nil {
		t.Fatalf("SaveFullConfig() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(testDataDir, "nodered"), 0o755); err != nil {
		t.Fatalf("MkdirAll(nodered) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDataDir, "nodered", "flows.json"), []byte(`[
		{"id":"tab-a","type":"tab","label":"Main"},
		{"id":"inject-1","type":"inject","z":"tab-a","name":"Start","wires":[]}
	]`), 0o644); err != nil {
		t.Fatalf("WriteFile(flows.json) error = %v", err)
	}

	backupSvc := service.NewBackupService(testDataDir)
	router := chi.NewRouter()
	registerAPIRoutes(
		router,
		nil,
		authService,
		configService,
		service.NewManagedEnvService(testDataDir),
		backupSvc,
		service.NewLibraryService(testDataDir),
		service.NewFlowService(testDataDir),
		service.NewUpdateService(testDataDir, &backupSvc),
		service.NewOperationLock(),
		nil,
		nil,
	)

	if _, _, err := authService.RegisterInitial("alice", "password123"); err != nil {
		t.Fatalf("RegisterInitial() error = %v", err)
	}

	loginReq := newJSONRequest(t, http.MethodPost, "/api/auth/login", map[string]string{
		"username": "alice",
		"password": "password123",
	})
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", loginRec.Code, http.StatusOK)
	}
	cookie := sessionCookieFromResponse(t, loginRec.Result())

	listReq := httptest.NewRequest(http.MethodGet, "/api/flows", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("flows list status = %d, want %d body=%s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var listResp model.APIResponse[model.FlowList]
	decodeResponse(t, listRec.Body.Bytes(), &listResp)
	if !listResp.Success || len(listResp.Data.Items) != 1 || listResp.Data.Items[0].ID != "tab-a" {
		t.Fatalf("flows list response = %+v", listResp)
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/api/flows/tab-a", nil)
	detailReq.AddCookie(cookie)
	detailRec := httptest.NewRecorder()
	router.ServeHTTP(detailRec, detailReq)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("flow detail status = %d, want %d body=%s", detailRec.Code, http.StatusOK, detailRec.Body.String())
	}

	var detailResp model.APIResponse[model.FlowDetailResponse]
	decodeResponse(t, detailRec.Body.Bytes(), &detailResp)
	if !detailResp.Success || detailResp.Data.Flow.ID != "tab-a" || len(detailResp.Data.Flow.Nodes) != 1 {
		t.Fatalf("flow detail response = %+v", detailResp)
	}

	notFoundReq := httptest.NewRequest(http.MethodGet, "/api/flows/missing", nil)
	notFoundReq.AddCookie(cookie)
	notFoundRec := httptest.NewRecorder()
	router.ServeHTTP(notFoundRec, notFoundReq)
	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("missing flow status = %d, want %d body=%s", notFoundRec.Code, http.StatusNotFound, notFoundRec.Body.String())
	}
}

func TestEnvironmentAPIHidesSecretValues(t *testing.T) {
	t.Parallel()

	const strongPassword = "NRCCTestVault847"

	authService, configService := newTestServices(t)
	testDataDir := t.TempDir()
	managedEnvService := service.NewManagedEnvService(testDataDir)
	if _, err := managedEnvService.Apply([]model.ManagedEnvVar{{Name: "API_KEY", Value: "super-secret", Secret: true}}); err != nil {
		t.Fatalf("Apply(managed env) error = %v", err)
	}

	backupSvc := service.NewBackupService(testDataDir)
	router := chi.NewRouter()
	registerAPIRoutes(
		router,
		nil,
		authService,
		configService,
		managedEnvService,
		backupSvc,
		service.NewLibraryService(testDataDir),
		service.NewFlowService(testDataDir),
		service.NewUpdateService(testDataDir, &backupSvc),
		service.NewOperationLock(),
		nil,
		nil,
	)

	if _, _, err := authService.RegisterInitial("alice", strongPassword); err != nil {
		t.Fatalf("RegisterInitial() error = %v", err)
	}

	loginReq := newJSONRequest(t, http.MethodPost, "/api/auth/login", map[string]string{
		"username": "alice",
		"password": strongPassword,
	})
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", loginRec.Code, http.StatusOK)
	}

	cookie := sessionCookieFromResponse(t, loginRec.Result())
	getReq := httptest.NewRequest(http.MethodGet, "/api/environment", nil)
	getReq.AddCookie(cookie)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET /api/environment status = %d, want %d body=%s", getRec.Code, http.StatusOK, getRec.Body.String())
	}

	var getResp model.APIResponse[model.ManagedEnvState]
	decodeResponse(t, getRec.Body.Bytes(), &getResp)
	if len(getResp.Data.Variables) != 1 {
		t.Fatalf("GET /api/environment len = %d, want 1", len(getResp.Data.Variables))
	}
	if !getResp.Data.Variables[0].Secret || getResp.Data.Variables[0].Value != "" || !getResp.Data.Variables[0].HasValue {
		t.Fatalf("GET /api/environment variable = %+v, want masked secret", getResp.Data.Variables[0])
	}

	raw, err := os.ReadFile(filepath.Join(testDataDir, ".env.managed"))
	if err != nil {
		t.Fatalf("ReadFile(.env.managed) error = %v", err)
	}
	if strings.Contains(string(raw), "super-secret") {
		t.Fatalf(".env.managed leaked plaintext secret: %q", string(raw))
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

	return authService, service.NewConfigService(dataDir, nil)
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
