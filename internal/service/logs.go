package service

import (
	"sync"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// LogBuffer implements a thread-safe ring buffer for log entries with pub/sub
type LogBuffer struct {
	mu          sync.RWMutex
	entries     []model.LogEntry
	maxSize     int
	head        int // index of next write position
	count       int // number of entries in buffer
	subscribers []chan model.LogEntry
	subMu       sync.Mutex
}

// NewLogBuffer creates a new LogBuffer with specified max size
func NewLogBuffer(maxSize int) *LogBuffer {
	return &LogBuffer{
		entries:     make([]model.LogEntry, maxSize),
		maxSize:     maxSize,
		subscribers: make([]chan model.LogEntry, 0),
	}
}

// Push adds a new log entry to the buffer and notifies subscribers
func (lb *LogBuffer) Push(entry model.LogEntry) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Add entry to ring buffer
	lb.entries[lb.head] = entry
	lb.head = (lb.head + 1) % lb.maxSize
	if lb.count < lb.maxSize {
		lb.count++
	}

	// Notify all subscribers (non-blocking, drop if buffer full)
	lb.subMu.Lock()
	subscribers := make([]chan model.LogEntry, len(lb.subscribers))
	copy(subscribers, lb.subscribers)
	lb.subMu.Unlock()

	for _, ch := range subscribers {
		select {
		case ch <- entry:
		default:
			// Drop if subscriber's buffer is full
		}
	}
}

// Recent returns the last n log entries
func (lb *LogBuffer) Recent(n int) []model.LogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if n > lb.count {
		n = lb.count
	}

	result := make([]model.LogEntry, n)
	for i := 0; i < n; i++ {
		// Calculate index from the end of the buffer
		idx := (lb.head - n + i + lb.maxSize) % lb.maxSize
		result[i] = lb.entries[idx]
	}

	return result
}

// All returns all log entries in chronological order
func (lb *LogBuffer) All() []model.LogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make([]model.LogEntry, lb.count)
	for i := 0; i < lb.count; i++ {
		idx := (lb.head - lb.count + i + lb.maxSize) % lb.maxSize
		result[i] = lb.entries[idx]
	}

	return result
}

// Clear removes all entries from the buffer
func (lb *LogBuffer) Clear() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.entries = make([]model.LogEntry, lb.maxSize)
	lb.head = 0
	lb.count = 0
}

// Subscribe creates a channel for receiving new log entries
// Returns the channel and an unsubscribe function
func (lb *LogBuffer) Subscribe() (ch <-chan model.LogEntry, unsub func()) {
	ch_buf := make(chan model.LogEntry, 64)

	lb.subMu.Lock()
	lb.subscribers = append(lb.subscribers, ch_buf)
	lb.subMu.Unlock()

	unsub = func() {
		lb.subMu.Lock()
		// Find and remove this channel from subscribers
		for i, sub := range lb.subscribers {
			if sub == ch_buf {
				lb.subscribers = append(lb.subscribers[:i], lb.subscribers[i+1:]...)
				break
			}
		}
		lb.subMu.Unlock()
		close(ch_buf)
	}

	return ch_buf, unsub
}

// Count returns the number of entries in the buffer
func (lb *LogBuffer) Count() int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	return lb.count
}
