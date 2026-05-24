package service

import (
	"sync"

	"github.com/composedof2/nrcc/internal/model"
)

// MetricsBuffer is a thread-safe ring buffer for MetricsSnapshot entries.
// It follows the same structural pattern as LogBuffer.
type MetricsBuffer struct {
	mu       sync.RWMutex
	entries  []model.MetricsSnapshot
	head     int // index of the next write position
	count    int // number of valid entries currently stored
	capacity int
}

// NewMetricsBuffer creates a new MetricsBuffer with the given capacity.
func NewMetricsBuffer(capacity int) *MetricsBuffer {
	return &MetricsBuffer{
		entries:  make([]model.MetricsSnapshot, capacity),
		capacity: capacity,
	}
}

// Push adds a snapshot to the ring buffer, overwriting the oldest entry when full.
func (mb *MetricsBuffer) Push(snap model.MetricsSnapshot) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	mb.entries[mb.head] = snap
	mb.head = (mb.head + 1) % mb.capacity
	if mb.count < mb.capacity {
		mb.count++
	}
}

// Recent returns the last n snapshots in chronological order (oldest first).
// If n exceeds the number of stored entries, all stored entries are returned.
func (mb *MetricsBuffer) Recent(n int) []model.MetricsSnapshot {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	if n > mb.count {
		n = mb.count
	}

	result := make([]model.MetricsSnapshot, n)
	for i := 0; i < n; i++ {
		idx := (mb.head - n + i + mb.capacity) % mb.capacity
		result[i] = mb.entries[idx]
	}
	return result
}

// Last returns the most recently pushed snapshot and true, or the zero value
// and false when the buffer is empty.
func (mb *MetricsBuffer) Last() (model.MetricsSnapshot, bool) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	if mb.count == 0 {
		return model.MetricsSnapshot{}, false
	}

	idx := (mb.head - 1 + mb.capacity) % mb.capacity
	return mb.entries[idx], true
}
