package model

import "time"

// LogEntry represents a structured log event
type LogEntry struct {
	ID          string         `json:"id"`
	Timestamp   time.Time      `json:"timestamp"`
	Level       string         `json:"level"` // debug, info, warn, error
	Source      string         `json:"source"`
	Event       string         `json:"event"`
	Message     string         `json:"message"`
	OperationID string         `json:"operationId,omitempty"`
	JobID       string         `json:"jobId,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Log levels
const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

// Event sources
const (
	SourceRuntime   = "runtime"
	SourceOperation = "operation"
	SourceJob       = "job"
	SourceDoctor    = "doctor"
	SourceAuth      = "auth"
	SourceConfig    = "config"
	SourceBackup    = "backup"
	SourceLibrary   = "library"
	SourceUpdate    = "update"
)

// Event types
const (
	EventRuntimeStdout     = "runtime.stdout"
	EventRuntimeStderr     = "runtime.stderr"
	EventRuntimeLifecycle  = "runtime.lifecycle"
	EventJobStarted        = "job.started"
	EventJobFinished       = "job.finished"
	EventJobFailed         = "job.failed"
	EventOperationLocked   = "operation.locked"
	EventOperationReleased = "operation.released"
	EventDoctorCheck       = "doctor.check"
	EventAuthAudit         = "auth.audit"
	EventConfigApply       = "config.apply"
	EventBackupCreate      = "backup.create"
	EventBackupRestore     = "backup.restore"
	EventLibraryInstall    = "library.install"
	EventLibraryUninstall  = "library.uninstall"
	EventUpdateCheck       = "update.check"
	EventUpdateApply       = "update.apply"
)
