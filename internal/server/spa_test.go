package server

import (
	"embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestSPAHandlerUnknownAPIRouteReturnsJSON404(t *testing.T) {
	h := SPAHandler(embed.FS{})
	req := httptest.NewRequest(http.MethodPost, "/api/does-not-exist", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want=404 body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type=%q want application/json", got)
	}
	var response model.ApiErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error == nil || response.Error.Code != "API_ROUTE_NOT_FOUND" {
		t.Fatalf("response=%+v", response)
	}
}
