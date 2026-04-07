package model

import "time"

// JobRecord represents a job/operation history entry
type JobRecord struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"startedAt"`
	FinishedAt  *time.Time `json:"finishedAt,omitempty"`
	TriggeredBy string     `json:"triggeredBy"`
	Summary     string     `json:"summary"`
	Error       string     `json:"error,omitempty"`
}

// Job types
const (
	JobTypeRestart      = "restart"
	JobTypeBackup       = "backup"
	JobTypeRestore      = "restore"
	JobTypeNpmInstall   = "npm-install"
	JobTypeNpmUninstall = "npm-uninstall"
	JobTypeUpdateCheck  = "update-check"
	JobTypeUpdateApply  = "update-apply"
	JobTypeConfigApply  = "config-apply"
)

// Job statuses
const (
	JobStatusPending   = "pending"
	JobStatusRunning   = "running"
	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"
)
