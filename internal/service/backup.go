package service

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/google/uuid"
)

const backupConfigFile = "backup_config.json"

var defaultBackupConfig = model.BackupConfig{
	Enabled:             false,
	Schedule:            "disabled",
	RetentionManual:     10,
	RetentionAuto:       30,
	RetentionPreRestore: 5,
	IncludeConfig:       true,
	IncludeSettings:     true,
	IncludeFlowsCred:    true,
	IncludePackageJSON:  true,
}

var ErrInvalidBackupConfig = errors.New("invalid backup config")

// BackupService handles backup operations.
type BackupService struct {
	dataDir    string
	scheduler  *backupScheduler
	eventStore *backupEventStore
}

type createBackupOptions struct {
	Type model.BackupType
	Name string
}

type backupMetadata struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Type        model.BackupType `json:"type"`
	CreatedAt   string           `json:"createdAt"`
	TriggeredBy string           `json:"triggeredBy"`
}

// NewBackupService creates a new backup service.
func NewBackupService(dataDir string) *BackupService {
	svc := &BackupService{dataDir: dataDir}
	svc.eventStore = newBackupEventStore(dataDir)
	svc.scheduler = newBackupScheduler(svc)
	return svc
}

// Start begins background scheduler processing for automatic backups.
func (s *BackupService) Start(ctx context.Context) {
	if s.scheduler != nil {
		s.scheduler.Start(ctx)
	}
}

// SchedulerStatus returns the current runtime scheduler state.
func (s *BackupService) SchedulerStatus() model.BackupSchedulerStatus {
	if s.scheduler == nil {
		return model.BackupSchedulerStatus{Schedule: defaultBackupConfig.Schedule}
	}
	return s.scheduler.Status()
}

// Observability returns scheduler status, storage summary, latest backup and recent events.
func (s *BackupService) Observability() (model.BackupObservability, error) {
	storage, err := s.Storage()
	if err != nil {
		return model.BackupObservability{}, err
	}

	backups, err := s.List()
	if err != nil {
		return model.BackupObservability{}, err
	}

	events := make([]model.BackupEvent, 0)
	if s.eventStore != nil {
		events, err = s.eventStore.List()
		if err != nil {
			return model.BackupObservability{}, err
		}
	}

	result := model.BackupObservability{
		Scheduler:    s.SchedulerStatus(),
		Storage:      storage,
		RecentEvents: events,
	}
	if len(backups) > 0 {
		latest := backups[0]
		result.LatestBackup = &latest
	}

	return result, nil
}

// List returns all available backups.
func (s *BackupService) List() ([]model.Backup, error) {
	backupDir := filepath.Join(s.dataDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backups directory: %w", err)
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backups directory: %w", err)
	}

	backups := make([]model.Backup, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".zip") {
			continue
		}

		backup, err := s.describeBackup(filepath.Join(backupDir, entry.Name()))
		if err != nil {
			continue
		}
		backups = append(backups, backup)
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt > backups[j].CreatedAt
	})

	return backups, nil
}

