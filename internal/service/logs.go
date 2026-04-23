package service

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

const (
	logRotationThreshold = 10 * 1024 * 1024 // 10MB
	logRotationCheckFreq = 100              // Check every N writes
	maxLogFiles          = 5                // Keep up to 5 rotated files
)

// LogService handles structured logging with ring buffer and JSONL persistence
type LogService struct {
	dataDir    string
	buffer     *LogBuffer
	logFile    *os.File
	db         *sql.DB
	writeCount int
}

// NewLogService creates a new LogService
func NewLogService(dataDir string, db *sql.DB) (*LogService, error) {
	// Ensure logs directory exists
	logsDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logsDir, 0700); err != nil {
		return nil, fmt.Errorf("create logs directory: %w", err)
	}

	// Open or create the log file
	logPath := filepath.Join(logsDir, "app.log")
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	return &LogService{
		dataDir: dataDir,
		buffer:  NewLogBuffer(500),
		logFile: logFile,
		db:      db,
	}, nil
}

// Write writes a LogEntry to both the ring buffer and the JSONL log file
func (ls *LogService) Write(entry model.LogEntry) error {
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("log_%d_%s", time.Now().UnixNano(), randomID(8))
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	// Add to ring buffer
	ls.buffer.Add(entry)

	// Write to JSONL file
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal log entry: %w", err)
	}

	if _, err := ls.logFile.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write log file: %w", err)
	}

	// Check if rotation is needed
	ls.writeCount++
	if ls.writeCount >= logRotationCheckFreq {
		ls.writeCount = 0
		_ = ls.rotate() // Ignore rotation errors - logging shouldn't fail
	}

	return nil
}

// Get retrieves filtered logs from the ring buffer
func (ls *LogService) Get(limit int, level, source string) []model.LogEntry {
	return ls.buffer.GetFiltered(level, source, limit)
}

// Close closes the log file
func (ls *LogService) Close() error {
	if ls.logFile != nil {
		return ls.logFile.Close()
	}
	return nil
}

// rotate performs log rotation when the file exceeds the threshold
func (ls *LogService) rotate() error {
	// Check file size
	info, err := ls.logFile.Stat()
	if err != nil {
		return err
	}

	if info.Size() < logRotationThreshold {
		return nil
	}

	// Close current file
	if err := ls.logFile.Close(); err != nil {
		return err
	}

	logsDir := filepath.Join(ls.dataDir, "logs")
	oldPath := filepath.Join(logsDir, "app.log")

	// Rotate existing files
	for i := maxLogFiles - 1; i >= 1; i-- {
		oldRotPath := filepath.Join(logsDir, fmt.Sprintf("app.log.%d", i))
		newRotPath := filepath.Join(logsDir, fmt.Sprintf("app.log.%d", i+1))
		if platform.Exists(oldRotPath) {
			_ = os.Rename(oldRotPath, newRotPath)
		}
	}

	// Rename current log to .1
	newPath := filepath.Join(logsDir, "app.log.1")
	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}

	// Open new log file
	logFile, err := os.OpenFile(oldPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	ls.logFile = logFile
	return nil
}

// InitLogSchema creates the database tables for logging
func InitLogSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS job_history (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		status TEXT NOT NULL,
		started_at TEXT NOT NULL,
		finished_at TEXT,
		triggered_by TEXT,
		summary TEXT,
		error TEXT,
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS doctor_runs (
		id TEXT PRIMARY KEY,
		generated_at TEXT NOT NULL,
		overall_status TEXT NOT NULL,
		checks_json TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_job_history_type_status ON job_history(type, status);
	CREATE INDEX IF NOT EXISTS idx_job_history_created_at ON job_history(created_at);
	CREATE INDEX IF NOT EXISTS idx_doctor_runs_created_at ON doctor_runs(created_at);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("initialize log schema: %w", err)
	}
	return nil
}

// randomID generates a cryptographically secure random string ID
func randomID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	
	// Generate random bytes using crypto/rand for secure entropy
	randBytes := make([]byte, length)
	if _, err := rand.Read(randBytes); err != nil {
		// Fallback to simple sequential approach if rand fails (should never happen)
		for i := 0; i < length; i++ {
			result[i] = charset[i%len(charset)]
		}
		return string(result)
	}
	
	// Map random bytes to charset
	for i := 0; i < length; i++ {
		result[i] = charset[randBytes[i]%byte(len(charset))]
	}
	return string(result)
}
