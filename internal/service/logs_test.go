package service

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"nrcc/internal/model"
)

func setupTestLogService(t *testing.T) (*LogService, *sql.DB, string) {
	// Create temp directory
	tempDir := t.TempDir()

	// Setup in-memory SQLite
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory database: %v", err)
	}

	// Initialize schema
	if err := InitLogSchema(db); err != nil {
		t.Fatalf("failed to initialize log schema: %v", err)
	}

	// Create LogService
	logService, err := NewLogService(tempDir, db)
	if err != nil {
		t.Fatalf("failed to create LogService: %v", err)
	}

	t.Cleanup(func() {
		logService.Close()
		db.Close()
	})

	return logService, db, tempDir
}

func TestLogServiceWrite(t *testing.T) {
	t.Parallel()

	logService, _, _ := setupTestLogService(t)

	entry := model.LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     model.LogLevelInfo,
		Source:    model.SourceRuntime,
		Event:     model.EventRuntimeLifecycle,
		Message:   "Test log message",
	}

	err := logService.Write(entry)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify entry was added to buffer
	entries := logService.Get(100, "", "")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry in buffer, got %d", len(entries))
	}

	if entries[0].Message != "Test log message" {
		t.Errorf("expected message 'Test log message', got %s", entries[0].Message)
	}
}

func TestLogServiceWriteGeneratesID(t *testing.T) {
	t.Parallel()

	logService, _, _ := setupTestLogService(t)

	entry := model.LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     model.LogLevelInfo,
		Source:    model.SourceRuntime,
		Event:     model.EventRuntimeLifecycle,
		Message:   "Test",
	}

	err := logService.Write(entry)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	entries := logService.Get(100, "", "")
	if entries[0].ID == "" {
		t.Fatal("expected auto-generated ID, got empty string")
	}

	if !startsWith(entries[0].ID, "log_") {
		t.Errorf("expected ID to start with 'log_', got %s", entries[0].ID)
	}
}

func TestLogServiceWriteToFile(t *testing.T) {
	t.Parallel()

	logService, _, tempDir := setupTestLogService(t)

	entry := model.LogEntry{
		ID:        "test_log_1",
		Timestamp: time.Now().UTC(),
		Level:     model.LogLevelInfo,
		Source:    model.SourceRuntime,
		Event:     model.EventRuntimeLifecycle,
		Message:   "Test log entry",
		Metadata: map[string]any{
			"key": "value",
		},
	}

	err := logService.Write(entry)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Read the log file
	logPath := filepath.Join(tempDir, "logs", "app.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	// Verify JSONL format
	var loggedEntry model.LogEntry
	err = json.Unmarshal(content, &loggedEntry)
	if err != nil {
		t.Fatalf("failed to unmarshal logged entry: %v", err)
	}

	if loggedEntry.ID != "test_log_1" {
		t.Errorf("expected ID 'test_log_1', got %s", loggedEntry.ID)
	}
	if loggedEntry.Message != "Test log entry" {
		t.Errorf("expected message 'Test log entry', got %s", loggedEntry.Message)
	}
}

func TestLogServiceMultipleWrites(t *testing.T) {
	t.Parallel()

	logService, _, tempDir := setupTestLogService(t)

	// Write multiple entries
	messages := []string{"msg1", "msg2", "msg3"}
	for i, msg := range messages {
		entry := model.LogEntry{
			ID:        string(rune(48 + i)),
			Timestamp: time.Now().UTC(),
			Level:     model.LogLevelInfo,
			Source:    model.SourceRuntime,
			Event:     model.EventRuntimeLifecycle,
			Message:   msg,
		}
		if err := logService.Write(entry); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}

	// Verify all entries in buffer
	entries := logService.Get(100, "", "")
	if len(entries) != len(messages) {
		t.Fatalf("expected %d entries, got %d", len(messages), len(entries))
	}

	// Verify all entries in file
	logPath := filepath.Join(tempDir, "logs", "app.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	lines := countLines(string(content))
	if lines != len(messages) {
		t.Fatalf("expected %d lines in log file, got %d", len(messages), lines)
	}
}

func TestLogServiceGet(t *testing.T) {
	t.Parallel()

	logService, _, _ := setupTestLogService(t)

	// Write entries with different levels
	logService.Write(model.LogEntry{
		ID:      "log1",
		Level:   model.LogLevelInfo,
		Source:  model.SourceRuntime,
		Event:   model.EventRuntimeLifecycle,
		Message: "info",
	})
	logService.Write(model.LogEntry{
		ID:      "log2",
		Level:   model.LogLevelWarn,
		Source:  model.SourceRuntime,
		Event:   model.EventRuntimeStderr,
		Message: "warn",
	})
	logService.Write(model.LogEntry{
		ID:      "log3",
		Level:   model.LogLevelError,
		Source:  model.SourceJob,
		Event:   model.EventJobFailed,
		Message: "error",
	})

	// Get with level filter
	entries := logService.Get(100, model.LogLevelInfo, "")
	if len(entries) != 1 {
		t.Fatalf("expected 1 info entry, got %d", len(entries))
	}

	// Get with source filter
	entries = logService.Get(100, "", model.SourceJob)
	if len(entries) != 1 {
		t.Fatalf("expected 1 job entry, got %d", len(entries))
	}

	// Get with limit
	entries = logService.Get(2, "", "")
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries with limit, got %d", len(entries))
	}
}

func TestLogServiceClose(t *testing.T) {
	t.Parallel()

	logService, _, _ := setupTestLogService(t)

	// Write an entry before closing
	logService.Write(model.LogEntry{
		Level:   model.LogLevelInfo,
		Source:  model.SourceRuntime,
		Event:   model.EventRuntimeLifecycle,
		Message: "Test",
	})

	// Close should not error
	err := logService.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestLogServiceDirectoryCreation(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	InitLogSchema(db)

	logService, err := NewLogService(tempDir, db)
	if err != nil {
		t.Fatalf("NewLogService() error = %v", err)
	}
	defer logService.Close()

	// Verify logs directory was created
	logsDir := filepath.Join(tempDir, "logs")
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		t.Fatalf("logs directory was not created at %s", logsDir)
	}

	// Verify app.log file was created
	logPath := filepath.Join(logsDir, "app.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatalf("app.log was not created at %s", logPath)
	}
}

// Helper functions
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func countLines(content string) int {
	count := 0
	for _, c := range content {
		if c == '\n' {
			count++
		}
	}
	// Count last line if it doesn't end with newline
	if len(content) > 0 && content[len(content)-1] != '\n' {
		count++
	}
	return count
}
