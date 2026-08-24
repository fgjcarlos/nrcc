package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// JSONStore is a generic JSON file store protected by a per-instance lock.
// It is not safe across processes or multiple instances using the same path.
type JSONStore[T any] struct {
	path string
	mu   sync.RWMutex
}

// NewJSONStore creates a new JSON store for the given path
func NewJSONStore[T any](path string) *JSONStore[T] {
	return &JSONStore[T]{path: path}
}

// Read reads and unmarshals the JSON file. If the file does not exist, Read
// returns the zero value and the filesystem error. Read is not atomic with a
// subsequent Write; use Update for compound mutations.
func (s *JSONStore[T]) Read() (T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.readUnlocked()
}

func (s *JSONStore[T]) readUnlocked() (T, error) {
	var val T
	data, err := os.ReadFile(s.path)
	if err != nil {
		return val, err
	}

	err = json.Unmarshal(data, &val)
	return val, err
}

// Write overwrites the file atomically with mode 0600 via a temporary file
// and rename. Write is not atomic with a preceding Read; use Update for
// compound mutations.
//
// Durability: the temp file is fsync'd before rename so its contents are
// guaranteed on stable storage when the rename publishes the file name.
// The parent directory is fsync'd afterwards so the rename itself is
// durable across a power loss — without that, a crash can leave a
// correctly-named but zero-length file (the rename published before the
// dir-entry change hit disk).
func (s *JSONStore[T]) Write(val T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.writeUnlocked(val)
}

func (s *JSONStore[T]) writeUnlocked(val T) error {
	data, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file first (atomic on POSIX), fsync the contents, then
	// rename. Fsync the parent directory afterwards so the rename is durable.
	tmpPath := s.path + ".tmp"
	// ponytail: gosec G304 — tmpPath is derived from s.path (the JSON store
	// path the caller already controls) plus the literal suffix ".tmp". Not
	// user input; safe to include.
	// #nosec G304
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	// Write the data; on error, capture the close error and join with the
	// write error so neither is swallowed.
	if _, werr := f.Write(data); werr != nil {
		if cerr := f.Close(); cerr != nil {
			return errors.Join(werr, cerr)
		}
		return werr
	}
	if serr := f.Sync(); serr != nil {
		if cerr := f.Close(); cerr != nil {
			return errors.Join(serr, cerr)
		}
		return serr
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return err
	}
	if d, err := os.Open(filepath.Dir(s.path)); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// Update is the supported API for compound read-modify-write mutations. The
// callback must not re-enter this store because doing so deadlocks, and it must
// not perform external I/O because it blocks readers and writers. Update
// returns nil on success and propagates callback errors without writing.
func (s *JSONStore[T]) Update(fn func(*T) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if fn == nil {
		return errors.New("nil callback")
	}

	current, err := s.readUnlocked()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	if err := fn(&current); err != nil {
		return err
	}

	return s.writeUnlocked(current)
}

// Exists checks if the file exists
func (s *JSONStore[T]) Exists() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err := os.Stat(s.path)
	return err == nil
}
