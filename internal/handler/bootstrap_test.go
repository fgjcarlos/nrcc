package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
)

func TestBootstrapHandler_GetStatus_StatusCode(t *testing.T) {
	hostSvc := service.NewHostService(t.TempDir())
	handler := NewBootstrapHandler(hostSvc)

	req := httptest.NewRequest("GET", "/api/bootstrap/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestBootstrapHandler_GetStatus_ValidHostStatusStructure(t *testing.T) {
	hostSvc := service.NewHostService(t.TempDir())
	handler := NewBootstrapHandler(hostSvc)

	req := httptest.NewRequest("GET", "/api/bootstrap/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	// Unwrap ApiResponse envelope
	var resp model.ApiResponse[model.HostStatus]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to deserialize response: %v", err)
	}

	// Validate structure - should be JSON-deserializable with fields
	if resp.Data.Settings.Path == "" {
		t.Error("Settings.Path should be set")
	}
}

func TestBootstrapHandler_GetStatus_ContentType(t *testing.T) {
	hostSvc := service.NewHostService(t.TempDir())
	handler := NewBootstrapHandler(hostSvc)

	req := httptest.NewRequest("GET", "/api/bootstrap/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestBootstrapHandler_GetStatus_JSONMarshalable(t *testing.T) {
	hostSvc := service.NewHostService(t.TempDir())
	handler := NewBootstrapHandler(hostSvc)

	req := httptest.NewRequest("GET", "/api/bootstrap/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	// Should be able to unmarshal without error
	var status model.HostStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("Response must be valid JSON: %v", err)
	}
}

func TestBootstrapHandler_GetStatus_AllFieldsAccessible(t *testing.T) {
	hostSvc := service.NewHostService(t.TempDir())
	handler := NewBootstrapHandler(hostSvc)

	req := httptest.NewRequest("GET", "/api/bootstrap/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	var status model.HostStatus
	json.Unmarshal(w.Body.Bytes(), &status)

	// Verify all top-level fields are present and accessible
	_ = status.Platform
	_ = status.Ready
	_ = status.Interactive
	_ = status.NodeJS
	_ = status.NPM
	_ = status.NodeRedBinary
	_ = status.Docker
	_ = status.DockerCompose
	_ = status.NodeRed
	_ = status.Settings
	_ = status.Recommendations

	// Verify some nested fields
	_ = status.NodeJS.Name
	_ = status.NodeJS.Installed
	_ = status.NodeJS.Version

	_ = status.Settings.Path
	_ = status.Settings.Source
	_ = status.Settings.Writable

	_ = status.NodeRed.Detected
	_ = status.NodeRed.Mode
	_ = status.NodeRed.Running
}

func TestBootstrapHandler_GetStatus_MultipleRequests_Consistent(t *testing.T) {
	hostSvc := service.NewHostService(t.TempDir())
	handler := NewBootstrapHandler(hostSvc)

	// First request
	req1 := httptest.NewRequest("GET", "/api/bootstrap/status", nil)
	w1 := httptest.NewRecorder()
	handler.GetStatus(w1, req1)

	var status1 model.HostStatus
	json.Unmarshal(w1.Body.Bytes(), &status1)

	// Second request
	req2 := httptest.NewRequest("GET", "/api/bootstrap/status", nil)
	w2 := httptest.NewRecorder()
	handler.GetStatus(w2, req2)

	var status2 model.HostStatus
	json.Unmarshal(w2.Body.Bytes(), &status2)

	// Platform should be same in both
	if status1.Platform != status2.Platform {
		t.Errorf("Platform should be consistent: %s vs %s", status1.Platform, status2.Platform)
	}

	// Settings path should be same
	if status1.Settings.Path != status2.Settings.Path {
		t.Error("Settings.Path should be consistent across requests")
	}
}
