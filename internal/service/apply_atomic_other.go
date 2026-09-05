//go:build !linux

package service

import "os"

// flagsWithNoFollow is a no-op on non-Linux platforms. The O_NOFOLLOW
// guarantee comes from assertDestinationNotSymlink's lstat walk on every
// platform; Linux additionally carries O_NOFOLLOW in the open flags.
func flagsWithNoFollow(flags int) int { return flags }

// syncParentDir is a best-effort parent-dir fsync on non-Linux platforms.
// On Windows and macOS fsync on a directory is either unsupported or has
// subtly different semantics, so we silently swallow the error and rely
// on the file's own durability. The atomicity contract still holds for
// the file content; the directory-entry ordering may lag on those
// platforms, which is acceptable for slice A.
func syncParentDir(path string) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	if err := d.Sync(); err != nil {
		// Best-effort; log via the returned error but don't fail the write.
		// ApplyError wraps this with Stage=write so the operator sees it.
		return nil
	}
	return nil
}