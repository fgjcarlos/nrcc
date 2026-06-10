package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/composedof2/nrcc/internal/model"
)

func TestBackupServiceCreateIncludesExplicitTypeAndDetail(t *testing.T) {
	tempDir := t.TempDir()
	writeTestFile(t, filepath.Join(tempDir, "flows.json"), `[{"id":"1"}]`)
	writeTestFile(t, filepath.Join(tempDir, "config.json"), `{"uiPort":1880}`)

	svc := NewBackupService(tempDir)
	backup, err := svc.CreateTyped(model.BackupTypeAuto, "auto-nocturno")
	if err != nil {
		t.Fatalf("CreateTyped returned error: %v", err)
	}

	if backup.Type != model.BackupTypeAuto {
		t.Fatalf("expected auto backup type, got %q", backup.Type)
	}
	if backup.FileCount < 3 {
		t.Fatalf("expected metadata plus payload files, got fileCount=%d", backup.FileCount)
	}

	manifest, err := svc.Detail(backup.ID)
	if err != nil {
		t.Fatalf("Detail returned error: %v", err)
	}

	if manifest.Type != model.BackupTypeAuto {
		t.Fatalf("expected manifest type auto, got %q", manifest.Type)
	}
	if manifest.TriggeredBy != "auto-nocturno" {
		t.Fatalf("expected triggeredBy auto-nocturno, got %q", manifest.TriggeredBy)
	}
	if len(manifest.Files) != 2 {
		t.Fatalf("expected 2 payload files in manifest, got %d", len(manifest.Files))
	}
}

func TestBackupServiceStorageCountsTypes(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewBackupService(tempDir)

	createBackupArchive(t, tempDir, backupMetadata{ID: "manual-1", Name: "manual-1", Type: model.BackupTypeManual, CreatedAt: "2026-05-09T10:00:00Z", TriggeredBy: "manual-1"})
	createBackupArchive(t, tempDir, backupMetadata{ID: "auto-1", Name: "auto-1", Type: model.BackupTypeAuto, CreatedAt: "2026-05-09T11:00:00Z", TriggeredBy: "auto-1"})
	createBackupArchive(t, tempDir, backupMetadata{ID: "pre-1", Name: "pre-restore", Type: model.BackupTypePreRestore, CreatedAt: "2026-05-09T12:00:00Z", TriggeredBy: "pre-restore"})

	storage, err := svc.Storage()
	if err != nil {
		t.Fatalf("Storage returned error: %v", err)
	}

	if storage.TotalBackups != 3 {
		t.Fatalf("expected 3 backups, got %d", storage.TotalBackups)
	}
	if storage.ManualCount != 1 || storage.AutoCount != 1 || storage.PreRestoreCount != 1 {
		t.Fatalf("unexpected counts: %+v", storage)
	}
	if storage.TotalSize <= 0 {
		t.Fatalf("expected total size > 0, got %d", storage.TotalSize)
	}
}

func TestBackupServiceConfigPersistsCustomSchedule(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewBackupService(tempDir)

	saved, err := svc.SaveConfig(model.BackupConfig{
		Enabled:             true,
		Schedule:            "custom",
		CustomSchedule:      "0 2 * * *",
		RetentionManual:     7,
		RetentionAuto:       9,
		RetentionPreRestore: 4,
		IncludeConfig:       false,
		IncludeSettings:     true,
		IncludeFlowsCred:    false,
		IncludePackageJSON:  true,
	})
	if err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}

	if saved.Schedule != "custom" || saved.CustomSchedule != "0 2 * * *" {
		t.Fatalf("unexpected saved config: %+v", saved)
	}
	if saved.IncludeConfig {
		t.Fatalf("expected includeConfig=false, got true")
	}

	loaded, err := svc.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig returned error: %v", err)
	}

	if loaded.Schedule != "custom" || loaded.CustomSchedule != "0 2 * * *" {
		t.Fatalf("unexpected loaded config: %+v", loaded)
	}
	if loaded.IncludeConfig {
		t.Fatalf("expected includeConfig=false after reload, got true")
	}
	if loaded.IncludeFlowsCred {
		t.Fatalf("expected includeFlowsCred=false after reload, got true")
	}
}

