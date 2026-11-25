//go:build linux

package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

// acquireDataDirLock serialises restore, backup creation and apply on one
// data directory (issue #1024). The returned release func must be called
// exactly once, and the lock is never nested: entry points that need both a
// snapshot and a swap take it once and call the unlocked helpers.
//
// The lock is advisory flock on <dataDir>/nrcc-data-op.lock, so it also covers
// a second NRCC process and a scheduled backup, and the kernel drops it when
// the process dies — an interrupted restore can never lock the next start out.
func acquireDataDirLock(dataDir string) (func(), error) {
	lockPath := filepath.Join(dataDir, "nrcc-data-op.lock")
	// #nosec G304,G703 -- lockPath is the operator-configured data directory plus a constant filename; nothing here is request-derived.
	lock, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open data dir lock %s: %w", lockPath, err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = lock.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("%w (%s)", ErrDataDirBusy, lockPath)
		}
		return nil, fmt.Errorf("lock data dir %s: %w", lockPath, err)
	}
	var once sync.Once
	return func() { once.Do(func() { _ = lock.Close() }) }, nil
}
