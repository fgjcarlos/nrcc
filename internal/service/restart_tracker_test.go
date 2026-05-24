package service

import (
	"testing"
	"time"

	"github.com/composedof2/nrcc/internal/model"
)

func TestRestartTracker_Push_RecordsCorrectFields(t *testing.T) {
	rt := newRestartTracker(50)

	evt := model.RestartEvent{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		ExitCode:    1,
		Attempt:     1,
		MaxAttempts: 10,
	}
	rt.push(evt)

	events := rt.restartEvents()
	if len(events) != 1 {
		t.Fatalf("restartEvents() len = %d, want 1", len(events))
	}

	got := events[0]
	if got.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", got.ExitCode)
	}
	if got.Attempt != 1 {
		t.Errorf("Attempt = %d, want 1", got.Attempt)
	}
	if got.MaxAttempts != 10 {
		t.Errorf("MaxAttempts = %d, want 10", got.MaxAttempts)
	}
	if got.Timestamp == "" {
		t.Error("Timestamp must not be empty")
	}
}

func TestRestartTracker_RestartEvents_ChronologicalOrder(t *testing.T) {
	rt := newRestartTracker(50)

	for i := 1; i <= 3; i++ {
		rt.push(model.RestartEvent{
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
			ExitCode:    i,
			Attempt:     i,
			MaxAttempts: 10,
		})
	}

	events := rt.restartEvents()
	if len(events) != 3 {
		t.Fatalf("restartEvents() len = %d, want 3", len(events))
	}

	// Must be chronological (oldest first — same order they were pushed)
	for i, evt := range events {
		if evt.Attempt != i+1 {
			t.Errorf("events[%d].Attempt = %d, want %d (chronological order broken)", i, evt.Attempt, i+1)
		}
	}
}

func TestRestartTracker_Cap_AtFifty(t *testing.T) {
	rt := newRestartTracker(50)

	// Push 60 events — only the last 50 should remain
	for i := 1; i <= 60; i++ {
		rt.push(model.RestartEvent{
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
			ExitCode:    i,
			Attempt:     i,
			MaxAttempts: 10,
		})
	}

	events := rt.restartEvents()
	if len(events) != 50 {
		t.Fatalf("restartEvents() len = %d, want 50 (cap enforced)", len(events))
	}

	// The oldest 10 are gone; first remaining event should have ExitCode=11
	if events[0].ExitCode != 11 {
		t.Errorf("events[0].ExitCode = %d, want 11 (oldest 10 overwritten)", events[0].ExitCode)
	}
	// Last event should be the 60th
	if events[49].ExitCode != 60 {
		t.Errorf("events[49].ExitCode = %d, want 60", events[49].ExitCode)
	}
}

func TestRestartTracker_Empty_ReturnsNilOrEmptySlice(t *testing.T) {
	rt := newRestartTracker(50)

	events := rt.restartEvents()
	if len(events) != 0 {
		t.Fatalf("restartEvents() on empty tracker = %d, want 0", len(events))
	}
}
