package service

import (
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// TestGetLogs_Empty tests that GetLogs returns empty slice for new ProcessManager
func TestGetLogs_Empty(t *testing.T) {
	logBuf := NewLogBuffer(100)
	pm := NewProcessManager("node-red", t.TempDir(), logBuf)

	result := pm.GetLogs(10)

	if result == nil {
		t.Error("GetLogs should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("GetLogs(10) on empty buffer should return empty slice, got %d entries", len(result))
	}
}

// TestGetLogs_LimitZero tests that GetLogs(0) returns all available logs
func TestGetLogs_LimitZero(t *testing.T) {
	logBuf := NewLogBuffer(100)

	// Add some test log entries
	for i := 0; i < 5; i++ {
		logBuf.Push(model.LogEntry{
			ID:        "test-" + string(rune(i)),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Level:     "info",
			Source:    "test",
			Message:   "Test message " + string(rune('0'+i)),
		})
	}

	pm := NewProcessManager("node-red", t.TempDir(), logBuf)

	result := pm.GetLogs(0)

	if len(result) != 5 {
		t.Errorf("GetLogs(0) should return all 5 entries, got %d", len(result))
	}

	// Verify messages are in the result
	for i := 0; i < 5; i++ {
		found := false
		for _, msg := range result {
			if msg == "Test message "+string(rune('0'+i)) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected message 'Test message %d' not found in results", i)
		}
	}
}

// TestGetLogs_LimitNegative tests that GetLogs with negative limit returns all logs
func TestGetLogs_LimitNegative(t *testing.T) {
	logBuf := NewLogBuffer(100)

	// Add test log entries
	for i := 0; i < 3; i++ {
		logBuf.Push(model.LogEntry{
			ID:        "test-" + string(rune(i)),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Level:     "info",
			Source:    "test",
			Message:   "Message " + string(rune('0'+i)),
		})
	}

	pm := NewProcessManager("node-red", t.TempDir(), logBuf)

	result := pm.GetLogs(-1)

	if len(result) != 3 {
		t.Errorf("GetLogs(-1) should return all 3 entries, got %d", len(result))
	}
}

// TestGetLogs_WithLimit tests that GetLogs respects the limit parameter
func TestGetLogs_WithLimit(t *testing.T) {
	logBuf := NewLogBuffer(100)

	// Add 10 test log entries
	for i := 0; i < 10; i++ {
		logBuf.Push(model.LogEntry{
			ID:        "test-" + string(rune('0'+(i/10))) + string(rune('0'+(i%10))),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Level:     "info",
			Source:    "test",
			Message:   "Message " + string(rune('0'+(i%10))),
		})
	}

	pm := NewProcessManager("node-red", t.TempDir(), logBuf)

	tests := []struct {
		name     string
		limit    int
		minCount int
		maxCount int
	}{
		{"limit 5", 5, 5, 5},
		{"limit 1", 1, 1, 1},
		{"limit 10", 10, 10, 10},
		{"limit 20 (more than available)", 20, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pm.GetLogs(tt.limit)
			if len(result) < tt.minCount || len(result) > tt.maxCount {
				t.Errorf("GetLogs(%d) returned %d entries, want between %d and %d",
					tt.limit, len(result), tt.minCount, tt.maxCount)
			}
		})
	}
}

// TestGetLogs_NilBuffer tests that GetLogs handles nil buffer gracefully
func TestGetLogs_NilBuffer(t *testing.T) {
	pm := NewProcessManager("node-red", t.TempDir(), nil)

	result := pm.GetLogs(10)

	if result == nil {
		t.Error("GetLogs should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Error("GetLogs with nil buffer should return empty slice")
	}
}

// TestGetLogs_MessageIntegrity tests that message content is preserved
func TestGetLogs_MessageIntegrity(t *testing.T) {
	logBuf := NewLogBuffer(100)

	testMessages := []string{
		"Node-RED started",
		"Flow deployed successfully",
		"Listening on port 1880",
		"Server running on http://localhost:1880",
	}

	for _, msg := range testMessages {
		logBuf.Push(model.LogEntry{
			ID:        "test-" + msg,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Level:     "info",
			Source:    "stdout",
			Message:   msg,
		})
	}

	pm := NewProcessManager("node-red", t.TempDir(), logBuf)

	result := pm.GetLogs(0)

	if len(result) != len(testMessages) {
		t.Fatalf("Expected %d messages, got %d", len(testMessages), len(result))
	}

	for i, expectedMsg := range testMessages {
		if result[i] != expectedMsg {
			t.Errorf("Message %d: expected %q, got %q", i, expectedMsg, result[i])
		}
	}
}
