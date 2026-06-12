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
// auth claims, which GetSystemInfo requires.
func authedSystemInfoRequest() *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/system/info", nil)
	ctx := context.WithValue(req.Context(), middleware.CtxKeyUser, &model.Claims{
		UserID:   "u-1",
		Username: "admin",
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
