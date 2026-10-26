// Package service — the settings snapshot writer.
//
// Every mutation of settings.js keeps a recovery snapshot of the state it
// replaced. The snapshot is the rollback target: when a post-write stage
// fails, the pipeline restores the bytes the snapshot captured (see
// apply_rollback.go).
//
// Snapshot contract
// =================
//
//  1. Every transaction keeps its own snapshot. Two applies that land inside
//     the same second must not share a file: the second one would overwrite
//     the first transaction's recovery point, and the operator would lose
//     the state that existed before the first change (issue #883).
//
//  2. The name is collision-resistant. It keeps the UTC timestamp — so the
//     directory listing stays readable and sorts chronologically — and adds a
//     random token, so two snapshots created within the same second never
//     share a name.
//
//  3. The write is exclusive. The file is created with O_EXCL, so an existing
//     snapshot is never overwritten, not even partially. When the drawn name
//     is already taken the writer draws a new one and tries again; it never
//     falls back to an overwrite-capable write.
//
//  4. The bytes are durable. The snapshot is fsync'ed before the caller
//     continues, so a crash cannot leave a truncated recovery point.
//
// The writer is shared by the apply pipeline (apply.go) and the structured
// save (config.go): both keep their snapshots in the same backups/settings
// directory, so they must not be able to clobber each other.

package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// settingsSnapshotAttempts is the number of names the writer draws
	// before it gives up. Every attempt draws a new random token, so the
	// attempts only fail when the directory itself refuses the file.
	settingsSnapshotAttempts = 8
	// settingsSnapshotTokenBytes is the size of the random token that
	// makes a snapshot name collision-resistant.
	settingsSnapshotTokenBytes = 6
)

// settingsSnapshotName builds the file name of a settings snapshot.
//
// The name keeps the established shape — "settings-<timestamp>.js.bak" — so
// the discovery tooling keeps finding the snapshots: the settings-*.js.bak
// glob, the *.js.bak glob and latestBackupFile (which reads the
// modification time, not the name). The random token between the timestamp
// and the extension is what makes two snapshots of the same second
// distinguishable.
func settingsSnapshotName(stamp time.Time) (string, error) {
	token, err := randomHex(settingsSnapshotTokenBytes)
	if err != nil {
		return "", fmt.Errorf("generate snapshot token: %w", err)
	}
	return fmt.Sprintf("settings-%s-%s.js.bak", stamp.UTC().Format("20060102-150405"), token), nil
}

// writeSettingsSnapshot writes data into a new snapshot inside dir and
// returns the path of the created file.
//
// The snapshot is created exclusively: an existing file is never
// overwritten. When the drawn name is already taken the writer draws a new
// one and tries again. A failure to create the snapshot is returned to the
// caller, which rejects the transaction instead of writing the live file
// without a recovery point.
func writeSettingsSnapshot(dir string, stamp time.Time, data []byte) (string, error) {
	return writeSettingsSnapshotNamed(dir, stamp, data, settingsSnapshotName)
}

// writeSettingsSnapshotNamed is writeSettingsSnapshot with an injectable
// name generator. The seam lets the tests pin a name and exercise the
// "the name is already taken" path deterministically.
func writeSettingsSnapshotNamed(dir string, stamp time.Time, data []byte, name func(time.Time) (string, error)) (string, error) {
	if dir == "" {
		return "", errors.New("create settings snapshot: backup directory is empty")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("create backup dir %s: %w", dir, err)
	}
	var lastErr error
	for attempt := 0; attempt < settingsSnapshotAttempts; attempt++ {
		fileName, err := name(stamp)
		if err != nil {
			return "", err
		}
		target := filepath.Join(dir, fileName)
		if err := createSnapshotFile(target, data); err != nil {
			if !errors.Is(err, os.ErrExist) {
				return "", err
			}
			// The name is taken: draw a new one. The existing snapshot
			// keeps its bytes — it is never rewritten.
			lastErr = err
			continue
		}
		return target, nil
	}
	return "", fmt.Errorf("create settings snapshot in %s after %d attempts: %w", dir, settingsSnapshotAttempts, lastErr)
}

// createSnapshotFile creates target exclusively and writes data into it.
//
// O_EXCL makes the creation fail when the file already exists, so an
// existing snapshot keeps its bytes. The file is fsync'ed before it is
// closed and the directory entry is fsync'ed afterwards, so the snapshot is
// durable before the caller continues.
func createSnapshotFile(target string, data []byte) error {
	return writeFileExclusive(target, data, settingsFileMode)
}

// writeFileExclusive creates path with O_EXCL and writes data into it.
//
// The creation fails with os.ErrExist when the file already exists, so an
// existing file is never overwritten, not even partially. The file is
// fsync'ed before it is closed and the directory entry is fsync'ed
// afterwards, so the content is durable before the caller continues.
func writeFileExclusive(path string, data []byte, mode os.FileMode) error {
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	flags = flagsWithNoFollow(flags)
	// #nosec G304 -- path is built from a validated directory plus a generated file name; not request-derived.
	f, err := os.OpenFile(path, flags, mode)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return fmt.Errorf("fsync %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close %s: %w", path, err)
	}
	if err := syncParentDir(filepath.Dir(path)); err != nil {
		// The file itself is durable; only the directory entry is
		// best-effort. The file is kept: dropping a valid recovery
		// point over a directory fsync would be the worse outcome.
		return fmt.Errorf("fsync dir %s: %w", filepath.Dir(path), err)
	}
	return nil
}
