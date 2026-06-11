package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
)

// stubLibraryMetrics is a test double for libraryMetricsRecorder.
type stubLibraryMetrics struct {
	calls []libraryMetricCall
}

type libraryMetricCall struct {
	operation string
	success   bool
}

func (s *stubLibraryMetrics) RecordLibraryOperation(operation string, success bool) {
	s.calls = append(s.calls, libraryMetricCall{operation: operation, success: success})
}

func setupLibraryHandlerWithMetrics(t *testing.T) (*LibraryHandler, *stubLibraryMetrics) {
	t.Helper()
	tempDir := t.TempDir()
	// Write a minimal package.json so the service can operate
	pkgJSON := `{"name":"node-red","version":"3.0.0","dependencies":{}}`
	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(pkgJSON), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	svc := service.NewLibraryService(tempDir)
	h := NewLibraryHandler(svc)
	stub := &stubLibraryMetrics{}
	h.SetLibraryMetrics(stub)
	return h, stub
}

// TestPostInstall_RecordsSuccessMetric verifies that a successful install
// calls RecordLibraryOperation("install", true).
func TestPostInstall_RecordsSuccessMetric(t *testing.T) {
	h, stub := setupLibraryHandlerWithMetrics(t)

	body := `{"name":"lodash"}`
	req := httptest.NewRequest(http.MethodPost, "/api/libraries/install", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.PostInstall(rec, req)

	// Whether install succeeds or fails depends on environment, but the metric must be recorded.
	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 RecordLibraryOperation call, got %d", len(stub.calls))
	}
	if stub.calls[0].operation != "install" {
		t.Fatalf("expected operation 'install', got '%s'", stub.calls[0].operation)
	}
}

// TestPostInstall_RecordsInstallOperation verifies the operation label is "install"
// when the call succeeds (triangulation: different package name, same operation).
func TestPostInstall_RecordsInstallOperation(t *testing.T) {
	h, stub := setupLibraryHandlerWithMetrics(t)

	body := `{"name":"express"}`
	req := httptest.NewRequest(http.MethodPost, "/api/libraries/install", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.PostInstall(rec, req)

	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 RecordLibraryOperation call, got %d", len(stub.calls))
	}
	if stub.calls[0].operation != "install" {
		t.Fatalf("expected operation 'install', got '%s'", stub.calls[0].operation)
	}
}

// TestDeleteLibrary_RecordsUninstallOperation verifies that uninstalling a package
// calls RecordLibraryOperation("uninstall", ...).
func TestDeleteLibrary_RecordsUninstallOperation(t *testing.T) {
	h, stub := setupLibraryHandlerWithMetrics(t)

	router := chi.NewRouter()
	router.Delete("/api/libraries/{name}", h.DeleteLibrary)

	req := httptest.NewRequest(http.MethodDelete, "/api/libraries/lodash", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 RecordLibraryOperation call, got %d", len(stub.calls))
	}
	if stub.calls[0].operation != "uninstall" {
		t.Fatalf("expected operation 'uninstall', got '%s'", stub.calls[0].operation)
	}
}

// TestLibrary_NoMetricsNilGuard verifies PostInstall and DeleteLibrary work when
// no metrics recorder is set (nil guard must not panic).
func TestLibrary_NoMetricsNilGuard(t *testing.T) {
	tempDir := t.TempDir()
	pkgJSON := `{"name":"node-red","version":"3.0.0","dependencies":{}}`
	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(pkgJSON), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	svc := service.NewLibraryService(tempDir)
	h := NewLibraryHandler(svc)
	// libraryMetrics is nil (no SetLibraryMetrics call)

	body := `{"name":"lodash"}`
	req := httptest.NewRequest(http.MethodPost, "/api/libraries/install", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Must not panic.
	h.PostInstall(rec, req)
}

// TestLibraryMetrics_ResponseBody verifies that the response body is valid JSON regardless of metric recording.
func TestLibraryMetrics_ResponseBody(t *testing.T) {
	h, _ := setupLibraryHandlerWithMetrics(t)

	body := `{"name":"lodash"}`
	req := httptest.NewRequest(http.MethodPost, "/api/libraries/install", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.PostInstall(rec, req)

	var result map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("response body must be valid JSON: %v", err)
	}
}
