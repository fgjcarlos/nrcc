package service

import (
	"sync"

	"nrcc/internal/model"
)

// LogBuffer is a thread-safe ring buffer for structured LogEntry objects
type LogBuffer struct {
	mu       sync.RWMutex
	entries  []model.LogEntry
	maxSize  int
	writeIdx int
}

// NewLogBuffer creates a new LogBuffer with the given capacity
func NewLogBuffer(maxSize int) *LogBuffer {
	if maxSize <= 0 {
		maxSize = 500
	}
	return &LogBuffer{
		entries: make([]model.LogEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add adds a LogEntry to the buffer
func (lb *LogBuffer) Add(entry model.LogEntry) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(lb.entries) >= lb.maxSize {
		// Shift all entries and replace the first one
		copy(lb.entries, lb.entries[1:])
		lb.entries[len(lb.entries)-1] = entry
	} else {
		lb.entries = append(lb.entries, entry)
	}
}

// GetAll returns all entries in the buffer
func (lb *LogBuffer) GetAll() []model.LogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make([]model.LogEntry, len(lb.entries))
	copy(result, lb.entries)
	return result
}

// GetFiltered returns entries filtered by level, source, and limited to the specified count
// Empty strings for level or source mean no filter for those fields
func (lb *LogBuffer) GetFiltered(level, source string, limit int) []model.LogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if limit <= 0 {
		limit = len(lb.entries)
	}

	var result []model.LogEntry
	// Iterate backwards through entries to get most recent first
	for i := len(lb.entries) - 1; i >= 0 && len(result) < limit; i-- {
		entry := lb.entries[i]

		// Check filters
		if level != "" && entry.Level != level {
			continue
		}
		if source != "" && entry.Source != source {
			continue
		}

		result = append(result, entry)
	}

	// Reverse to get chronological order
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}
