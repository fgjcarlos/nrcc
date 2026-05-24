package metrics

import (
	"net/http"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestNewCollector_ReturnsNonNil verifies that NewCollector returns a non-nil *MetricsCollector.
func TestNewCollector_ReturnsNonNil(t *testing.T) {
	mc := NewCollector()
	if mc == nil {
		t.Fatal("NewCollector() returned nil, expected non-nil *MetricsCollector")
	}
}

// TestCollector_Handler_ReturnsNonNil verifies that Handler() returns a non-nil http.Handler.
func TestCollector_Handler_ReturnsNonNil(t *testing.T) {
	mc := NewCollector()
	h := mc.Handler()
	if h == nil {
		t.Fatal("Handler() returned nil, expected non-nil http.Handler")
	}
	// Verify the handler implements the http.Handler interface.
	var _ http.Handler = h
}

// TestRecordLoginAttempt verifies that RecordLoginAttempt increments the correct label.
func TestRecordLoginAttempt(t *testing.T) {
	tests := []struct {
		name    string
		success bool
		want    string
	}{
		{
			name:    "success increments success label",
			success: true,
			want: `
# HELP nrcc_login_attempts_total Total number of login attempts, labeled by result (success or failure).
# TYPE nrcc_login_attempts_total counter
nrcc_login_attempts_total{result="success"} 1
`,
		},
		{
			name:    "failure increments failure label",
			success: false,
			want: `
# HELP nrcc_login_attempts_total Total number of login attempts, labeled by result (success or failure).
# TYPE nrcc_login_attempts_total counter
nrcc_login_attempts_total{result="failure"} 1
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mc := NewCollector()
			mc.RecordLoginAttempt(tc.success)

			if err := testutil.CollectAndCompare(mc.LoginAttempts, strings.NewReader(tc.want)); err != nil {
				t.Errorf("unexpected metric output: %v", err)
			}
		})
	}
}

// TestRecordBackupCreated verifies that RecordBackupCreated increments the correct type label.
func TestRecordBackupCreated(t *testing.T) {
	tests := []struct {
		name       string
		backupType string
		want       string
	}{
		{
			name:       "manual backup increments manual label",
			backupType: "manual",
			want: `
# HELP nrcc_backup_created_total Total number of backups created, labeled by type (manual, auto, pre_restore).
# TYPE nrcc_backup_created_total counter
nrcc_backup_created_total{type="manual"} 1
`,
		},
		{
			name:       "auto backup increments auto label",
			backupType: "auto",
			want: `
# HELP nrcc_backup_created_total Total number of backups created, labeled by type (manual, auto, pre_restore).
# TYPE nrcc_backup_created_total counter
nrcc_backup_created_total{type="auto"} 1
`,
		},
		{
			name:       "pre_restore backup increments pre_restore label",
			backupType: "pre_restore",
			want: `
# HELP nrcc_backup_created_total Total number of backups created, labeled by type (manual, auto, pre_restore).
# TYPE nrcc_backup_created_total counter
nrcc_backup_created_total{type="pre_restore"} 1
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mc := NewCollector()
			mc.RecordBackupCreated(tc.backupType)

			if err := testutil.CollectAndCompare(mc.BackupCreated, strings.NewReader(tc.want)); err != nil {
				t.Errorf("unexpected metric output: %v", err)
			}
		})
	}
}

// TestRecordRestoreAttempt verifies that RecordRestoreAttempt increments the correct label.
func TestRecordRestoreAttempt(t *testing.T) {
	tests := []struct {
		name    string
		success bool
		want    string
	}{
		{
			name:    "success increments success label",
			success: true,
			want: `
# HELP nrcc_restore_attempts_total Total number of restore attempts, labeled by result (success or failure).
# TYPE nrcc_restore_attempts_total counter
nrcc_restore_attempts_total{result="success"} 1
`,
		},
		{
			name:    "failure increments failure label",
			success: false,
			want: `
# HELP nrcc_restore_attempts_total Total number of restore attempts, labeled by result (success or failure).
# TYPE nrcc_restore_attempts_total counter
nrcc_restore_attempts_total{result="failure"} 1
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mc := NewCollector()
			mc.RecordRestoreAttempt(tc.success)

			if err := testutil.CollectAndCompare(mc.RestoreAttempts, strings.NewReader(tc.want)); err != nil {
				t.Errorf("unexpected metric output: %v", err)
			}
		})
	}
}

// TestRecordUpdateAttempt verifies that RecordUpdateAttempt increments the correct label.
func TestRecordUpdateAttempt(t *testing.T) {
	tests := []struct {
		name    string
		success bool
		want    string
	}{
		{
			name:    "success increments success label",
			success: true,
			want: `
# HELP nrcc_update_attempts_total Total number of update attempts, labeled by result (success or failure).
# TYPE nrcc_update_attempts_total counter
nrcc_update_attempts_total{result="success"} 1
`,
		},
		{
			name:    "failure increments failure label",
			success: false,
			want: `
# HELP nrcc_update_attempts_total Total number of update attempts, labeled by result (success or failure).
# TYPE nrcc_update_attempts_total counter
nrcc_update_attempts_total{result="failure"} 1
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mc := NewCollector()
			mc.RecordUpdateAttempt(tc.success)

			if err := testutil.CollectAndCompare(mc.UpdateAttempts, strings.NewReader(tc.want)); err != nil {
				t.Errorf("unexpected metric output: %v", err)
			}
		})
	}
}

// TestRecordLibraryOperation verifies that RecordLibraryOperation increments the correct labels.
func TestRecordLibraryOperation(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		success   bool
		want      string
	}{
		{
			name:      "install success",
			operation: "install",
			success:   true,
			want: `
# HELP nrcc_library_operations_total Total number of library operations, labeled by operation (install, uninstall) and result (success, failure).
# TYPE nrcc_library_operations_total counter
nrcc_library_operations_total{operation="install",result="success"} 1
`,
		},
		{
			name:      "uninstall failure",
			operation: "uninstall",
			success:   false,
			want: `
# HELP nrcc_library_operations_total Total number of library operations, labeled by operation (install, uninstall) and result (success, failure).
# TYPE nrcc_library_operations_total counter
nrcc_library_operations_total{operation="uninstall",result="failure"} 1
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mc := NewCollector()
			mc.RecordLibraryOperation(tc.operation, tc.success)

			if err := testutil.CollectAndCompare(mc.LibraryOps, strings.NewReader(tc.want)); err != nil {
				t.Errorf("unexpected metric output: %v", err)
			}
		})
	}
}
