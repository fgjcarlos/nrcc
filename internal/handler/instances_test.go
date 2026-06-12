package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

func TestGetInstances_ReturnsDefaultInstanceOnly(t *testing.T) {
	const dataDir = "/srv/nrcc/data"
	h := NewInstanceHandler(service.NewInstanceStore(dataDir))

	req := httptest.NewRequest(http.MethodGet, "/api/instances", nil)
	w := httptest.NewRecorder()
	h.GetInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp model.ApiResponse[[]model.Instance]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body must be valid JSON: %v\nbody: %s", err, w.Body.String())
	}

	if !resp.Success {
		t.Errorf("success = false, want true")
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data has %d instances, want 1 (default only)", len(resp.Data))
	}

	def := resp.Data[0]
	if def.ID != model.DefaultInstanceID {
		t.Errorf("instance ID = %q, want %q", def.ID, model.DefaultInstanceID)
	}
	if def.Kind != model.InstanceKindLocal {
		t.Errorf("instance Kind = %q, want %q", def.Kind, model.InstanceKindLocal)
	}
	if def.DataDir != dataDir {
		t.Errorf("instance DataDir = %q, want %q", def.DataDir, dataDir)
	}
}
