package handler

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	appmiddleware "github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
)

func loggedBackupRouter(handler *BackupHandler, logs *bytes.Buffer) http.Handler {
	router := chi.NewRouter()
	router.Use(appmiddleware.RequestID)
	router.Use(appmiddleware.Logger(slog.New(slog.NewJSONHandler(logs, nil))))
	router.Get("/api/backups/{id}/download", handler.DownloadBackup)
	return router
}

// TestDownloadBackup_MissingReturnsCleanError is the #290 regression: a missing
// backup must not produce a 200 response with zip headers and an empty/partial
// body — the error must be detected before any file headers are written.
func TestDownloadBackup_MissingReturnsCleanError(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Get("/api/backups/{id}/download", handler.DownloadBackup)

	req := httptest.NewRequest(http.MethodGet, "/api/backups/does-not-exist/download", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("missing backup must not return 200; body: %s", w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct == "application/zip" {
		t.Errorf("error response must not claim application/zip, got %q", ct)
	}
}

// TestDownloadBackup_SuccessSetsContentLength verifies a successful download
// advertises Content-Length so a client can detect a truncated stream.
func TestDownloadBackup_SuccessSetsContentLength(t *testing.T) {
	tempDir := t.TempDir()
	writeBackupFixture(t, tempDir)

	svc := service.NewBackupService(tempDir)
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Get("/api/backups/{id}/download", handler.DownloadBackup)

	req := httptest.NewRequest(http.MethodGet, "/api/backups/fixture-auto/download", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Content-Length") == "" {
		t.Error("expected Content-Length header on successful download")
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty body")
	}
}

func TestDownloadBackup_StreamsThroughRequestLogging(t *testing.T) {
	tempDir := t.TempDir()
	writeBackupFixture(t, tempDir)
	// #nosec G304 -- tempDir is test-owned and the remaining path is constant.
	wantRaw, err := os.ReadFile(filepath.Join(tempDir, "backups", "fixture-auto.zip"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	for _, test := range []struct {
		name, query, contentType, disposition string
		raw                                   bool
		header                                string
	}{
		{name: "raw", contentType: "application/zip", disposition: "backup-fixture-auto.zip", raw: true},
		{name: "password", query: "?password=secret", contentType: "application/octet-stream", disposition: "backup-fixture-auto.zip.enc"},
		// #670: passphrase now travels in the X-Backup-Password header so
		// it does not end up in proxy access logs or browser history.
		{name: "password-header", header: "X-Backup-Password: secret", contentType: "application/octet-stream", disposition: "backup-fixture-auto.zip.enc"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			router := loggedBackupRouter(NewBackupHandler(service.NewBackupService(tempDir)), &logs)
			req := httptest.NewRequest(http.MethodGet, "/api/backups/fixture-auto/download"+test.query, nil)
			if test.header != "" {
				parts := strings.SplitN(test.header, ": ", 2)
				req.Header.Set(parts[0], parts[1])
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != test.contentType {
				t.Fatalf("status=%d content-type=%q body=%q", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
			}
			if !strings.Contains(rec.Header().Get("Content-Disposition"), test.disposition) || rec.Body.Len() == 0 {
				t.Fatalf("disposition=%q body length=%d", rec.Header().Get("Content-Disposition"), rec.Body.Len())
			}
			if test.raw {
				if !bytes.Equal(rec.Body.Bytes(), wantRaw) || rec.Header().Get("Content-Length") != strconv.Itoa(len(wantRaw)) {
					t.Fatal("raw download bytes or length changed through request logger")
				}
			} else if rec.Header().Get("Content-Length") != "" || bytes.Equal(rec.Body.Bytes(), wantRaw) {
				t.Fatal("encrypted download must be chunkable ciphertext without Content-Length")
			}
			if got := logs.String(); strings.Count(got, `"msg":"http_request_completed"`) != 1 || strings.Contains(got, "secret") {
				t.Fatalf("unsafe or duplicate completion log: %s", got)
			}
		})
	}
}

func TestDownloadBackup_ErrorsThroughRequestLogging(t *testing.T) {
	for _, test := range []struct {
		name, id string
		status   int
	}{
		{name: "invalid", id: "bad..id", status: http.StatusBadRequest},
		{name: "missing", id: "does-not-exist", status: http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			router := loggedBackupRouter(NewBackupHandler(service.NewBackupService(t.TempDir())), &logs)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/backups/"+test.id+"/download", nil))
			if rec.Code != test.status {
				t.Fatalf("status=%d, want %d; body=%s", rec.Code, test.status, rec.Body.String())
			}
			if strings.Count(logs.String(), `"msg":"http_request_completed"`) != 1 || !strings.Contains(logs.String(), `"status":`+strconv.Itoa(test.status)) {
				t.Fatalf("completion log missing status %d: %s", test.status, logs.String())
			}
		})
	}
}