// ListPaginated returns paginated backups with sorting and filtering.
// Implements Task 1.2-1.3: pagination logic with defaults, clamping, sorting
func (s *BackupService) ListPaginated(opts model.PaginationOpts) (model.PaginatedBackups, error) {
	// Get all backups using existing List method
	allBackups, err := s.List()
	if err != nil {
		return model.PaginatedBackups{}, err
	}

	total := len(allBackups)

	// Clamp and normalize pagination parameters
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.Limit < 1 {
		opts.Limit = 20
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	// Default sort and order
	if opts.Sort == "" {
		opts.Sort = "date"
	}
	if opts.Order == "" {
		opts.Order = "desc"
	}

	// Apply sorting (overrides the default sort.Slice from List())
	// Re-sort based on opts.Sort if not "date"
	if opts.Sort == "size" {
		sort.Slice(allBackups, func(i, j int) bool {
			if opts.Order == "asc" {
				return allBackups[i].SizeBytes < allBackups[j].SizeBytes
			}
			return allBackups[i].SizeBytes > allBackups[j].SizeBytes
		})
	} else if opts.Sort == "status" {
		// Sort by Type (status proxy)
		sort.Slice(allBackups, func(i, j int) bool {
			if opts.Order == "asc" {
				return allBackups[i].Type < allBackups[j].Type
			}
			return allBackups[i].Type > allBackups[j].Type
		})
	} else {
		// Default: sort by date
		sort.Slice(allBackups, func(i, j int) bool {
			if opts.Order == "asc" {
				return allBackups[i].CreatedAt < allBackups[j].CreatedAt
			}
			return allBackups[i].CreatedAt > allBackups[j].CreatedAt
		})
	}

	// Calculate pagination offsets
	offset := (opts.Page - 1) * opts.Limit

	// Handle out-of-range pages
	if offset >= total && total > 0 {
		// Return empty page but still valid response
		return model.PaginatedBackups{
			Items: []model.Backup{},
			Total: total,
			Page:  opts.Page,
			Limit: opts.Limit,
		}, nil
	}

	// Slice the backups
	end := offset + opts.Limit
	if end > total {
		end = total
	}

	items := allBackups
	if offset < total {
		items = allBackups[offset:end]
	} else {
		items = []model.Backup{}
	}

	return model.PaginatedBackups{
		Items: items,
		Total: total,
		Page:  opts.Page,
		Limit: opts.Limit,
	}, nil
}

// Create creates a new backup using the default manual type.
func (s *BackupService) Create(name string) (model.Backup, error) {
	return s.createBackup(createBackupOptions{Type: model.BackupTypeManual, Name: name})
}

// CreateTyped creates a new backup with an explicit type.
func (s *BackupService) CreateTyped(backupType model.BackupType, name string) (model.Backup, error) {
	return s.createBackup(createBackupOptions{Type: normalizeBackupType(string(backupType)), Name: name})
}

// ValidateBackupID rejects identifiers that could escape the backups directory
// when joined into a file path. Backup ids are server-generated (uuids / typed
// names), so requiring a single clean path component is safe and sufficient.
func ValidateBackupID(id string) error {
	if id == "" || id != filepath.Base(id) || strings.Contains(id, "..") {
		return fmt.Errorf("invalid backup id: %q", id)
	}
	return nil
}

// Detail returns authoritative metadata for a backup file.
func (s *BackupService) Detail(id string) (model.BackupManifest, error) {
	if err := ValidateBackupID(id); err != nil {
		return model.BackupManifest{}, err
	}
	backupPath := filepath.Join(s.dataDir, "backups", id+".zip")
	manifest, err := s.inspectBackup(backupPath)
	if err != nil {
		return model.BackupManifest{}, err
	}
	return manifest, nil
}

// Storage returns aggregate local backup stats.
func (s *BackupService) Storage() (model.BackupStorageInfo, error) {
	backups, err := s.List()
	if err != nil {
		return model.BackupStorageInfo{}, err
	}

	storage := model.BackupStorageInfo{}
	for _, backup := range backups {
		storage.TotalBackups++
		storage.TotalSize += backup.SizeBytes

		switch backup.Type {
		case model.BackupTypeAuto:
			storage.AutoCount++
		case model.BackupTypePreRestore:
			storage.PreRestoreCount++
		default:
			storage.ManualCount++
		}
	}

	return storage, nil
}

// GetConfig loads persisted backup config or defaults.
func (s *BackupService) GetConfig() (model.BackupConfig, error) {
	path := filepath.Join(s.dataDir, backupConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultBackupConfig, nil
		}
		return model.BackupConfig{}, fmt.Errorf("failed to read backup config: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return model.BackupConfig{}, fmt.Errorf("failed to parse backup config: %w", err)
	}

	cfg := defaultBackupConfig
	decodeJSONField(raw, "enabled", &cfg.Enabled)
	decodeJSONField(raw, "schedule", &cfg.Schedule)
	decodeJSONField(raw, "customSchedule", &cfg.CustomSchedule)
	decodeJSONField(raw, "retentionManual", &cfg.RetentionManual)
	decodeJSONField(raw, "retentionAuto", &cfg.RetentionAuto)
	decodeJSONField(raw, "retentionPreRestore", &cfg.RetentionPreRestore)
	decodeJSONField(raw, "includeConfig", &cfg.IncludeConfig)
	decodeJSONField(raw, "includeSettings", &cfg.IncludeSettings)
	decodeJSONField(raw, "includeFlowsCred", &cfg.IncludeFlowsCred)
	decodeJSONField(raw, "includePackageJson", &cfg.IncludePackageJSON)

	return normalizeBackupConfig(cfg), nil
}

// SaveConfig persists backup config and returns the normalized result.
func (s *BackupService) SaveConfig(cfg model.BackupConfig) (model.BackupConfig, error) {
	normalized := normalizeBackupConfig(cfg)
	if err := validateBackupConfig(normalized); err != nil {
		return model.BackupConfig{}, err
	}
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return model.BackupConfig{}, fmt.Errorf("failed to create data directory: %w", err)
	}

	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return model.BackupConfig{}, fmt.Errorf("failed to encode backup config: %w", err)
	}

	path := filepath.Join(s.dataDir, backupConfigFile)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return model.BackupConfig{}, fmt.Errorf("failed to save backup config: %w", err)
	}

	if s.scheduler != nil {
		if err := s.scheduler.ApplyConfig(normalized); err != nil {
			return model.BackupConfig{}, err
		}
	}

	s.recordEvent(model.BackupEvent{
		Type:       model.BackupEventTypeSchedulerConfig,
		Status:     "success",
		Message:    "Backup scheduler configuration updated",
		Schedule:   normalized.Schedule,
		ActiveSpec: s.SchedulerStatus().ActiveSpec,
		Trigger:    "config-save",
	})

	return normalized, nil
}

