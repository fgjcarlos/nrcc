package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// snapshotStamp is the pinned timestamp the snapshot tests use. Every
// snapshot created with it lands in the same second, which is exactly the
// collision the issue reports.
var snapshotStamp = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

// TestSettingsSnapshotName_StaysDiscoverable proves the new snapshot
// names keep the established discovery shape, so the tooling that finds
// the snapshots (the settings-*.js.bak glob, the *.js.bak glob and
// latestBackupFile) keeps finding them (issue #883, acceptance criterion
// "keep retention/cleanup compatible with the new naming scheme").
func TestSettingsSnapshotName_StaysDiscoverable(t *testing.T) {
	name, err := settingsSnapshotName(snapshotStamp)
	if err != nil {
		t.Fatalf("settingsSnapshotName: %v", err)
	}
	if !strings.HasPrefix(name, "settings-20261014-120000-") {
		t.Errorf("snapshot name = %q, want the settings-<timestamp>- prefix", name)
	}
	if !strings.HasSuffix(name, ".js.bak") {
		t.Errorf("snapshot name = %q, want the .js.bak suffix", name)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("snapshot"), 0o600); err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	for _, pattern := range []string{"settings-*.js.bak", "*.js.bak"} {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			t.Fatalf("glob %s: %v", pattern, err)
		}
		if len(matches) != 1 {
			t.Errorf("glob %s matched %d files, want 1 (%v)", pattern, len(matches), matches)
		}
	}

	// The host service picks the newest snapshot by modification time; the
	// new name must not break that lookup.
	newest, err := latestBackupFile(dir)
	if err != nil {
		t.Fatalf("latestBackupFile: %v", err)
	}
	if newest != filepath.Join(dir, name) {
		t.Errorf("latestBackupFile = %q, want %q", newest, filepath.Join(dir, name))
	}
}

// TestSettingsSnapshotName_DrawsADifferentToken proves the random token
// is what makes two snapshots of the same second distinguishable.
func TestSettingsSnapshotName_DrawsADifferentToken(t *testing.T) {
	seen := make(map[string]struct{}, 16)
	for i := 0; i < 16; i++ {
		name, err := settingsSnapshotName(snapshotStamp)
		if err != nil {
			t.Fatalf("settingsSnapshotName: %v", err)
		}
		if _, duplicate := seen[name]; duplicate {
			t.Fatalf("settingsSnapshotName drew the same name twice: %s", name)
		}
		seen[name] = struct{}{}
	}
}

// TestWriteSettingsSnapshot_KeepsDistinctNamesForTheSameSecond proves
// two snapshots created within the same second land in two files, each
// holding its own bytes.
func TestWriteSettingsSnapshot_KeepsDistinctNamesForTheSameSecond(t *testing.T) {
	dir := t.TempDir()

	first, err := writeSettingsSnapshot(dir, snapshotStamp, []byte("first"))
	if err != nil {
		t.Fatalf("first snapshot: %v", err)
	}
	second, err := writeSettingsSnapshot(dir, snapshotStamp, []byte("second"))
	if err != nil {
		t.Fatalf("second snapshot: %v", err)
	}
	if first == second {
		t.Fatalf("two snapshots of the same second share the path %s", first)
	}

	for path, want := range map[string]string{first: "first", second: "second"} {
		// #nosec G304 -- path is the snapshot path the writer returned under the test's temp dir.
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read snapshot %s: %v", path, err)
		}
		if string(data) != want {
			t.Errorf("snapshot %s holds %q, want %q", path, data, want)
		}
	}
}

// TestWriteSettingsSnapshot_ExistingFileIsNeverOverwritten proves the
// exclusive creation: an existing snapshot keeps its bytes and the writer
// reports the conflict instead of clobbering it.
func TestWriteSettingsSnapshot_ExistingFileIsNeverOverwritten(t *testing.T) {
	dir := t.TempDir()
	pinned := "settings-20261014-120000-pinned.js.bak"
	target := filepath.Join(dir, pinned)
	if err := os.WriteFile(target, []byte("original"), 0o600); err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}

	name := func(time.Time) (string, error) { return pinned, nil }
	_, err := writeSettingsSnapshotNamed(dir, snapshotStamp, []byte("overwritten"), name)
	if err == nil {
		t.Fatal("expected the writer to refuse an existing snapshot")
	}
	if !errors.Is(err, os.ErrExist) {
		t.Errorf("err = %v, want an os.ErrExist in the chain", err)
	}

	// #nosec G304 -- target is the pinned snapshot path under the test's temp dir.
	data, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read snapshot: %v", readErr)
	}
	if string(data) != "original" {
		t.Errorf("the existing snapshot was rewritten: got %q, want %q", data, "original")
	}
}

// TestWriteSettingsSnapshot_ReportsCreationFailure proves an unusable
// backup directory is reported instead of silently skipped.
func TestWriteSettingsSnapshot_ReportsCreationFailure(t *testing.T) {
	dir := t.TempDir()
	// A regular file where the backup directory belongs makes the
	// directory creation fail with ENOTDIR.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("not a dir"), 0o600); err != nil {
		t.Fatalf("blocker: %v", err)
	}

	_, err := writeSettingsSnapshot(filepath.Join(blocker, "backups"), snapshotStamp, []byte("payload"))
	if err == nil {
		t.Fatal("expected the writer to fail on an unusable backup directory")
	}
	if !strings.Contains(err.Error(), "create backup dir") {
		t.Errorf("err = %v, want it to mention the backup directory", err)
	}
}

// TestWriteSettingsSnapshot_RejectsAnEmptyDirectory proves the writer
// refuses an empty backup directory instead of writing into the current
// working directory.
func TestWriteSettingsSnapshot_RejectsAnEmptyDirectory(t *testing.T) {
	_, err := writeSettingsSnapshot("", snapshotStamp, []byte("payload"))
	if err == nil {
		t.Fatal("expected the writer to reject an empty backup directory")
	}
	if !strings.Contains(err.Error(), "backup directory is empty") {
		t.Errorf("err = %v, want it to mention the empty backup directory", err)
	}
}
