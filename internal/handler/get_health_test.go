package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
)

// healthPayload is the shape of the data field returned by GetHealth.
type healthPayload struct {
	Status       string `json:"status"`
	Uptime       int    `json:"uptime"`
	RestartCount int    `json:"restartCount"`
}

// TestGetHealth_Returns200WithRequiredFields verifies GetHealth returns 200
// with status, integer uptime, and integer restartCount.
func TestGetHealth_Returns200WithRequiredFields(t *testing.T) {
	h := NewSystemHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	h.GetHealth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp model.ApiResponse[healthPayload]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body must be valid JSON: %v\nbody: %s", err, w.Body.String())
	}

	if resp.Data.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Data.Status, "ok")
	}
	if resp.Data.Uptime < 0 {
		t.Errorf("uptime = %d, want >= 0", resp.Data.Uptime)
	}
	if resp.Data.RestartCount < 0 {
		t.Errorf("restartCount = %d, want >= 0", resp.Data.RestartCount)
	}
}

// TestGetHealth_NilProcessManager_ReturnsZeroNoPanic verifies that with a nil
// ProcessManager GetHealth returns restartCount 0 without panicking. (uptime
// still reflects real elapsed time; only restartCount falls back to 0.)
func TestGetHealth_NilProcessManager_ReturnsZeroNoPanic(t *testing.T) {
	h := NewSystemHandler()
	// processManager is nil by default on a freshly created handler.

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	// Must not panic.
	h.GetHealth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp model.ApiResponse[healthPayload]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body must be valid JSON: %v", err)
	}

	if resp.Data.RestartCount != 0 {
		t.Errorf("restartCount with nil PM = %d, want 0", resp.Data.RestartCount)
	}
}

// seedRestartCountFile writes the restart_count.json file directly so the
// handler test can prime a known cumulative restart count without needing to
// export internal store types.
func seedRestartCountFile(t *testing.T, dir string, count int) {
	t.Helper()
	content := []byte(`{"cumulativeRestarts":` + itoa(count) + `}`)
	if err := os.WriteFile(filepath.Join(dir, "restart_count.json"), content, 0644); err != nil {
		t.Fatalf("seedRestartCountFile: %v", err)
	}
}

// itoa is a minimal int-to-string helper to avoid importing strconv here.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

// TestGetHealth_RestartCountReflectsCumulativeNotBackoff verifies that the
// restartCount in /api/health comes from CumulativeRestarts(), not the backoff
// counter (pm.restartCount).
func TestGetHealth_RestartCountReflectsCumulativeNotBackoff(t *testing.T) {
	dir := t.TempDir()
	seedRestartCountFile(t, dir, 7)

	logBuf := service.NewLogBuffer(100)
	pm := service.NewProcessManager("node-red", dir, logBuf)

	h := NewSystemHandler()
	h.SetProcessManager(pm)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	h.GetHealth(w, req)

	var resp model.ApiResponse[healthPayload]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body must be valid JSON: %v", err)
	}

	// The health endpoint must report 7 (the cumulative count), not 0 (the backoff counter).
	if resp.Data.RestartCount != 7 {
		t.Errorf("restartCount = %d, want 7 (cumulative)", resp.Data.RestartCount)
	}
}
