package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/store"
	"github.com/go-chi/chi/v5"
)

// buildAuthRouter returns a chi router wired with Auth middleware protecting
// the system and runtime history endpoints (mirrors the production setup).
func buildAuthRouter(t *testing.T) *chi.Mux {
	t.Helper()
	tempDir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](tempDir + "/users.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](tempDir + "/sessions.json")
	authSvc := service.NewAuthService("test-secret", userStore, sessionStore)

	h := NewSystemHandler()

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(authSvc))
		r.Get("/api/system/history", h.GetSystemHistory)
		r.Get("/api/runtime/history", h.GetRuntimeHistory)
	})
	return r
}

// TestGetSystemHistory_NoAuth_Returns401 asserts that an unauthenticated
// request to /api/system/history is rejected with HTTP 401.
func TestGetSystemHistory_NoAuth_Returns401(t *testing.T) {
	r := buildAuthRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/system/history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/system/history without auth: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// TestGetRuntimeHistory_NoAuth_Returns401 asserts that an unauthenticated
// request to /api/runtime/history is rejected with HTTP 401.
func TestGetRuntimeHistory_NoAuth_Returns401(t *testing.T) {
	r := buildAuthRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/runtime/history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/runtime/history without auth: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// TestGetRuntimeHistory_NilProcessManager_ReturnsEmptyEvents verifies that
// GetRuntimeHistory with a nil ProcessManager returns events:[] and a zero
// status without panicking (WARN-2 coverage gap).
func TestGetRuntimeHistory_NilProcessManager_ReturnsEmptyEvents(t *testing.T) {
	h := NewSystemHandler()
	// processManager is nil — no SetProcessManager call.

	req := httptest.NewRequest(http.MethodGet, "/api/runtime/history", nil)
	w := httptest.NewRecorder()

	// Must not panic.
	h.GetRuntimeHistory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Data struct {
			Events []json.RawMessage `json:"events"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body must be valid JSON: %v\nbody: %s", err, w.Body.String())
	}

	if len(resp.Data.Events) != 0 {
		t.Errorf("events with nil PM = %d, want 0", len(resp.Data.Events))
	}
}
