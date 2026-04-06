package model

type BackupFile struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	SHA256    string `json:"sha256"`
}

type BackupManifest struct {
	ID             string       `json:"id"`
	SchemaVersion  int          `json:"schemaVersion"`
	Reason         string       `json:"reason"`
	CreatedAt      string       `json:"createdAt"`
	ArchiveName    string       `json:"archiveName"`
	ArchiveSHA256  string       `json:"archiveSha256"`
	ArchiveBytes   int64        `json:"archiveBytes"`
	Files          []BackupFile `json:"files"`
}

type BackupSummary struct {
	ID            string `json:"id"`
	Reason        string `json:"reason"`
	CreatedAt     string `json:"createdAt"`
	ArchiveName   string `json:"archiveName"`
	ArchiveBytes  int64  `json:"archiveBytes"`
	ArchiveSHA256 string `json:"archiveSha256"`
}

type BackupList struct {
	Items []BackupSummary `json:"items"`
}