// Restore restores a backup by ID.
func (s *BackupService) Restore(id string) error {
	if err := ValidateBackupID(id); err != nil {
		return err
	}
	backupDir := filepath.Join(s.dataDir, "backups")
	backupPath := filepath.Join(backupDir, id+".zip")

	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	zipFile, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup: %w", err)
	}
	defer zipFile.Close()

	zipReader, err := zip.NewReader(zipFile, int64(getFileSize(backupPath)))
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	for _, file := range zipReader.File {
		if err := s.extractFileFromZip(file, s.dataDir); err != nil {
			return fmt.Errorf("failed to extract file from backup: %w", err)
		}
	}

	return nil
}

// RestoreWithSafetyBackup creates a pre-restore snapshot before restoring.
func (s *BackupService) RestoreWithSafetyBackup(id string) (string, error) {
	if err := ValidateBackupID(id); err != nil {
		return "", err
	}
	backupPath := filepath.Join(s.dataDir, "backups", id+".zip")
	if _, err := os.Stat(backupPath); err != nil {
		return "", fmt.Errorf("backup not found: %w", err)
	}
	restoredBackup, _ := s.describeBackup(backupPath)

	preRestore, err := s.CreateTyped(model.BackupTypePreRestore, "pre-restore")
	if err != nil {
		return "", err
	}

	if err := s.Restore(id); err != nil {
		s.recordEvent(model.BackupEvent{
			Type:       model.BackupEventTypeRestore,
			Status:     "error",
			BackupID:   id,
			BackupName: restoredBackup.Name,
			BackupType: restoredBackup.Type,
			Message:    "Backup restore failed",
			Trigger:    "restore",
			Error:      err.Error(),
		})
		return preRestore.ID, err
	}

	s.recordEvent(model.BackupEvent{
		Type:       model.BackupEventTypeRestore,
		Status:     "success",
		BackupID:   id,
		BackupName: restoredBackup.Name,
		BackupType: restoredBackup.Type,
		Message:    "Backup restored successfully",
		Trigger:    "restore",
	})

	return preRestore.ID, nil
}

// Delete deletes a backup by ID.
func (s *BackupService) Delete(id string) error {
	if err := ValidateBackupID(id); err != nil {
		return err
	}
	backupDir := filepath.Join(s.dataDir, "backups")
	backupPath := filepath.Join(backupDir, id+".zip")
	backup, _ := s.describeBackup(backupPath)

	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	s.recordEvent(model.BackupEvent{
		Type:       model.BackupEventTypeDelete,
		Status:     "success",
		BackupID:   id,
		BackupName: backup.Name,
		BackupType: backup.Type,
		Message:    "Backup deleted",
		Trigger:    "delete",
	})

	return nil
}