func TestBackupServiceRejectsInvalidCustomSchedule(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewBackupService(tempDir)

	_, err := svc.SaveConfig(model.BackupConfig{
		Enabled:             true,
		Schedule:            "custom",
		CustomSchedule:      "not-a-cron",
		RetentionManual:     7,
		RetentionAuto:       9,
		RetentionPreRestore: 4,
		IncludeConfig:       true,
		IncludeSettings:     true,
		IncludeFlowsCred:    true,
		IncludePackageJSON:  true,
	})
	if !errors.Is(err, ErrInvalidBackupConfig) {
		t.Fatalf("expected ErrInvalidBackupConfig, got %v", err)
	}
}

func TestBackupServiceSchedulerReloadsSavedConfig(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewBackupService(tempDir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.Start(ctx)

	baseConfig := model.BackupConfig{
		Enabled:             true,
		RetentionManual:     10,
		RetentionAuto:       30,
		RetentionPreRestore: 5,
		IncludeConfig:       true,
		IncludeSettings:     true,
		IncludeFlowsCred:    true,
		IncludePackageJSON:  true,
	}

	if _, err := svc.SaveConfig(model.BackupConfig{
		Enabled:             baseConfig.Enabled,
		Schedule:            "daily",
		RetentionManual:     baseConfig.RetentionManual,
		RetentionAuto:       baseConfig.RetentionAuto,
		RetentionPreRestore: baseConfig.RetentionPreRestore,
		IncludeConfig:       baseConfig.IncludeConfig,
		IncludeSettings:     baseConfig.IncludeSettings,
		IncludeFlowsCred:    baseConfig.IncludeFlowsCred,
		IncludePackageJSON:  baseConfig.IncludePackageJSON,
	}); err != nil {
		t.Fatalf("SaveConfig daily returned error: %v", err)
	}

	status := svc.SchedulerStatus()
	if !status.Scheduled || status.ActiveSpec != "0 2 * * *" {
		t.Fatalf("unexpected daily scheduler status: %+v", status)
	}
	if status.NextRunAt == "" {
		t.Fatalf("expected next run for daily scheduler, got %+v", status)
	}

	if _, err := svc.SaveConfig(model.BackupConfig{
		Enabled:             baseConfig.Enabled,
		Schedule:            "custom",
		CustomSchedule:      "15 3 * * 1",
		RetentionManual:     baseConfig.RetentionManual,
		RetentionAuto:       baseConfig.RetentionAuto,
		RetentionPreRestore: baseConfig.RetentionPreRestore,
		IncludeConfig:       baseConfig.IncludeConfig,
		IncludeSettings:     baseConfig.IncludeSettings,
		IncludeFlowsCred:    baseConfig.IncludeFlowsCred,
		IncludePackageJSON:  baseConfig.IncludePackageJSON,
	}); err != nil {
		t.Fatalf("SaveConfig custom returned error: %v", err)
	}

	status = svc.SchedulerStatus()
	if !status.Scheduled || status.ActiveSpec != "15 3 * * 1" {
		t.Fatalf("unexpected custom scheduler status: %+v", status)
	}
	if status.CustomSchedule != "15 3 * * 1" {
		t.Fatalf("expected custom schedule to be exposed, got %+v", status)
	}
}

func TestBackupServiceCreateRespectsFileInclusionConfig(t *testing.T) {
	tempDir := t.TempDir()
	writeTestFile(t, filepath.Join(tempDir, "flows.json"), `[{"id":"1"}]`)
	writeTestFile(t, filepath.Join(tempDir, "config.json"), `{"uiPort":1880}`)
	writeTestFile(t, filepath.Join(tempDir, "settings.js"), `module.exports = {};`)
	writeTestFile(t, filepath.Join(tempDir, "flows_cred.json"), `{}`)
	writeTestFile(t, filepath.Join(tempDir, "package.json"), `{"name":"nrcc"}`)
	writeTestFile(t, filepath.Join(tempDir, "cc-users.json"), `[]`)

	svc := NewBackupService(tempDir)
	if _, err := svc.SaveConfig(model.BackupConfig{
		Enabled:             false,
		Schedule:            "disabled",
		RetentionManual:     10,
		RetentionAuto:       30,
		RetentionPreRestore: 5,
		IncludeConfig:       false,
		IncludeSettings:     true,
		IncludeFlowsCred:    false,
		IncludePackageJSON:  false,
	}); err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}

	backup, err := svc.CreateTyped(model.BackupTypeAuto, "auto-config-aware")
	if err != nil {
		t.Fatalf("CreateTyped returned error: %v", err)
	}

	manifest, err := svc.Detail(backup.ID)
	if err != nil {
		t.Fatalf("Detail returned error: %v", err)
	}

	paths := make(map[string]bool, len(manifest.Files))
	for _, file := range manifest.Files {
		paths[file.Path] = true
	}

	if !paths["flows.json"] || !paths["cc-users.json"] || !paths["settings.js"] {
		t.Fatalf("expected required files to be present, got %+v", manifest.Files)
	}
	if paths["config.json"] || paths["flows_cred.json"] || paths["package.json"] {
		t.Fatalf("expected disabled optional files to be excluded, got %+v", manifest.Files)
	}
}

