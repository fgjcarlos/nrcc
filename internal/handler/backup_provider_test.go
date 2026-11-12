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
	"github.com/go-chi/chi/v5"
)

// authedClaims returns a request whose context carries the given role.
func authedClaims(t *testing.T, username string, role model.UserRole) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/backups/provider", nil)
	ctx := context.WithValue(req.Context(), middleware.CtxKeyUser, &model.Claims{
		Username: username,
		Role:     role,
	})
	return req.WithContext(ctx)
}

// TestGetBackupProvider_AdminReturnsLocal proves the endpoint reports
// the noop provider when no remote one is configured (admin path).
func TestGetBackupProvider_AdminReturnsLocal(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Get("/api/backups/provider", handler.GetBackupProvider)

	req := authedClaims(t, "admin", model.RoleAdmin)
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

// TestGetBackupProvider_ViewerReturnsNull is the MEDIUM-017 RED case: viewers
// must not be able to tell whether a remote provider is configured. The
// endpoint must return 200 with an explicit JSON null for the provider field.
func TestGetBackupProvider_ViewerReturnsNull(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Get("/api/backups/provider", handler.GetBackupProvider)

	req := authedClaims(t, "viewer", model.RoleViewer)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var raw struct {
		Data map[string]*string `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("response must be valid JSON: %v\nbody: %s", err, w.Body.String())
	}
	got, exists := raw.Data["provider"]
	if !exists || got != nil {
		t.Fatalf("expected provider=null for viewer, got exists=%v value=%v (body=%s)", exists, got, w.Body.String())
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
	dataDir := t.TempDir()
	svc := service.NewBackupService(dataDir)
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Post("/api/backups/provider/restore", handler.RestoreProviderSnapshot)

	// Destination must resolve inside the service's dataDir so the new
	// validateRestoreDestination containment check (HIGH-007) does not
	// short-circuit with 400 before we reach the provider-availability
	// branch. The handler returns 503 because no remote provider is
	// configured on a fresh BackupService.
	body := `{"id":"abc","destination":"` + dataDir + `/restore-target"}`
	req := httptest.NewRequest(http.MethodPost, "/api/backups/provider/restore", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

// TestRestoreProviderSnapshotRejectsEscapeDestination covers HIGH-007 at the
// HTTP boundary: a destination that resolves outside the configured
// backup root (NRCC_BACKUP_ROOT env, fallback = service dataDir) must be
// rejected with HTTP 400 before the provider layer is touched.
func TestRestoreProviderSnapshotRejectsEscapeDestination(t *testing.T) {
	// Pin the root to a directory that is NOT a prefix of "/etc". The
	// handler resolves NRCC_BACKUP_ROOT and rejects any destination
	// that does not resolve inside it.
	root := t.TempDir()
	t.Setenv("NRCC_BACKUP_ROOT", root)

	svc := service.NewBackupService(t.TempDir()) // different from root
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Post("/api/backups/provider/restore", handler.RestoreProviderSnapshot)

	body := `{"id":"abc123def","destination":"/etc"}`
	req := httptest.NewRequest(http.MethodPost, "/api/backups/provider/restore", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestRestoreProviderSnapshotRejectsEscapeViaDotDot covers the canonical
// attack surface: a destination like "/<root>/../etc" that passes the
// no-`..` segment check at isSafeAbsoluteDestination but escapes via
// the .. after filepath.Clean.
func TestRestoreProviderSnapshotRejectsEscapeViaDotDot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NRCC_BACKUP_ROOT", root)

	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Post("/api/backups/provider/restore", handler.RestoreProviderSnapshot)

	body := `{"id":"abc123def","destination":"` + root + `/../escape"}`
	req := httptest.NewRequest(http.MethodPost, "/api/backups/provider/restore", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}