// OpenForDownload validates the id, confirms the backup exists and opens it,
// returning a reader and the file size. Callers (HTTP handlers) can use this to
// detect missing/unreadable backups and set Content-Length BEFORE writing any
// response body, avoiding truncated downloads served with a 200 status.
func (s *BackupService) OpenForDownload(id string) (io.ReadCloser, int64, error) {
	if err := ValidateBackupID(id); err != nil {
		return nil, 0, err
	}
	backupPath := filepath.Join(s.dataDir, "backups", id+".zip")

	info, err := os.Stat(backupPath)
	if err != nil {
		return nil, 0, fmt.Errorf("backup not found: %w", err)
	}

	file, err := os.Open(backupPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open backup: %w", err)
	}
	return file, info.Size(), nil
}

// Download streams a backup file to the response writer.
func (s *BackupService) Download(id string, w io.Writer) error {
	rc, _, err := s.OpenForDownload(id)
	if err != nil {
		return err
	}
	defer rc.Close()

	if _, err := io.Copy(w, rc); err != nil {
		return fmt.Errorf("failed to download backup: %w", err)
	}

	return nil
}

func (s *BackupService) createBackup(options createBackupOptions) (model.Backup, error) {
	backupDir := filepath.Join(s.dataDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return model.Backup{}, fmt.Errorf("failed to create backups directory: %w", err)
	}

	config, err := s.GetConfig()
	if err != nil {
		config = defaultBackupConfig
	}

	backupType := options.Type
	if backupType == "" {
		backupType = model.BackupTypeManual
	}

	backupID := uuid.New().String()
	backupPath := filepath.Join(backupDir, backupID+".zip")
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	name := strings.TrimSpace(options.Name)
	if name == "" {
		name = defaultBackupName(backupType, createdAt)
	}

	zipFile, err := os.Create(backupPath)
	if err != nil {
		return model.Backup{}, fmt.Errorf("failed to create backup file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	metadata := backupMetadata{
		ID:          backupID,
		Name:        name,
		Type:        backupType,
		CreatedAt:   createdAt,
		TriggeredBy: name,
	}

	filesToBackup := s.filesForBackup(config)

	if err := s.addMetadataToZip(zipWriter, metadata); err != nil {
		os.Remove(backupPath)
		return model.Backup{}, err
	}

	fileCount := 1
	for _, file := range filesToBackup {
		added, err := s.addFileToZip(zipWriter, file.src, file.dst)
		if err != nil {
			os.Remove(backupPath)
			return model.Backup{}, err
		}
		if added {
			fileCount++
		}
	}

	if err := zipWriter.Close(); err != nil {
		os.Remove(backupPath)
		return model.Backup{}, fmt.Errorf("failed to finalize backup: %w", err)
	}

	info, err := os.Stat(backupPath)
	if err != nil {
		return model.Backup{}, fmt.Errorf("failed to stat backup: %w", err)
	}

	backup := model.Backup{
		ID:          backupID,
		Name:        name,
		Type:        backupType,
		CreatedAt:   createdAt,
		TriggeredBy: name,
		FileCount:   fileCount,
		SizeBytes:   info.Size(),
		Path:        backupPath,
	}

	prunedIDs, pruneErr := s.pruneBackups(config, backupType, backupID)
	s.recordBackupCreated(backup, prunedIDs)

	return backup, pruneErr
}

func (s *BackupService) filesForBackup(cfg model.BackupConfig) []struct {
	src string
	dst string
} {
	files := []struct {
		src string
		dst string
	}{
		{filepath.Join(s.dataDir, "flows.json"), "flows.json"},
		{filepath.Join(s.dataDir, "cc-users.json"), "cc-users.json"},
	}

	if cfg.IncludeConfig {
		files = append(files, struct {
			src string
			dst string
		}{filepath.Join(s.dataDir, "config.json"), "config.json"})
	}
	if cfg.IncludeSettings {
		files = append(files, struct {
			src string
			dst string
		}{filepath.Join(s.dataDir, "settings.js"), "settings.js"})
	}
	if cfg.IncludeFlowsCred {
		files = append(files, struct {
			src string
			dst string
		}{filepath.Join(s.dataDir, "flows_cred.json"), "flows_cred.json"})
	}
	if cfg.IncludePackageJSON {
		files = append(files, struct {
			src string
			dst string
		}{filepath.Join(s.dataDir, "package.json"), "package.json"})
	}

	return files
}

func (s *BackupService) pruneBackups(cfg model.BackupConfig, backupType model.BackupType, keepID string) ([]string, error) {
	limit := retentionForType(cfg, backupType)
	if limit <= 0 {
		return nil, nil
	}

	backups, err := s.List()
	if err != nil {
		return nil, fmt.Errorf("prune backups: %w", err)
	}

	matching := make([]model.Backup, 0, len(backups))
	for _, backup := range backups {
		if backup.Type == backupType {
			matching = append(matching, backup)
		}
	}

	if len(matching) <= limit {
		return nil, nil
	}

	sort.Slice(matching, func(i, j int) bool {
		if matching[i].CreatedAt == matching[j].CreatedAt {
			leftInfo, leftErr := os.Stat(matching[i].Path)
			rightInfo, rightErr := os.Stat(matching[j].Path)
			if leftErr == nil && rightErr == nil && !leftInfo.ModTime().Equal(rightInfo.ModTime()) {
				return leftInfo.ModTime().After(rightInfo.ModTime())
			}
			return matching[i].ID > matching[j].ID
		}
		return matching[i].CreatedAt > matching[j].CreatedAt
	})

	prunedIDs := make([]string, 0, len(matching)-limit)
	for _, backup := range matching[limit:] {
		if backup.ID == keepID {
			continue
		}
		if err := os.Remove(backup.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return prunedIDs, fmt.Errorf("prune backup %s: %w", backup.ID, err)
		}
		prunedIDs = append(prunedIDs, backup.ID)
	}

	return prunedIDs, nil
}

func (s *BackupService) recordBackupCreated(backup model.Backup, prunedIDs []string) {
	eventType := model.BackupEventTypeManualCreate
	trigger := "manual"
	message := "Manual backup created"

	switch backup.Type {
	case model.BackupTypeAuto:
		eventType = model.BackupEventTypeAutoCreate
		trigger = "scheduler"
		message = "Automatic backup created"
	case model.BackupTypePreRestore:
		eventType = model.BackupEventTypePreRestoreCreate
		trigger = "pre-restore"
		message = "Pre-restore safety backup created"
	}

	s.recordEvent(model.BackupEvent{
		Type:       eventType,
		Status:     "success",
		OccurredAt: backup.CreatedAt,
		BackupID:   backup.ID,
		BackupName: backup.Name,
		BackupType: backup.Type,
		Message:    message,
		Trigger:    trigger,
	})

	if len(prunedIDs) > 0 {
		s.recordEvent(model.BackupEvent{
			Type:        model.BackupEventTypePrune,
			Status:      "success",
			BackupType:  backup.Type,
			Message:     "Retention policy pruned older backups",
			Trigger:     "retention",
			PrunedCount: len(prunedIDs),
			PrunedIDs:   prunedIDs,
		})
	}
}

func (s *BackupService) recordEvent(event model.BackupEvent) {
	if s.eventStore == nil {
		return
	}
	_ = s.eventStore.Append(event)
}

func (s *BackupService) describeBackup(backupPath string) (model.Backup, error) {
	manifest, err := s.inspectBackup(backupPath)
	if err != nil {
		return model.Backup{}, err
	}

	return model.Backup{
		ID:          manifest.ID,
		Name:        manifest.Name,
		Type:        manifest.Type,
		CreatedAt:   manifest.CreatedAt,
		TriggeredBy: manifest.TriggeredBy,
		FileCount:   len(manifest.Files),
		SizeBytes:   manifest.TotalSize,
		Path:        backupPath,
	}, nil
}

func (s *BackupService) inspectBackup(backupPath string) (model.BackupManifest, error) {
	info, err := os.Stat(backupPath)
	if err != nil {
		return model.BackupManifest{}, fmt.Errorf("backup not found: %w", err)
	}

	file, err := os.Open(backupPath)
	if err != nil {
		return model.BackupManifest{}, fmt.Errorf("failed to open backup: %w", err)
	}
	defer file.Close()

	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return model.BackupManifest{}, fmt.Errorf("failed to inspect backup: %w", err)
	}

	id := strings.TrimSuffix(filepath.Base(backupPath), ".zip")
	manifest := model.BackupManifest{
		ID:        id,
		Name:      id,
		Type:      inferBackupType(id),
		CreatedAt: info.ModTime().UTC().Format(time.RFC3339),
		TotalSize: info.Size(),
		Files:     make([]model.BackupFileEntry, 0, len(reader.File)),
	}

	for _, zippedFile := range reader.File {
		if zippedFile.FileInfo().IsDir() {
			continue
		}

		if zippedFile.Name == "backup-metadata.json" {
			metadata, err := readBackupMetadata(zippedFile)
			if err == nil {
				manifest.ID = firstNonEmpty(metadata.ID, manifest.ID)
				manifest.Name = firstNonEmpty(metadata.Name, manifest.Name)
				manifest.CreatedAt = firstNonEmpty(metadata.CreatedAt, manifest.CreatedAt)
				manifest.TriggeredBy = firstNonEmpty(metadata.TriggeredBy, metadata.Name)
				manifest.Type = normalizeBackupType(string(metadata.Type))
			}
			continue
		}

		checksum, err := checksumZipFile(zippedFile)
		if err != nil {
			checksum = ""
		}

		manifest.Files = append(manifest.Files, model.BackupFileEntry{
			Path:     zippedFile.Name,
			Size:     int64(zippedFile.UncompressedSize64),
			Checksum: checksum,
		})
	}

	if manifest.Type == "" {
		manifest.Type = inferBackupType(manifest.Name)
	}
	if manifest.TriggeredBy == "" {
		manifest.TriggeredBy = manifest.Name
	}

	sort.Slice(manifest.Files, func(i, j int) bool {
		return manifest.Files[i].Path < manifest.Files[j].Path
	})

	return manifest, nil
}

