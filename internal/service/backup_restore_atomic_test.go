package service

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// writeAtomicRestoreArchive hand-crafts a current-version archive whose payload
// is bodyByPath, with matching manifest checksums.
func writeAtomicRestoreArchive(t *testing.T, backupDir, id string, bodyByPath map[string]string) {
	t.Helper()
	writeAtomicRestoreArchiveVersion(t, backupDir, id, currentManifestVersion, bodyByPath)
}

func writeAtomicRestoreArchiveVersion(t *testing.T, backupDir, id string, version int, bodyByPath map[string]string) {
	t.Helper()
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		t.Fatalf("mkdir backup dir: %v", err)
	}
	paths := make([]string, 0, len(bodyByPath))
	for path := range bodyByPath {
		paths = append(paths, path)
	}
	// Deterministic order keeps the zip layout stable across runs.
	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			if paths[j] < paths[i] {
				paths[i], paths[j] = paths[j], paths[i]
			}
		}
	}

	manifest := backupMetadata{
		Version:   version,
		Algorithm: defaultChecksumAlgo,
		ID:        id,
		Name:      id,
		Type:      model.BackupTypeManual,
		CreatedAt: "2026-01-01T00:00:00Z",
	}
	for _, path := range paths {
		body := bodyByPath[path]
		manifest.Files = append(manifest.Files, model.BackupFileEntry{
			Path:     path,
			Size:     int64(len(body)),
			Checksum: sha256Hex(t, body),
		})
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	meta, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	mw, err := zw.Create("backup-metadata.json")
	if err != nil {
		t.Fatalf("create manifest entry: %v", err)
	}
	if _, err := mw.Write(meta); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	for _, path := range paths {
		w, err := zw.Create(path)
		if err != nil {
			t.Fatalf("create entry %s: %v", path, err)
		}
		if _, err := w.Write([]byte(bodyByPath[path])); err != nil {
			t.Fatalf("write entry %s: %v", path, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, id+".zip"), buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write archive: %v", err)
	}
}

// stageAndSwap starts a restore and dies in the middle of the rename loop,
// exactly like a process killed while it publishes the staged tree.
func stageAndSwap(t *testing.T, svc *BackupService, id string) {
	t.Helper()
	backupPath := filepath.Join(svc.backupDir, id+".zip")
	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatalf("stat archive: %v", err)
	}
	// #nosec G304 -- backupPath is built from t.TempDir() + a test constant.
	file, err := os.Open(backupPath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer func() { _ = file.Close() }()
	zipReader, err := zip.NewReader(file, info.Size())
	if err != nil {
		t.Fatalf("parse archive: %v", err)
	}
	manifest, err := verifyArchiveManifest(zipReader)
	if err != nil {
		t.Fatalf("verify manifest: %v", err)
	}
	stagingDir, err := svc.stageRestore(zipReader, manifest)
	if err != nil {
		t.Fatalf("stage restore: %v", err)
	}
	// Die after the first published file: old and new sets are now mixed.
	previous := restoreSwapHook
	restoreSwapHook = func(string) error { return errors.New("simulated process death") }
	defer func() { restoreSwapHook = previous }()
	if err := svc.swapStagingIntoDataDir(stagingDir, manifest); err == nil {
		t.Fatal("swapStagingIntoDataDir returned nil, want the simulated interruption")
	}
}

func atomicRestoreDataDir(t *testing.T) string {
	t.Helper()
	dataDir := t.TempDir()
	writeTestFile(t, filepath.Join(dataDir, "flows.json"), `[{"id":"OLD-FLOWS"}]`)
	writeTestFile(t, filepath.Join(dataDir, "settings.js"), `module.exports = {OLD:true};`)
	writeTestFile(t, filepath.Join(dataDir, "config.json"), `{"OLD":true}`)
	return dataDir
}

func assertRestoredSet(t *testing.T, dataDir string) {
	t.Helper()
	want := map[string]string{
		"flows.json":  `[{"id":"OLD-FLOWS"}]`,
		"settings.js": `module.exports = {OLD:true};`,
		"config.json": `{"OLD":true}`,
	}
	for name, wantBody := range want {
		// #nosec G304 -- dataDir is t.TempDir().
		got, err := os.ReadFile(filepath.Join(dataDir, name))
		if err != nil {
			t.Fatalf("read %s after recovery: %v", name, err)
		}
		if string(got) != wantBody {
			t.Fatalf("%s = %q after recovery, want %q", name, got, wantBody)
		}
	}
	// A file the archive introduced but the old set never had must be gone.
	if _, err := os.Stat(filepath.Join(dataDir, "cc-users.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cc-users.json survived the rollback (stat err=%v)", err)
	}
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatalf("read dataDir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), restoreStagingPrefix) {
			t.Fatalf("staging directory %s survived the rollback", entry.Name())
		}
	}
}

