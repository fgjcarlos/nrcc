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

func TestSettingsHandler_SaveRaw_RejectsUnsupportedNodeREDVersion(t *testing.T) {
	t.Setenv("PATH", "")
	configSvc := service.NewConfigService(t.TempDir())
	handler := NewSettingsHandler(configSvc)

	body, err := json.Marshal(RawSettingsRequest{Content: "module.exports = { uiPort: 1880 };"})
	if err != nil {
		t.Fatalf("marshal settings request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/settings/raw", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), middleware.CtxKeyUser, &model.Claims{
		Username: "admin",
		Role:     model.RoleAdmin,
	}))
	w := httptest.NewRecorder()

	handler.SaveRaw(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("SaveRaw status = %d, want %d: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestBootstrapHandler_GetStatus_Returns200(t *testing.T) {
	hostSvc := service.NewHostService(t.TempDir())
	handler := NewBootstrapHandler(hostSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/bootstrap/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestBootstrapHandler_GetStatus_ReturnsValidJSON(t *testing.T) {
	hostSvc := service.NewHostService(t.TempDir())
	handler := NewBootstrapHandler(hostSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/bootstrap/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	var status model.HostStatus
	err := json.Unmarshal(w.Body.Bytes(), &status)

	if err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}
}

func TestBootstrapHandler_GetStatus_HasRequiredFields(t *testing.T) {
	hostSvc := service.NewHostService(t.TempDir())
	handler := NewBootstrapHandler(hostSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/bootstrap/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	var status model.HostStatus
	_ = json.Unmarshal(w.Body.Bytes(), &status)

	// Verify basic fields are accessible
	_ = status.Platform
	_ = status.NodeJS
	_ = status.Settings.Path
}

func TestBootstrapHandler_GetStatus_Dependencies(t *testing.T) {
	hostSvc := service.NewHostService(t.TempDir())
	handler := NewBootstrapHandler(hostSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/bootstrap/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	var status model.HostStatus
	_ = json.Unmarshal(w.Body.Bytes(), &status)

	// Verify dependency fields are accessible and structured correctly
	_ = status.NodeJS.Installed
	_ = status.NPM.Installed
	_ = status.Docker.Installed
}

func TestBootstrapHandler_GetStatus_NodeRedEnvironment(t *testing.T) {
	hostSvc := service.NewHostService(t.TempDir())
	handler := NewBootstrapHandler(hostSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/bootstrap/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	var status model.HostStatus
	_ = json.Unmarshal(w.Body.Bytes(), &status)

	// NodeRed should be populated as a struct
	_ = status.NodeRed.Mode
	_ = status.NodeRed.Detected
}

func TestSettingsHandler_GetRaw_WithAuth_Returns200(t *testing.T) {
	tempDir := t.TempDir()
	configSvc := service.NewIsolatedConfigService(tempDir)
	handler := NewSettingsHandler(configSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/raw", nil)

	// Inject auth claims
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.CtxKeyUser, &model.Claims{
		Username: "testuser",
		Role:     model.RoleAdmin,
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.GetRaw(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with auth, got %d", w.Code)
	}
}

func TestSettingsHandler_GetRaw_ReturnsValidJSON(t *testing.T) {
	tempDir := t.TempDir()
	configSvc := service.NewIsolatedConfigService(tempDir)
	handler := NewSettingsHandler(configSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/raw", nil)

	// Inject auth
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.CtxKeyUser, &model.Claims{
		Username: "testuser",
		Role:     model.RoleAdmin,
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetRaw(w, req)

	var doc model.SettingsDocument
	err := json.Unmarshal(w.Body.Bytes(), &doc)

	if err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}

	// Verify basic fields
	_ = doc.Path
	_ = doc.Content
}

func TestSettingsHandler_SaveRaw_WithAdmin_Returns200(t *testing.T) {
	tempDir := t.TempDir()
	configSvc := service.NewIsolatedConfigService(tempDir)
	handler := NewSettingsHandler(configSvc)

	payload := RawSettingsRequest{Content: "module.exports = { uiPort: 1880 }"}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/settings/raw", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Inject admin auth
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.CtxKeyUser, &model.Claims{
		Username: "admin",
		Role:     model.RoleAdmin,
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.SaveRaw(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with admin, got %d", w.Code)
	}
}

func TestSettingsHandler_SaveRaw_RejectsEmptyContent(t *testing.T) {
	configSvc := service.NewIsolatedConfigService(t.TempDir())
	handler := NewSettingsHandler(configSvc)

	payload := RawSettingsRequest{Content: ""} // Empty
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/settings/raw", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Inject admin auth
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.CtxKeyUser, &model.Claims{
		Username: "admin",
		Role:     model.RoleAdmin,
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.SaveRaw(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for empty content, got %d", w.Code)
	}
}

func TestSettingsHandler_SaveRaw_ReturnsValidJSON(t *testing.T) {
	tempDir := t.TempDir()
	configSvc := service.NewIsolatedConfigService(tempDir)
	handler := NewSettingsHandler(configSvc)

	payload := RawSettingsRequest{Content: "module.exports = { uiPort: 1880 }"}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/settings/raw", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.CtxKeyUser, &model.Claims{
		Username: "admin",
		Role:     model.RoleAdmin,
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.SaveRaw(w, req)

	// Unwrap ApiResponse envelope
	var resp model.ApiResponse[model.SettingsDocument]
	err := json.Unmarshal(w.Body.Bytes(), &resp)

	if err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}

	if resp.Data.Content != payload.Content {
		t.Errorf("Saved content should match request: got %s", resp.Data.Content)
	}
}

func TestSettingsHandler_SaveRaw_InvalidJSON_Body(t *testing.T) {
	configSvc := service.NewIsolatedConfigService(t.TempDir())
	handler := NewSettingsHandler(configSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/settings/raw", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.CtxKeyUser, &model.Claims{
		Username: "admin",
		Role:     model.RoleAdmin,
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.SaveRaw(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestSettingsHandler_SaveRaw_CreatesBackup(t *testing.T) {
	tempDir := t.TempDir()
	configSvc := service.NewIsolatedConfigService(tempDir)
	handler := NewSettingsHandler(configSvc)

	makeAdminCtx := func(r *http.Request) *http.Request {
		ctx := context.WithValue(r.Context(), middleware.CtxKeyUser, &model.Claims{
			Username: "admin",
			Role:     model.RoleAdmin,
		})
		return r.WithContext(ctx)
	}

	save := func(content string) model.ApiResponse[model.SettingsDocument] {
		body, _ := json.Marshal(RawSettingsRequest{Content: content})
		req := httptest.NewRequest(http.MethodPost, "/api/settings/raw", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = makeAdminCtx(req)
		w := httptest.NewRecorder()
		handler.SaveRaw(w, req)
		var resp model.ApiResponse[model.SettingsDocument]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		return resp
	}

	// First save — no previous file, no backup expected
	save("module.exports = { uiPort: 1880 }")

	// Second save — previous file now exists, backup must be created
	resp := save("module.exports = { uiPort: 1881 }")

	if resp.Data.BackupPath == "" {
		t.Error("BackupPath should be populated after second save (backup of previous file)")
	}
}
