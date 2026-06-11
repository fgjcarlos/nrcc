package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
)

func TestBackupHandlerConfigRoundTrip(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)

	payload := model.BackupConfig{
		Enabled:             true,
		Schedule:            "custom",
		CustomSchedule:      "0 3 * * 1",
		RetentionManual:     8,
		RetentionAuto:       12,
		RetentionPreRestore: 3,
		IncludeConfig:       false,
		IncludeSettings:     true,
		IncludeFlowsCred:    false,
		IncludePackageJSON:  true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/backups/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.PostBackupConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/backups/config", nil)
	w = httptest.NewRecorder()
	handler.GetBackupConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response model.ApiResponse[model.BackupConfig]
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if response.Data.Schedule != "custom" || response.Data.CustomSchedule != "0 3 * * 1" {
		t.Fatalf("unexpected response config: %+v", response.Data)
	}
	if response.Data.IncludeConfig {
		t.Fatalf("expected includeConfig=false, got true")
	}
}

func TestBackupHandlerStorageAndDetailEndpoints(t *testing.T) {
	tempDir := t.TempDir()
	writeBackupFixture(t, tempDir)

	svc := service.NewBackupService(tempDir)
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Get("/api/backups/status", handler.GetBackupStatus)
	router.Get("/api/backups/observability", handler.GetBackupObservability)
	router.Get("/api/backups/storage", handler.GetBackupStorage)
	router.Get("/api/backups/{id}", handler.GetBackupDetail)

	req := httptest.NewRequest(http.MethodGet, "/api/backups/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var statusResp model.ApiResponse[model.BackupSchedulerStatus]
	if err := json.Unmarshal(w.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("Unmarshal status failed: %v", err)
	}
	if statusResp.Data.Schedule == "" {
		t.Fatalf("expected scheduler status schedule, got %+v", statusResp.Data)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/backups/observability", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var observabilityResp model.ApiResponse[model.BackupObservability]
	if err := json.Unmarshal(w.Body.Bytes(), &observabilityResp); err != nil {
		t.Fatalf("Unmarshal observability failed: %v", err)
	}
	if observabilityResp.Data.Storage.TotalBackups != 1 {
		t.Fatalf("unexpected observability storage response: %+v", observabilityResp.Data.Storage)
	}
	if len(observabilityResp.Data.RecentEvents) == 0 {
		t.Fatalf("expected observability events, got %+v", observabilityResp.Data)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/backups/storage", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var storageResp model.ApiResponse[model.BackupStorageInfo]
	if err := json.Unmarshal(w.Body.Bytes(), &storageResp); err != nil {
		t.Fatalf("Unmarshal storage failed: %v", err)
	}
	if storageResp.Data.TotalBackups != 1 || storageResp.Data.ManualCount != 0 || storageResp.Data.AutoCount != 1 {
		t.Fatalf("unexpected storage response: %+v", storageResp.Data)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/backups/fixture-auto", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var detailResp model.ApiResponse[model.BackupManifest]
	if err := json.Unmarshal(w.Body.Bytes(), &detailResp); err != nil {
		t.Fatalf("Unmarshal detail failed: %v", err)
	}
	if detailResp.Data.Type != model.BackupTypeAuto {
		t.Fatalf("expected auto type, got %q", detailResp.Data.Type)
	}
	if len(detailResp.Data.Files) != 1 {
		t.Fatalf("expected one payload file, got %d", len(detailResp.Data.Files))
	}
}

func writeBackupFixture(t *testing.T, dataDir string) {
	t.Helper()
	svc := service.NewBackupService(dataDir)
	if err := os.WriteFile(filepath.Join(dataDir, "flows.json"), []byte(`[{"id":"1"}]`), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	backup, err := svc.CreateTyped(model.BackupTypeAuto, "fixture-auto")
	if err != nil {
		t.Fatalf("CreateTyped failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "backups", backup.ID+".zip")); err != nil {
		t.Fatalf("expected created backup file: %v", err)
	}
	if err := os.Rename(filepath.Join(dataDir, "backups", backup.ID+".zip"), filepath.Join(dataDir, "backups", "fixture-auto.zip")); err != nil {
		t.Fatalf("Rename failed: %v", err)
	}
}

// === Task 1.7: Integration test for paginated GetBackups handler ===

// TestGetBackupsPaginatedIntegration: GET /api/backups?page=1&limit=10 returns paginated response
func TestGetBackupsPaginatedIntegration(t *testing.T) {
	tempDir := t.TempDir()
	svc := service.NewBackupService(tempDir)
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Get("/api/backups", handler.GetBackups)

	// Create 5 test backups by writing flows.json and calling CreateTyped
	if err := os.WriteFile(filepath.Join(tempDir, "flows.json"), []byte(`[{"id":"1"}]`), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	for i := 1; i <= 5; i++ {
		_, err := svc.CreateTyped(model.BackupTypeManual, fmt.Sprintf("test-backup-%d", i))
		if err != nil {
			t.Fatalf("CreateTyped failed: %v", err)
		}
	}

	// Test: page 1, limit 2
	req := httptest.NewRequest(http.MethodGet, "/api/backups?page=1&limit=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response model.ApiResponse[model.PaginatedBackups]
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	result := response.Data
	if result.Total != 5 {
		t.Fatalf("expected total=5, got %d", result.Total)
	}
	if result.Page != 1 {
		t.Fatalf("expected page=1, got %d", result.Page)
	}
	if result.Limit != 2 {
		t.Fatalf("expected limit=2, got %d", result.Limit)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items on page 1, got %d", len(result.Items))
	}

	// Test: page 2, limit 2
	req = httptest.NewRequest(http.MethodGet, "/api/backups?page=2&limit=2", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal page 2 failed: %v", err)
	}

	result = response.Data
	if result.Page != 2 {
		t.Fatalf("expected page=2, got %d", result.Page)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items on page 2, got %d", len(result.Items))
	}

	// Test: page 3, limit 2
	req = httptest.NewRequest(http.MethodGet, "/api/backups?page=3&limit=2", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal page 3 failed: %v", err)
	}

	result = response.Data
	if result.Page != 3 {
		t.Fatalf("expected page=3, got %d", result.Page)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item on page 3 (5 total), got %d", len(result.Items))
	}
}

// TestGetBackupsWithSortingIntegration: GET /api/backups?sort=date&order=asc returns properly sorted items
func TestGetBackupsWithSortingIntegration(t *testing.T) {
	tempDir := t.TempDir()
	svc := service.NewBackupService(tempDir)
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Get("/api/backups", handler.GetBackups)

	// Create test backups
	if err := os.WriteFile(filepath.Join(tempDir, "flows.json"), []byte(`[{"id":"1"}]`), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	for i := 1; i <= 3; i++ {
		_, err := svc.CreateTyped(model.BackupTypeManual, fmt.Sprintf("backup-%d", i))
		if err != nil {
			t.Fatalf("CreateTyped failed: %v", err)
		}
	}

	// Test: sort by date, ascending order (oldest first)
	req := httptest.NewRequest(http.MethodGet, "/api/backups?sort=date&order=asc&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response model.ApiResponse[model.PaginatedBackups]
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	result := response.Data
	if len(result.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.Items))
	}

	// Verify ascending order: first item should be oldest
	if len(result.Items) >= 2 {
		if result.Items[0].CreatedAt > result.Items[1].CreatedAt {
			t.Fatalf("expected asc order: first %s > second %s", result.Items[0].CreatedAt, result.Items[1].CreatedAt)
		}
	}
}

// TestGetBackupsDefaultsIntegration: GET /api/backups (no params) uses defaults
func TestGetBackupsDefaultsIntegration(t *testing.T) {
	tempDir := t.TempDir()
	svc := service.NewBackupService(tempDir)
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Get("/api/backups", handler.GetBackups)

	// Create test backup
	if err := os.WriteFile(filepath.Join(tempDir, "flows.json"), []byte(`[{"id":"1"}]`), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err := svc.CreateTyped(model.BackupTypeManual, "test-backup")
	if err != nil {
		t.Fatalf("CreateTyped failed: %v", err)
	}

	// Test: no query parameters (should use defaults)
	req := httptest.NewRequest(http.MethodGet, "/api/backups", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response model.ApiResponse[model.PaginatedBackups]
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	result := response.Data
	if result.Page != 1 {
		t.Fatalf("expected default page=1, got %d", result.Page)
	}
	if result.Limit != 20 {
		t.Fatalf("expected default limit=20, got %d", result.Limit)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

// === TDD: Scheduler Config Endpoint Tests ===

// TestPostSchedulerConfigValidCron: POST /api/scheduler/config with valid cron
func TestPostSchedulerConfigValidCron(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Post("/api/scheduler/config", handler.PostSchedulerConfig)

	payload := model.SchedulerConfigRequest{
		Cron: "0 2 * * *", // valid daily at 2am
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/scheduler/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response model.ApiResponse[model.SchedulerConfigResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if response.Data.Cron != "0 2 * * *" {
		t.Fatalf("expected cron '0 2 * * *', got '%s'", response.Data.Cron)
	}
	if !response.Data.Valid {
		t.Fatalf("expected valid=true for cron '0 2 * * *'")
	}
}

// TestPostSchedulerConfigInvalidCron: POST /api/scheduler/config with invalid cron returns 400
func TestPostSchedulerConfigInvalidCron(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Post("/api/scheduler/config", handler.PostSchedulerConfig)

	payload := model.SchedulerConfigRequest{
		Cron: "99 99 99 99 99", // invalid cron
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/scheduler/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var errResponse model.ApiErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResponse); err != nil {
		t.Fatalf("Unmarshal error response failed: %v", err)
	}

	if errResponse.Error == nil || errResponse.Error.Code != "INVALID_CRON" {
		t.Fatalf("expected error code 'INVALID_CRON', got '%+v'", errResponse.Error)
	}
}

// === TDD: Scheduler History Endpoint Tests ===

// TestGetSchedulerHistoryPaginatedHappyPath: GET /api/scheduler/history with pagination
func TestGetSchedulerHistoryPaginatedHappyPath(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Get("/api/scheduler/history", handler.GetSchedulerHistory)

	// Create some test events
	for i := 0; i < 25; i++ {
		svc.RecordSchedulerEvent(model.SchedulerHistoryEntry{
			Timestamp: "2026-05-11T10:00:00Z",
			Status:    "success",
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/scheduler/history?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response model.ApiResponse[model.PaginatedSchedulerHistory]
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	result := response.Data
	if result.Total < 25 {
		t.Fatalf("expected total >= 25, got %d", result.Total)
	}
	if result.Page != 1 {
		t.Fatalf("expected page=1, got %d", result.Page)
	}
	if result.Limit != 10 {
		t.Fatalf("expected limit=10, got %d", result.Limit)
	}
	if len(result.Entries) != 10 {
		t.Fatalf("expected 10 entries on page 1, got %d", len(result.Entries))
	}
}

// TestGetSchedulerHistoryEmpty: GET /api/scheduler/history returns empty list when no history
func TestGetSchedulerHistoryEmpty(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Get("/api/scheduler/history", handler.GetSchedulerHistory)

	req := httptest.NewRequest(http.MethodGet, "/api/scheduler/history", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response model.ApiResponse[model.PaginatedSchedulerHistory]
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	result := response.Data
	if result.Total != 0 {
		t.Fatalf("expected total=0, got %d", result.Total)
	}
	if len(result.Entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(result.Entries))
	}
}

// === TDD: Storage Retention Endpoint Tests ===

// TestPatchStorageRetentionValidDays: PATCH /api/storage/retention with valid days
func TestPatchStorageRetentionValidDays(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Patch("/api/storage/retention", handler.PatchStorageRetention)

	payload := model.RetentionConfigRequest{
		RetentionDays: 30,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/storage/retention", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response model.ApiResponse[model.RetentionConfigResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if response.Data.RetentionDays != 30 {
		t.Fatalf("expected retentionDays=30, got %d", response.Data.RetentionDays)
	}
}

// TestPatchStorageRetentionInvalidDays: PATCH /api/storage/retention with invalid days (< 1) returns 400
func TestPatchStorageRetentionInvalidDays(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Patch("/api/storage/retention", handler.PatchStorageRetention)

	payload := model.RetentionConfigRequest{
		RetentionDays: 0,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/storage/retention", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var errResponse model.ApiErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResponse); err != nil {
		t.Fatalf("Unmarshal error response failed: %v", err)
	}

	if errResponse.Error == nil || errResponse.Error.Code != "INVALID_REQUEST" {
		t.Fatalf("expected error code 'INVALID_REQUEST', got '%+v'", errResponse.Error)
	}
}

// TestPatchStorageRetentionMaxDays: PATCH /api/storage/retention accepts max 3650 days
func TestPatchStorageRetentionMaxDays(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Patch("/api/storage/retention", handler.PatchStorageRetention)

	payload := model.RetentionConfigRequest{
		RetentionDays: 3650,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/storage/retention", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response model.ApiResponse[model.RetentionConfigResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if response.Data.RetentionDays != 3650 {
		t.Fatalf("expected retentionDays=3650, got %d", response.Data.RetentionDays)
	}
}

// === TRIANGULATION: Additional scheduler config tests ===

// TestPostSchedulerConfigMultiplePresets: POST /api/scheduler/config with different valid presets
func TestPostSchedulerConfigMultiplePresets(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Post("/api/scheduler/config", handler.PostSchedulerConfig)

	testCases := []struct {
		name string
		cron string
	}{
		{"Hourly", "0 * * * *"},
		{"Every 6 hours", "0 */6 * * *"},
		{"Daily at 2am", "0 2 * * *"},
		{"Weekly", "0 2 * * 0"},
		{"Every 15 minutes", "*/15 * * * *"},
	}

	for _, tc := range testCases {
		payload := model.SchedulerConfigRequest{Cron: tc.cron}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/scheduler/config", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("test %s: expected 200, got %d: %s", tc.name, w.Code, w.Body.String())
		}

		var response model.ApiResponse[model.SchedulerConfigResponse]
		json.Unmarshal(w.Body.Bytes(), &response)
		if response.Data.Cron != tc.cron {
			t.Fatalf("test %s: expected cron %s, got %s", tc.name, tc.cron, response.Data.Cron)
		}
	}
}

// === TRIANGULATION: Pagination edge cases ===

// TestGetSchedulerHistoryPaginationSecondPage: Test pagination moves through pages
func TestGetSchedulerHistoryPaginationSecondPage(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Get("/api/scheduler/history", handler.GetSchedulerHistory)

	// Create 25 events
	for i := 0; i < 25; i++ {
		svc.RecordSchedulerEvent(model.SchedulerHistoryEntry{
			Timestamp: "2026-05-11T10:00:00Z",
			Status:    "success",
		})
	}

	// Page 2, limit 10
	req := httptest.NewRequest(http.MethodGet, "/api/scheduler/history?page=2&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response model.ApiResponse[model.PaginatedSchedulerHistory]
	json.Unmarshal(w.Body.Bytes(), &response)
	result := response.Data

	if result.Page != 2 {
		t.Fatalf("expected page=2, got %d", result.Page)
	}
	if len(result.Entries) != 10 {
		t.Fatalf("expected 10 entries on page 2, got %d", len(result.Entries))
	}
	if result.Total < 25 {
		t.Fatalf("expected total >= 25, got %d", result.Total)
	}
}

// === TRIANGULATION: Retention boundary tests ===

// TestPatchStorageRetentionMinDays: PATCH /api/storage/retention accepts min 1 day
func TestPatchStorageRetentionMinDays(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Patch("/api/storage/retention", handler.PatchStorageRetention)

	payload := model.RetentionConfigRequest{
		RetentionDays: 1,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/storage/retention", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response model.ApiResponse[model.RetentionConfigResponse]
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Data.RetentionDays != 1 {
		t.Fatalf("expected retentionDays=1, got %d", response.Data.RetentionDays)
	}
}

// TestPatchStorageRetentionAboveMax: PATCH /api/storage/retention rejects > 3650
func TestPatchStorageRetentionAboveMax(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Patch("/api/storage/retention", handler.PatchStorageRetention)

	payload := model.RetentionConfigRequest{
		RetentionDays: 3651,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/storage/retention", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for days > 3650, got %d: %s", w.Code, w.Body.String())
	}
}

// === Integration test: Full endpoint flow ===

// TestSchedulerConfigAndHistoryIntegration: Set config, record event, fetch history
func TestSchedulerConfigAndHistoryIntegration(t *testing.T) {
	svc := service.NewBackupService(t.TempDir())
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Post("/api/scheduler/config", handler.PostSchedulerConfig)
	router.Get("/api/scheduler/history", handler.GetSchedulerHistory)

	// 1. Set scheduler config
	configPayload := model.SchedulerConfigRequest{
		Cron: "0 2 * * *",
	}
	body, _ := json.Marshal(configPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/scheduler/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("set config failed: %d: %s", w.Code, w.Body.String())
	}

	// 2. Record scheduler event
	svc.RecordSchedulerEvent(model.SchedulerHistoryEntry{
		Timestamp: "2026-05-11T02:00:00Z",
		Status:    "success",
	})

	// 3. Fetch history
	req = httptest.NewRequest(http.MethodGet, "/api/scheduler/history?limit=5", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get history failed: %d: %s", w.Code, w.Body.String())
	}

	var response model.ApiResponse[model.PaginatedSchedulerHistory]
	json.Unmarshal(w.Body.Bytes(), &response)
	result := response.Data

	if result.Total != 1 {
		t.Fatalf("expected 1 event after recording, got %d", result.Total)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry in response, got %d", len(result.Entries))
	}
	if result.Entries[0].Status != "success" {
		t.Fatalf("expected status 'success', got '%s'", result.Entries[0].Status)
	}
}

// TestRetentionUpdateAndPersistence: Set retention and verify persistence
func TestRetentionUpdateAndPersistence(t *testing.T) {
	dataDir := t.TempDir()
	svc := service.NewBackupService(dataDir)
	handler := NewBackupHandler(svc)
	router := chi.NewRouter()
	router.Patch("/api/storage/retention", handler.PatchStorageRetention)
	router.Get("/api/backups/config", handler.GetBackupConfig)

	// 1. Update retention
	payload := model.RetentionConfigRequest{
		RetentionDays: 60,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/storage/retention", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("patch retention failed: %d: %s", w.Code, w.Body.String())
	}

	// 2. Verify by getting config
	req = httptest.NewRequest(http.MethodGet, "/api/backups/config", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get config failed: %d: %s", w.Code, w.Body.String())
	}

	var response model.ApiResponse[model.BackupConfig]
	json.Unmarshal(w.Body.Bytes(), &response)
	cfg := response.Data

	if cfg.RetentionManual != 60 {
		t.Fatalf("expected retentionManual=60, got %d", cfg.RetentionManual)
	}
	if cfg.RetentionAuto != 60 {
		t.Fatalf("expected retentionAuto=60, got %d", cfg.RetentionAuto)
	}
}