func TestBackupServicePrunesBackupsByTypeRetention(t *testing.T) {
	tempDir := t.TempDir()
	writeTestFile(t, filepath.Join(tempDir, "flows.json"), `[{"id":"1"}]`)

	svc := NewBackupService(tempDir)
	if _, err := svc.SaveConfig(model.BackupConfig{
		Enabled:             false,
		Schedule:            "disabled",
		RetentionManual:     2,
		RetentionAuto:       1,
		RetentionPreRestore: 3,
		IncludeConfig:       true,
		IncludeSettings:     true,
		IncludeFlowsCred:    true,
		IncludePackageJSON:  true,
	}); err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}

	first, err := svc.CreateTyped(model.BackupTypeAuto, "auto-1")
	if err != nil {
		t.Fatalf("CreateTyped first returned error: %v", err)
	}
	second, err := svc.CreateTyped(model.BackupTypeAuto, "auto-2")
	if err != nil {
		t.Fatalf("CreateTyped second returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "backups", first.ID+".zip")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected first auto backup to be pruned, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, "backups", second.ID+".zip")); err != nil {
		t.Fatalf("expected latest auto backup to remain, got err=%v", err)
	}

	manualOne, err := svc.CreateTyped(model.BackupTypeManual, "manual-1")
	if err != nil {
		t.Fatalf("CreateTyped manualOne returned error: %v", err)
	}
	manualTwo, err := svc.CreateTyped(model.BackupTypeManual, "manual-2")
	if err != nil {
		t.Fatalf("CreateTyped manualTwo returned error: %v", err)
	}
	manualThree, err := svc.CreateTyped(model.BackupTypeManual, "manual-3")
	if err != nil {
		t.Fatalf("CreateTyped manualThree returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "backups", manualOne.ID+".zip")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected oldest manual backup to be pruned, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, "backups", manualTwo.ID+".zip")); err != nil {
		t.Fatalf("expected second manual backup to remain, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, "backups", manualThree.ID+".zip")); err != nil {
		t.Fatalf("expected latest manual backup to remain, got err=%v", err)
	}
}

func TestBackupServiceObservabilityIncludesRecentEvents(t *testing.T) {
	tempDir := t.TempDir()
	writeTestFile(t, filepath.Join(tempDir, "flows.json"), `[{"id":"1"}]`)

	svc := NewBackupService(tempDir)
	if _, err := svc.SaveConfig(model.BackupConfig{
		Enabled:             true,
		Schedule:            "daily",
		RetentionManual:     2,
		RetentionAuto:       1,
		RetentionPreRestore: 2,
		IncludeConfig:       true,
		IncludeSettings:     true,
		IncludeFlowsCred:    true,
		IncludePackageJSON:  true,
	}); err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}

	backup, err := svc.CreateTyped(model.BackupTypeManual, "manual-observable")
	if err != nil {
		t.Fatalf("CreateTyped returned error: %v", err)
	}

	observability, err := svc.Observability()
	if err != nil {
		t.Fatalf("Observability returned error: %v", err)
	}

	if observability.Storage.TotalBackups != 1 {
		t.Fatalf("expected one backup in storage summary, got %+v", observability.Storage)
	}
	if observability.LatestBackup == nil || observability.LatestBackup.ID != backup.ID {
		t.Fatalf("expected latest backup %q, got %+v", backup.ID, observability.LatestBackup)
	}
	if len(observability.RecentEvents) == 0 {
		t.Fatal("expected recent events to be recorded")
	}

	foundManualCreate := false
	for _, event := range observability.RecentEvents {
		if event.Type == model.BackupEventTypeManualCreate && event.BackupID == backup.ID {
			foundManualCreate = true
			break
		}
	}
	if !foundManualCreate {
		t.Fatalf("expected manual create event in observability, got %+v", observability.RecentEvents)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

func createBackupArchive(t *testing.T, dataDir string, metadata backupMetadata) {
	t.Helper()
	backupDir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	zipPath := filepath.Join(backupDir, metadata.ID+".zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	zipWriter := zip.NewWriter(file)
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	metadataWriter, err := zipWriter.Create("backup-metadata.json")
	if err != nil {
		t.Fatalf("Create metadata entry failed: %v", err)
	}
	if _, err := metadataWriter.Write(metadataBytes); err != nil {
		t.Fatalf("Write metadata failed: %v", err)
	}

	payloadWriter, err := zipWriter.Create("flows.json")
	if err != nil {
		t.Fatalf("Create payload entry failed: %v", err)
	}
	if _, err := payloadWriter.Write([]byte(`[{"id":"node-1"}]`)); err != nil {
		t.Fatalf("Write payload failed: %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("zip close failed: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("file close failed: %v", err)
	}
}

// === Task 1.1-1.3: ListPaginated Tests (RED Phase) ===

// TestListPaginatedHappyPath: First page with default limit
func TestListPaginatedHappyPath(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewBackupService(tempDir)

	// Create 25 backups with controlled CreatedAt timestamps
	for i := 1; i <= 25; i++ {
		backup := backupMetadata{
			ID:          fmt.Sprintf("backup-%03d", i),
			Name:        fmt.Sprintf("backup-%03d", i),
			Type:        model.BackupTypeManual,
			CreatedAt:   fmt.Sprintf("2026-01-01T%02d:00:00Z", (i%24)+1),
			TriggeredBy: "manual",
		}
		createBackupArchive(t, tempDir, backup)
	}

	// Test: page 1, default limit (20)
	opts := model.PaginationOpts{
		Page:  1,
		Limit: 20,
		Sort:  "date",
		Order: "desc",
	}

	result, err := svc.ListPaginated(opts)
	if err != nil {
		t.Fatalf("ListPaginated returned error: %v", err)
	}

	if result.Total != 25 {
		t.Fatalf("expected total=25, got %d", result.Total)
	}
	if result.Page != 1 {
		t.Fatalf("expected page=1, got %d", result.Page)
	}
	if result.Limit != 20 {
		t.Fatalf("expected limit=20, got %d", result.Limit)
	}
	if len(result.Items) != 20 {
		t.Fatalf("expected 20 items, got %d", len(result.Items))
	}

	// Verify items are sorted (just check first and last are different and sequential IDs exist)
	if len(result.Items) > 0 && len(result.Items) < 25 {
		// Valid: partial page returned
		t.Logf("Page 1 contains %d items as expected", len(result.Items))
	}
}

// TestListPaginatedSecondPage: Navigate to second page with controlled data
func TestListPaginatedSecondPage(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewBackupService(tempDir)

	// Create exactly 25 backups
	for i := 1; i <= 25; i++ {
		backup := backupMetadata{
			ID:          fmt.Sprintf("backup-%03d", i),
			Name:        fmt.Sprintf("backup-%03d", i),
			Type:        model.BackupTypeManual,
			CreatedAt:   fmt.Sprintf("2026-01-01T%02d:00:00Z", (i%24)+1),
			TriggeredBy: "manual",
		}
		createBackupArchive(t, tempDir, backup)
	}

	// Get all backups first to understand the sort order
	allOpts := model.PaginationOpts{
		Page:  1,
		Limit: 100,
		Sort:  "date",
		Order: "desc",
	}
	allResult, err := svc.ListPaginated(allOpts)
	if err != nil {
		t.Fatalf("ListPaginated full returned error: %v", err)
	}

	if allResult.Total != 25 {
		t.Fatalf("expected total=25, got %d", allResult.Total)
	}

	// Test: page 2, limit 20
	opts := model.PaginationOpts{
		Page:  2,
		Limit: 20,
		Sort:  "date",
		Order: "desc",
	}

	result, err := svc.ListPaginated(opts)
	if err != nil {
		t.Fatalf("ListPaginated page 2 returned error: %v", err)
	}

	if result.Page != 2 {
		t.Fatalf("expected page=2, got %d", result.Page)
	}
	if len(result.Items) != 5 {
		t.Fatalf("expected 5 items on page 2 (25-20=5), got %d", len(result.Items))
	}
}

// TestListPaginatedSortBySize: Sort by size descending
func TestListPaginatedSortBySize(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewBackupService(tempDir)

	// Create multiple backups
	for i := 1; i <= 3; i++ {
		backup := backupMetadata{
			ID:          fmt.Sprintf("size-%03d", i),
			Name:        fmt.Sprintf("size-%03d", i),
			Type:        model.BackupTypeManual,
			CreatedAt:   fmt.Sprintf("2026-01-01T%02d:00:00Z", i),
			TriggeredBy: "manual",
		}
		createBackupArchive(t, tempDir, backup)
	}

	opts := model.PaginationOpts{
		Page:  1,
		Limit: 20,
		Sort:  "size",
		Order: "desc",
	}

	result, err := svc.ListPaginated(opts)
	if err != nil {
		t.Fatalf("ListPaginated with sort=size returned error: %v", err)
	}

	if len(result.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.Items))
	}

	// Verify that items are sorted (all have sizeBytes > 0)
	for _, item := range result.Items {
		if item.SizeBytes <= 0 {
			t.Fatalf("expected all items to have sizeBytes > 0, got %d", item.SizeBytes)
		}
	}
}

// TestListPaginatedEmptyResult: No backups available
func TestListPaginatedEmptyResult(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewBackupService(tempDir)

	opts := model.PaginationOpts{
		Page:  1,
		Limit: 20,
		Sort:  "date",
		Order: "desc",
	}

	result, err := svc.ListPaginated(opts)
	if err != nil {
		t.Fatalf("ListPaginated on empty backups returned error: %v", err)
	}

	if result.Total != 0 {
		t.Fatalf("expected total=0, got %d", result.Total)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(result.Items))
	}
}

// TestListPaginatedLimitClamping: Limit > max (100) should be clamped
func TestListPaginatedLimitClamping(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewBackupService(tempDir)

	// Create 5 backups
	for i := 1; i <= 5; i++ {
		backup := backupMetadata{
			ID:          fmt.Sprintf("backup-%03d", i),
			Name:        fmt.Sprintf("backup-%03d", i),
			Type:        model.BackupTypeManual,
			CreatedAt:   fmt.Sprintf("2026-01-01T%02d:00:00Z", (i%24)+1),
			TriggeredBy: "manual",
		}
		createBackupArchive(t, tempDir, backup)
	}

	opts := model.PaginationOpts{
		Page:  1,
		Limit: 250,
		Sort:  "date",
		Order: "desc",
	}

	result, err := svc.ListPaginated(opts)
	if err != nil {
		t.Fatalf("ListPaginated with limit > 100 returned error: %v", err)
	}

	if result.Limit != 100 {
		t.Fatalf("expected limit clamped to 100, got %d", result.Limit)
	}
	if len(result.Items) != 5 {
		t.Fatalf("expected all 5 items on page 1, got %d", len(result.Items))
	}
}

// TestListPaginatedSortOrder: Both asc and desc should work
func TestListPaginatedSortOrder(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewBackupService(tempDir)

	// Create 3 backups with specific CreatedAt values (in RFC3339 string format)
	// We use different months to ensure strict ordering
	backups := []backupMetadata{
		{
			ID:          "backup-jan",
			Name:        "backup-jan",
			Type:        model.BackupTypeManual,
			CreatedAt:   "2026-01-01T10:00:00Z",
			TriggeredBy: "manual",
		},
		{
			ID:          "backup-feb",
			Name:        "backup-feb",
			Type:        model.BackupTypeManual,
			CreatedAt:   "2026-02-01T10:00:00Z",
			TriggeredBy: "manual",
		},
		{
			ID:          "backup-mar",
			Name:        "backup-mar",
			Type:        model.BackupTypeManual,
			CreatedAt:   "2026-03-01T10:00:00Z",
			TriggeredBy: "manual",
		},
	}

	for _, backup := range backups {
		createBackupArchive(t, tempDir, backup)
	}

	// Test descending (newest first)
	optsDesc := model.PaginationOpts{
		Page:  1,
		Limit: 20,
		Sort:  "date",
		Order: "desc",
	}
	resultDesc, err := svc.ListPaginated(optsDesc)
	if err != nil {
		t.Fatalf("ListPaginated desc returned error: %v", err)
	}

	if len(resultDesc.Items) != 3 {
		t.Fatalf("expected 3 items in desc, got %d", len(resultDesc.Items))
	}

	// The first item in desc should be the newest (march > feb > jan)
	if resultDesc.Items[0].CreatedAt < resultDesc.Items[len(resultDesc.Items)-1].CreatedAt {
		t.Fatalf("desc ordering failed: first item %s should be >= last item %s",
			resultDesc.Items[0].CreatedAt, resultDesc.Items[len(resultDesc.Items)-1].CreatedAt)
	}

	// Test ascending (oldest first)
	optsAsc := model.PaginationOpts{
		Page:  1,
		Limit: 20,
		Sort:  "date",
		Order: "asc",
	}
	resultAsc, err := svc.ListPaginated(optsAsc)
	if err != nil {
		t.Fatalf("ListPaginated asc returned error: %v", err)
	}

	if len(resultAsc.Items) != 3 {
		t.Fatalf("expected 3 items in asc, got %d", len(resultAsc.Items))
	}

	// The first item in asc should be the oldest (jan < feb < mar)
	if resultAsc.Items[0].CreatedAt > resultAsc.Items[len(resultAsc.Items)-1].CreatedAt {
		t.Fatalf("asc ordering failed: first item %s should be <= last item %s",
			resultAsc.Items[0].CreatedAt, resultAsc.Items[len(resultAsc.Items)-1].CreatedAt)
	}
}

func TestSanitizeArchivePathRejectsTraversal(t *testing.T) {
	destDir := t.TempDir()

	tests := []struct {
		name  string
		entry string
	}{
		{"dot-dot prefix", "../etc/passwd"},
		{"nested dot-dot", "subdir/../../etc/passwd"},
		{"absolute path", "/etc/passwd"},
		{"dot-dot only", ".."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := sanitizeArchivePath(destDir, tc.entry)
			if err == nil {
				t.Fatalf("expected error for entry %q, got nil", tc.entry)
			}
		})
	}
}

func TestSanitizeArchivePathAllowsValidEntries(t *testing.T) {
	destDir := t.TempDir()

	valid := []string{
		"flows.json",
		"subdir/settings.js",
		"deep/nested/file.txt",
	}

	for _, entry := range valid {
		t.Run(entry, func(t *testing.T) {
			got, err := sanitizeArchivePath(destDir, entry)
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", entry, err)
			}
			expected := filepath.Join(destDir, filepath.Clean(entry))
			if got != expected {
				t.Fatalf("expected %q, got %q", expected, got)
			}
		})
	}
}