// TestRestoreInterruptedSwapRecoversOldSet is the acceptance test for S1/S2:
// a restore that dies inside the rename loop ends, after a restart, with the
// full old set — never a mix of old and new files.
func TestRestoreInterruptedSwapRecoversOldSet(t *testing.T) {
	dataDir := atomicRestoreDataDir(t)
	backupDir := filepath.Join(t.TempDir(), "backups")
	svc := NewBackupServiceWithBackupDir(dataDir, backupDir)
	writeAtomicRestoreArchive(t, backupDir, "mix-1", map[string]string{
		"flows.json":    `[{"id":"NEW-FLOWS"}]`,
		"settings.js":   `module.exports = {NEW:true};`,
		"config.json":   `{"NEW":true}`,
		"cc-users.json": `[]`,
	})

	stageAndSwap(t, svc, "mix-1")

	recovered, err := RecoverInterruptedRestore(dataDir)
	if err != nil {
		t.Fatalf("RecoverInterruptedRestore: %v", err)
	}
	if recovered == "" {
		t.Fatal("RecoverInterruptedRestore reported no interrupted restore")
	}
	assertRestoredSet(t, dataDir)
}

// TestRecoverInterruptedRestoreIsIdempotent proves a crash during the
// rollback itself cannot delete an already-restored file on the next boot.
func TestRecoverInterruptedRestoreIsIdempotent(t *testing.T) {
	dataDir := atomicRestoreDataDir(t)
	backupDir := filepath.Join(t.TempDir(), "backups")
	svc := NewBackupServiceWithBackupDir(dataDir, backupDir)
	writeAtomicRestoreArchive(t, backupDir, "mix-2", map[string]string{
		"flows.json":  `[{"id":"NEW-FLOWS"}]`,
		"settings.js": `module.exports = {NEW:true};`,
	})
	stageAndSwap(t, svc, "mix-2")

	if _, err := RecoverInterruptedRestore(dataDir); err != nil {
		t.Fatalf("first recovery: %v", err)
	}
	second, err := RecoverInterruptedRestore(dataDir)
	if err != nil {
		t.Fatalf("second recovery: %v", err)
	}
	if second != "" {
		t.Fatalf("second recovery claimed work to do: %s", second)
	}
	assertRestoredSet(t, dataDir)
}

// TestRollbackAfterJournalOnlyCrashKeepsOldSet pins the swap ordering: the
// journal line is durable before anything in dataDir moves, so a crash right
// after the append must still leave the old set intact.
func TestRollbackAfterJournalOnlyCrashKeepsOldSet(t *testing.T) {
	dataDir := atomicRestoreDataDir(t)
	stagingDir := filepath.Join(dataDir, restoreStagingPrefix+"journal-only")
	if err := os.MkdirAll(stagingDir, 0o750); err != nil {
		t.Fatalf("mkdir staging: %v", err)
	}
	journal := "keep\tflows.json\nnew\tcc-users.json\n"
	if err := os.WriteFile(filepath.Join(stagingDir, restoreJournalName), []byte(journal), 0o600); err != nil { // #nosec G306 -- test fixture.
		t.Fatalf("write journal: %v", err)
	}
	// No .replaced file and nothing published: the crash landed between the
	// durable append and the renames.
	recovered, err := RecoverInterruptedRestore(dataDir)
	if err != nil {
		t.Fatalf("RecoverInterruptedRestore: %v", err)
	}
	if recovered == "" {
		t.Fatal("recovery ignored the journal")
	}
	assertRestoredSet(t, dataDir)
}

// TestRestoreSwapFailureRollsBackImmediately proves a swap that fails without
// a crash (e.g. a full disk) leaves the old set in place in the same run.
func TestRestoreSwapFailureRollsBackImmediately(t *testing.T) {
	dataDir := atomicRestoreDataDir(t)
	backupDir := filepath.Join(t.TempDir(), "backups")
	svc := NewBackupServiceWithBackupDir(dataDir, backupDir)
	writeAtomicRestoreArchive(t, backupDir, "mix-3", map[string]string{
		"flows.json":  `[{"id":"NEW-FLOWS"}]`,
		"settings.js": `module.exports = {NEW:true};`,
	})

	previous := restoreSwapHook
	calls := 0
	restoreSwapHook = func(string) error {
		calls++
		if calls == 2 {
			return errors.New("simulated rename failure")
		}
		return nil
	}
	defer func() { restoreSwapHook = previous }()

	if err := svc.Restore("mix-3"); err == nil {
		t.Fatal("Restore returned nil, want the simulated rename failure")
	}
	// The failed attempt is already undone; no boot recovery is needed.
	recovered, err := RecoverInterruptedRestore(dataDir)
	if err != nil {
		t.Fatalf("RecoverInterruptedRestore: %v", err)
	}
	if recovered != "" {
		t.Fatalf("failed restore left a journal behind: %s", recovered)
	}
	assertRestoredSet(t, dataDir)
}
