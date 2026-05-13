package model

import "time"

// UpdateState represents the current state of an update flow.
type UpdateState string

const (
	StateIdle      UpdateState = "Idle"
	StateBackingUp UpdateState = "BackingUp"
	StateApplying  UpdateState = "Applying"
	StateCompleted UpdateState = "Completed"
	StateFailed    UpdateState = "Failed"
)

// UpdateFlowState represents the current state of an update operation.
// This struct tracks multi-stage update flows: BackingUp → Applying → Completed/Failed.
type UpdateFlowState struct {
	State            UpdateState `json:"state"`
	Phase            string      `json:"phase"`
	BackupID         string      `json:"backupId,omitempty"`
	Error            string      `json:"error,omitempty"`
	AvailableVersion string      `json:"availableVersion,omitempty"`
}

// BackupEntry represents a single backup of Node-RED user data.
// Status values: "pending", "completed", "failed"
type BackupEntry struct {
	ID          string    `json:"id"`
	Path        string    `json:"path"`
	SizeBytes   int64     `json:"sizeBytes"`
	Timestamp   time.Time `json:"timestamp"`
	FromVersion string    `json:"fromVersion"`
	Status      string    `json:"status"` // "pending" | "completed" | "failed"
}

// UpdateCacheEntry is the persisted and in-memory cache for update status.
// This replaces the old UpdateStatus (which lacked checkedAt/error).
//
// Cache Location: ./data/update_cache.json
// Cache Format: JSON (see example below)
// Cache Lifetime: Persists across server restarts; updated every 4 hours by background polling
// Cache Invalidation: Automatic refresh at poll interval; manual refresh via GET /api/updates/check
//
// Example cache file (./data/update_cache.json):
//
//	{
//	  "currentVersion": "4.0.1",
//	  "latestVersion": "4.0.2",
//	  "updateAvailable": true,
//	  "checkedAt": "2026-05-09T14:30:00Z",
//	  "error": ""
//	}
//
// Example error state (when npm fails):
//
//	{
//	  "currentVersion": "4.0.1",
//	  "latestVersion": "",
//	  "updateAvailable": false,
//	  "checkedAt": "2026-05-09T14:35:00Z",
//	  "error": "context deadline exceeded"
//	}
type UpdateCacheEntry struct {
	CurrentVersion  string    `json:"currentVersion"`
	LatestVersion   string    `json:"latestVersion"`
	UpdateAvailable bool      `json:"updateAvailable"`
	CheckedAt       time.Time `json:"checkedAt"`
	Error           string    `json:"error,omitempty"` // last check error; empty if success
}

// UpdateApplyResult is returned by POST /api/updates/apply.
type UpdateApplyResult struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	FromVersion string `json:"fromVersion"`
	ToVersion   string `json:"toVersion"`
	BackupID    string `json:"backupId,omitempty"`
}
