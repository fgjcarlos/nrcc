package model

// BackupType identifies how a backup was created.
type BackupType string

const (
	BackupTypeManual     BackupType = "manual"
	BackupTypeAuto       BackupType = "auto"
	BackupTypePreRestore BackupType = "pre-restore"
)

// PaginationOpts specifies pagination, sorting, and filtering parameters.
type PaginationOpts struct {
	Page  int    // 1-based; default 1
	Limit int    // default 20; max 100
	Sort  string // "date", "size", "status"; default "date"
	Order string // "asc" or "desc"; default "desc"
}

// PaginatedBackups is the response wrapper for paginated backup lists.
type PaginatedBackups struct {
	Items []Backup `json:"items"`
	Total int      `json:"total"`
	Page  int      `json:"page"`
	Limit int      `json:"limit"`
}

// Backup represents a backup entry.
type Backup struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Type        BackupType `json:"type"`
	CreatedAt   string     `json:"createdAt"`
	TriggeredBy string     `json:"triggeredBy"`
	FileCount   int        `json:"fileCount"`
	SizeBytes   int64      `json:"sizeBytes"`
	Path        string     `json:"-"` // internal, not exposed in JSON
}

// BackupFileEntry represents a file stored inside a backup.
type BackupFileEntry struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
}

// BackupManifestV1 is the on-disk manifest embedded as `backup-metadata.json`
// inside each backup archive. The Algorithm field pins the checksum scheme so
// older backups remain verifiable when the default algorithm changes.
type BackupManifestV1 struct {
	Version      int              `json:"version"`
	Algorithm    string           `json:"algorithm"`
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Type         BackupType       `json:"type"`
	CreatedAt    string           `json:"createdAt"`
	TriggeredBy  string           `json:"triggeredBy"`
	Files        []BackupFileEntry `json:"files"`
}

// BackupManifest represents backup metadata.
type BackupManifest struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        BackupType        `json:"type"`
	CreatedAt   string            `json:"createdAt"`
	TriggeredBy string            `json:"triggeredBy"`
	Files       []BackupFileEntry `json:"files"`
	TotalSize   int64             `json:"totalSize"`
}

// BackupStorageInfo contains aggregate local backup stats.
type BackupStorageInfo struct {
	TotalBackups    int   `json:"totalBackups"`
	TotalSize       int64 `json:"totalSize"`
	ManualCount     int   `json:"manualCount"`
	AutoCount       int   `json:"autoCount"`
	PreRestoreCount int   `json:"preRestoreCount"`
}

// BackupConfig stores scheduler and retention settings for backups.
type BackupConfig struct {
	Enabled             bool   `json:"enabled"`
	Schedule            string `json:"schedule"`
	CustomSchedule      string `json:"customSchedule,omitempty"`
	RetentionManual     int    `json:"retentionManual"`
	RetentionAuto       int    `json:"retentionAuto"`
	RetentionPreRestore int    `json:"retentionPreRestore"`
	IncludeConfig       bool   `json:"includeConfig"`
	IncludeSettings     bool   `json:"includeSettings"`
	IncludeFlowsCred    bool   `json:"includeFlowsCred"`
	IncludePackageJSON  bool   `json:"includePackageJson"`
}

// BackupSchedulerStatus reports the runtime scheduler state.
type BackupSchedulerStatus struct {
	Enabled        bool   `json:"enabled"`
	Scheduled      bool   `json:"scheduled"`
	Schedule       string `json:"schedule"`
	CustomSchedule string `json:"customSchedule,omitempty"`
	ActiveSpec     string `json:"activeSpec,omitempty"`
	NextRunAt      string `json:"nextRunAt,omitempty"`
	LastRunAt      string `json:"lastRunAt,omitempty"`
	LastSuccessAt  string `json:"lastSuccessAt,omitempty"`
	LastBackupID   string `json:"lastBackupId,omitempty"`
	LastError      string `json:"lastError,omitempty"`
}

// BackupEventType identifies an observability event emitted by backup flows.
type BackupEventType string

const (
	BackupEventTypeManualCreate     BackupEventType = "manual-create"
	BackupEventTypeAutoCreate       BackupEventType = "auto-create"
	BackupEventTypePreRestoreCreate BackupEventType = "pre-restore-create"
	BackupEventTypeRestore          BackupEventType = "restore"
	BackupEventTypeDelete           BackupEventType = "delete"
	BackupEventTypePrune            BackupEventType = "prune"
	BackupEventTypeSchedulerConfig  BackupEventType = "scheduler-config"
	BackupEventTypeSchedulerRun     BackupEventType = "scheduler-run"
	BackupEventTypeSchedulerError   BackupEventType = "scheduler-error"
)

// BackupEvent captures a recent backup/scheduler event for observability.
type BackupEvent struct {
	ID          string          `json:"id"`
	Type        BackupEventType `json:"type"`
	Status      string          `json:"status"`
	OccurredAt  string          `json:"occurredAt"`
	BackupID    string          `json:"backupId,omitempty"`
	BackupName  string          `json:"backupName,omitempty"`
	BackupType  BackupType      `json:"backupType,omitempty"`
	Message     string          `json:"message,omitempty"`
	Schedule    string          `json:"schedule,omitempty"`
	ActiveSpec  string          `json:"activeSpec,omitempty"`
	Trigger     string          `json:"trigger,omitempty"`
	PrunedCount int             `json:"prunedCount,omitempty"`
	PrunedIDs   []string        `json:"prunedIds,omitempty"`
	Error       string          `json:"error,omitempty"`
}

// BackupObservability summarizes current backup health plus recent events.
type BackupObservability struct {
	Scheduler    BackupSchedulerStatus `json:"scheduler"`
	Storage      BackupStorageInfo     `json:"storage"`
	LatestBackup *Backup               `json:"latestBackup,omitempty"`
	RecentEvents []BackupEvent         `json:"recentEvents"`
}

// SchedulerConfigRequest is the request body for POST /api/scheduler/config.
type SchedulerConfigRequest struct {
	Cron string `json:"cron"`
}

// SchedulerConfigResponse is the response body for POST /api/scheduler/config.
type SchedulerConfigResponse struct {
	Cron  string `json:"cron"`
	Valid bool   `json:"valid"`
}

// SchedulerHistoryEntry represents a single scheduler run in the history.
type SchedulerHistoryEntry struct {
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"` // "success", "failure", "skipped"
	Error     string `json:"error,omitempty"`
}

// PaginatedSchedulerHistory is the response wrapper for paginated scheduler history.
type PaginatedSchedulerHistory struct {
	Entries []SchedulerHistoryEntry `json:"entries"`
	Total   int                     `json:"total"`
	Page    int                     `json:"page"`
	Limit   int                     `json:"limit"`
}

// RetentionConfigRequest is the request body for PATCH /api/storage/retention.
type RetentionConfigRequest struct {
	RetentionDays int `json:"retentionDays"`
}

// RetentionConfigResponse is the response body for PATCH /api/storage/retention.
type RetentionConfigResponse struct {
	RetentionDays int `json:"retentionDays"`
}
