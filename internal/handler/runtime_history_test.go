package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
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

// TestGetRuntimeHistory_StatusRestartCountIsCumulativeNotBackoff is the
// regression guard for ADR 0001 (restart-count semantics, #244/#249).
//
// The ADR splits two counters that used to share the name restartCount:
//   - status.restartCount        -> durable CUMULATIVE auto-restart count
//   - status.consecutiveFailures -> in-session backoff/give-up counter
//
// /api/runtime/history must report the CUMULATIVE counter under restartCount.
// The original confusion was that this endpoint exposed the backoff counter
// instead. We pin the distinction by seeding a cumulative count that differs
// from the (fresh, zero) backoff counter through the persisted store seam: a
// ProcessManager loads its cumulative count from <dataDir>/restart_count.json
// at construction, while the backoff counter starts at 0. If Status() ever
// regresses to report the backoff counter under restartCount, this test fails.
func TestGetRuntimeHistory_StatusRestartCountIsCumulativeNotBackoff(t *testing.T) {
	const cumulative = 7

	dataDir := t.TempDir()
	// Seed the durable cumulative counter before the ProcessManager loads it.
	seed := []byte(`{"cumulativeRestarts":7}`)
	if err := os.WriteFile(filepath.Join(dataDir, "restart_count.json"), seed, 0600); err != nil {
		t.Fatalf("seeding restart_count.json: %v", err)
	}

	logBuf := service.NewLogBuffer(100)
	pm := service.NewProcessManager("node-red", dataDir, logBuf)

	h := NewSystemHandler()
	h.SetProcessManager(pm)

	req := httptest.NewRequest(http.MethodGet, "/api/runtime/history", nil)
	w := httptest.NewRecorder()

	h.GetRuntimeHistory(w, req)

	var resp model.ApiResponse[runtimeHistoryResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response must be valid JSON: %v", err)
	}

	// restartCount MUST be the cumulative counter (7), never the backoff
	// counter — which is 0 on a freshly constructed, never-crashed manager.
	if resp.Data.Status.RestartCount != cumulative {
		t.Errorf("Status.RestartCount = %d, want %d (the durable cumulative count, not the backoff counter)",
			resp.Data.Status.RestartCount, cumulative)
	}

	// consecutiveFailures is the separate backoff counter and must stay 0 here,
	// proving the two fields are not conflated.
	if resp.Data.Status.ConsecutiveFailures != 0 {
		t.Errorf("Status.ConsecutiveFailures = %d, want 0 (backoff counter on a fresh manager)",
			resp.Data.Status.ConsecutiveFailures)
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
