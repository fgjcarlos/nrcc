package service

import (
	"sync"
	"testing"
	"time"

	"nrcc/internal/model"
)

func TestLogBufferAdd(t *testing.T) {
	t.Parallel()

	buffer := NewLogBuffer(3)

	entry1 := model.LogEntry{
		ID:        "log1",
		Timestamp: time.Now(),
		Level:     model.LogLevelInfo,
		Source:    model.SourceRuntime,
		Event:     model.EventRuntimeLifecycle,
		Message:   "Runtime started",
	}

	buffer.Add(entry1)
	entries := buffer.GetAll()

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ID != "log1" {
		t.Errorf("expected log1, got %s", entries[0].ID)
	}
}

func TestLogBufferEviction(t *testing.T) {
	t.Parallel()

	buffer := NewLogBuffer(3) // Capacity of 3

	// Add 5 entries
	for i := 1; i <= 5; i++ {
		entry := model.LogEntry{
			ID:        string(rune(48 + i)), // "1", "2", "3", "4", "5"
			Timestamp: time.Now(),
			Level:     model.LogLevelInfo,
			Source:    model.SourceRuntime,
			Event:     model.EventRuntimeLifecycle,
			Message:   "Test",
		}
		buffer.Add(entry)
	}

	entries := buffer.GetAll()

	// Should only have 3 entries (oldest 2 evicted)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries after eviction, got %d", len(entries))
	}

	// Should have entries 3, 4, 5 (IDs "3", "4", "5")
	expectedIDs := []string{"3", "4", "5"}
	for i, expected := range expectedIDs {
		if entries[i].ID != expected {
			t.Errorf("entry %d: expected ID %s, got %s", i, expected, entries[i].ID)
		}
	}
}

func TestLogBufferGetAll(t *testing.T) {
	t.Parallel()

	buffer := NewLogBuffer(5)

	// Add entries with different levels
	levels := []string{model.LogLevelDebug, model.LogLevelInfo, model.LogLevelWarn, model.LogLevelError}
	for i, level := range levels {
		buffer.Add(model.LogEntry{
			ID:        string(rune(48 + i)),
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Level:     level,
			Source:    model.SourceRuntime,
			Event:     model.EventRuntimeLifecycle,
			Message:   "Test",
		})
	}

	entries := buffer.GetAll()

	if len(entries) != len(levels) {
		t.Fatalf("expected %d entries, got %d", len(levels), len(entries))
	}

	// Verify order is preserved
	for i := 0; i < len(levels); i++ {
		if entries[i].Level != levels[i] {
			t.Errorf("entry %d: expected level %s, got %s", i, levels[i], entries[i].Level)
		}
	}
}

func TestLogBufferFilterByLevel(t *testing.T) {
	t.Parallel()

	buffer := NewLogBuffer(10)

	// Add entries with different levels
	testCases := []struct {
		level string
		count int
	}{
		{model.LogLevelInfo, 2},
		{model.LogLevelWarn, 3},
		{model.LogLevelError, 1},
	}

	entryCount := 0
	for _, tc := range testCases {
		for i := 0; i < tc.count; i++ {
			buffer.Add(model.LogEntry{
				ID:        string(rune(48 + entryCount)),
				Timestamp: time.Now(),
				Level:     tc.level,
				Source:    model.SourceRuntime,
				Event:     model.EventRuntimeLifecycle,
				Message:   "Test",
			})
			entryCount++
		}
	}

	// Filter by info level
	entries := buffer.GetFiltered(model.LogLevelInfo, "", 100)
	if len(entries) != 2 {
		t.Fatalf("expected 2 info entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Level != model.LogLevelInfo {
			t.Errorf("expected info level, got %s", e.Level)
		}
	}

	// Filter by warn level
	entries = buffer.GetFiltered(model.LogLevelWarn, "", 100)
	if len(entries) != 3 {
		t.Fatalf("expected 3 warn entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Level != model.LogLevelWarn {
			t.Errorf("expected warn level, got %s", e.Level)
		}
	}

	// Filter by error level
	entries = buffer.GetFiltered(model.LogLevelError, "", 100)
	if len(entries) != 1 {
		t.Fatalf("expected 1 error entry, got %d", len(entries))
	}
}

