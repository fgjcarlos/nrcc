package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"nrcc/internal/middleware"
	"nrcc/internal/model"
	"nrcc/internal/service"
)

type stubFlowAnalysisProvider struct {
	metadata model.FlowAnalysisProvider
	result   service.FlowAnalysisPayload
	err      error
}

func (s stubFlowAnalysisProvider) Metadata() model.FlowAnalysisProvider {
	return s.metadata
}

func (s stubFlowAnalysisProvider) Analyze(_ context.Context, _ string) (service.FlowAnalysisPayload, error) {
	if s.err != nil {
		return service.FlowAnalysisPayload{}, s.err
	}
	return s.result, nil
}

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
		"password": "Alice2025!secure",
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
	if _, _, err := authService.RegisterInitial("alice", "Alice2025!secure"); err != nil {
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
		"password": "Alice2025!secure",
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
	router.Use(middleware.RequestID)
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

	if _, _, err := authService.RegisterInitial("alice", "Alice2025!secure"); err != nil {
		t.Fatalf("RegisterInitial() error = %v", err)
	}

	loginReq := newJSONRequest(t, http.MethodPost, "/api/auth/login", map[string]string{
		"username": "alice",
		"password": "Alice2025!secure",
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

	var conflictResp model.APIResponse[any]
	decodeResponse(t, restoreRec.Body.Bytes(), &conflictResp)
	if !strings.EqualFold(conflictResp.Error.Code, "OPERATION_IN_PROGRESS") {
		t.Fatalf("conflict code = %q, want OPERATION_IN_PROGRESS", conflictResp.Error.Code)
	}
	if conflictResp.RequestID == "" || conflictResp.Error.RequestID == "" {
		t.Fatalf("request ids missing in conflict response: %+v", conflictResp)
	}
	details, ok := conflictResp.Error.Details.(map[string]any)
	if !ok {
		t.Fatalf("conflict details type = %T, want map", conflictResp.Error.Details)
	}
	if busy, _ := details["busy"].(bool); !busy {
		t.Fatalf("conflict details busy = %v, want true", details["busy"])
	}
	if opType, _ := details["type"].(string); opType != "installing" {
		t.Fatalf("conflict details type = %q, want installing", opType)
	}
}

func TestLockedRuntimeAndConfigRestoreReturnConflict(t *testing.T) {
	t.Parallel()

	testDataDir := t.TempDir()
	lock := service.NewOperationLock()
	release, err := lock.Acquire("updating", "node-red")
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer release()

	backupSvc := service.NewBackupService(testDataDir)
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	registerAPIRoutes(
		router,
		service.NewProcessManager(service.ProcessConfig{DataDir: testDataDir, Port: 1880}),
		nil,
		service.NewConfigService(testDataDir, nil),
		service.NewManagedEnvService(testDataDir),
		backupSvc,
		service.NewLibraryService(testDataDir),
		service.NewFlowService(testDataDir),
		service.NewUpdateService(testDataDir, &backupSvc),
		lock,
		nil,
		nil,
	)

	for _, path := range []string{"/api/runtime/restart", "/api/config/backups/snapshot-1/restore"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Fatalf("%s status = %d, want %d body=%s", path, rec.Code, http.StatusConflict, rec.Body.String())
		}

		var resp model.APIResponse[any]
		decodeResponse(t, rec.Body.Bytes(), &resp)
		if resp.Error == nil || resp.Error.Code != "OPERATION_IN_PROGRESS" {
			t.Fatalf("%s error = %+v", path, resp.Error)
		}
	}
}

func TestRuntimeRestartErrorIncludesRequestID(t *testing.T) {
	t.Parallel()

	testDataDir := t.TempDir()
	backupSvc := service.NewBackupService(testDataDir)
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	registerAPIRoutes(
		router,
		service.NewProcessManager(service.ProcessConfig{DataDir: testDataDir, Port: 1880}),
		nil,
		service.NewConfigService(testDataDir, nil),
		service.NewManagedEnvService(testDataDir),
		backupSvc,
		service.NewLibraryService(testDataDir),
		service.NewFlowService(testDataDir),
		service.NewUpdateService(testDataDir, &backupSvc),
		service.NewOperationLock(),
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/api/runtime/restart", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("restart status = %d, want %d body=%s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}

	var resp model.APIResponse[any]
	decodeResponse(t, rec.Body.Bytes(), &resp)
	if resp.Error == nil || resp.Error.Code != "RUNTIME_RESTART_FAILED" {
		t.Fatalf("restart error = %+v", resp.Error)
	}
	if resp.RequestID == "" || resp.Error.RequestID == "" {
		t.Fatalf("request ids missing in restart error response: %+v", resp)
	}
}

func TestEnvironmentAPIHidesSecretValues(t *testing.T) {
	t.Parallel()

	strongPassword := strings.Repeat("Ab9!", 3)

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

func TestUsersAPIRequiresAdminAndAppliesSafeguards(t *testing.T) {
	t.Parallel()

	authService, configService := newTestServices(t)
	testDataDir := t.TempDir()
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

	adminSession := loginSession(t, router, authService, "admin1", "Admin2025!pass", model.RoleAdmin)
	createReq := newJSONRequest(t, http.MethodPost, "/api/users", map[string]string{
		"username": "viewer1",
		"password": "Viewer2025!pass",
		"role":     "viewer",
	})
	createReq.AddCookie(adminSession.Cookie)
	createReq.Header.Set("X-CSRF-Token", adminSession.CSRFToken)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("POST /api/users status = %d, want %d body=%s", createRec.Code, http.StatusOK, createRec.Body.String())
	}

	var createResp model.APIResponse[userMutationResponse]
	decodeResponse(t, createRec.Body.Bytes(), &createResp)
	if !createResp.Success || createResp.Data.User == nil || createResp.Data.User.Role != model.RoleViewer {
		t.Fatalf("POST /api/users response = %+v", createResp)
	}
	viewerID := createResp.Data.User.ID

	viewerSession := loginSession(t, router, authService, "viewer1", "Viewer2025!pass", model.RoleViewer)
	viewerListReq := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	viewerListReq.AddCookie(viewerSession.Cookie)
	viewerListRec := httptest.NewRecorder()
	router.ServeHTTP(viewerListRec, viewerListReq)
	if viewerListRec.Code != http.StatusForbidden {
		t.Fatalf("viewer GET /api/users status = %d, want %d", viewerListRec.Code, http.StatusForbidden)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	listReq.AddCookie(adminSession.Cookie)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("GET /api/users status = %d, want %d", listRec.Code, http.StatusOK)
	}

	lastAdminReq := newJSONRequest(t, http.MethodPatch, "/api/users/"+adminSession.User.ID+"/role", map[string]string{"role": "operator"})
	lastAdminReq.AddCookie(adminSession.Cookie)
	lastAdminReq.Header.Set("X-CSRF-Token", adminSession.CSRFToken)
	lastAdminRec := httptest.NewRecorder()
	router.ServeHTTP(lastAdminRec, lastAdminReq)
	if lastAdminRec.Code != http.StatusConflict {
		t.Fatalf("PATCH last admin status = %d, want %d body=%s", lastAdminRec.Code, http.StatusConflict, lastAdminRec.Body.String())
	}

	createAdminReq := newJSONRequest(t, http.MethodPost, "/api/users", map[string]string{
		"username": "admin2",
		"password": "Admin22025!pass",
		"role":     "admin",
	})
	createAdminReq.AddCookie(adminSession.Cookie)
	createAdminReq.Header.Set("X-CSRF-Token", adminSession.CSRFToken)
	createAdminRec := httptest.NewRecorder()
	router.ServeHTTP(createAdminRec, createAdminReq)
	if createAdminRec.Code != http.StatusOK {
		t.Fatalf("POST /api/users admin2 status = %d, want %d body=%s", createAdminRec.Code, http.StatusOK, createAdminRec.Body.String())
	}

	var createAdminResp model.APIResponse[userMutationResponse]
	decodeResponse(t, createAdminRec.Body.Bytes(), &createAdminResp)
	admin2ID := createAdminResp.Data.User.ID

	resetReq := newJSONRequest(t, http.MethodPost, "/api/users/"+viewerID+"/reset-password", map[string]string{"password": "Viewer2025!next"})
	resetReq.AddCookie(adminSession.Cookie)
	resetReq.Header.Set("X-CSRF-Token", adminSession.CSRFToken)
	resetRec := httptest.NewRecorder()
	router.ServeHTTP(resetRec, resetReq)
	if resetRec.Code != http.StatusOK {
		t.Fatalf("POST reset password status = %d, want %d body=%s", resetRec.Code, http.StatusOK, resetRec.Body.String())
	}
	if _, _, err := authService.Login("viewer1", "Viewer2025!pass", "127.0.0.1:1234"); err == nil {
		t.Fatal("Login(old reset password) error = nil, want failure")
	}
	if _, _, err := authService.Login("viewer1", "Viewer2025!next", "127.0.0.1:1234"); err != nil {
		t.Fatalf("Login(new reset password) error = %v", err)
	}

	deleteSelfReq := newJSONRequest(t, http.MethodDelete, "/api/users/"+adminSession.User.ID, nil)
	deleteSelfReq.AddCookie(adminSession.Cookie)
	deleteSelfReq.Header.Set("X-CSRF-Token", adminSession.CSRFToken)
	deleteSelfRec := httptest.NewRecorder()
	router.ServeHTTP(deleteSelfRec, deleteSelfReq)
	if deleteSelfRec.Code != http.StatusOK {
		t.Fatalf("DELETE self status = %d, want %d body=%s", deleteSelfRec.Code, http.StatusOK, deleteSelfRec.Body.String())
	}
	deletedCookie := sessionCookieFromResponse(t, deleteSelfRec.Result())
	if deletedCookie.MaxAge >= 0 {
		t.Fatalf("DELETE self cookie MaxAge = %d, want negative", deletedCookie.MaxAge)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.AddCookie(adminSession.Cookie)
	meRec := httptest.NewRecorder()
	router.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("deleted self me status = %d, want %d", meRec.Code, http.StatusUnauthorized)
	}

	admin2Session := loginSession(t, router, authService, "admin2", "Admin22025!pass", model.RoleAdmin)
	deleteLastAdminReq := newJSONRequest(t, http.MethodDelete, "/api/users/"+admin2ID, nil)
	deleteLastAdminReq.AddCookie(admin2Session.Cookie)
	deleteLastAdminReq.Header.Set("X-CSRF-Token", admin2Session.CSRFToken)
	deleteLastAdminRec := httptest.NewRecorder()
	router.ServeHTTP(deleteLastAdminRec, deleteLastAdminReq)
	if deleteLastAdminRec.Code != http.StatusConflict {
		t.Fatalf("DELETE last admin status = %d, want %d body=%s", deleteLastAdminRec.Code, http.StatusConflict, deleteLastAdminRec.Body.String())
	}

	var deleteLastAdminResp model.APIResponse[userDeleteResponse]
	decodeResponse(t, deleteLastAdminRec.Body.Bytes(), &deleteLastAdminResp)
	if deleteLastAdminResp.Error == nil || deleteLastAdminResp.Error.Code != "LAST_ADMIN_REQUIRED" {
		t.Fatalf("DELETE last admin response = %+v", deleteLastAdminResp)
	}
}

func TestFlowsEndpoints(t *testing.T) {
	t.Parallel()

	strongPassword := strings.Repeat("Ab9!", 3)

	authService, configService := newTestServices(t)
	testDataDir := t.TempDir()
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

func TestFlowAnalysisEndpoint(t *testing.T) {
	t.Parallel()

	strongPassword := strings.Repeat("Ab9!", 3)

	authService, configService := newTestServices(t)
	testDataDir := t.TempDir()
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
	flowSvc := service.NewFlowServiceWithProvider(testDataDir, stubFlowAnalysisProvider{
		metadata: model.FlowAnalysisProvider{Name: "ollama", Model: "llama3.2", Local: true},
		result: service.FlowAnalysisPayload{
			Summary:     "Resumen del flujo.",
			Strengths:   []string{"Entrada clara"},
			Issues:      []string{"Falta manejo de errores"},
			Suggestions: []string{"Agregar validacion"},
		},
	})

	router := chi.NewRouter()
	registerAPIRoutes(
		router,
		nil,
		authService,
		configService,
		service.NewManagedEnvService(testDataDir),
		backupSvc,
		service.NewLibraryService(testDataDir),
		flowSvc,
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

	var loginResp model.APIResponse[authResponse]
	decodeResponse(t, loginRec.Body.Bytes(), &loginResp)

	analysisReq := newJSONRequest(t, http.MethodPost, "/api/flows/tab-a/analysis", nil)
	analysisReq.AddCookie(cookie)
	analysisReq.Header.Set("X-CSRF-Token", loginResp.Data.CSRFToken)
	analysisRec := httptest.NewRecorder()
	router.ServeHTTP(analysisRec, analysisReq)
	if analysisRec.Code != http.StatusOK {
		t.Fatalf("analysis status = %d, want %d body=%s", analysisRec.Code, http.StatusOK, analysisRec.Body.String())
	}

	var analysisResp model.APIResponse[model.FlowAnalysis]
	decodeResponse(t, analysisRec.Body.Bytes(), &analysisResp)
	if !analysisResp.Success || analysisResp.Data.Flow.ID != "tab-a" || analysisResp.Data.Provider.Name != "ollama" {
		t.Fatalf("analysis response = %+v", analysisResp)
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

type testLoginSession struct {
	User      model.UserPublic
	Cookie    *http.Cookie
	CSRFToken string
}

func loginSession(t *testing.T, router http.Handler, authService *service.AuthService, username, password string, role model.UserRole) testLoginSession {
	t.Helper()

	hasUsers, err := authService.HasUsers()
	if err != nil {
		t.Fatalf("HasUsers() error = %v", err)
	}
	if !hasUsers {
		if _, _, err := authService.RegisterInitial(username, password); err != nil {
			t.Fatalf("RegisterInitial(%s) error = %v", username, err)
		}
	} else if role != model.RoleAdmin || username != "admin1" {
		if _, err := authService.CreateUser(username, password, role, "admin1"); err != nil {
			if actionErr, ok := service.IsUserActionError(err); !ok || actionErr.Code != "USER_EXISTS" {
				t.Fatalf("CreateUser(%s) error = %v", username, err)
			}
		}
	}

	loginReq := newJSONRequest(t, http.MethodPost, "/api/auth/login", map[string]string{
		"username": username,
		"password": password,
	})
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("loginSession(%s) status = %d, want %d body=%s", username, loginRec.Code, http.StatusOK, loginRec.Body.String())
	}

	var loginResp model.APIResponse[authResponse]
	decodeResponse(t, loginRec.Body.Bytes(), &loginResp)
	return testLoginSession{
		User:      *loginResp.Data.User,
		Cookie:    sessionCookieFromResponse(t, loginRec.Result()),
		CSRFToken: loginResp.Data.CSRFToken,
	}
}
