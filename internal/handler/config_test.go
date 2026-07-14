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
)

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
	req := httptest.NewRequest("POST", "/api/config", bytes.NewReader(body))
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
					"password":    "password123",
					"permissions": "*",
				},
			},
		},
	}

	body, _ := json.Marshal(frontendPayload)
	req := httptest.NewRequest("POST", "/api/config", bytes.NewReader(body))
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

	frontendPayload := map[string]interface{}{
		"uiPort":        1880,
		"httpAdminRoot": "/",
		"httpNodeRoot":  "/",
		"adminAuth": map[string]interface{}{
			"type": "credentials",
			"users": []map[string]interface{}{
				{
					"username":    "", // EMPTY username
					"password":    "password123",
					"permissions": "*",
				},
			},
		},
	}

	body, _ := json.Marshal(frontendPayload)
	req := httptest.NewRequest("POST", "/api/config", bytes.NewReader(body))
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
	req := httptest.NewRequest("POST", "/api/config", bytes.NewReader(body))
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
					Password:    "hashedpassword123",
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
	req := httptest.NewRequest("POST", "/api/config", bytes.NewReader(body))
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

	if savedCfg.AdminAuth.Users[0].Password != "hashedpassword123" {
		t.Errorf("Expected password 'hashedpassword123', got '%s'", savedCfg.AdminAuth.Users[0].Password)
	}
}
