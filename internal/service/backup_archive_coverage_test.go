package service

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// TestBackupCoversEverythingNRCCOwns is the acceptance test for S3/S6: a
// backup restored onto a clean volume brings back the environment file, the
// MFA store, uploads and the runtime config alongside the Node-RED files.
func TestBackupCoversEverythingNRCCOwns(t *testing.T) {
	dataDir := t.TempDir()
	owned := map[string]string{
		".env":                 "NODE_RED_ADMIN=1\n",
		"mfa.json":             `{"users":[]}`,
		".config.runtime.json": `{"port":1880}`,
		"uploads/logo.png":     "PNGDATA",
		"uploads/sub/deep.txt": "DEEP",
		"flows.json":           `[{"id":"1"}]`,
	}
	for name, body := range owned {
		writeTestFile(t, filepath.Join(dataDir, name), body)
	}

	svc := NewBackupService(dataDir)
	backup, err := svc.CreateTyped(model.BackupTypeManual, "everything")
	if err != nil {
		t.Fatalf("CreateTyped: %v", err)
	}

	// Wipe the volume: every owned file is gone.
	for name := range owned {
		if err := os.Remove(filepath.Join(dataDir, name)); err != nil {
			t.Fatalf("remove %s: %v", name, err)
		}
	}

	if err := svc.Restore(backup.ID); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	for name, want := range owned {
		// #nosec G304 -- dataDir is t.TempDir().
		got, err := os.ReadFile(filepath.Join(dataDir, name))
		if err != nil {
			t.Fatalf("read restored %s: %v", name, err)
		}
		if string(got) != want {
			t.Fatalf("restored %s = %q, want %q", name, got, want)
		}
	}
}

// TestBackupManifestVersionIsCurrent pins the archive format: the missing
// files changed what a v1 archive means, so the writer advertises v2.
func TestBackupManifestVersionIsCurrent(t *testing.T) {
	dataDir := t.TempDir()
	writeTestFile(t, filepath.Join(dataDir, "flows.json"), `[{"id":"1"}]`)
	svc := NewBackupService(dataDir)
	backup, err := svc.CreateTyped(model.BackupTypeManual, "version")
	if err != nil {
		t.Fatalf("CreateTyped: %v", err)
	}
	// #nosec G304 -- backup.Path comes from t.TempDir().
	file, err := os.Open(backup.Path)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		t.Fatalf("stat archive: %v", err)
	}
	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		t.Fatalf("parse archive: %v", err)
	}
	manifest, err := verifyArchiveManifest(reader)
	if err != nil {
		t.Fatalf("verify manifest: %v", err)
	}
	if manifest.Version != 2 {
		t.Fatalf("manifest version = %d, want 2", manifest.Version)
	}
}

// TestRestoreAcceptsVersionOneArchive keeps backups written before the bump
// restorable.
func TestRestoreAcceptsVersionOneArchive(t *testing.T) {
	dataDir := t.TempDir()
	backupDir := filepath.Join(t.TempDir(), "backups")
	svc := NewBackupServiceWithBackupDir(dataDir, backupDir)
	writeAtomicRestoreArchiveVersion(t, backupDir, "legacy-1", 1, map[string]string{
		"flows.json": `[{"id":"LEGACY"}]`,
	})

	if err := svc.Restore("legacy-1"); err != nil {
		t.Fatalf("Restore v1 archive: %v", err)
	}
	// #nosec G304 -- dataDir is t.TempDir().
	got, err := os.ReadFile(filepath.Join(dataDir, "flows.json"))
	if err != nil {
		t.Fatalf("read flows.json: %v", err)
	}
	if string(got) != `[{"id":"LEGACY"}]` {
		t.Fatalf("flows.json = %q, want the v1 payload", got)
	}
}

// TestBackupOwnsNRCCFilesEvenWhenIncludesAreOff pins that the files NRCC
// itself manages are never behind a per-type switch (issue #1024).
func TestBackupOwnsNRCCFilesEvenWhenIncludesAreOff(t *testing.T) {
	dataDir := t.TempDir()
	writeTestFile(t, filepath.Join(dataDir, "uploads", "logo.png"), "PNGDATA")
	svc := NewBackupService(dataDir)

	files, err := svc.filesForBackup(model.BackupConfig{})
	if err != nil {
		t.Fatalf("filesForBackup: %v", err)
	}
	got := make(map[string]bool, len(files))
	for _, file := range files {
		got[file.dst] = true
	}
	for _, want := range []string{".env", "mfa.json", ".config.runtime.json", "uploads/logo.png", "flows.json", "cc-users.json"} {
		if !got[want] {
			t.Fatalf("filesForBackup omitted %s (got %v)", want, got)
		}
	}
	for _, unwanted := range []string{"settings.js", "config.json", "package.json"} {
		if got[unwanted] {
			t.Fatalf("filesForBackup included %s with every Include switch off", unwanted)
		}
	}
}

// TestRestoreRejectsUnsupportedManifestVersion keeps the reader strict about
// archive formats it does not know.
func TestRestoreRejectsUnsupportedManifestVersion(t *testing.T) {
	dataDir := atomicRestoreDataDir(t)
	backupDir := filepath.Join(t.TempDir(), "backups")
	svc := NewBackupServiceWithBackupDir(dataDir, backupDir)
	writeAtomicRestoreArchiveVersion(t, backupDir, "future-1", currentManifestVersion+1, map[string]string{
		"flows.json": `[{"id":"FUTURE"}]`,
	})

	if err := svc.Restore("future-1"); !errors.Is(err, ErrBackupCorrupt) {
		t.Fatalf("Restore of a future manifest = %v, want ErrBackupCorrupt", err)
	}
	assertRestoredSet(t, dataDir)
}