func (s *BackupService) addMetadataToZip(zipWriter *zip.Writer, metadata backupMetadata) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to encode backup metadata: %w", err)
	}

	writer, err := zipWriter.Create("backup-metadata.json")
	if err != nil {
		return fmt.Errorf("failed to create backup metadata: %w", err)
	}

	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("failed to write backup metadata: %w", err)
	}

	return nil
}

func (s *BackupService) addFileToZip(zipWriter *zip.Writer, srcPath, dstPath string) (bool, error) {
	file, err := os.Open(srcPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("failed to open %s: %w", srcPath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return false, fmt.Errorf("failed to stat %s: %w", srcPath, err)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return false, fmt.Errorf("failed to build zip header for %s: %w", srcPath, err)
	}
	header.Name = dstPath
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return false, fmt.Errorf("failed to add %s to zip: %w", srcPath, err)
	}

	if _, err := io.Copy(writer, file); err != nil {
		return false, fmt.Errorf("failed to write %s to zip: %w", srcPath, err)
	}

	return true, nil
}

// maxBackupEntrySize caps the uncompressed size of a single backup entry to
// guard against decompression bombs during restore. It is a var (not const) so
// tests can lower it. 200 MiB comfortably covers real Node-RED data files.
var maxBackupEntrySize int64 = 200 * 1024 * 1024

