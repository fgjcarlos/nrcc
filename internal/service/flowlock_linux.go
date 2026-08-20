//go:build linux

package service

import (
	"fmt"
	"os"
	"syscall"
)

// flockExclusive acquires an exclusive, non-blocking advisory lock on the
// file at path's sibling lockfile (path + ".lock"). The returned *os.File
// owns the lock; closing it releases the lock. Callers MUST defer Close.
//
// flock is OS-managed and survives process death, unlike a Go mutex which
// only serializes NRCC-internal goroutines. flock is advisory: a writer
// that does not respect flock will not be blocked. For NRCC's use case
// (Node-RED flows.json on a local POSIX FS, same UID) this is sufficient.
// LOCK_NB surfaces contention immediately so request handlers do not hang
// waiting for a lock held by another goroutine or process.
func flockExclusive(path string) (*os.File, error) {
	lockPath := path + ".lock"
	// #nosec G304 -- lockPath is the sibling of an already-validated target path; not request-derived.
	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file %s: %w", lockPath, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("acquire lock on %s: %w", lockPath, err)
	}
	return f, nil
}

// openNoFollow opens path for reading without following a symlink at the
// final component. If the last path component is a symlink, the kernel
// returns ELOOP and this function surfaces that error. After the open it
// also calls Fstat and rejects any fd that turns out to be a symlink —
// defense-in-depth against a race that replaces a regular file with a
// symlink via rename(2) on the same directory entry between open and
// fstat. On the same fd this is atomic on Linux; the check is cheap.
func openNoFollow(path string) (*os.File, error) {
	// #nosec G304 -- path is supplied by the caller after ValidateBackupID + filepath.Join against an operator-supplied dataDir; not request-derived. openNoFollow is the choke point that enforces O_NOFOLLOW.
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s without follow: %w", path, err)
	}
	var st syscall.Stat_t
	if err := syscall.Fstat(int(f.Fd()), &st); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("fstat %s: %w", path, err)
	}
	if st.Mode&syscall.S_IFMT == syscall.S_IFLNK {
		_ = f.Close()
		return nil, fmt.Errorf("refuse to read symlink %s", path)
	}
	return f, nil
}