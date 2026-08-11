package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/service"
)

func TestBulkEndpoint_1000Entries(t *testing.T) {
	status, persisted := runBulkEndpoint(t, 1000)
	if status != http.StatusOK || persisted != 1000 {
		t.Fatalf("status=%d persisted=%d, want 200/1000", status, persisted)
	}
}

func TestBulkEndpoint_400OnCapExceeded(t *testing.T) {
	t.Setenv("NRCC_BULK_MAX_ENTRIES", "2")
	status, persisted := runBulkEndpoint(t, 3)
	if status != http.StatusBadRequest || persisted != 0 {
		t.Fatalf("status=%d persisted=%d, want 400/0", status, persisted)
	}
}

func TestBulkEndpoint_FinalSyncFailure500(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "flows.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	handler := NewEnvHandler(service.NewEnvService(service.NewIsolatedConfigService(dir)), dir)
	w := postBulkEnv(t, handler, "COMMITTED=value", true)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s, want 500", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "BULK_IMPORT_FAILED") || !strings.Contains(w.Body.String(), "committed") {
		t.Fatalf("response lacks failure code or committed-state context: %s", w.Body.String())
	}
	config, err := service.NewIsolatedConfigService(dir).Get()
	if err != nil {
		t.Fatal(err)
	}
	if len(config.EnvVars) != 1 || config.EnvVars[0].Key != "COMMITTED" || config.EnvVars[0].Value != "value" {
		t.Fatalf("committed state was not preserved: %+v", config.EnvVars)
	}
}

func BenchmarkBulkImport_1000(b *testing.B) {
	content := bulkEnvContent(1000)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result, err := service.ParseBulkEnvWithLimits(content, service.DefaultBulkLimits())
		if err != nil || !result.Valid || len(result.Lines) != 1000 {
			b.Fatalf("valid=%v lines=%d err=%v", result.Valid, len(result.Lines), err)
		}
	}
}

func runBulkEndpoint(t *testing.T, count int) (int, int) {
	t.Helper()
	dir := t.TempDir()
	handler := NewEnvHandler(service.NewEnvService(service.NewIsolatedConfigService(dir)), dir)
	body := fmt.Sprintf(`{"content":%q,"commit":true}`, bulkEnvContent(count))
	req := httptest.NewRequest(http.MethodPost, "/api/env/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.BulkEnv(w, req)
	config, err := service.NewIsolatedConfigService(dir).Get()
	if err != nil {
		t.Fatal(err)
	}
	return w.Code, len(config.EnvVars)
}

func postBulkEnv(t testing.TB, handler *EnvHandler, content string, commit bool) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"content":%q,"commit":%t}`, content, commit)
	req := httptest.NewRequest(http.MethodPost, "/api/env/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.BulkEnv(w, req)
	return w
}

func bulkEnvContent(count int) string {
	var content strings.Builder
	for i := 0; i < count; i++ {
		fmt.Fprintf(&content, "KEY_%04d=value_%04d\n", i, i)
	}
	return content.String()
}