func (s *BackupService) extractFileFromZip(file *zip.File, destDir string) error {
	if file.FileInfo().IsDir() || file.Name == "backup-metadata.json" {
		return nil
	}

	// Reject symlinks and hardlinks.
	if file.FileInfo().Mode()&(os.ModeSymlink|os.ModeNamedPipe|os.ModeDevice) != 0 {
		return fmt.Errorf("unsafe archive entry type: %s", file.Name)
	}

	// Decompression-bomb guard: reject entries whose declared uncompressed size
	// already exceeds the limit before reading a single byte.
	if file.UncompressedSize64 > uint64(maxBackupEntrySize) {
		return fmt.Errorf("archive entry %s exceeds maximum allowed size", file.Name)
	}

	destPath, err := sanitizeArchivePath(destDir, file.Name)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	writer, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer writer.Close()

	// The declared size can lie; cap the actual bytes copied. Reading one byte
	// past the limit means the entry is oversized.
	written, err := io.Copy(writer, io.LimitReader(reader, maxBackupEntrySize+1))
	if err != nil {
		return err
	}
	if written > maxBackupEntrySize {
		writer.Close()
		os.Remove(destPath)
		return fmt.Errorf("archive entry %s exceeds maximum allowed size", file.Name)
	}
	return nil
}

