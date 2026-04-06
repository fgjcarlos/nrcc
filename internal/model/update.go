package model

type UpdateStatus struct {
	InstalledVersion string `json:"installedVersion"`
	AvailableVersion string `json:"availableVersion"`
	UpdateAvailable  bool   `json:"updateAvailable"`
}

type UpdateApplyResult struct {
	FromVersion       string `json:"fromVersion"`
	ToVersion         string `json:"toVersion"`
	PreventiveBackupID string `json:"preventiveBackupId"`
	RolledBack        bool   `json:"rolledBack"`
	Message           string `json:"message"`
}
