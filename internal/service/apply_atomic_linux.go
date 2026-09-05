//go:build linux

package service

import (
	"os"
	"syscall"
)

// flagsWithNoFollow adds the Linux-specific O_NOFOLLOW flag to flags so
// opening the tmp sibling never follows a symlink at the destination.
// On non-Linux platforms this is a no-op (see apply_atomic_other.go).
func flagsWithNoFollow(flags int) int {
	return flags | syscall.O_NOFOLLOW
}

// syncParentDir fsyncs the parent directory so the rename's directory
// entry change is durable across a power loss. The error is returned to
// the caller; on Linux this is universally supported.
func syncParentDir(path string) error {
	// #nosec G304 -- path is the parent directory derived from the caller-validated settings.js destination; not request-derived.
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	return d.Sync()
}