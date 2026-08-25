package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func mustConfigPasswordHash(t *testing.T) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("admin-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("generate bcrypt hash: %v", err)
	}
	return string(hash)
}

func TestSaveConfigWithFrontendPayload(t *testing.T) {
	// Create a temporary config service
	configSvc := service.NewIsolatedConfigService(t.TempDir())
	handler := NewConfigHandler(configSvc)

	// Frontend payload from Configuration.tsx
	frontendPayload := map[string]interface{}{
		"uiPort":          1880,
		"uiHost":          "0.0.0.0",
		"httpAdminRoot":   "/",
		"httpNodeRoot":    "/",
		"disableEditor":   false,
		"flowFile":        "flows.json",
		"userDir":         "",
		"nodesDir":        "",
		"projectsEnabled": false,
		"logging": map[string]interface{}{
			"console": map[string]interface{}{
				"level":   "info",
				"metrics": false,
			},
		},
		"editorTheme": map[string]interface{}{
			"page": map[string]interface{}{
				"title": "Node-RED",
			},
		},
		"lang": "en-US",
	}

	body, _ := json.Marshal(frontendPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Inject admin claims into context
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.CtxKeyUser, &model.Claims{
		Username: "admin",
		Role:     model.RoleAdmin,
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.SaveConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d\nResponse: %s", w.Code, w.Body.String())
	}

	// Verify the config was saved
	savedCfg, err := configSvc.Get()
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	if savedCfg.UIPort != 1880 {
		t.Errorf("Expected UIPort 1880, got %d", savedCfg.UIPort)
	}
}

// TestSaveConfigWithAdminAuth verifies admin auth can be saved with valid credentials
func TestSaveConfigWithAdminAuth(t *testing.T) {
	configSvc := service.NewIsolatedConfigService(t.TempDir())
	handler := NewConfigHandler(configSvc)
	hash := mustConfigPasswordHash(t)

	// Frontend payload with admin auth enabled
	frontendPayload := map[string]interface{}{
		"uiPort":          1880,
		"uiHost":          "0.0.0.0",
		"httpAdminRoot":   "/",
		"httpNodeRoot":    "/",
		"projectsEnabled": false,
		"adminAuth": map[string]interface{}{
			"type": "credentials",
			"users": []map[string]interface{}{
				{
					"username":    "admin",
					"password":    hash,
					"permissions": "*",
				},
			},
		},
	}

	body, _ := json.Marshal(frontendPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.CtxKeyUser, &model.Claims{
		Username: "admin",
		Role:     model.RoleAdmin,
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.SaveConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d\nResponse: %s", w.Code, w.Body.String())
	}

	savedCfg, err := configSvc.Get()
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	if savedCfg.AdminAuth == nil {
		t.Errorf("AdminAuth was not saved")
	}
	if len(savedCfg.AdminAuth.Users) == 0 {
		t.Errorf("AdminAuth users were not saved")
	}
	if savedCfg.AdminAuth.Users[0].Username != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", savedCfg.AdminAuth.Users[0].Username)
	}
}

// TestSaveConfigAdminAuthRequiresUsername verifies that empty username is rejected
func TestSaveConfigAdminAuthRequiresUsername(t *testing.T) {
	configSvc := service.NewIsolatedConfigService(t.TempDir())
	handler := NewConfigHandler(configSvc)
	hash := mustConfigPasswordHash(t)

	frontendPayload := map[string]interface{}{
		"uiPort":        1880,
		"httpAdminRoot": "/",
		"httpNodeRoot":  "/",
		"adminAuth": map[string]interface{}{
			"type": "credentials",
			"users": []map[string]interface{}{
				{
					"username":    "", // EMPTY username
					"password":    hash,
					"permissions": "*",
				},
			},
		},
	}

	body, _ := json.Marshal(frontendPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.CtxKeyUser, &model.Claims{
		Username: "admin",
		Role:     model.RoleAdmin,
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.SaveConfig(w, req)

	if w.Code == http.StatusOK {
		t.Errorf("Expected validation error (400), but got 200 OK - empty username was accepted!")
	}
}

// TestSaveConfigAdminAuthRequiresPassword verifies that empty password is rejected when no existing config
func TestSaveConfigAdminAuthRequiresPassword(t *testing.T) {
	configSvc := service.NewIsolatedConfigService(t.TempDir())
	handler := NewConfigHandler(configSvc)

	frontendPayload := map[string]interface{}{
		"uiPort":        1880,
		"httpAdminRoot": "/",
		"httpNodeRoot":  "/",
		"adminAuth": map[string]interface{}{
			"type": "credentials",
			"users": []map[string]interface{}{
				{
					"username":    "admin",
					"password":    "", // EMPTY password
					"permissions": "*",
				},
			},
		},
	}

	body, _ := json.Marshal(frontendPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.CtxKeyUser, &model.Claims{
		Username: "admin",
		Role:     model.RoleAdmin,
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.SaveConfig(w, req)

	if w.Code == http.StatusOK {
		t.Errorf("Expected validation error (400), but got 200 OK - empty password was accepted!")
	}
}

// TestSaveConfigAdminAuthPreservePassword verifies that empty password preserves existing hash
func TestSaveConfigAdminAuthPreservePassword(t *testing.T) {
	tempDir := t.TempDir()
	configSvc := service.NewIsolatedConfigService(tempDir)
	hash := mustConfigPasswordHash(t)

	// First, save an initial config with a password
	initialCfg := model.NodeRedConfig{
		Port:          1880,
		UIPort:        1880,
		HTTPAdminRoot: "/",
		HTTPNodeRoot:  "/",
		AdminAuth: &model.AdminAuth{
			Type: "credentials",
			Users: []model.AdminAuthUser{
				{
					Username:    "admin",
					Password:    hash,
					Permissions: "*",
				},
			},
		},
	}
	_ = configSvc.Save(initialCfg)

	// Now update with empty password (frontend sends empty to mean "don't change")
	handler := NewConfigHandler(configSvc)
	updatePayload := map[string]interface{}{
		"uiPort":        1880,
		"httpAdminRoot": "/",
		"httpNodeRoot":  "/",
		"adminAuth": map[string]interface{}{
			"type": "credentials",
			"users": []map[string]interface{}{
				{
					"username":    "admin",
					"password":    "", // Empty - should preserve "hashedpassword123"
					"permissions": "*",
				},
			},
		},
	}

	body, _ := json.Marshal(updatePayload)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.CtxKeyUser, &model.Claims{
		Username: "admin",
		Role:     model.RoleAdmin,
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.SaveConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d\nResponse: %s", w.Code, w.Body.String())
	}

	savedCfg, err := configSvc.Get()
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	if savedCfg.AdminAuth == nil || len(savedCfg.AdminAuth.Users) == 0 {
		t.Fatalf("AdminAuth was not saved")
	}

	if savedCfg.AdminAuth.Users[0].Password != hash {
		t.Errorf("Expected password %q, got '%s'", hash, savedCfg.AdminAuth.Users[0].Password)
	}
}

// seedRedactionConfig writes a NodeRedConfig containing a bcrypt password and
// two env vars (one encrypted blob, one cleartext secret) into the isolated
// service so the GetConfig redaction tests have a deterministic fixture.
func seedRedactionConfig(t *testing.T, configSvc *service.ConfigService, hash string) {
	t.Helper()

	initial := model.NodeRedConfig{
		Port:          1880,
		UIPort:        1880,
		HTTPAdminRoot: "/",
		HTTPNodeRoot:  "/",
		AdminAuth: &model.AdminAuth{
			Type: "credentials",
			Users: []model.AdminAuthUser{
				{
					Username:    "admin",
					Password:    hash,
					Permissions: "*",
				},
			},
		},
		EnvVars: []model.EnvVar{
			{Key: "PLAINTEXT_SECRET", Value: "supersecret-cleartext", Type: "string"},
			{Key: "ENCRYPTED_SECRET", Value: "v1:already-encrypted-blob", Type: "secret", Encrypted: true},
		},
	}
	if err := configSvc.Save(initial); err != nil {
		t.Fatalf("seed config: %v", err)
	}
}

// TestConfigHandler_GetConfig_Viewer_RedactsSecrets is the MEDIUM-015 RED
// case: viewers must NOT see AdminAuth.Users[*].Password or
// EnvVars[*].Value (cleartext). Encrypted env blobs are not in cleartext, so
// they pass through unchanged.
func TestConfigHandler_GetConfig_Viewer_RedactsSecrets(t *testing.T) {
	configSvc := service.NewIsolatedConfigService(t.TempDir())
	handler := NewConfigHandler(configSvc)
	hash := mustConfigPasswordHash(t)
	seedRedactionConfig(t, configSvc, hash)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.CtxKeyUser, &model.Claims{
		Username: "viewer",
		Role:     model.RoleViewer,
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d\nResponse: %s", w.Code, w.Body.String())
	}

	var resp model.ApiResponse[model.NodeRedConfig]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response must be valid JSON: %v\nbody: %s", err, w.Body.String())
	}

	cfg := resp.Data
	if cfg.AdminAuth == nil || len(cfg.AdminAuth.Users) == 0 {
		t.Fatalf("expected adminAuth.users to be present")
	}
	for i, u := range cfg.AdminAuth.Users {
		if u.Password != "" {
			t.Errorf("AdminAuth.Users[%d].Password must be redacted for viewer; got non-empty value (len=%d)", i, len(u.Password))
		}
	}

	if len(cfg.EnvVars) != 2 {
		t.Fatalf("expected 2 env vars in fixture, got %d", len(cfg.EnvVars))
	}
	for i, ev := range cfg.EnvVars {
		if !ev.Encrypted {
			if ev.Value != "********" {
				t.Errorf("EnvVars[%d] (non-encrypted, key=%q): expected value %q for viewer, got %q", i, ev.Key, "********", ev.Value)
			}
		}
	}

	// Encrypted blob must pass through unchanged — the redaction must NOT
	// touch the cipher blob, only cleartext values.
	var foundEncrypted bool
	for _, ev := range cfg.EnvVars {
		if ev.Encrypted {
			foundEncrypted = true
			if ev.Value != "v1:already-encrypted-blob" {
				t.Errorf("encrypted env var value must be preserved; got %q", ev.Value)
			}
		}
	}
	if !foundEncrypted {
		t.Errorf("expected encrypted env var to be present in response")
	}

	if cfg.Port != 1880 {
		t.Errorf("expected Port=1880 (non-sensitive field preserved), got %d", cfg.Port)
	}
}

// TestConfigHandler_GetConfig_Admin_ReturnsFullConfig is the MEDIUM-015 GREEN
// regression guard: admins still see AdminAuth.Users[*].Password and cleartext
// EnvVar values without modification.
func TestConfigHandler_GetConfig_Admin_ReturnsFullConfig(t *testing.T) {
	configSvc := service.NewIsolatedConfigService(t.TempDir())
	handler := NewConfigHandler(configSvc)
	hash := mustConfigPasswordHash(t)
	seedRedactionConfig(t, configSvc, hash)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.CtxKeyUser, &model.Claims{
		Username: "admin",
		Role:     model.RoleAdmin,
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d\nResponse: %s", w.Code, w.Body.String())
	}

	var resp model.ApiResponse[model.NodeRedConfig]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response must be valid JSON: %v\nbody: %s", err, w.Body.String())
	}

	cfg := resp.Data
	if cfg.AdminAuth == nil || cfg.AdminAuth.Users[0].Password != hash {
		t.Errorf("admin must see full password hash; got %q", cfg.AdminAuth.Users[0].Password)
	}
	var foundPlain bool
	for _, ev := range cfg.EnvVars {
		if !ev.Encrypted && ev.Value == "supersecret-cleartext" {
			foundPlain = true
		}
		if ev.Encrypted && ev.Value != "v1:already-encrypted-blob" {
			t.Errorf("encrypted env var corrupted: got %q", ev.Value)
		}
	}
	if !foundPlain {
		t.Errorf("admin must see cleartext env var value; not found in response")
	}
}

// TestSaveConfig_NoProcessManager_DoesNotPanic is the regression guard for
// #715: ConfigHandler.SaveConfig previously had no dependency on a
// ProcessManager; adding the auto-restart hook must not break any existing
// test setup that uses NewConfigHandler without SetProcessManager.
func TestSaveConfig_NoProcessManager_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("SaveConfig panicked without ProcessManager: %v", r)
		}
	}()
	configSvc := service.NewIsolatedConfigService(t.TempDir())
	handler := NewConfigHandler(configSvc) // processManager intentionally nil

	payload := map[string]interface{}{
		"uiPort":          1880,
		"uiHost":          "0.0.0.0",
		"httpAdminRoot":   "/",
		"httpNodeRoot":    "/",
		"flowFile":        "flows.json",
		"projectsEnabled": false,
		"logging":         map[string]interface{}{},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/config/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.CtxKeyUser, &model.Claims{
		Username: "admin",
		Role:     model.RoleAdmin,
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.SaveConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

// TestSaveConfig_WiredButStopped_DoesNotCallRestart pins that an unwired
// or stopped Node-RED process does NOT receive a Restart call after a
// successful Save. The dashboard's editor-theme save should auto-restart
// only when the runtime is actually running (#715).
func TestSaveConfig_WiredButStopped_DoesNotCallRestart(t *testing.T) {
	// We cannot substitute a mock ProcessManager (it is a concrete type),
	// so the strongest claim we can make without starting a real Node-RED
	// is: the handler completes synchronously without panicking when the
	// wired PM reports Status() == "stopped" (the default for a freshly
	// constructed PM). This is the negative half of the auto-restart
	// contract: the path the user expects (running → restart) is covered
	// by the integration test scripts/acceptance/docker-stacks.sh.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("SaveConfig panicked with stopped ProcessManager: %v", r)
		}
	}()

	configSvc := service.NewIsolatedConfigService(t.TempDir())
	handler := NewConfigHandler(configSvc)
	handler.SetProcessManager(service.NewProcessManager("node-red", t.TempDir()))

	payload := map[string]interface{}{
		"uiPort":          1880,
		"uiHost":          "0.0.0.0",
		"httpAdminRoot":   "/",
		"httpNodeRoot":    "/",
		"flowFile":        "flows.json",
		"projectsEnabled": false,
		"logging":         map[string]interface{}{},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/config/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.CtxKeyUser, &model.Claims{
		Username: "admin",
		Role:     model.RoleAdmin,
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.SaveConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}
