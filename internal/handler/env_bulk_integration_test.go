package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
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
	if err := os.Mkdir(filepath.Join(dir, "flows.json"), 0o750); err != nil {
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

func TestBulkEndpoint_CapRejectionDryRun(t *testing.T) {
	testBulkEndpointCapRejection(t, false)
}

func TestBulkEndpoint_CapRejectionCommit(t *testing.T) {
	testBulkEndpointCapRejection(t, true)
}

func testBulkEndpointCapRejection(t *testing.T, commit bool) {
	t.Helper()
	t.Setenv("NRCC_BULK_MAX_ENTRIES", "2")
	dir := t.TempDir()
	svc := service.NewEnvService(service.NewIsolatedConfigService(dir))
	if err := svc.Set("EXISTING", "keep", "string", "", false); err != nil {
		t.Fatalf("seed state: %v", err)
	}
	// #nosec G304 -- test fixture: dir comes from t.TempDir().
	beforeConfig, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	// #nosec G304 -- test fixture: dir comes from t.TempDir().
	beforeFlows, err := os.ReadFile(filepath.Join(dir, "flows.json"))
	if err != nil {
		t.Fatal(err)
	}

	w := postBulkEnv(t, NewEnvHandler(svc, dir), bulkEnvContent(3), commit)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", w.Code, w.Body.String())
	}
	var response model.ApiResponse[service.BulkEnvResult]
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	result := response.Data
	if result.Valid || len(result.Issues) != 1 || !strings.Contains(result.Issues[0].Reason, "maximum 2") {
		t.Fatalf("unexpected cap payload: %+v", result)
	}
	// #nosec G304 -- test fixture: dir comes from t.TempDir().
	afterConfig, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	// #nosec G304 -- test fixture: dir comes from t.TempDir().
	afterFlows, err := os.ReadFile(filepath.Join(dir, "flows.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeConfig, afterConfig) || !bytes.Equal(beforeFlows, afterFlows) {
		t.Fatalf("cap rejection mutated state (commit=%v)", commit)
	}
}

func TestEnvHandler_RemainingEnvironmentEndpoints(t *testing.T) {
	dir := t.TempDir()
	svc := service.NewEnvService(service.NewIsolatedConfigService(dir))
	handler := NewEnvHandler(svc, dir)
	handler.SetManagedRuntime(true)
	handler.SetProcessManager(nil)
	handler.SetAuditService(nil)
	if err := svc.Set("DELETE_ME", "value", "string", "", false); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	handler.GetEnv(w, httptest.NewRequest(http.MethodGet, "/api/env", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetEnv status=%d body=%s", w.Code, w.Body.String())
	}
	router := chi.NewRouter()
	router.Delete("/api/env/{key}", handler.DeleteEnv)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/env/DELETE_ME", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("DeleteEnv status=%d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	handler.ImportFromNodeRedEnv(w, httptest.NewRequest(http.MethodPost, "/api/env/import-from-node-red", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ImportFromNodeRedEnv status=%d body=%s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	handler.GetDotenv(w, httptest.NewRequest(http.MethodGet, "/api/env/dotenv", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetDotenv status=%d body=%s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	handler.PutDotenv(w, httptest.NewRequest(http.MethodPut, "/api/env/dotenv", strings.NewReader(`{"content":"A=1"}`)))
	if w.Code != http.StatusOK || handler.restartIfRunning() {
		t.Fatalf("PutDotenv status=%d restarted=true body=%s", w.Code, w.Body.String())
	}
	if _, err := handler.withManagedNodeRedStopped(func() error { return errors.New("change failed") }); err == nil {
		t.Fatal("withManagedNodeRedStopped error=nil")
	}

	handler.SetManagedRuntime(false)
	w = postBulkEnv(t, handler, "A=1", true)
	if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "BULK_IMPORT_FAILED") {
		t.Fatalf("unmanaged bulk status=%d body=%s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	handler.ImportFromNodeRedEnv(w, httptest.NewRequest(http.MethodPost, "/api/env/import-from-node-red", nil))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "NODE_RED_SYNC_UNAVAILABLE") {
		t.Fatalf("unavailable Node-RED import status=%d body=%s", w.Code, w.Body.String())
	}
	handler.SetManagedRuntime(true)
	t.Setenv("NRCC_BULK_MAX_ENTRIES", "-1")
	w = postBulkEnv(t, handler, "A=1", false)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "cannot be negative") {
		t.Fatalf("invalid-limit status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestEnvHandler_EnvironmentErrorResponses(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	handler := NewEnvHandler(service.NewEnvService(service.NewIsolatedConfigService(dir)), dir)
	w := httptest.NewRecorder()
	handler.GetEnv(w, httptest.NewRequest(http.MethodGet, "/api/env", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("GetEnv status=%d", w.Code)
	}

	dir = t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "flows.json"), []byte(`[{"type":"global-config","env":{}}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	handler = NewEnvHandler(service.NewEnvService(service.NewIsolatedConfigService(dir)), dir)
	w = httptest.NewRecorder()
	handler.ImportFromNodeRedEnv(w, httptest.NewRequest(http.MethodPost, "/api/env/import-from-node-red", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("ImportFromNodeRedEnv status=%d", w.Code)
	}

	dataFile := filepath.Join(t.TempDir(), "data-file")
	if err := os.WriteFile(dataFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	handler = NewEnvHandler(service.NewEnvService(service.NewIsolatedConfigService(t.TempDir())), dataFile)
	for _, call := range []func(http.ResponseWriter, *http.Request){handler.GetDotenv, handler.PutDotenv} {
		w = httptest.NewRecorder()
		call(w, httptest.NewRequest(http.MethodPut, "/api/env/dotenv", strings.NewReader(`{"content":"A=1"}`)))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("dotenv error status=%d body=%s", w.Code, w.Body.String())
		}
	}
}

func BenchmarkParseBulkEnv_1000(b *testing.B) {
	content := bulkEnvContent(1000)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result, err := service.ParseBulkEnvWithLimits(content, service.DefaultBulkLimits())
		if err != nil || !result.Valid || len(result.Lines) != 1000 {
			b.Fatalf("valid=%v lines=%d err=%v", result.Valid, len(result.Lines), err)
		}
	}
}

func BenchmarkBulkImportCommit_1000(b *testing.B) {
	dir := b.TempDir()
	configSvc := service.NewIsolatedConfigService(dir)
	handler := NewEnvHandler(service.NewEnvService(configSvc), dir)
	content := bulkEnvContent(1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		started := time.Now()
		w := postBulkEnv(b, handler, content, true)
		elapsed := time.Since(started)
		if w.Code != http.StatusOK {
			b.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		config, err := configSvc.Get()
		if err != nil {
			b.Fatal(err)
		}
		if len(config.EnvVars) != 1000 {
			b.Fatalf("persisted=%d, want 1000", len(config.EnvVars))
		}
		if elapsed >= time.Second {
			b.Fatalf("committed import took %s, want <1s", elapsed)
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