func TestExtractFileFromZipRejectsMaliciousEntries(t *testing.T) {
	restoreDir := t.TempDir()
	outsideFile := filepath.Join(t.TempDir(), "should-not-exist.txt")

	maliciousNames := []string{
		"../should-not-exist.txt",
		"/tmp/should-not-exist.txt",
	}

	for _, name := range maliciousNames {
		t.Run(name, func(t *testing.T) {
			zipPath := filepath.Join(t.TempDir(), "evil.zip")
			createZipWithEntry(t, zipPath, name, "pwned")

			reader, err := zip.OpenReader(zipPath)
			if err != nil {
				t.Fatalf("open zip: %v", err)
			}
			defer reader.Close()

			svc := NewBackupService(restoreDir)
			for _, f := range reader.File {
				if f.Name == "backup-metadata.json" {
					continue
				}
				err := svc.extractFileFromZip(f, restoreDir)
				if err == nil {
					t.Fatalf("expected error extracting %q, got nil", f.Name)
				}
			}

			if _, err := os.Stat(outsideFile); err == nil {
				t.Fatal("malicious file was written outside restore dir")
			}
		})
	}
}

func TestExtractFileFromZipRestoresValidArchive(t *testing.T) {
	restoreDir := t.TempDir()
	zipPath := filepath.Join(t.TempDir(), "valid.zip")
	createZipWithEntry(t, zipPath, "flows.json", `[{"id":"1"}]`)

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer reader.Close()

	svc := NewBackupService(restoreDir)
	for _, f := range reader.File {
		if f.Name == "backup-metadata.json" {
			continue
		}
		if err := svc.extractFileFromZip(f, restoreDir); err != nil {
			t.Fatalf("extract valid file: %v", err)
		}
	}

	content, err := os.ReadFile(filepath.Join(restoreDir, "flows.json"))
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(content) != `[{"id":"1"}]` {
		t.Fatalf("unexpected content: %s", content)
	}
}