// sanitizeArchivePath validates that a zip entry resolves within destDir.
func sanitizeArchivePath(destDir, entryName string) (string, error) {
	if filepath.IsAbs(entryName) {
		return "", fmt.Errorf("absolute path not allowed: %s", entryName)
	}

	destPath := filepath.Join(destDir, filepath.Clean(entryName))

	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		return "", err
	}
	absDestPath, err := filepath.Abs(destPath)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(absDestPath, absDestDir+string(os.PathSeparator)) {
		return "", fmt.Errorf("path traversal detected: %s", entryName)
	}

	return destPath, nil
}

func normalizeBackupConfig(cfg model.BackupConfig) model.BackupConfig {
	normalized := defaultBackupConfig
	normalized.Enabled = cfg.Enabled
	normalized.Schedule = normalizeSchedule(cfg.Schedule)
	normalized.CustomSchedule = strings.TrimSpace(cfg.CustomSchedule)
	normalized.RetentionManual = maxInt(1, cfg.RetentionManual, defaultBackupConfig.RetentionManual)
	normalized.RetentionAuto = maxInt(1, cfg.RetentionAuto, defaultBackupConfig.RetentionAuto)
	normalized.RetentionPreRestore = maxInt(1, cfg.RetentionPreRestore, defaultBackupConfig.RetentionPreRestore)
	normalized.IncludeConfig = cfg.IncludeConfig
	normalized.IncludeSettings = cfg.IncludeSettings
	normalized.IncludeFlowsCred = cfg.IncludeFlowsCred
	normalized.IncludePackageJSON = cfg.IncludePackageJSON

	if normalized.Schedule == "disabled" {
		normalized.Enabled = false
	}
	if normalized.Schedule != "custom" {
		normalized.CustomSchedule = ""
	}

	return normalized
}

func normalizeSchedule(schedule string) string {
	value := strings.TrimSpace(strings.ToLower(schedule))
	switch value {
	case "disabled", "hourly", "every6h", "daily", "weekly", "custom":
		return value
	default:
		return defaultBackupConfig.Schedule
	}
}

func validateBackupConfig(cfg model.BackupConfig) error {
	_, err := scheduleSpec(cfg)
	return err
}

func scheduleSpec(cfg model.BackupConfig) (string, error) {
	normalized := normalizeBackupConfig(cfg)
	if !normalized.Enabled || normalized.Schedule == "disabled" {
		return "", nil
	}

	switch normalized.Schedule {
	case "hourly":
		return "0 * * * *", nil
	case "every6h":
		return "0 */6 * * *", nil
	case "daily":
		return "0 2 * * *", nil
	case "weekly":
		return "0 2 * * 0", nil
	case "custom":
		if normalized.CustomSchedule == "" {
			return "", fmt.Errorf("%w: customSchedule is required when schedule is custom", ErrInvalidBackupConfig)
		}
		if _, err := backupCronParser.Parse(normalized.CustomSchedule); err != nil {
			return "", fmt.Errorf("%w: invalid customSchedule: %v", ErrInvalidBackupConfig, err)
		}
		return normalized.CustomSchedule, nil
	default:
		return "", fmt.Errorf("%w: unsupported schedule %q", ErrInvalidBackupConfig, normalized.Schedule)
	}
}

func normalizeBackupType(value string) model.BackupType {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case string(model.BackupTypeAuto):
		return model.BackupTypeAuto
	case string(model.BackupTypePreRestore):
		return model.BackupTypePreRestore
	default:
		return model.BackupTypeManual
	}
}

func inferBackupType(name string) model.BackupType {
	normalizedName := strings.ToLower(name)
	if strings.Contains(normalizedName, "pre-restore") || strings.Contains(normalizedName, "prerestore") {
		return model.BackupTypePreRestore
	}
	if strings.Contains(normalizedName, "auto") {
		return model.BackupTypeAuto
	}
	return model.BackupTypeManual
}

func defaultBackupName(backupType model.BackupType, createdAt string) string {
	suffix := strings.ReplaceAll(strings.ReplaceAll(createdAt, ":", "-"), "T", "_")
	suffix = strings.TrimSuffix(suffix, "Z")
	if backupType == model.BackupTypeManual {
		return "manual-" + suffix
	}
	return string(backupType) + "-" + suffix
}

