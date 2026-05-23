package store

import (
	"encoding/json"
	"os"
	"sync"
)

// JSONStore is a generic JSON file store with mutex protection
type JSONStore[T any] struct {
	path string
	mu   sync.RWMutex
}

// NewJSONStore creates a new JSON store for the given path
func NewJSONStore[T any](path string) *JSONStore[T] {
	return &JSONStore[T]{path: path}
}

// Read reads and unmarshals the JSON file
func (s *JSONStore[T]) Read() (T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var val T
	data, err := os.ReadFile(s.path)
	if err != nil {
		return val, err
	}

	err = json.Unmarshal(data, &val)
	return val, err
}

// Write marshals and writes the value to JSON file (atomic via temp file)
func (s *JSONStore[T]) Write(val T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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

// Exists checks if the file exists
func (s *JSONStore[T]) Exists() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err := os.Stat(s.path)
	return err == nil
}
