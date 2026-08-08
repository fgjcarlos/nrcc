package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
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

// Write overwrites the file atomically with mode 0600 via a temporary file and
// rename. Write is not atomic with a preceding Read; use Update for compound
// mutations.
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

	// Write to temp file first (atomic on POSIX)
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpPath, s.path)
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