func readBackupMetadata(file *zip.File) (backupMetadata, error) {
	reader, err := file.Open()
	if err != nil {
		return backupMetadata{}, err
	}
	defer reader.Close()

	var metadata backupMetadata
	if err := json.NewDecoder(reader).Decode(&metadata); err != nil {
		return backupMetadata{}, err
	}

	metadata.Type = normalizeBackupType(string(metadata.Type))
	return metadata, nil
}

func checksumZipFile(file *zip.File) (string, error) {
	reader, err := file.Open()
	if err != nil {
		return "", err
	}
	defer reader.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, reader); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func decodeJSONField[T any](raw map[string]json.RawMessage, key string, target *T) {
	value, ok := raw[key]
	if !ok {
		return
	}
	_ = json.Unmarshal(value, target)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func maxInt(minimum int, value int, fallback int) int {
	if value <= 0 {
		value = fallback
	}
	if value < minimum {
		return minimum
	}
	return value
}

func retentionForType(cfg model.BackupConfig, backupType model.BackupType) int {
	switch backupType {
	case model.BackupTypeAuto:
		return cfg.RetentionAuto
	case model.BackupTypePreRestore:
		return cfg.RetentionPreRestore
	default:
		return cfg.RetentionManual
	}
}

func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// RecordSchedulerEvent records a single scheduler execution event in history.
func (s *BackupService) RecordSchedulerEvent(entry model.SchedulerHistoryEntry) {
	if s.eventStore == nil {
		return
	}
	// Store in eventStore as a backup event
	event := model.BackupEvent{
		ID:         uuid.New().String(),
		Type:       model.BackupEventTypeSchedulerRun,
		Status:     entry.Status,
		OccurredAt: entry.Timestamp,
		Error:      entry.Error,
	}
	s.recordEvent(event)
}

// GetSchedulerHistory returns paginated scheduler execution history from recent events.
func (s *BackupService) GetSchedulerHistory(opts model.PaginationOpts) (model.PaginatedSchedulerHistory, error) {
	if s.eventStore == nil {
		return model.PaginatedSchedulerHistory{}, nil
	}

	// Get all events
	allEvents, err := s.eventStore.List()
	if err != nil {
		return model.PaginatedSchedulerHistory{}, err
	}

	// Filter to only scheduler-related events
	var schedulerEvents []model.BackupEvent
	for _, event := range allEvents {
		if event.Type == model.BackupEventTypeSchedulerRun ||
			event.Type == model.BackupEventTypeSchedulerError {
			schedulerEvents = append(schedulerEvents, event)
		}
	}

	// Sort by date descending (most recent first)
	sort.Slice(schedulerEvents, func(i, j int) bool {
		return schedulerEvents[i].OccurredAt > schedulerEvents[j].OccurredAt
	})

	// Apply pagination
	total := len(schedulerEvents)
	start := (opts.Page - 1) * opts.Limit
	if start < 0 {
		start = 0
	}
	if start >= total {
		start = total
	}

	end := start + opts.Limit
	if end > total {
		end = total
	}

	var entries []model.SchedulerHistoryEntry
	for i := start; i < end; i++ {
		entries = append(entries, model.SchedulerHistoryEntry{
			Timestamp: schedulerEvents[i].OccurredAt,
			Status:    schedulerEvents[i].Status,
			Error:     schedulerEvents[i].Error,
		})
	}

	if entries == nil {
		entries = make([]model.SchedulerHistoryEntry, 0)
	}

	return model.PaginatedSchedulerHistory{
		Entries: entries,
		Total:   total,
		Page:    opts.Page,
		Limit:   opts.Limit,
	}, nil
}

// IsValidCron checks if a cron expression is valid.
func IsValidCron(cronExpr string) bool {
	// Use robfig/cron for validation (same as scheduleSpec does)
	if cronExpr == "" {
		return false
	}

	// Use the backupCronParser which is defined in backup_scheduler.go
	// Note: backupCronParser is package-private, so we recreate a parser here
	// or we could import directly. Let's use a basic check first.
	_, err := backupCronParser.Parse(cronExpr)
	return err == nil
}
