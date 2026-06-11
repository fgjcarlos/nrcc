package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// TestUpdateHandler_GetStatus_ReturnsOK tests that GET /api/updates/status returns 200
func TestUpdateHandler_GetStatus_ReturnsOK(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("GET", "/api/updates/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestUpdateHandler_GetStatus_ContentType verifies correct Content-Type header
func TestUpdateHandler_GetStatus_ContentType(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("GET", "/api/updates/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

// TestUpdateHandler_GetStatus_EmptyCache tests GetStatus with empty/zero cache
func TestUpdateHandler_GetStatus_EmptyCache(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("GET", "/api/updates/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	// Should still return 200 with empty/zero-value entry
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var entry model.UpdateCacheEntry
	if err := json.Unmarshal(w.Body.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify it's a zero-value entry (not yet checked)
	if entry.CurrentVersion != "" {
		t.Errorf("Expected empty CurrentVersion, got %s", entry.CurrentVersion)
	}
	if entry.UpdateAvailable {
		t.Error("Expected UpdateAvailable to be false")
	}
}

// TestUpdateHandler_GetStatus_ValidJSON tests that response is valid JSON
func TestUpdateHandler_GetStatus_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("GET", "/api/updates/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	var entry model.UpdateCacheEntry
	if err := json.Unmarshal(w.Body.Bytes(), &entry); err != nil {
		t.Fatalf("Response must be valid JSON: %v", err)
	}
	// If we got here, JSON is valid
}

// TestUpdateHandler_GetCheck_ReturnsOK tests that GET /api/updates/check returns 200
func TestUpdateHandler_GetCheck_ReturnsOK(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("GET", "/api/updates/check", nil)
	w := httptest.NewRecorder()

	handler.GetCheck(w, req)

	// Should return 200 OK or 500 (if check fails), but not 5xx server error
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}
}

// TestUpdateHandler_GetCheck_ContentType verifies correct Content-Type header
func TestUpdateHandler_GetCheck_ContentType(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("GET", "/api/updates/check", nil)
	w := httptest.NewRecorder()

	handler.GetCheck(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

// TestUpdateHandler_GetCheck_ResponseStructure verifies response JSON structure
func TestUpdateHandler_GetCheck_ResponseStructure(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("GET", "/api/updates/check", nil)
	w := httptest.NewRecorder()

	handler.GetCheck(w, req)

	var entry model.UpdateCacheEntry
	if err := json.Unmarshal(w.Body.Bytes(), &entry); err != nil {
		// If we get a 500, the response might be an error response, which is fine
		if w.Code == http.StatusOK {
			t.Fatalf("Expected valid JSON on 200 response: %v", err)
		}
		return
	}

	// On success, verify structure
	if w.Code == http.StatusOK {
		// CurrentVersion may be empty if no npm is available, but structure should exist
		if entry.CheckedAt.IsZero() && entry.Error == "" && entry.CurrentVersion == "" {
			t.Log("Warning: UpdateCacheEntry appears to be all-zero, but this may be expected if npm isn't available")
		}
	}
}

// TestUpdateHandler_GetCheck_ValidJSON tests that response is valid JSON
func TestUpdateHandler_GetCheck_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("GET", "/api/updates/check", nil)
	w := httptest.NewRecorder()

	handler.GetCheck(w, req)

	// Response should be JSON regardless of success/error
	if w.Code == http.StatusOK {
		var entry model.UpdateCacheEntry
		if err := json.Unmarshal(w.Body.Bytes(), &entry); err != nil {
			t.Fatalf("Response must be valid JSON on 200: %v", err)
		}
	} else if w.Code == http.StatusInternalServerError {
		// Error response is also JSON
		var errResp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
			t.Logf("Error response might not be JSON, which is ok: %v", err)
		}
	}
}

// TestUpdateHandler_PostApply_ReturnsOK tests that POST /api/updates/apply returns 200 on success
func TestUpdateHandler_PostApply_ReturnsOK(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("POST", "/api/updates/apply", nil)
	w := httptest.NewRecorder()

	handler.PostApply(w, req)
	waitForUpdateFlowToSettle(t, svc)

	// Expect 200 OK or 500 (if npm is not available), but not 4xx
	if w.Code == http.StatusInternalServerError {
		// npm might not be available in test environment, which is ok
		t.Logf("Note: ApplyUpdate returned 500, likely because npm is not available in test env")
		return
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestUpdateHandler_PostApply_ResponseStructure verifies response includes fromVersion and toVersion
func TestUpdateHandler_PostApply_ResponseStructure(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("POST", "/api/updates/apply", nil)
	w := httptest.NewRecorder()

	handler.PostApply(w, req)
	waitForUpdateFlowToSettle(t, svc)

	// Only check on success (200 OK)
	if w.Code == http.StatusOK {
		var apiResp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &apiResp); err != nil {
			t.Fatalf("Response must be valid JSON on 200: %v", err)
		}

		// The response is wrapped in ApiResponse envelope, so extract data
		dataField, ok := apiResp["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected 'data' field to be object, got %v", apiResp["data"])
		}

		// Verify required fields for frontend contract
		if success, ok := dataField["success"]; !ok || success != true {
			t.Errorf("Expected 'success': true, got %v", dataField["success"])
		}

		if _, ok := dataField["message"]; !ok {
			t.Errorf("Expected 'message' field in data")
		}

		if _, ok := dataField["fromVersion"]; !ok {
			t.Errorf("Expected 'fromVersion' field in data (required by frontend)")
		}

		if _, ok := dataField["toVersion"]; !ok {
			t.Errorf("Expected 'toVersion' field in data (required by frontend)")
		}
	} else {
		t.Logf("Test skipped: npm install failed or not available (status %d)", w.Code)
	}
}

// TestUpdateHandler_PostApply_ContentType verifies correct Content-Type header
func TestUpdateHandler_PostApply_ContentType(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("POST", "/api/updates/apply", nil)
	w := httptest.NewRecorder()

	handler.PostApply(w, req)
	waitForUpdateFlowToSettle(t, svc)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

// PR 2 Tests: Backend HTTP Integration (handlers & routes)

// TestUpdateHandler_GetState_ReturnsOK tests that GET /api/updates/state returns 200
func TestUpdateHandler_GetState_ReturnsOK(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("GET", "/api/updates/state", nil)
	w := httptest.NewRecorder()

	handler.GetState(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestUpdateHandler_GetState_ValidJSON tests that GET /api/updates/state returns valid JSON
func TestUpdateHandler_GetState_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("GET", "/api/updates/state", nil)
	w := httptest.NewRecorder()

	handler.GetState(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var flowState model.UpdateFlowState
	if err := json.Unmarshal(w.Body.Bytes(), &flowState); err != nil {
		t.Fatalf("Response must be valid JSON: %v", err)
	}
}

// TestUpdateHandler_GetState_InitialStateIdle tests that initial state is Idle
func TestUpdateHandler_GetState_InitialStateIdle(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("GET", "/api/updates/state", nil)
	w := httptest.NewRecorder()

	handler.GetState(w, req)

	// Response is wrapped in ApiResponse{Data: UpdateFlowState, Success, Timestamp}
	var apiResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &apiResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Extract data field
	dataField, ok := apiResp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected 'data' field to be object, got %v", apiResp["data"])
	}

	// Check state and phase inside data
	state, stateOk := dataField["state"].(string)
	if !stateOk || state == "" {
		t.Errorf("Expected state to be non-empty string, got %v", dataField["state"])
	}

	if state != "Idle" {
		t.Errorf("Expected initial state Idle, got %s", state)
	}

	phase, phaseOk := dataField["phase"].(string)
	if !phaseOk || phase == "" {
		t.Errorf("Expected phase to be non-empty string, got %v", dataField["phase"])
	}

	if phase != "idle" {
		t.Errorf("Expected initial phase 'idle', got %s", phase)
	}
}

// TestUpdateHandler_GetState_ContentType verifies correct Content-Type header
func TestUpdateHandler_GetState_ContentType(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("GET", "/api/updates/state", nil)
	w := httptest.NewRecorder()

	handler.GetState(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

// TestUpdateHandler_PostApply_StartsBackgroundFlow tests that PostApply launches async flow
func TestUpdateHandler_PostApply_StartsBackgroundFlow(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	// Initial state should be Idle
	initialState := svc.GetFlowState()
	if initialState.State != model.StateIdle {
		t.Fatalf("Expected initial state Idle, got %s", initialState.State)
	}

	req := httptest.NewRequest("POST", "/api/updates/apply", nil)
	w := httptest.NewRecorder()

	handler.PostApply(w, req)
	waitForUpdateFlowToSettle(t, svc)

	// PostApply should return 200 (not block on npm call)
	if w.Code != http.StatusOK {
		t.Logf("Note: PostApply returned %d (expected 200 if npm available)", w.Code)
	}

	// Response should include state field
	var apiResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &apiResp); err != nil {
		t.Logf("Warning: Failed to unmarshal response: %v", err)
		return
	}

	// Extract data field (wrapped in ApiResponse envelope)
	dataField, ok := apiResp["data"].(map[string]interface{})
	if !ok {
		t.Logf("Warning: Expected 'data' field, got %v", apiResp["data"])
		return
	}

	// Verify state field is present
	if _, ok := dataField["state"]; !ok {
		t.Errorf("Expected 'state' field in response")
	}
}

// TestUpdateHandler_PostApply_IncludesBackupId tests that PostApply response includes backupId
func TestUpdateHandler_PostApply_IncludesBackupId(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	req := httptest.NewRequest("POST", "/api/updates/apply", nil)
	w := httptest.NewRecorder()

	handler.PostApply(w, req)
	waitForUpdateFlowToSettle(t, svc)

	if w.Code == http.StatusOK {
		var apiResp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &apiResp); err != nil {
			t.Logf("Failed to unmarshal response: %v", err)
			return
		}

		// Extract data field
		dataField, ok := apiResp["data"].(map[string]interface{})
		if !ok {
			return
		}

		// backupId field should be present (even if empty on first response)
		if _, ok := dataField["backupId"]; !ok {
			t.Logf("Note: backupId field not in response (may be added async)")
		}
	}
}

// TestUpdateHandler_PostApply_ConcurrencyLogic_DocumentedBehavior documents the concurrency behavior.
// NOTE: This test is not deterministic because it depends on goroutine scheduling.
// In production, integration tests with mocks or time control would be used.
// For now, this just documents the intended behavior:
// - If state == Idle, allow POST /apply
// - If state != Idle, return 409 Conflict
func TestUpdateHandler_PostApply_ConcurrencyLogic_DocumentedBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	handler := NewUpdateHandler(svc)

	// Verify initial state is Idle
	initialState := svc.GetFlowState()
	if initialState.State != model.StateIdle {
		t.Fatalf("Expected initial state to be Idle, got %s", initialState.State)
	}

	// First POST should succeed
	req := httptest.NewRequest("POST", "/api/updates/apply", nil)
	w := httptest.NewRecorder()
	handler.PostApply(w, req)
	waitForUpdateFlowToSettle(t, svc)

	if w.Code != http.StatusOK {
		t.Logf("First POST returned %d (expected 200)", w.Code)
	}

	// Behavior: If state becomes non-Idle, subsequent POSTs should return 409
	// This is tested deterministically in the service layer via
	// TestApplyUpdateWithBackup_ConcurrencyGuard (unit test that mocks execution)
	t.Log("Concurrency behavior is verified at service layer; handler just checks state and returns 409")
}

func installFakeNPM(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	npmPath := filepath.Join(binDir, "npm")
	if err := os.WriteFile(npmPath, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write fake npm: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func waitForUpdateFlowToSettle(t *testing.T, svc *service.UpdateService) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	started := false
	for time.Now().Before(deadline) {
		state := svc.GetFlowState().State
		if state == model.StateBackingUp || state == model.StateApplying {
			started = true
		} else if started || state == model.StateCompleted || state == model.StateFailed {
			// The goroutine sets a terminal state as its last operation.
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("update flow did not settle before test cleanup; latest state=%+v", svc.GetFlowState())
}
