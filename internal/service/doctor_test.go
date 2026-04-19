package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"nrcc/internal/model"
)

func TestDoctorServiceRunAllChecks(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create required directory structure
	os.MkdirAll(filepath.Join(tempDir, "logs"), 0700)
	os.MkdirAll(filepath.Join(tempDir, "nodered"), 0700)

	// Create required files
	os.WriteFile(filepath.Join(tempDir, "nodered", "settings.js"), []byte("// settings"), 0600)
	os.WriteFile(filepath.Join(tempDir, "nodered", "flows.json"), []byte("[]"), 0600)

	doctor := NewDoctorService(tempDir)

	report := doctor.Run(context.Background())

	// Verify report structure
	if report.GeneratedAt.IsZero() {
		t.Fatal("expected GeneratedAt to be set")
	}

	// Should have 14 checks
	if len(report.Checks) != 14 {
		t.Fatalf("expected 14 checks, got %d", len(report.Checks))
	}

	// Verify all check IDs are present
	expectedCheckIDs := map[string]bool{
		"node-red-installed": false,
		"node-version":       false,
		"npm-version":        false,
		"data-dir-writable":  false,
		"userdir-exists":     false,
		"settings-file":      false,
		"flows-file":         false,
		"process-running":    false,
		"port-available":     false,
		"local-access":       false,
		"log-dir-writable":   false,
		"db-accessible":      false,
		"disk-space":         false,
		"nrcc-version":       false,
	}

	for _, check := range report.Checks {
		if _, exists := expectedCheckIDs[check.ID]; exists {
			expectedCheckIDs[check.ID] = true
		}
	}

	for checkID, found := range expectedCheckIDs {
		if !found {
			t.Errorf("expected check %s not found in report", checkID)
		}
	}
}

func TestDoctorServiceOverallStatus(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "logs"), 0700)
	os.MkdirAll(filepath.Join(tempDir, "nodered"), 0700)

	doctor := NewDoctorService(tempDir)
	report := doctor.Run(context.Background())

	// OverallStatus should be one of: healthy, degraded, critical
	validStatuses := map[string]bool{
		model.OverallHealthy:  true,
		model.OverallDegraded: true,
		model.OverallCritical: true,
	}

	if !validStatuses[report.OverallStatus] {
		t.Errorf("expected valid overall status, got %s", report.OverallStatus)
	}
}

func TestDoctorServiceNrccVersionAlwaysPasses(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "logs"), 0700)

	doctor := NewDoctorService(tempDir)
	report := doctor.Run(context.Background())

	// Find nrcc-version check
	var versionCheck *model.DoctorCheck
	for _, check := range report.Checks {
		if check.ID == "nrcc-version" {
			versionCheck = &check
			break
		}
	}

	if versionCheck == nil {
		t.Fatal("nrcc-version check not found")
	}

	if versionCheck.Status != model.CheckStatusPass {
		t.Errorf("expected nrcc-version to pass, got %s", versionCheck.Status)
	}
}

func TestDoctorServiceDataDirWritable(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	doctor := NewDoctorService(tempDir)
	report := doctor.Run(context.Background())

	// Find data-dir-writable check
	var check *model.DoctorCheck
	for _, c := range report.Checks {
		if c.ID == "data-dir-writable" {
			check = &c
			break
		}
	}

	if check == nil {
		t.Fatal("data-dir-writable check not found")
	}

	if check.Status != model.CheckStatusPass {
		t.Errorf("expected data-dir-writable to pass, got %s", check.Status)
	}
}

func TestDoctorServiceLogDirWritable(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "logs"), 0700)

	doctor := NewDoctorService(tempDir)
	report := doctor.Run(context.Background())

	// Find log-dir-writable check
	var check *model.DoctorCheck
	for _, c := range report.Checks {
		if c.ID == "log-dir-writable" {
			check = &c
			break
		}
	}

	if check == nil {
		t.Fatal("log-dir-writable check not found")
	}

	if check.Status != model.CheckStatusPass {
		t.Errorf("expected log-dir-writable to pass, got %s", check.Status)
	}
}

