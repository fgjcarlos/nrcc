package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
)

func TestFlowHandler_PostRevertPreservesResponseWhenAuditUnavailable(t *testing.T) {
	dataDir := t.TempDir()
	flowsPath := filepath.Join(dataDir, "flows.json")
	original := []byte(`[{"id":"original","type":"inject"}]`)
	if err := os.WriteFile(flowsPath, original, 0600); err != nil {
		t.Fatal(err)
	}

	versionSvc := service.NewFlowVersionService(dataDir)
	if err := versionSvc.CaptureNow(); err != nil {
		t.Fatal(err)
	}
	versions, err := versionSvc.ListVersions()
	if err != nil || len(versions) != 1 {
		t.Fatalf("ListVersions = (%v, %v), want one version", versions, err)
	}
	versionID := versions[0].ID
	if err := os.WriteFile(flowsPath, []byte(`[{"id":"changed","type":"debug"}]`), 0600); err != nil {
		t.Fatal(err)
	}

	h := NewFlowHandler(nil)
	h.SetVersionService(versionSvc)
	request := httptest.NewRequest(http.MethodPost, "/api/flows/versions/"+versionID+"/revert", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", versionID)
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	response := httptest.NewRecorder()

	h.PostRevert(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	var body model.ApiResponse[map[string]string]
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Success || body.Data["message"] != "Flows reverted" || body.Data["versionId"] != versionID {
		t.Fatalf("body = %#v, want successful revert response", body)
	}
	// #nosec G304 -- flowsPath is rooted in t.TempDir().
	gotFlows, err := os.ReadFile(flowsPath)
	if err != nil || string(gotFlows) != string(original) {
		t.Fatalf("flows after revert = (%s, %v), want %s", gotFlows, err, original)
	}
}
