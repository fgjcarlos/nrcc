package service

import (
	"sync"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// restartTracker is a thread-safe in-memory ring buffer for the most
// recent Node-RED restart events. It is intentionally separate from
// the durable persistent counter kept by `restartCountStore`:
//
//   - The ring buffer (`restartTracker`, this file) holds the last N
//     events with their context (timestamp, reason, etc.) for the
//     runtime history endpoint. Capacity is bounded so memory is
//     finite; data is lost on process restart.
//   - The persistent counter (`restartCountStore`) holds a single
//     monotonic integer — total cumulative restarts since install —
//     that survives process restarts and is exposed as
//     `ProcessManager.CumulativeRestarts()`.
//
// The two sources answer different questions and have different
// lifetimes. They are not a duplication; do not collapse them without
// confirming the runtime history endpoint still has data to serve
// after a restart.
type restartTracker struct {
	mu       sync.RWMutex
	entries  []model.RestartEvent
	head     int // index of the next write position
	count    int // number of valid entries currently stored
	capacity int
}

// newRestartTracker creates a restartTracker with the given capacity (e.g. 50).
func newRestartTracker(capacity int) *restartTracker {
	return &restartTracker{
		entries:  make([]model.RestartEvent, capacity),
		capacity: capacity,
	}
}

// push adds a RestartEvent to the ring buffer, overwriting the oldest when full.
func (rt *restartTracker) push(evt model.RestartEvent) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.entries[rt.head] = evt
	rt.head = (rt.head + 1) % rt.capacity
	if rt.count < rt.capacity {
		rt.count++
	}
}

// restartEvents returns all stored events in chronological order (oldest first).
func (rt *restartTracker) restartEvents() []model.RestartEvent {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	n := rt.count
	result := make([]model.RestartEvent, n)
	for i := 0; i < n; i++ {
		idx := (rt.head - n + i + rt.capacity) % rt.capacity
		result[i] = rt.entries[idx]
	}
	return result
}
