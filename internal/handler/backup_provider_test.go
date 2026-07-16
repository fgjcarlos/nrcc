package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
)

// TestGetBackupProviderReturnsLocalByDefault proves the endpoint reports
// the noop provider when no remote one is configured.
func TestGetBackupProviderReturnsLocalByDefault(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Get("/api/backups/provider", handler.GetBackupProvider)

	req := httptest.NewRequest(http.MethodGet, "/api/backups/provider", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp model.ApiResponse[map[string]string]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data["provider"] != "local" {
		t.Fatalf("expected provider=local, got %q", resp.Data["provider"])
	}
}

// TestListProviderSnapshotsReturns503WhenDisabled ensures the endpoint
// reports a clean 503 (not a 500) when no provider is configured.
func TestListProviderSnapshotsReturns503WhenDisabled(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Get("/api/backups/provider/snapshots", handler.ListProviderSnapshots)

	req := httptest.NewRequest(http.MethodGet, "/api/backups/provider/snapshots", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

// TestRestoreProviderSnapshotRejectsMissingID sends an explicit empty id
// so the JSON decoder succeeds and the handler's dedicated 400 branch
// ("id is required") is the one that runs.
func TestRestoreProviderSnapshotRejectsMissingID(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Post("/api/backups/provider/restore", handler.RestoreProviderSnapshot)

	req := httptest.NewRequest(http.MethodPost, "/api/backups/provider/restore", bytes.NewReader([]byte(`{"id":"","destination":"/tmp/x"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestRestoreProviderSnapshotRejectsNonAbsoluteDestination ensures the
// handler refuses to hand a relative path to the provider layer.
func TestRestoreProviderSnapshotRejectsNonAbsoluteDestination(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Post("/api/backups/provider/restore", handler.RestoreProviderSnapshot)

	req := httptest.NewRequest(http.MethodPost, "/api/backups/provider/restore", bytes.NewReader([]byte(`{"id":"abc123def","destination":"relative/path"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestRestoreProviderSnapshotReturns503WhenDisabled ensures the write
// endpoint also gates on the provider being configured.
func TestRestoreProviderSnapshotReturns503WhenDisabled(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Post("/api/backups/provider/restore", handler.RestoreProviderSnapshot)

	body := `{"id":"abc","destination":"/tmp/x"}`
	req := httptest.NewRequest(http.MethodPost, "/api/backups/provider/restore", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}