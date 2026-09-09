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

func TestDashboardDiscoveryHandler(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"node-red-dashboard":"3.6.6"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	handler := NewDashboardHandler(service.NewDashboardService(dir))
	recorder := httptest.NewRecorder()
	handler.GetDiscovery(recorder, httptest.NewRequest(http.MethodGet, "/api/dashboards/discovery", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var response model.ApiResponse[model.DashboardDiscovery]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	discovery := response.Data
	if discovery.Legacy == nil || discovery.Legacy.Path != "/ui" {
		t.Fatalf("discovery = %#v", discovery)
	}
}
