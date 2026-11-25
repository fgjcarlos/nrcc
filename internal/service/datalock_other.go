//go:build !linux

package service

import (
	"fmt"
	"path/filepath"
	"sync"
)

// dataDirLocks keeps one mutex per data directory for platforms without
// flock. NRCC is deployed on Linux; this fallback only serialises goroutines
// inside one process.
var dataDirLocks sync.Map

// acquireDataDirLock serialises restore, backup creation and apply on one
// data directory (issue #1024). The returned release func must be called
// exactly once, and the lock is never nested.
func acquireDataDirLock(dataDir string) (func(), error) {
	abs, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve data dir: %w", err)
	}
	value, _ := dataDirLocks.LoadOrStore(abs, &sync.Mutex{})
	lock, ok := value.(*sync.Mutex)
	if !ok {
		return nil, fmt.Errorf("unexpected lock type for %s", abs)
	}
	if !lock.TryLock() {
		return nil, fmt.Errorf("%w (%s)", ErrDataDirBusy, abs)
	}
	return lock.Unlock, nil
}