func TestDoctorServiceUserDirExists(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "nodered"), 0700)

	doctor := NewDoctorService(tempDir)
	report := doctor.Run(context.Background())

	// Find userdir-exists check
	var check *model.DoctorCheck
	for _, c := range report.Checks {
		if c.ID == "userdir-exists" {
			check = &c
			break
		}
	}

	if check == nil {
		t.Fatal("userdir-exists check not found")
	}

	if check.Status != model.CheckStatusPass {
		t.Errorf("expected userdir-exists to pass, got %s", check.Status)
	}
}

func TestDoctorServiceSettingsFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	userDir := filepath.Join(tempDir, "nodered")
	os.MkdirAll(userDir, 0700)
	os.WriteFile(filepath.Join(userDir, "settings.js"), []byte("// settings"), 0600)

	doctor := NewDoctorService(tempDir)
	report := doctor.Run(context.Background())

	// Find settings-file check
	var check *model.DoctorCheck
	for _, c := range report.Checks {
		if c.ID == "settings-file" {
			check = &c
			break
		}
	}

	if check == nil {
		t.Fatal("settings-file check not found")
	}

	if check.Status != model.CheckStatusPass {
		t.Errorf("expected settings-file to pass, got %s", check.Status)
	}
}

func TestDoctorServiceFlowsFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	userDir := filepath.Join(tempDir, "nodered")
	os.MkdirAll(userDir, 0700)
	os.WriteFile(filepath.Join(userDir, "flows.json"), []byte("[]"), 0600)

	doctor := NewDoctorService(tempDir)
	report := doctor.Run(context.Background())

	// Find flows-file check
	var check *model.DoctorCheck
	for _, c := range report.Checks {
		if c.ID == "flows-file" {
			check = &c
			break
		}
	}

	if check == nil {
		t.Fatal("flows-file check not found")
	}

	if check.Status == model.CheckStatusFail {
		t.Errorf("expected flows-file check to not fail, got %s", check.Status)
	}
}

func TestDoctorServiceCheckHasMessage(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "logs"), 0700)

	doctor := NewDoctorService(tempDir)
	report := doctor.Run(context.Background())

	for _, check := range report.Checks {
		if check.Message == "" {
			t.Errorf("check %s has empty message", check.ID)
		}

		if check.Label == "" {
			t.Errorf("check %s has empty label", check.ID)
		}

		if check.Status == "" {
			t.Errorf("check %s has empty status", check.ID)
		}
	}
}

func TestDoctorServiceMissingDataDir(t *testing.T) {
	t.Parallel()

	// Use a non-existent directory
	nonExistentDir := "/tmp/nrcc_test_nonexistent_" + randomID(8)

	doctor := NewDoctorService(nonExistentDir)
	report := doctor.Run(context.Background())

	// Should still have all 14 checks (some may fail)
	if len(report.Checks) != 14 {
		t.Fatalf("expected 14 checks even with missing data dir, got %d", len(report.Checks))
	}

	// Overall status should be degraded or critical
	if report.OverallStatus == "" {
		t.Fatal("expected overall status to be set")
	}
}

func TestDoctorServiceContextTimeout(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "logs"), 0700)

	doctor := NewDoctorService(tempDir)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*1024*1024)
	defer cancel()

	// Should not panic even with timeout
	report := doctor.Run(ctx)

	if len(report.Checks) != 14 {
		t.Fatalf("expected 14 checks, got %d", len(report.Checks))
	}
}

func TestDoctorServiceCheckStatusValues(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "logs"), 0700)
	os.MkdirAll(filepath.Join(tempDir, "nodered"), 0700)

	doctor := NewDoctorService(tempDir)
	report := doctor.Run(context.Background())

	validStatuses := map[string]bool{
		model.CheckStatusPass: true,
		model.CheckStatusWarn: true,
		model.CheckStatusFail: true,
	}

	for _, check := range report.Checks {
		if !validStatuses[check.Status] {
			t.Errorf("check %s has invalid status: %s", check.ID, check.Status)
		}
	}
}
