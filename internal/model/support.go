package model

import "time"

// SupportBundleManifest represents metadata about a generated support bundle
type SupportBundleManifest struct {
	FileName    string    `json:"fileName"`
	GeneratedAt time.Time `json:"generatedAt"`
	BundleSize  int64     `json:"bundleSizeBytes"`
	FileCount   int       `json:"fileCount"`
	GeneratedBy string    `json:"generatedBy"`
}