func TestLogBufferFilterBySource(t *testing.T) {
	t.Parallel()

	buffer := NewLogBuffer(10)

	sources := []string{model.SourceRuntime, model.SourceJob, model.SourceRuntime, model.SourceDoctor}
	for i, source := range sources {
		buffer.Add(model.LogEntry{
			ID:        string(rune(48 + i)),
			Timestamp: time.Now(),
			Level:     model.LogLevelInfo,
			Source:    source,
			Event:     model.EventRuntimeLifecycle,
			Message:   "Test",
		})
	}

	// Filter by runtime source
	entries := buffer.GetFiltered("", model.SourceRuntime, 100)
	if len(entries) != 2 {
		t.Fatalf("expected 2 runtime entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Source != model.SourceRuntime {
			t.Errorf("expected runtime source, got %s", e.Source)
		}
	}

	// Filter by job source
	entries = buffer.GetFiltered("", model.SourceJob, 100)
	if len(entries) != 1 {
		t.Fatalf("expected 1 job entry, got %d", len(entries))
	}
	if entries[0].Source != model.SourceJob {
		t.Errorf("expected job source, got %s", entries[0].Source)
	}
}

func TestLogBufferFilterCombined(t *testing.T) {
	t.Parallel()

	buffer := NewLogBuffer(20)

	// Add entries with various combinations
	buffer.Add(model.LogEntry{
		ID:     "log1",
		Level:  model.LogLevelInfo,
		Source: model.SourceRuntime,
		Event:  model.EventRuntimeLifecycle,
	})
	buffer.Add(model.LogEntry{
		ID:     "log2",
		Level:  model.LogLevelError,
		Source: model.SourceRuntime,
		Event:  model.EventRuntimeStderr,
	})
	buffer.Add(model.LogEntry{
		ID:     "log3",
		Level:  model.LogLevelInfo,
		Source: model.SourceJob,
		Event:  model.EventJobStarted,
	})

	// Filter by level AND source
	entries := buffer.GetFiltered(model.LogLevelInfo, model.SourceRuntime, 100)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ID != "log1" {
		t.Errorf("expected log1, got %s", entries[0].ID)
	}
}

func TestLogBufferLimit(t *testing.T) {
	t.Parallel()

	buffer := NewLogBuffer(100)

	// Add 10 entries
	for i := 0; i < 10; i++ {
		buffer.Add(model.LogEntry{
			ID:        string(rune(48 + i)),
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Level:     model.LogLevelInfo,
			Source:    model.SourceRuntime,
			Event:     model.EventRuntimeLifecycle,
			Message:   "Test",
		})
	}

	// Get all with limit 5
	entries := buffer.GetFiltered("", "", 5)
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries with limit, got %d", len(entries))
	}

	// Most recent 5 should be returned (IDs 5-9)
	// (GetFiltered iterates backwards, then reverses)
	// So we should get the last 5
	expectedStart := 5
	for i, e := range entries {
		expectedID := string(rune(48 + expectedStart + i))
		if e.ID != expectedID {
			t.Errorf("entry %d: expected ID %s, got %s", i, expectedID, e.ID)
		}
	}
}

func TestLogBufferThreadSafety(t *testing.T) {
	t.Parallel()

	buffer := NewLogBuffer(1000)
	numGoroutines := 10
	entriesPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < entriesPerGoroutine; i++ {
				buffer.Add(model.LogEntry{
					ID:        string(rune(48+goroutineID)) + "_" + string(rune(48+i%10)),
					Timestamp: time.Now(),
					Level:     model.LogLevelInfo,
					Source:    model.SourceRuntime,
					Event:     model.EventRuntimeLifecycle,
					Message:   "Concurrent test",
				})
			}
		}(g)
	}

	wg.Wait()

	entries := buffer.GetAll()
	if len(entries) != numGoroutines*entriesPerGoroutine {
		t.Fatalf("expected %d entries, got %d", numGoroutines*entriesPerGoroutine, len(entries))
	}
}

func TestLogBufferEmptyBuffer(t *testing.T) {
	t.Parallel()

	buffer := NewLogBuffer(10)

	entries := buffer.GetAll()
	if len(entries) != 0 {
		t.Fatalf("expected empty buffer, got %d entries", len(entries))
	}

	entries = buffer.GetFiltered(model.LogLevelInfo, "", 100)
	if len(entries) != 0 {
		t.Fatalf("expected empty filter result, got %d entries", len(entries))
	}
}

func TestLogBufferDefaultSize(t *testing.T) {
	t.Parallel()

	buffer := NewLogBuffer(0) // 0 should default to 500
	if buffer.maxSize != 500 {
		t.Fatalf("expected default size 500, got %d", buffer.maxSize)
	}

	buffer = NewLogBuffer(-1) // Negative should also default to 500
	if buffer.maxSize != 500 {
		t.Fatalf("expected default size 500, got %d", buffer.maxSize)
	}
}
