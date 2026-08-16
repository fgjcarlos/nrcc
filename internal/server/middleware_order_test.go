package server

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/store"
)

func discardHTTPLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewServerWithConfig_MiddlewareComposition(t *testing.T) {
	dir := t.TempDir()
	authSvc := service.NewAuthService(
		"test-secret",
		store.NewJSONStore[model.CCUsers](dir+"/users.json"),
		store.NewJSONStore[model.RefreshSessions](dir+"/sessions.json"),
	)
	var logs bytes.Buffer
	srv := NewServerWithConfig(authSvc, Config{
		DataDir:    dir,
		CORS:       middleware.CORSConfig{},
		HTTPLogger: slog.New(slog.NewTextHandler(&logs, nil)),
	})
	t.Cleanup(srv.Shutdown)

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz?password=secret", nil))
	requestID := rec.Header().Get("X-Request-Id")
	if rec.Code != http.StatusOK || requestID == "" {
		t.Fatalf("health response status=%d request_id=%q", rec.Code, requestID)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("security middleware header = %q, want nosniff", got)
	}
	if got := logs.String(); !strings.Contains(got, "http_request_completed") || !strings.Contains(got, requestID) || strings.Contains(got, "secret") {
		t.Fatalf("injected logger record is unsafe or incomplete: %s", got)
	}

	srv.router.Get("/panic-test", func(http.ResponseWriter, *http.Request) { panic("private detail") })
	panicRec := httptest.NewRecorder()
	srv.ServeHTTP(panicRec, httptest.NewRequest(http.MethodGet, "/panic-test", nil))
	if panicRec.Code != http.StatusInternalServerError || strings.Contains(panicRec.Body.String(), "private detail") {
		t.Fatalf("panic response status=%d body=%q", panicRec.Code, panicRec.Body.String())
	}
	if !strings.Contains(logs.String(), "status=500") {
		t.Fatalf("panic completion was not logged as 500: %s", logs.String())
	}
}

func TestNewServerWithConfig_NilHTTPLoggerPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("nil HTTP logger must panic during construction")
		}
	}()
	NewServerWithConfig(nil, Config{DataDir: t.TempDir()})
}
