package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
)

// stubBackupMetrics is a test double for backupMetricsRecorder.
type stubBackupMetrics struct {
	backupCreatedCalls []string
	restoreCalls       []bool
}

func (s *stubBackupMetrics) RecordBackupCreated(backupType string) {
	s.backupCreatedCalls = append(s.backupCreatedCalls, backupType)
}

func (s *stubBackupMetrics) RecordRestoreAttempt(success bool) {
	s.restoreCalls = append(s.restoreCalls, success)
}

func setupBackupHandlerWithMetrics(t *testing.T) (*BackupHandler, *stubBackupMetrics, *service.BackupService) {
	t.Helper()
	tempDir := t.TempDir()
	// Write flows.json so CreateTyped can succeed
	if err := os.WriteFile(filepath.Join(tempDir, "flows.json"), []byte(`[{"id":"1"}]`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	svc := service.NewBackupService(tempDir)
	h := NewBackupHandler(svc)
	stub := &stubBackupMetrics{}
	h.SetBackupMetrics(stub)
	return h, stub, svc
}

// TestPostBackup_RecordsBackupCreated verifies that creating a backup calls
// RecordBackupCreated("manual") on success.
func TestPostBackup_RecordsBackupCreated(t *testing.T) {
	h, stub, _ := setupBackupHandlerWithMetrics(t)
	// audit must not be nil for PostBackup to work
	h.SetAuditService(newNoopAudit(t))

	body := `{"type":"manual","name":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/backups", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.PostBackup(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(stub.backupCreatedCalls) != 1 {
		t.Fatalf("expected 1 RecordBackupCreated call, got %d", len(stub.backupCreatedCalls))
	}
	if stub.backupCreatedCalls[0] != "manual" {
		t.Fatalf("expected type 'manual', got '%s'", stub.backupCreatedCalls[0])
	}
}

// TestPostBackup_RecordsBackupCreatedAutoType verifies that creating an auto backup
// calls RecordBackupCreated("auto") — triangulation with different type.
func TestPostBackup_RecordsBackupCreatedAutoType(t *testing.T) {
	h, stub, _ := setupBackupHandlerWithMetrics(t)
	h.SetAuditService(newNoopAudit(t))

	body := `{"type":"auto","name":"test-auto"}`
	req := httptest.NewRequest(http.MethodPost, "/api/backups", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.PostBackup(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(stub.backupCreatedCalls) != 1 {
		t.Fatalf("expected 1 RecordBackupCreated call, got %d", len(stub.backupCreatedCalls))
	}
	if stub.backupCreatedCalls[0] != "auto" {
		t.Fatalf("expected type 'auto', got '%s'", stub.backupCreatedCalls[0])
	}
}

// TestRestoreBackup_RecordsSuccessMetric verifies that a successful restore
// calls RecordRestoreAttempt(true).
func TestRestoreBackup_RecordsSuccessMetric(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "flows.json"), []byte(`[{"id":"1"}]`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	svc := service.NewBackupService(tempDir)

	// Create a real backup first so we can restore it
	backup, err := svc.CreateTyped("manual", "test-backup")
	if err != nil {
		t.Fatalf("CreateTyped: %v", err)
	}

	h := NewBackupHandler(svc)
	stub := &stubBackupMetrics{}
	h.SetBackupMetrics(stub)
	h.SetAuditService(newNoopAudit(t))

	router := chi.NewRouter()
	router.Post("/api/backups/{id}/restore", h.RestoreBackup)

	req := httptest.NewRequest(http.MethodPost, "/api/backups/"+backup.ID+"/restore", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on restore, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(stub.restoreCalls) != 1 {
		t.Fatalf("expected 1 RecordRestoreAttempt call, got %d", len(stub.restoreCalls))
	}
	if stub.restoreCalls[0] != true {
		t.Fatalf("expected RecordRestoreAttempt(true) on success")
	}
}

// TestRestoreBackup_RecordsFailureMetric verifies that a failed restore
// calls RecordRestoreAttempt(false).
func TestRestoreBackup_RecordsFailureMetric(t *testing.T) {
	tempDir := t.TempDir()
	svc := service.NewBackupService(tempDir)
	h := NewBackupHandler(svc)
	stub := &stubBackupMetrics{}
	h.SetBackupMetrics(stub)
	h.SetAuditService(newNoopAudit(t))

	router := chi.NewRouter()
	router.Post("/api/backups/{id}/restore", h.RestoreBackup)

	// Restore a non-existent backup — should fail
	req := httptest.NewRequest(http.MethodPost, "/api/backups/nonexistent-id/restore", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("expected error status on restore of missing backup, got 200")
	}
	if len(stub.restoreCalls) != 1 {
		t.Fatalf("expected 1 RecordRestoreAttempt call, got %d", len(stub.restoreCalls))
	}
	if stub.restoreCalls[0] != false {
		t.Fatalf("expected RecordRestoreAttempt(false) on failure")
	}
}

// TestBackup_NoMetricsNilGuard verifies PostBackup and RestoreBackup work when
// no metrics recorder is set.
func TestBackup_NoMetricsNilGuard(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "flows.json"), []byte(`[{"id":"1"}]`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	svc := service.NewBackupService(tempDir)
	h := NewBackupHandler(svc)
	h.SetAuditService(newNoopAudit(t))
	// backupMetrics is nil (no SetBackupMetrics call)

	body := `{"type":"manual","name":"nil-guard"}`
	req := httptest.NewRequest(http.MethodPost, "/api/backups", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Must not panic.
	h.PostBackup(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

// newNoopAudit returns a real *audit.Service backed by a temp dir.
// It acts as a no-op for tests that don't care about audit output.
func newNoopAudit(t *testing.T) *audit.Service {
	t.Helper()
	svc, err := audit.NewService(t.TempDir())
	if err != nil {
		t.Fatalf("audit.NewService: %v", err)
	}
	return svc
}

func setupBackupHandlerWithAudit(t *testing.T) (*BackupHandler, *stubBackupMetrics) {
	t.Helper()
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "flows.json"), []byte(`[{"id":"1"}]`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	svc := service.NewBackupService(tempDir)
	h := NewBackupHandler(svc)
	stub := &stubBackupMetrics{}
	h.SetBackupMetrics(stub)
	return h, stub
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}
