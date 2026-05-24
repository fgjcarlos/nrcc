package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
)

// runtimeHistoryResponse matches the JSON shape returned by GetRuntimeHistory.
type runtimeHistoryResponse struct {
	Events []model.RestartEvent `json:"events"`
	Status model.RuntimeStatus  `json:"status"`
}

func TestGetRuntimeHistory_Returns200WithEmptyEvents(t *testing.T) {
	logBuf := service.NewLogBuffer(100)
	pm := service.NewProcessManager("node-red", t.TempDir(), logBuf)

	h := NewSystemHandler()
	h.SetProcessManager(pm)

	req := httptest.NewRequest(http.MethodGet, "/api/runtime/history", nil)
	w := httptest.NewRecorder()

	h.GetRuntimeHistory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp model.ApiResponse[runtimeHistoryResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response must be valid JSON: %v", err)
	}

	if !resp.Success {
		t.Error("response success must be true")
	}

	// With no crashes recorded, events should be empty (nil or empty slice is fine)
	if len(resp.Data.Events) != 0 {
		t.Errorf("expected 0 restart events, got %d", len(resp.Data.Events))
	}

	// Status must be present and have a status field
	if resp.Data.Status.Status == "" {
		t.Error("Status.Status must not be empty")
	}
}

func TestGetRuntimeHistory_StatusReflectsProcessManagerState(t *testing.T) {
	logBuf := service.NewLogBuffer(100)
	pm := service.NewProcessManager("node-red", t.TempDir(), logBuf)

	h := NewSystemHandler()
	h.SetProcessManager(pm)

	req := httptest.NewRequest(http.MethodGet, "/api/runtime/history", nil)
	w := httptest.NewRecorder()

	h.GetRuntimeHistory(w, req)

	var resp model.ApiResponse[runtimeHistoryResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response must be valid JSON: %v", err)
	}

	// A freshly created ProcessManager that hasn't been started should report "stopped"
	if resp.Data.Status.Status != "stopped" {
		t.Errorf("Status.Status = %q, want \"stopped\" for a new unstarted ProcessManager", resp.Data.Status.Status)
	}
}

func TestGetRuntimeHistory_ContentTypeIsJSON(t *testing.T) {
	logBuf := service.NewLogBuffer(100)
	pm := service.NewProcessManager("node-red", t.TempDir(), logBuf)

	h := NewSystemHandler()
	h.SetProcessManager(pm)

	req := httptest.NewRequest(http.MethodGet, "/api/runtime/history", nil)
	w := httptest.NewRecorder()

	h.GetRuntimeHistory(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want \"application/json\"", contentType)
	}
}
