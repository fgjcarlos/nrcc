package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/composedof2/nrcc/internal/model"
)

// mockRunner implements execRunner for testing
type mockRunner struct {
	output    []byte
	err       error
	lastCalls []struct {
		name string
		args []string
	}
}

func (m *mockRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	m.lastCalls = append(m.lastCalls, struct {
		name string
		args []string
	}{name, args})
	return m.output, m.err
}

// TestNewUpdateService tests basic initialization
func TestNewUpdateService(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	if svc == nil {
		t.Fatal("NewUpdateService should not return nil")
	}

	if svc.dataDir != tmpDir {
		t.Errorf("Expected dataDir %s, got %s", tmpDir, svc.dataDir)
	}
}

// TestGetCachedStatus_Empty tests that GetCachedStatus now performs an initial check on startup
// This ensures the UI never shows "unknown" on first page load
func TestGetCachedStatus_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create a mock service with controlled version functions
	svc := NewUpdateService(tmpDir)
	
	// Override the version functions to return controlled values for this test
	svc.getInstalledVersionFn = func(ctx context.Context) string {
		return "4.0.1"
	}
	svc.getLatestVersionFn = func(ctx context.Context) (string, error) {
		return "4.0.1", nil
	}
	
	status := svc.GetCachedStatus()

	// After initialization, cache should be populated (not empty)
	if status.CurrentVersion == "" {
		t.Error("Expected CurrentVersion to be populated after initialization")
	}
	if status.LatestVersion == "" {
		t.Error("Expected LatestVersion to be populated after initialization")
	}
}

// TestCompareVersions tests version comparison logic
func TestCompareVersions(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	tests := []struct {
		v1       string
		v2       string
		expected int
		desc     string
	}{
		{"1.0.0", "1.0.0", 0, "equal versions"},
		{"1.0.0", "2.0.0", -1, "v1 < v2 (major)"},
		{"2.0.0", "1.0.0", 1, "v1 > v2 (major)"},
		{"1.1.0", "1.2.0", -1, "v1 < v2 (minor)"},
		{"1.2.0", "1.1.0", 1, "v1 > v2 (minor)"},
		{"1.0.1", "1.0.2", -1, "v1 < v2 (patch)"},
		{"1.0.2", "1.0.1", 1, "v1 > v2 (patch)"},
		{"v1.0.0", "1.0.0", 0, "v-prefix normalization"},
		{"1.0", "1.0.0", 0, "missing patch version"},
		{"1", "1.0.0", 0, "missing minor and patch"},
	}

	for _, tt := range tests {
		result := svc.compareVersions(tt.v1, tt.v2)
		if result != tt.expected {
			t.Errorf("%s: compareVersions(%s, %s) = %d, expected %d", tt.desc, tt.v1, tt.v2, result, tt.expected)
		}
	}
}

