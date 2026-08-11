package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/store"
)

func TestBodyLimit_ServerAppliesGloballyAfterCORS(t *testing.T) {
	dir := t.TempDir()
	authSvc := service.NewAuthService(
		"test-secret",
		store.NewJSONStore[model.CCUsers](dir+"/users.json"),
		store.NewJSONStore[model.RefreshSessions](dir+"/sessions.json"),
	)
	const origin = "https://app.example.com"
	srv := NewServerWithConfig(authSvc, dir, middleware.CORSConfig{AllowedOrigins: []string{origin}})
	t.Cleanup(srv.Shutdown)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/setup", strings.NewReader(`{"username":"`+strings.Repeat("x", 2<<20)+`","password":"password123"}`))
	req.Header.Set("Origin", origin)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, origin)
	}
	if got := rec.Body.String(); got != `{"error":"request body too large"}` {
		t.Errorf("body = %q, want exact overflow contract", got)
	}
}
