package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// TestRestoreAndApplyCannotOverlap is the acceptance test for S4: one data
// dir runs at most one restore, backup or apply at a time.
func TestRestoreAndApplyCannotOverlap(t *testing.T) {
	dataDir := t.TempDir()
	writeTestFile(t, filepath.Join(dataDir, "flows.json"), `[{"id":"1"}]`)
	svc := NewBackupService(dataDir)
	backup, err := svc.CreateTyped(model.BackupTypeManual, "snap")
	if err != nil {
		t.Fatalf("CreateTyped: %v", err)
	}

	settingsPath := filepath.Join(dataDir, "settings.js")
	writeTestFile(t, settingsPath, validSettingsJS(t))
	coordinator := NewApplyCoordinator(NewApplyService(NewConfigService(dataDir), nil))

	release, err := acquireDataDirLock(dataDir)
	if err != nil {
		t.Fatalf("acquireDataDirLock: %v", err)
	}

	if err := svc.Restore(backup.ID); !errors.Is(err, ErrDataDirBusy) {
		t.Fatalf("Restore during another operation = %v, want ErrDataDirBusy", err)
	}
	if _, err := svc.CreateTyped(model.BackupTypeAuto, ""); !errors.Is(err, ErrDataDirBusy) {
		t.Fatalf("scheduled backup during another operation = %v, want ErrDataDirBusy", err)
	}
	_, applyErr := coordinator.Apply(context.Background(), ApplyRequest{Path: settingsPath})
	if !errors.Is(applyErr, ErrApplyInFlight) {
		t.Fatalf("apply during another operation = %v, want ErrApplyInFlight", applyErr)
	}

	release()

	// The lock is exclusive, not sticky: both paths work again once it is free.
	if err := svc.Restore(backup.ID); err != nil {
		t.Fatalf("Restore after release: %v", err)
	}
	if _, err := svc.CreateTyped(model.BackupTypeManual, "after"); err != nil {
		t.Fatalf("CreateTyped after release: %v", err)
	}
	spec := ApplyRequest{Path: settingsPath, Content: validSettingsJS(t), BackupDir: filepath.Join(dataDir, "backups", "settings")}
	if _, err := coordinator.Apply(context.Background(), spec); err != nil {
		t.Fatalf("apply after release: %v", err)
	}
}

// TestSafetyBackupRestoreDoesNotDeadlockOnItsOwnLock proves the restore entry
// point takes the lock exactly once while it creates its pre-restore
// snapshot and swaps the files.
func TestSafetyBackupRestoreDoesNotDeadlockOnItsOwnLock(t *testing.T) {
	dataDir := t.TempDir()
	writeTestFile(t, filepath.Join(dataDir, "flows.json"), `[{"id":"1"}]`)
	svc := NewBackupService(dataDir)
	backup, err := svc.CreateTyped(model.BackupTypeManual, "snap")
	if err != nil {
		t.Fatalf("CreateTyped: %v", err)
	}
	if _, err := svc.RestoreWithSafetyBackup(backup.ID); err != nil {
		t.Fatalf("RestoreWithSafetyBackup: %v", err)
	}
}

// TestDataDirLockIsReentrantFree documents the contract the restore entry
// points rely on: the lock is held per call, never nested.
func TestDataDirLockIsReentrantFree(t *testing.T) {
	dataDir := t.TempDir()
	release, err := acquireDataDirLock(dataDir)
	if err != nil {
		t.Fatalf("acquireDataDirLock: %v", err)
	}
	if _, err := acquireDataDirLock(dataDir); !errors.Is(err, ErrDataDirBusy) {
		t.Fatalf("second acquire = %v, want ErrDataDirBusy", err)
	}
	release()
	if release2, err := acquireDataDirLock(dataDir); err != nil {
		t.Fatalf("acquire after release: %v", err)
	} else {
		release2()
	}
}
