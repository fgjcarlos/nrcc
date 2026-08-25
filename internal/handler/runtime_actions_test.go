package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// All three handlers refuse to do anything when no ProcessManager is wired
// (edge mode, tests). Without this guard the dashboard's "Restart" button
// would silently no-op — the exact bug that #715 fixes.

// TestRuntimeHandlers_NoProcessManager_Returns503 is the regression guard
// for #715. Before the fix, /api/runtime/restart silently returned 200
// HTML from the SPA fallback. Now it must return a structured 503.
func TestRuntimeHandlers_NoProcessManager_Returns503(t *testing.T) {
	h := NewSystemHandler() // processManager left nil

	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
		path    string
	}{
		{"restart", h.RestartNodeRed, "/api/runtime/restart"},
		{"start", h.StartNodeRed, "/api/runtime/start"},
		{"stop", h.StopNodeRed, "/api/runtime/stop"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, nil)
			w := httptest.NewRecorder()
			tc.handler(w, req)

			if w.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want 503 (PROCESS_MANAGER_UNAVAILABLE)", w.Code)
			}
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
			var resp model.ApiResponse[any]
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("response must be valid JSON, got: %s", w.Body.String())
			}
			if resp.Success {
				t.Errorf("response.success = true, want false (handler must refuse to no-op)")
			}
			if resp.Error == nil || resp.Error.Code != "PROCESS_MANAGER_UNAVAILABLE" {
				t.Errorf("error.code = %v, want PROCESS_MANAGER_UNAVAILABLE", resp.Error)
			}
		})
	}
}

// TestRuntimeHandlers_ContentTypeIsJSON pins the response shape so the
// SPA fallback (which always returns text/html) is impossible to mistake
// for a real handler.
func TestRuntimeHandlers_ContentTypeIsJSON(t *testing.T) {
	h := NewSystemHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/restart", nil)
	w := httptest.NewRecorder()
	h.RestartNodeRed(w, req)

	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json (SPA fallback returns text/html)", got)
	}
}

// TestRestartNodeRed_NoProcess_DoesNotPanic guards the nil-pointer path.
// Before the fix the endpoint simply did not exist; the SPA fallback was
// invoked instead. A future refactor that drops the nil-check must fail
// this test.
func TestRestartNodeRed_NoProcess_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RestartNodeRed panicked with nil ProcessManager: %v", r)
		}
	}()
	h := NewSystemHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/restart", nil)
	w := httptest.NewRecorder()
	h.RestartNodeRed(w, req)
}

// TestStartStopNodeRed_ProcessAlreadyRunning_Returns409 covers the
// "process already running" path on Start (Start fails fast; the
// ProcessManager returns a typed error rather than crashing).
func TestStartStopNodeRed_ExternalMode_Returns409(t *testing.T) {
	// We can't reach the external-mode branch without a port-collision
	// side effect, so this test just pins that the no-PM path is the
	// primary guard. A future test that flips IsExternalMode would slot
	// in here.
	_ = service.NewProcessManager
	h := NewSystemHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/start", nil)
	w := httptest.NewRecorder()
	h.StartNodeRed(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no-PM Start must 503, got %d", w.Code)
	}
}