// TestForceCheck_Success tests a successful force check
func TestForceCheck_Success(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	// Setup mock functions
	svc.getInstalledVersionFn = func(ctx context.Context) string {
		return "3.0.0"
	}
	svc.getLatestVersionFn = func(ctx context.Context) (string, error) {
		return "3.1.0", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entry, err := svc.ForceCheck(ctx)
	if err != nil {
		t.Errorf("ForceCheck should not error: %v", err)
	}

	if entry.CurrentVersion != "3.0.0" {
		t.Errorf("Expected CurrentVersion 3.0.0, got %s", entry.CurrentVersion)
	}
	if entry.LatestVersion != "3.1.0" {
		t.Errorf("Expected LatestVersion 3.1.0, got %s", entry.LatestVersion)
	}
	if !entry.UpdateAvailable {
		t.Error("Expected UpdateAvailable to be true")
	}
	if entry.Error != "" {
		t.Errorf("Expected no error, got %s", entry.Error)
	}
	if entry.CheckedAt.IsZero() {
		t.Error("Expected CheckedAt to be set")
	}
}

// TestForceCheck_Error tests force check with an error
func TestForceCheck_Error(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	// Setup mock functions
	svc.getInstalledVersionFn = func(ctx context.Context) string {
		return "3.0.0"
	}
	svc.getLatestVersionFn = func(ctx context.Context) (string, error) {
		return "", context.DeadlineExceeded
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entry, err := svc.ForceCheck(ctx)
	if err != nil {
		t.Errorf("ForceCheck should not return error itself: %v", err)
	}

	if entry.CurrentVersion != "3.0.0" {
		t.Errorf("Expected CurrentVersion 3.0.0, got %s", entry.CurrentVersion)
	}
	if entry.Error == "" {
		t.Error("Expected Error to be set")
	}
	if entry.UpdateAvailable {
		t.Error("Expected UpdateAvailable to be false on error")
	}
}

// TestCacheWritten tests that cache is written to disk after ForceCheck
func TestCacheWritten(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	// Setup mock functions
	svc.getInstalledVersionFn = func(ctx context.Context) string {
		return "3.0.0"
	}
	svc.getLatestVersionFn = func(ctx context.Context) (string, error) {
		return "3.1.0", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := svc.ForceCheck(ctx)
	if err != nil {
		t.Fatalf("ForceCheck failed: %v", err)
	}

	// Verify cache file exists
	cacheFile := filepath.Join(tmpDir, "update_cache.json")
	if _, err := os.Stat(cacheFile); err != nil {
		t.Errorf("Expected cache file to exist at %s: %v", cacheFile, err)
	}

	// Verify we can read it back
	newSvc := NewUpdateService(tmpDir)
	status := newSvc.GetCachedStatus()

	if status.CurrentVersion != "3.0.0" {
		t.Errorf("Expected persisted CurrentVersion 3.0.0, got %s", status.CurrentVersion)
	}
	if status.LatestVersion != "3.1.0" {
		t.Errorf("Expected persisted LatestVersion 3.1.0, got %s", status.LatestVersion)
	}
}

// TestStartStop tests goroutine lifecycle
func TestStartStop(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	// Override the poll interval to be very short for testing
	svc.pollInterval = 100 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start the polling loop
	svc.Start(ctx)

	// Give it time to run
	time.Sleep(150 * time.Millisecond)

	// Stop should not hang
	done := make(chan struct{})
	go func() {
		svc.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Stop() took too long, possible goroutine leak")
	}
}

// TestCheckForUpdate_BackwardCompatibility tests backward compat wrapper
func TestCheckForUpdate_BackwardCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	svc.getInstalledVersionFn = func(ctx context.Context) string {
		return "3.0.0"
	}
	svc.getLatestVersionFn = func(ctx context.Context) (string, error) {
		return "3.1.0", nil
	}

	status, err := svc.CheckForUpdate()
	if err != nil {
		t.Errorf("CheckForUpdate should not error: %v", err)
	}

	if status.CurrentVersion != "3.0.0" {
		t.Errorf("Expected CurrentVersion 3.0.0, got %s", status.CurrentVersion)
	}
	if status.LatestVersion != "3.1.0" {
		t.Errorf("Expected LatestVersion 3.1.0, got %s", status.LatestVersion)
	}
	if !status.UpdateAvailable {
		t.Error("Expected UpdateAvailable to be true")
	}
}

// TestConcurrentForceCheck tests that concurrent force checks don't run concurrently
func TestConcurrentForceCheck(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	checkCount := 0
	svc.getInstalledVersionFn = func(ctx context.Context) string {
		checkCount++
		time.Sleep(100 * time.Millisecond) // Simulate slow check
		return "3.0.0"
	}
	svc.getLatestVersionFn = func(ctx context.Context) (string, error) {
		return "3.1.0", nil
	}

	ctx := context.Background()

	// Start two concurrent force checks
	done := make(chan struct{})
	go func() {
		svc.ForceCheck(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond) // Let first one start
	svc.ForceCheck(ctx)               // This should block until first completes

	<-done

	// Both calls should have completed
	// The second call should have returned cached status without re-running the check
	if checkCount != 1 {
		t.Errorf("Expected 1 actual check, got %d (concurrency guard may be broken)", checkCount)
	}
}

// TestSanitizeErrorMessage tests error message sanitization
func TestSanitizeErrorMessage(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	tests := []struct {
		input    error
		expected string
		desc     string
	}{
		{nil, "", "nil error returns empty string"},
		{
			fmt.Errorf("exit status 234"),
			"Update check failed. Please ensure npm is properly installed and Node-RED is accessible.",
			"npm exit status is sanitized",
		},
		{
			fmt.Errorf("exit status 1"),
			"Update check failed. Please ensure npm is properly installed and Node-RED is accessible.",
			"npm exit status 1 is sanitized",
		},
		{
			fmt.Errorf("context deadline exceeded"),
			"Update check timed out. Please try again.",
			"timeout error is user-friendly",
		},
		{
			fmt.Errorf("i/o timeout"),
			"Update check timed out. Please try again.",
			"io timeout is user-friendly",
		},
		{
			fmt.Errorf("connection refused"),
			"Network error while checking for updates. Please check your internet connection.",
			"connection error is user-friendly",
		},
	}

	for _, tt := range tests {
		result := svc.sanitizeErrorMessage(tt.input)
		if result != tt.expected {
			t.Errorf("%s: got %q, expected %q", tt.desc, result, tt.expected)
		}
	}
}

// TestGetFlowState tests the GetFlowState getter method
func TestGetFlowState(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	// Initially should be Idle
	state := svc.GetFlowState()
	if state.State != "Idle" {
		t.Errorf("Expected initial state Idle, got %s", state.State)
	}

	// Set flowState and verify retrieval
	svc.setFlowState(model.UpdateFlowState{
		State:    "Applying",
		Phase:    "applying",
		BackupID: "test-backup-123",
	})

	state = svc.GetFlowState()
	if state.State != "Applying" {
		t.Errorf("Expected state Applying, got %s", state.State)
	}
	if state.BackupID != "test-backup-123" {
		t.Errorf("Expected BackupID test-backup-123, got %s", state.BackupID)
	}
}

// TestCreateBackup tests backup entry creation
// failingBackupCreator simulates a backup engine that cannot write to disk.
type failingBackupCreator struct{}

func (failingBackupCreator) CreateTyped(model.BackupType, string) (model.Backup, error) {
	return model.Backup{}, fmt.Errorf("disk full")
}

func TestCreateBackup(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)
	svc.SetBackupCreator(NewBackupService(tmpDir))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entry, err := svc.CreateBackup(ctx, "4.0.1")
	if err != nil {
		t.Fatalf("CreateBackup should not error: %v", err)
	}

	if entry.ID == "" {
		t.Error("Expected BackupEntry ID to be set")
	}
	if entry.FromVersion != "4.0.1" {
		t.Errorf("Expected FromVersion 4.0.1, got %s", entry.FromVersion)
	}
	if entry.Status != "completed" {
		t.Errorf("Expected Status completed, got %s", entry.Status)
	}
	if entry.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be set")
	}

	// Regression for #276: the backup must be a REAL file on disk, not a phantom record.
	if entry.Path == "" {
		t.Fatal("Expected BackupEntry Path to be set")
	}
	info, statErr := os.Stat(entry.Path)
	if statErr != nil {
		t.Fatalf("Expected backup archive to exist at %s: %v", entry.Path, statErr)
	}
	if info.Size() == 0 {
		t.Error("Expected backup archive to be non-empty")
	}
	if entry.SizeBytes != info.Size() {
		t.Errorf("Expected SizeBytes %d to match file size %d", entry.SizeBytes, info.Size())
	}
}

// TestCreateBackup_FailsWithoutBackupCreator is part of the #276 regression:
// the service must refuse to report a "completed" backup when no real backup
// engine is wired in, rather than fabricating a placeholder entry.
func TestCreateBackup_FailsWithoutBackupCreator(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := svc.CreateBackup(ctx, "4.0.1"); err == nil {
		t.Fatal("expected error when no backup creator is configured")
	}
}

// TestCreateBackup_PropagatesError ensures a real backup failure surfaces as an
// error so the caller can abort the update.
func TestCreateBackup_PropagatesError(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)
	svc.SetBackupCreator(failingBackupCreator{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := svc.CreateBackup(ctx, "4.0.1"); err == nil {
		t.Fatal("expected CreateBackup to propagate the backup engine error")
	}
}

// TestAppendBackup tests backup catalog persistence and trimming
func TestAppendBackup(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	// Add 5 backups
	for i := 1; i <= 5; i++ {
		entry := model.BackupEntry{
			ID:          fmt.Sprintf("backup_%d", i),
			FromVersion: fmt.Sprintf("4.0.%d", i),
			Status:      "completed",
			Timestamp:   time.Now().UTC().Add(time.Duration(i) * time.Hour),
		}
		if err := svc.AppendBackup(entry); err != nil {
			t.Fatalf("AppendBackup failed: %v", err)
		}
	}

	// Verify catalog file exists and has 5 entries
	backups, err := svc.backupStore.Read()
	if err != nil {
		t.Fatalf("Failed to read backup catalog: %v", err)
	}
	if len(backups) != 5 {
		t.Errorf("Expected 5 backups, got %d", len(backups))
	}

	// Add 6th backup; should trim oldest
	entry6 := model.BackupEntry{
		ID:          "backup_6",
		FromVersion: "4.0.6",
		Status:      "completed",
		Timestamp:   time.Now().UTC().Add(6 * time.Hour),
	}
	if err := svc.AppendBackup(entry6); err != nil {
		t.Fatalf("AppendBackup failed: %v", err)
	}

	// Verify still 5 entries and oldest was removed
	backups, err = svc.backupStore.Read()
	if err != nil {
		t.Fatalf("Failed to read backup catalog: %v", err)
	}
	if len(backups) != 5 {
		t.Errorf("Expected 5 backups after trim, got %d", len(backups))
	}

	// First entry should now be backup_2 (backup_1 was trimmed)
	if backups[0].ID != "backup_2" {
		t.Errorf("Expected first backup to be backup_2 after trim, got %s", backups[0].ID)
	}
	if backups[4].ID != "backup_6" {
		t.Errorf("Expected last backup to be backup_6, got %s", backups[4].ID)
	}
}

// TestApplyUpdateWithBackup_CannotStartIfNotIdle tests that 409-like error is returned
func TestApplyUpdateWithBackup_CannotStartIfNotIdle(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	// Manually set state to Applying to simulate in-progress update
	svc.setFlowState(model.UpdateFlowState{
		State: "Applying",
		Phase: "applying",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := svc.ApplyUpdateWithBackup(ctx)
	if err == nil {
		t.Error("Expected error when state is not Idle")
	}
	if err.Error() != fmt.Sprintf("update cannot start: state is %s", "Applying") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

// TestApplyUpdateWithBackup_ConcurrencyGuard tests that only one update can proceed
func TestApplyUpdateWithBackup_ConcurrencyGuard(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	// Override ApplyUpdate to block for a bit
	blockChan := make(chan struct{})
	svc.runner = &mockRunner{output: []byte(""), err: nil}
	
	// Manually acquire lock to simulate first call holding it
	svc.applyMu.Lock()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to acquire; should fail immediately
	err := svc.ApplyUpdateWithBackup(ctx)
	
	// Release lock
	svc.applyMu.Unlock()
	
	if err == nil {
		t.Error("Expected error when applyMu is held")
	}
	if err.Error() != "update already in progress" {
		t.Errorf("Expected 'update already in progress' error, got: %v", err)
	}

	// Verify state is still Idle (no partial change)
	state := svc.GetFlowState()
	if state.State != "Idle" {
		t.Errorf("Expected state to remain Idle on error, got %s", state.State)
	}

	close(blockChan)
}

// TestApplyUpdateWithBackup_SuccessfulFlow tests successful flow: BackingUp → Applying → Completed
func TestApplyUpdateWithBackup_SuccessfulFlow(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)
	svc.SetBackupCreator(NewBackupService(tmpDir))

	// Mock version functions
	svc.getInstalledVersionFn = func(ctx context.Context) string {
		return "4.0.1"
	}
	svc.getLatestVersionFn = func(ctx context.Context) (string, error) {
		return "4.0.2", nil
	}

	// Force a check so cache has versions
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, _ = svc.ForceCheck(ctx)
	cancel()

	// Override runner to succeed
	svc.runner = &mockRunner{output: []byte("success"), err: nil}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := svc.ApplyUpdateWithBackup(ctx)
	if err != nil {
		t.Errorf("ApplyUpdateWithBackup should succeed: %v", err)
	}

	// Verify final state is Completed
	state := svc.GetFlowState()
	if state.State != "Completed" {
		t.Errorf("Expected final state Completed, got %s", state.State)
	}

	// Verify backup was persisted
	backups, err := svc.backupStore.Read()
	if err != nil {
		t.Errorf("Failed to read backup catalog: %v", err)
	}
	if len(backups) != 1 {
		t.Errorf("Expected 1 backup persisted, got %d", len(backups))
	}
	if backups[0].FromVersion != "4.0.1" {
		t.Errorf("Expected backup FromVersion 4.0.1, got %s", backups[0].FromVersion)
	}
}

// TestApplyUpdateWithBackup_NpmFailure tests failure during npm update phase
func TestApplyUpdateWithBackup_NpmFailure(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)
	svc.SetBackupCreator(NewBackupService(tmpDir))

	// Mock version functions
	svc.getInstalledVersionFn = func(ctx context.Context) string {
		return "4.0.1"
	}
	svc.getLatestVersionFn = func(ctx context.Context) (string, error) {
		return "4.0.2", nil
	}

	// Force a check so cache has versions
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, _ = svc.ForceCheck(ctx)
	cancel()

	// Override runner to fail at npm step
	svc.runner = &mockRunner{output: []byte(""), err: fmt.Errorf("npm install failed")}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := svc.ApplyUpdateWithBackup(ctx)
	if err == nil {
		t.Error("Expected error when npm install fails")
	}

	// Verify state is Failed
	state := svc.GetFlowState()
	if state.State != "Failed" {
		t.Errorf("Expected state Failed, got %s", state.State)
	}
	if state.Error != "update_failed" {
		t.Errorf("Expected error reason 'update_failed', got %s", state.Error)
	}

	// Verify backup was still persisted even though npm failed
	backups, err := svc.backupStore.Read()
	if err != nil {
		t.Errorf("Failed to read backup catalog: %v", err)
	}
	if len(backups) != 1 {
		t.Errorf("Expected 1 backup persisted despite npm failure, got %d", len(backups))
	}
}

// TestApplyUpdateWithBackup_AbortsWhenBackupFails is the core #276 regression:
// if the pre-update backup cannot be created, the npm install must NOT run and
// the flow must end in Failed/backup_failed.
func TestApplyUpdateWithBackup_AbortsWhenBackupFails(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)
	svc.SetBackupCreator(failingBackupCreator{})

	svc.getInstalledVersionFn = func(ctx context.Context) string { return "4.0.1" }
	svc.getLatestVersionFn = func(ctx context.Context) (string, error) { return "4.0.2", nil }

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, _ = svc.ForceCheck(ctx)
	cancel()

	runner := &mockRunner{output: []byte("success"), err: nil}
	svc.runner = runner

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := svc.ApplyUpdateWithBackup(ctx)
	if err == nil {
		t.Fatal("expected error when backup creation fails")
	}

	// npm install must never have been invoked.
	for _, call := range runner.lastCalls {
		for _, a := range call.args {
			if a == "install" {
				t.Fatalf("npm install must not run when the pre-update backup fails; calls: %v", runner.lastCalls)
			}
		}
	}

	state := svc.GetFlowState()
	if state.State != "Failed" {
		t.Errorf("expected state Failed, got %s", state.State)
	}
	if state.Error != "backup_failed" {
		t.Errorf("expected error reason 'backup_failed', got %s", state.Error)
	}
}

// sequenceRunner returns different results per call index.
type sequenceRunner struct {
	results []struct {
		output []byte
		err    error
	}
	calls []struct {
		name string
		args []string
	}
}

func (s *sequenceRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	idx := len(s.calls)
	s.calls = append(s.calls, struct {
		name string
		args []string
	}{name, args})
	if idx < len(s.results) {
		return s.results[idx].output, s.results[idx].err
	}
	return nil, nil
}

func TestApplyUpdate_PinsResolvedVersion(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	svc.getInstalledVersionFn = func(ctx context.Context) string { return "4.0.1" }
	svc.getLatestVersionFn = func(ctx context.Context) (string, error) { return "4.0.2", nil }

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, _ = svc.ForceCheck(ctx)
	cancel()

	runner := &sequenceRunner{
		results: []struct {
			output []byte
			err    error
		}{
			{[]byte("installed"), nil}, // npm install
			{[]byte("ok"), nil},        // npm audit
		},
	}
	svc.runner = runner

	err := svc.ApplyUpdate()
	if err != nil {
		t.Fatalf("ApplyUpdate should succeed: %v", err)
	}

	if len(runner.calls) < 1 {
		t.Fatal("expected at least one call")
	}

	installArgs := runner.calls[0].args
	found := false
	for _, arg := range installArgs {
		if arg == "node-red@4.0.2" {
			found = true
		}
		if arg == "node-red@latest" {
			t.Fatal("ApplyUpdate must not use unpinned @latest")
		}
	}
	if !found {
		t.Fatalf("expected pinned specifier node-red@4.0.2, got args: %v", installArgs)
	}
}

func TestApplyUpdate_RejectsWithoutResolvedVersion(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	// Clear cache so LatestVersion is empty
	svc.cacheMu.Lock()
	svc.cache = model.UpdateCacheEntry{}
	svc.cacheMu.Unlock()

	svc.runner = &mockRunner{output: nil, err: nil}

	err := svc.ApplyUpdate()
	if err == nil {
		t.Fatal("expected error when no resolved version is cached")
	}
}

func TestApplyUpdate_BlocksOnAuditFailure(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	svc.getInstalledVersionFn = func(ctx context.Context) string { return "4.0.1" }
	svc.getLatestVersionFn = func(ctx context.Context) (string, error) { return "4.0.2", nil }

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, _ = svc.ForceCheck(ctx)
	cancel()

	runner := &sequenceRunner{
		results: []struct {
			output []byte
			err    error
		}{
			{[]byte("installed"), nil},                                  // npm install succeeds
			{[]byte("critical vuln found"), fmt.Errorf("exit status 1")}, // npm audit fails
		},
	}
	svc.runner = runner

	err := svc.ApplyUpdate()
	if err == nil {
		t.Fatal("expected error when post-install audit finds critical vulnerabilities")
	}
	if len(runner.calls) != 2 {
		t.Fatalf("expected 2 runner calls (install + audit), got %d", len(runner.calls))
	}
}

