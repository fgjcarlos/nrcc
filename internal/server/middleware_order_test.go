package server

import (
	"bytes"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	setupstate "github.com/fgjcarlos/nrcc/internal/setup"
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

func TestNewServerWithConfig_LogsSetupTokenOnlyOnFreshBoot(t *testing.T) {
	originalOutput := log.Writer()
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(originalOutput) })

	dir := t.TempDir()
	authSvc := service.NewAuthService("test-secret",
		store.NewJSONStore[model.CCUsers](dir+"/users.json"),
		store.NewJSONStore[model.RefreshSessions](dir+"/sessions.json"))
	srv := NewServerWithConfig(authSvc, Config{DataDir: dir, CORS: middleware.CORSConfig{}, HTTPLogger: discardHTTPLogger()})
	srv.Shutdown()
	token, err := setupstate.ReadTokenFile(filepath.Join(dir, setupstate.SetupTokenFileName))
	if err != nil {
		t.Fatal(err)
	}
	if got := logs.String(); strings.Count(got, "auth setup token: ") != 1 || !strings.Contains(got, "auth setup token: "+token.Raw) {
		t.Fatalf("fresh startup log=%q", got)
	}

	logs.Reset()
	now := model.NowISO8601()
	if err := authSvc.CreateUser(&model.CCUser{ID: "existing", Username: "admin", PasswordHash: "hash", Role: model.RoleAdmin, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	srv = NewServerWithConfig(authSvc, Config{DataDir: dir, CORS: middleware.CORSConfig{}, HTTPLogger: discardHTTPLogger()})
	srv.Shutdown()
	if strings.Contains(logs.String(), "auth setup token: ") {
		t.Fatalf("configured startup logged a setup token: %s", logs.String())
	}

	logs.Reset()
	brokenDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(brokenDir, "users.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	brokenAuthSvc := service.NewAuthService("test-secret",
		store.NewJSONStore[model.CCUsers](filepath.Join(brokenDir, "users.json")),
		store.NewJSONStore[model.RefreshSessions](filepath.Join(brokenDir, "sessions.json")))
	srv = NewServerWithConfig(brokenAuthSvc, Config{DataDir: brokenDir, CORS: middleware.CORSConfig{}, HTTPLogger: discardHTTPLogger()})
	srv.Shutdown()
	if strings.Contains(logs.String(), "auth setup token: ") {
		t.Fatalf("unreadable user state logged a setup token: %s", logs.String())
	}
	if _, err := os.Stat(filepath.Join(brokenDir, setupstate.SetupTokenFileName)); !os.IsNotExist(err) {
		t.Fatalf("unreadable user state generated a setup token: %v", err)
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