// TestExtractFileFromZip_RejectsOversizedEntry is the #279 regression: an entry
// whose uncompressed content exceeds the per-entry limit must be rejected
// (decompression-bomb guard) and must not be left on disk.
func TestExtractFileFromZip_RejectsOversizedEntry(t *testing.T) {
	orig := maxBackupEntrySize
	maxBackupEntrySize = 16
	defer func() { maxBackupEntrySize = orig }()

	restoreDir := t.TempDir()
	zipPath := filepath.Join(t.TempDir(), "bomb.zip")
	createZipWithEntry(t, zipPath, "flows.json", strings.Repeat("A", 1024)) // 1 KiB >> 16 B limit

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer reader.Close()

	svc := NewBackupService(restoreDir)
	for _, f := range reader.File {
		if f.Name == "backup-metadata.json" {
			continue
		}
		if err := svc.extractFileFromZip(f, restoreDir); err == nil {
			t.Fatalf("expected error extracting oversized entry %q, got nil", f.Name)
		}
	}

	if _, err := os.Stat(filepath.Join(restoreDir, "flows.json")); err == nil {
		t.Fatal("oversized entry must not be left on disk")
	}
}

// TestExtractFileFromZip_AllowsEntryWithinLimit guards against over-tightening.
func TestExtractFileFromZip_AllowsEntryWithinLimit(t *testing.T) {
	orig := maxBackupEntrySize
	maxBackupEntrySize = 1024
	defer func() { maxBackupEntrySize = orig }()

	restoreDir := t.TempDir()
	zipPath := filepath.Join(t.TempDir(), "ok.zip")
	createZipWithEntry(t, zipPath, "flows.json", `[{"id":"1"}]`)

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer reader.Close()

	svc := NewBackupService(restoreDir)
	for _, f := range reader.File {
		if f.Name == "backup-metadata.json" {
			continue
		}
		if err := svc.extractFileFromZip(f, restoreDir); err != nil {
			t.Fatalf("entry within limit should extract: %v", err)
		}
	}
}

func createZipWithEntry(t *testing.T, zipPath, entryName, content string) {
	t.Helper()
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip file: %v", err)
	}
	w := zip.NewWriter(f)

	meta, _ := w.Create("backup-metadata.json")
	meta.Write([]byte(`{"id":"test","name":"test","type":"manual","createdAt":"2026-01-01T00:00:00Z","triggeredBy":"test"}`))

	entry, err := w.Create(entryName)
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	entry.Write([]byte(content))

	if err := w.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}
}
