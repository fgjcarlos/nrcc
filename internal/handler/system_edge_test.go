package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
)

// authedSystemInfoRequest builds a GET /api/system/info request carrying valid
// admin auth claims, which GetSystemInfo requires.
func authedSystemInfoRequest() *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/system/info", nil)
	ctx := context.WithValue(req.Context(), middleware.CtxKeyUser, &model.Claims{
		UserID:   "u-1",
		Username: "admin",
		Role:     model.RoleAdmin,
	})
	return req.WithContext(ctx)
}

// authedViewerSystemInfoRequest builds a GET /api/system/info request carrying
// viewer auth claims; used by MEDIUM-018 redaction tests.
func authedViewerSystemInfoRequest() *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/system/info", nil)
	ctx := context.WithValue(req.Context(), middleware.CtxKeyUser, &model.Claims{
		UserID:   "u-2",
		Username: "viewer",
		Role:     model.RoleViewer,
	})
	return req.WithContext(ctx)
}

func decodeSystemInfo(t *testing.T, w *httptest.ResponseRecorder) SystemInfo {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp model.ApiResponse[SystemInfo]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body must be valid JSON: %v\nbody: %s", err, w.Body.String())
	}
	return resp.Data
}

func TestGetSystemInfo_EdgeModeDefaultsFalse(t *testing.T) {
	// A freshly constructed handler (no SetEdgeMode) must report edge mode off,
	// proving the default is unchanged for existing deployments.
	h := NewSystemHandler()

	w := httptest.NewRecorder()
	h.GetSystemInfo(w, authedSystemInfoRequest())

	if got := decodeSystemInfo(t, w); got.EdgeMode {
		t.Errorf("edgeMode = true, want false by default")
	}
}

func TestGetSystemInfo_EdgeModeReflectsSetter(t *testing.T) {
	h := NewSystemHandler()
	h.SetEdgeMode(true)

	w := httptest.NewRecorder()
	h.GetSystemInfo(w, authedSystemInfoRequest())

	if got := decodeSystemInfo(t, w); !got.EdgeMode {
		t.Errorf("edgeMode = false, want true after SetEdgeMode(true)")
	}
}

// TestGetSystemInfo_Viewer_HostnameEmpty is the MEDIUM-018 RED case: viewers
// must NOT see the hostname, since it leaks internal network identity
// ("prod-web-01.internal.example.com" and similar). Other system metrics must
// stay populated so the operator dashboard still works for read-only users.
func TestGetSystemInfo_Viewer_HostnameEmpty(t *testing.T) {
	h := NewSystemHandler()

	w := httptest.NewRecorder()
	h.GetSystemInfo(w, authedViewerSystemInfoRequest())

	got := decodeSystemInfo(t, w)
	if got.Hostname != "" {
		t.Errorf("hostname must be redacted for viewer; got %q", got.Hostname)
	}
	if got.Platform == "" {
		t.Errorf("platform must remain populated for viewer diagnostics; got empty")
	}
	if got.Arch == "" {
		t.Errorf("arch must remain populated for viewer diagnostics; got empty")
	}
}

// TestGetSystemInfo_Admin_HostnamePopulated ensures admins still see the
// real hostname (no regression).
func TestGetSystemInfo_Admin_HostnamePopulated(t *testing.T) {
	h := NewSystemHandler()

	w := httptest.NewRecorder()
	h.GetSystemInfo(w, authedSystemInfoRequest())

	got := decodeSystemInfo(t, w)
	if got.Hostname == "" {
		t.Errorf("admin must see real hostname; got empty")
	}
}
