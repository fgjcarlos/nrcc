package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
)

// TestDownloadBackup_MissingReturnsCleanError is the #290 regression: a missing
// backup must not produce a 200 response with zip headers and an empty/partial
// body — the error must be detected before any file headers are written.
func TestDownloadBackup_MissingReturnsCleanError(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Get("/api/backups/{id}/download", handler.DownloadBackup)

	req := httptest.NewRequest(http.MethodGet, "/api/backups/does-not-exist/download", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("missing backup must not return 200; body: %s", w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct == "application/zip" {
		t.Errorf("error response must not claim application/zip, got %q", ct)
	}
}

// TestDownloadBackup_SuccessSetsContentLength verifies a successful download
// advertises Content-Length so a client can detect a truncated stream.
func TestDownloadBackup_SuccessSetsContentLength(t *testing.T) {
	tempDir := t.TempDir()
	writeBackupFixture(t, tempDir)

	svc := service.NewBackupService(tempDir)
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Get("/api/backups/{id}/download", handler.DownloadBackup)

	req := httptest.NewRequest(http.MethodGet, "/api/backups/fixture-auto/download", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Content-Length") == "" {
		t.Error("expected Content-Length header on successful download")
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty body")
	}
}
