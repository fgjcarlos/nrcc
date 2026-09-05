// Package service — slice A of issue #758 introduces an apply pipeline
// that orchestrates validate → backup → atomic write → audit for every
// settings.js mutation. The HTTP handlers and ProcessManager restart
// wiring belong to slices B and C; this file only delivers the core
// transaction.
//
// Atomic write contract
// =====================
//
// The atomic-write helper in this file replaces the un-atomic
// os.WriteFile call that used to live in ConfigService.writeSettingsFile.
// The requirements are:
//
//   1. Write content to a tmp sibling, fsync the file, then rename(2) over
//      the live path so a crash never publishes a half-written settings.js.
//
//   2. Refuse to follow a symlink substituted into the destination path
//      between the open and the rename. The choke point is the rename
//      itself: rename(2) on POSIX does NOT follow the destination's
//      symlink (it unlinks the symlink and creates a new file in its
//      place), but we additionally lstat+assert before the rename so a
//      defender who reads the contract cannot accidentally rely on
//      POSIX semantics.
//
//   3. Fsync the parent directory after the rename so the directory
//      entry change is durable across a power loss. Without that, the
//      rename can be published (file appears in directory listings)
//      before the entry hits disk.
//
//   4. Reject path-traversal (`..` components) at the boundary before
//      any I/O happens. This is a defence-in-depth check; the live
//      SettingsDocument.Path is authoritative and is normally
//      populated from host detection, not user input.
//
// Atomicity is best-effort across crashes; the durability guarantee is
// "either the previous settings.js is intact or the new one is fully on
// disk and durable". There is no third observable state.

package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// settingsFileMode is the mode applied to the freshly written settings.js.
// settings.js contains credentials and adminAuth hashes; 0600 is the
// repository-wide policy (#620).
const settingsFileMode os.FileMode = 0o600

// settingsTmpSuffix is appended to the destination path when writing the
// temporary file. The trailing dot+random token guarantees uniqueness
// across concurrent apply attempts on the same destination.
const settingsTmpSuffix = ".apply-"

// ErrSettingsPathInvalid is returned by AtomicWriteSettings when the path
// fails the boundary validation (traversal components, non-clean path,
// parent directory escape). The ApplyError wraps it with Stage=write.
var ErrSettingsPathInvalid = errors.New("settings.js path failed boundary validation")

// ErrSettingsSymlinkRejected is returned when the destination path or any
// parent component turns out to be a symlink at write time. The apply
// transaction must refuse such paths instead of silently following them
// to an attacker-controlled location.
var ErrSettingsSymlinkRejected = errors.New("settings.js destination resolves to a symlink")

// AtomicWriteSettings atomically replaces settings.js at path with content.
//
// On POSIX the implementation:
//
//   1. Validates path for traversal/cleanliness.
//   2. mkdir -p the parent directory (idempotent; safe across Apply calls).
//   3. Opens a tmp sibling with O_WRONLY|O_CREATE|O_TRUNC, writes content,
//      fsyncs, closes.
//   4. Asserts the destination is not a symlink (lstat + Fstat-style check).
//   5. rename(2) tmp → path.
//   6. fsync the parent directory (best-effort; failure is logged via
//      the returned error but the file itself is durable).
//
// ctx is honoured for cancellation between the write and the rename so a
// caller-supplied timeout does not leave a stale .apply-* sibling on disk
// forever. On cancellation the tmp file is removed before returning.
//
// The function does NOT consult audit, do revision checks, or touch the
// backup directory; those are the orchestrator's responsibility (see
// ApplyService.Apply).
func AtomicWriteSettings(ctx context.Context, path, content string) error {
	if path == "" {
		return fmt.Errorf("%w: path is empty", ErrSettingsPathInvalid)
	}
	if err := validateSettingsPath(path); err != nil {
		return err
	}

	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return fmt.Errorf("create parent dir %s: %w", parent, err)
	}

	tmpPath, err := writeSettingsTmp(path, content)
	if err != nil {
		return err
	}
	// Track whether the tmp has been published so the deferred cleanup
	// only fires when the rename did NOT succeed. After a successful
	// rename the tmp path is gone (it is now `path`); after a failed
	// rename the tmp file is still on disk and must be removed.
	var published bool
	defer func() {
		if !published {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := assertDestinationNotSymlink(path); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename tmp to %s: %w", path, err)
	}
	published = true
	if err := syncParentDir(parent); err != nil {
		// File is durable on stable storage; dir-entry change is best-effort.
		// Surface the failure but do not undo the rename.
		return fmt.Errorf("fsync parent dir %s: %w", parent, err)
	}
	if err := os.Chmod(path, settingsFileMode); err != nil {
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	return nil
}

// writeSettingsTmp writes content to a unique tmp sibling of path and
// returns the tmp path. The caller is responsible for renaming or
// removing it.
func writeSettingsTmp(path, content string) (string, error) {
	suffix, err := randomHex(8)
	if err != nil {
		return "", fmt.Errorf("generate tmp suffix: %w", err)
	}
	tmpPath := path + settingsTmpSuffix + suffix
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	flags = flagsWithNoFollow(flags) // platform-specific O_NOFOLLOW on Linux
	// #nosec G304 -- tmpPath is derived from path (already validated above) plus a constant suffix and a random hex token; not request-derived.
	f, err := os.OpenFile(tmpPath, flags, settingsFileMode)
	if err != nil {
		return "", fmt.Errorf("open tmp %s: %w", tmpPath, err)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		_ = f.Close()
		return "", fmt.Errorf("write tmp %s: %w", tmpPath, err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return "", fmt.Errorf("fsync tmp %s: %w", tmpPath, err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("close tmp %s: %w", tmpPath, err)
	}
	return tmpPath, nil
}

// validateSettingsPath enforces the boundary contract documented on
// AtomicWriteSettings: no `..` components and the cleaned path must equal
// the input. The check happens before any I/O.
func validateSettingsPath(path string) error {
	cleaned := filepath.Clean(path)
	if cleaned != path {
		return fmt.Errorf("%w: %q is not clean (cleaned=%q)", ErrSettingsPathInvalid, path, cleaned)
	}
	for _, part := range strings.Split(cleaned, string(filepath.Separator)) {
		if part == ".." {
			return fmt.Errorf("%w: %q contains traversal component", ErrSettingsPathInvalid, path)
		}
	}
	return nil
}

// assertDestinationNotSymlink rejects a destination that is (or has a
// parent that is) a symlink. We lstat the deepest existing component and
// walk up — a symlink at any level would let the rename resolve to an
// attacker-controlled location. Platforms with stricter open() guarantees
// (Linux + O_NOFOLLOW) wrap this with an additional pre-rename check.
func assertDestinationNotSymlink(path string) error {
	// First check the path itself.
	if err := assertNotSymlink(path); err != nil {
		return err
	}
	// Then walk up parents to make sure no intermediate component is a
	// symlink that could redirect the rename.
	current := path
	for {
		parent := filepath.Dir(current)
		if parent == current || parent == "" || parent == "." {
			break
		}
		if err := assertNotSymlink(parent); err != nil {
			return fmt.Errorf("%w: parent %s", ErrSettingsSymlinkRejected, parent)
		}
		current = parent
	}
	return nil
}

// assertNotSymlink returns ErrSettingsSymlinkRejected if path exists and
// is a symlink. A missing path is acceptable — the rename will create it.
func assertNotSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("lstat %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: %s", ErrSettingsSymlinkRejected, path)
	}
	return nil
}

// randomHex returns n bytes of crypto-random data encoded as hex.
// Used to suffix the tmp file so concurrent Apply calls cannot collide.
func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}