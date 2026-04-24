package service

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"nrcc/internal/db"
	"nrcc/internal/model"
)

func setupTestLogService(t *testing.T) (*LogService, *sql.DB, string) {
	// Create temp directory
	tempDir := t.TempDir()

	// Setup in-memory SQLite with migrations
	testDB, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("failed to create in-memory database: %v", err)
	}

	// Create LogService
	logService, err := NewLogService(tempDir, testDB)
	if err != nil {
		t.Fatalf("failed to create LogService: %v", err)
	}

	t.Cleanup(func() {
		logService.Close()
		testDB.Close()
	})

	return logService, testDB, tempDir
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
	testDB, _ := db.OpenMemory()
	defer testDB.Close()

	logService, err := NewLogService(tempDir, testDB)
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

func TestLogServiceConcurrentWritesDuringRotation(t *testing.T) {
	t.Parallel()

	logService, _, tempDir := setupTestLogService(t)
	logPath := filepath.Join(tempDir, "logs", "app.log")

	seedLine := []byte("{\"seed\":true}\n")
	seedData := bytes.Repeat(seedLine, (logRotationThreshold/len(seedLine))+1)
	if err := os.WriteFile(logPath, seedData, 0600); err != nil {
		t.Fatalf("failed to seed log file for rotation test: %v", err)
	}
	logService.writeCount = logRotationCheckFreq - 1

	const writers = 32
	errCh := make(chan error, writers)
	var wg sync.WaitGroup

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			entry := model.LogEntry{
				ID:        fmt.Sprintf("concurrent-%d", i),
				Timestamp: time.Now().UTC(),
				Level:     model.LogLevelInfo,
				Source:    model.SourceRuntime,
				Event:     model.EventRuntimeLifecycle,
				Message:   strings.Repeat("x", 128),
			}

			if err := logService.Write(entry); err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent Write() error = %v", err)
	}

	entries := logService.Get(100, "", "")
	if len(entries) != writers {
		t.Fatalf("expected %d entries in buffer, got %d", writers, len(entries))
	}

	totalLines := 0
	seedLines := 0
	for _, path := range []string{logPath, filepath.Join(tempDir, "logs", "app.log.1")} {
		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatalf("failed to read %s: %v", path, err)
		}

		for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
			if line == "" {
				continue
			}

			var entry model.LogEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				t.Fatalf("failed to unmarshal log line from %s: %v", path, err)
			}
			if entry.ID == "" && entry.Message == "" {
				seedLines++
				continue
			}
			totalLines++
		}
	}

	if totalLines != writers {
		t.Fatalf("expected %d log lines across rotated files, got %d", writers, totalLines)
	}
	if seedLines == 0 {
		t.Fatal("expected seeded lines to remain readable after rotation")
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
