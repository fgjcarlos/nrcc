package service

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestBackupService_RejectsTraversalID is the #278 regression: every id-taking
// method must reject identifiers that could escape the backups directory.
func TestBackupService_RejectsTraversalID(t *testing.T) {
	svc := NewBackupService(t.TempDir())

	badIDs := []string{"../config", "../../etc/passwd", "foo/bar", "..", ""}
	for _, id := range badIDs {
		if _, err := svc.Detail(id); err == nil {
			t.Errorf("Detail(%q): expected error", id)
		}
		if err := svc.Restore(id); err == nil {
			t.Errorf("Restore(%q): expected error", id)
		}
		if _, err := svc.RestoreWithSafetyBackup(id); err == nil {
			t.Errorf("RestoreWithSafetyBackup(%q): expected error", id)
		}
		if err := svc.Delete(id); err == nil {
			t.Errorf("Delete(%q): expected error", id)
		}
		if err := svc.Download(id, io.Discard); err == nil {
			t.Errorf("Download(%q): expected error", id)
		}
	}
}

// TestBackupService_DeleteCannotEscapeBackupsDir proves the traversal is real:
// "../victim" resolves to <dataDir>/victim.zip under the naive filepath.Join,
// so a buggy Delete would remove a file outside the backups directory.
func TestBackupService_DeleteCannotEscapeBackupsDir(t *testing.T) {
	dataDir := t.TempDir()
	svc := NewBackupService(dataDir)

	victim := filepath.Join(dataDir, "victim.zip")
	if err := os.WriteFile(victim, []byte("sensitive"), 0o600); err != nil {
		t.Fatalf("seed victim: %v", err)
	}

	if err := svc.Delete("../victim"); err == nil {
		t.Error("Delete must reject a traversal id")
	}
	if _, err := os.Stat(victim); err != nil {
		t.Fatalf("traversal Delete removed a file outside the backups dir: %v", err)
	}
}

// TestValidateBackupID_AllowsLegitimateIDs guards against over-tightening:
// server-generated ids (uuids, typed names) must pass validation.
func TestValidateBackupID_AllowsLegitimateIDs(t *testing.T) {
	good := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"manual-1",
		"backup_1700000000",
	}
	for _, id := range good {
		if err := ValidateBackupID(id); err != nil {
			t.Errorf("ValidateBackupID(%q) should pass, got %v", id, err)
		}
	}
}
