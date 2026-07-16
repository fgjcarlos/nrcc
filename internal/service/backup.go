package service

import (
	"archive/zip"
	"bytes"
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
	"sync"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/google/uuid"
)

const (
	backupConfigFile       = "backup_config.json"
	defaultChecksumAlgo    = "sha256"
	currentManifestVersion = 1
)

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

// ErrBackupCorrupt is returned when an archive's manifest or a payload entry
// fails integrity validation during restore.
var ErrBackupCorrupt = errors.New("backup integrity check failed")

// BackupService handles backup operations.
type BackupService struct {
	dataDir    string
	backupDir  string
	scheduler  *backupScheduler
	eventStore *backupEventStore

	mu           sync.Mutex
	quiesceFunc  func() error
	restartFunc  func() error
}

type createBackupOptions struct {
	Type model.BackupType
	Name string
}

// backupMetadata is the in-process shape used during creation. The on-disk
// shape is model.BackupManifestV1; this struct mirrors only the creator-side
// fields and is converted when written into the zip.
type backupMetadata struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Type        model.BackupType    `json:"type"`
	CreatedAt   string              `json:"createdAt"`
	TriggeredBy string              `json:"triggeredBy"`
	Algorithm   string              `json:"algorithm"`
	Version     int                 `json:"version"`
	Files       []model.BackupFileEntry `json:"files"`
}

// NewBackupService creates a new backup service that stores archives under
// <dataDir>/backups. Use NewBackupServiceWithBackupDir to override the
// archive directory (e.g. to point at a dedicated per-instance volume via the
// NRCC_BACKUP_DIR env var).
func NewBackupService(dataDir string) *BackupService {
	return NewBackupServiceWithBackupDir(dataDir, os.Getenv("NRCC_BACKUP_DIR"))
}

// NewBackupServiceWithBackupDir is NewBackupService plus an explicit archive
// directory. An empty backupDir falls back to <dataDir>/backups.
func NewBackupServiceWithBackupDir(dataDir, backupDir string) *BackupService {
	if strings.TrimSpace(backupDir) == "" {
		backupDir = filepath.Join(dataDir, "backups")
	}
	svc := &BackupService{dataDir: dataDir, backupDir: backupDir}
	svc.eventStore = newBackupEventStore(dataDir)
	svc.scheduler = newBackupScheduler(svc)
	return svc
}

// SetRestoreHooks wires optional Node-RED lifecycle hooks so the service can
// quiesce the runtime during a restore and (best-effort) restart it after.
// Both funcs may be nil; in that case restore skips the corresponding step.
func (s *BackupService) SetRestoreHooks(quiesce, restart func() error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quiesceFunc = quiesce
	s.restartFunc = restart
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
	backupDir := s.backupDir
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backups directory: %w", err)
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backups directory: %w", err)
	}

	backups := make([]model.Backup, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".zip") || strings.HasSuffix(entry.Name(), ".zip.tmp") {
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

	// Apply sorting (overrides the default sort.Slice from List()).
	switch opts.Sort {
	case "size":
		sort.Slice(allBackups, func(i, j int) bool {
			if opts.Order == "asc" {
				return allBackups[i].SizeBytes < allBackups[j].SizeBytes
			}
			return allBackups[i].SizeBytes > allBackups[j].SizeBytes
		})
	case "status":
		// Sort by Type (status proxy)
		sort.Slice(allBackups, func(i, j int) bool {
			if opts.Order == "asc" {
				return allBackups[i].Type < allBackups[j].Type
			}
			return allBackups[i].Type > allBackups[j].Type
		})
	default:
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

	items := []model.Backup{}
	if offset < total {
		items = allBackups[offset:end]
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
	backupPath := filepath.Join(s.backupDir, id+".zip")
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
	if _, err := scheduleSpec(normalized); err != nil {
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

// Restore restores a backup by ID. The archive is validated (manifest +
// per-entry checksum) before any dataDir file is overwritten. Files are
// extracted to a staging directory and only swapped into dataDir after the
// full archive is verified intact. On validation failure the staging tree
// is removed and dataDir is untouched.
func (s *BackupService) Restore(id string) error {
	if err := ValidateBackupID(id); err != nil {
		return err
	}
	backupPath := filepath.Join(s.backupDir, id+".zip")

	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	// Read the archive into memory in one pass so we can both verify
	// checksums and stream files into staging. Backups are bounded by the
	// per-entry size cap (see extractFileFromZip); the full archive is small
	// enough to keep the implementation linear and atomic at the swap step.
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to parse backup: %w", err)
	}

	manifest, err := verifyArchiveManifest(zipReader)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBackupCorrupt, err)
	}

	stagingDir, err := s.stageRestore(zipReader, manifest)
	if err != nil {
		return err
	}

	if err := s.swapStagingIntoDataDir(stagingDir, manifest); err != nil {
		_ = os.RemoveAll(stagingDir)
		return err
	}
	return nil
}

// RestoreWithSafetyBackup creates a pre-restore snapshot before restoring.
func (s *BackupService) RestoreWithSafetyBackup(id string) (string, error) {
	if err := ValidateBackupID(id); err != nil {
		return "", err
	}
	backupPath := filepath.Join(s.backupDir, id+".zip")
	if _, err := os.Stat(backupPath); err != nil {
		return "", fmt.Errorf("backup not found: %w", err)
	}
	restoredBackup, _ := s.describeBackup(backupPath)

	s.mu.Lock()
	quiesce := s.quiesceFunc
	restart := s.restartFunc
	s.mu.Unlock()

	// Quiesce Node-RED before touching its files so we never observe a
	// partially-restored on-disk state from a running process. The hooks
	// are best-effort: if no hook is wired (e.g. external Node-RED) we
	// proceed. The defer below always fires on the way out so Node-RED
	// picks up the restored flows on the success path AND recovers from
	// any restore error. The defer is registered before quiesce() so a
	// future edit that adds an early return between this point and the
	// pre-restore create still triggers the restart path.
	if quiesce != nil {
		if err := quiesce(); err != nil {
			s.recordEvent(model.BackupEvent{
				Type:    model.BackupEventTypeRestore,
				Status:  "error",
				Message: "Failed to quiesce Node-RED before restore",
				Trigger: "restore",
				Error:   err.Error(),
			})
			return "", fmt.Errorf("quiesce before restore: %w", err)
		}
	}
	defer func() {
		if restart == nil {
			return
		}
		if rerr := restart(); rerr != nil {
			s.recordEvent(model.BackupEvent{
				Type:    model.BackupEventTypeRestore,
				Status:  "error",
				Message: "Failed to restart Node-RED after restore",
				Trigger: "restore",
				Error:   rerr.Error(),
			})
		}
	}()

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
	backupPath := filepath.Join(s.backupDir, id+".zip")
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
	backupPath := filepath.Join(s.backupDir, id+".zip")

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

// Download streams a backup file to the response writer. If password is
// non-empty the zip bytes are wrapped with AES-256-GCM (see Encrypt); the
// client decrypts with the same passphrase. This lets an operator transfer a
// backup containing credentials/secrets off-host without exposing them in the
// raw archive.
func (s *BackupService) Download(id string, w io.Writer, password string) error {
	rc, _, err := s.OpenForDownload(id)
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()

	if password == "" {
		_, err := io.Copy(w, rc)
		if err != nil {
			return fmt.Errorf("failed to download backup: %w", err)
		}
		return nil
	}

	data, err := io.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}
	encrypted, err := EncryptBytes(data, password)
	if err != nil {
		return fmt.Errorf("failed to encrypt backup: %w", err)
	}
	if _, err := io.WriteString(w, encrypted); err != nil {
		return fmt.Errorf("failed to stream encrypted backup: %w", err)
	}
	return nil
}

func (s *BackupService) createBackup(options createBackupOptions) (model.Backup, error) {
	if err := os.MkdirAll(s.backupDir, 0755); err != nil {
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
	backupPath := filepath.Join(s.backupDir, backupID+".zip")
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	name := strings.TrimSpace(options.Name)
	if name == "" {
		suffix := strings.ReplaceAll(strings.ReplaceAll(createdAt, ":", "-"), "T", "_")
		suffix = strings.TrimSuffix(suffix, "Z")
		if backupType == model.BackupTypeManual {
			name = "manual-" + suffix
		} else {
			name = string(backupType) + "-" + suffix
		}
	}

	filesToBackup := s.filesForBackup(config)
	manifest := backupMetadata{
		ID:          backupID,
		Name:        name,
		Type:        backupType,
		CreatedAt:   createdAt,
		TriggeredBy: name,
		Algorithm:   defaultChecksumAlgo,
		Version:     currentManifestVersion,
		Files:       make([]model.BackupFileEntry, 0, len(filesToBackup)),
	}

	// Write to a sibling temp file in the same directory so the final
	// os.Rename below is atomic on POSIX (rename within a filesystem).
	// ponytail: relies on the temp file living in s.backupDir; if a caller
	// ever points s.backupDir at a special tmpfs without rename support,
	// upgrade to fsync + explicit fsync-of-directory.
	tmpPath := backupPath + ".tmp"
	zipFile, err := os.Create(tmpPath)
	if err != nil {
		return model.Backup{}, fmt.Errorf("failed to create backup file: %w", err)
	}
	zipWriter := zip.NewWriter(zipFile)

	if err := s.addMetadataToZip(zipWriter, manifest); err != nil {
		_ = zipWriter.Close()
		_ = zipFile.Close()
		_ = os.Remove(tmpPath)
		return model.Backup{}, err
	}

	fileCount := 1
	for _, file := range filesToBackup {
		entry, added, err := s.addFileToZip(zipWriter, file.src, file.dst)
		if err != nil {
			_ = zipWriter.Close()
			_ = zipFile.Close()
			_ = os.Remove(tmpPath)
			return model.Backup{}, err
		}
		if added {
			manifest.Files = append(manifest.Files, entry)
			fileCount++
		}
	}

	// Re-write the manifest with the populated Files slice so the on-disk
	// manifest carries the per-entry checksums.
	if err := zipWriter.Close(); err != nil {
		_ = zipFile.Close()
		_ = os.Remove(tmpPath)
		return model.Backup{}, fmt.Errorf("failed to finalize backup: %w", err)
	}
	// Force the OS to flush the archive to durable storage before we
	// publish it via rename. Without fsync, a crash between close and
	// rename can leave the published file empty on disk despite the
	// "atomic" claim. ponytail: no dir-fsync; in practice the rename +
	// the next open on the published path is enough to surface any
	// missing data. Upgrade to dir-fsync if a deployment needs
	// power-loss guarantees.
	if err := zipFile.Sync(); err != nil {
		_ = zipFile.Close()
		_ = os.Remove(tmpPath)
		return model.Backup{}, fmt.Errorf("failed to flush backup: %w", err)
	}
	if err := zipFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return model.Backup{}, fmt.Errorf("failed to close backup file: %w", err)
	}

	if err := s.patchManifestChecksums(tmpPath, manifest); err != nil {
		_ = os.Remove(tmpPath)
		return model.Backup{}, fmt.Errorf("failed to write manifest checksums: %w", err)
	}

	if err := os.Rename(tmpPath, backupPath); err != nil {
		_ = os.Remove(tmpPath)
		return model.Backup{}, fmt.Errorf("failed to publish backup: %w", err)
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

// patchManifestChecksums rewrites the on-disk archive's manifest entry to
// include the per-entry checksums computed during zip creation. We do this as a
// second pass over the just-written file because the manifest must be written
// before any payload entry (zip readers scan in order) but checksums are only
// known once the payload has been streamed.
func (s *BackupService) patchManifestChecksums(zipPath string, manifest backupMetadata) error {
	// ponytail: small backup, full rewrite is fine. If archives grow past
	// tens of MiB, switch to a streaming rewrite that only re-encodes the
	// manifest entry while copying the rest.
	updated, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	in, err := os.Open(zipPath)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	info, err := in.Stat()
	if err != nil {
		return err
	}
	reader, err := zip.NewReader(in, info.Size())
	if err != nil {
		return err
	}

	out, err := os.CreateTemp(filepath.Dir(zipPath), filepath.Base(zipPath)+".rewriting-")
	if err != nil {
		return err
	}
	tmpOut := out.Name()
	defer func() {
		_ = out.Close()
		_ = os.Remove(tmpOut)
	}()

	zw := zip.NewWriter(out)
	for _, f := range reader.File {
		header := &zip.FileHeader{
			Name:     f.Name,
			Method:   f.Method,
			Modified: f.Modified,
		}
		w, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		if f.Name == "backup-metadata.json" {
			if _, err := w.Write(updated); err != nil {
				return err
			}
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		if _, err := io.Copy(w, rc); err != nil {
			_ = rc.Close()
			return err
		}
		_ = rc.Close()
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(tmpOut, zipPath)
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
	defer func() { _ = file.Close() }()

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

func (s *BackupService) addFileToZip(zipWriter *zip.Writer, srcPath, dstPath string) (model.BackupFileEntry, bool, error) {
	file, err := os.Open(srcPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return model.BackupFileEntry{}, false, nil
		}
		return model.BackupFileEntry{}, false, fmt.Errorf("failed to open %s: %w", srcPath, err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return model.BackupFileEntry{}, false, fmt.Errorf("failed to stat %s: %w", srcPath, err)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return model.BackupFileEntry{}, false, fmt.Errorf("failed to build zip header for %s: %w", srcPath, err)
	}
	header.Name = dstPath
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return model.BackupFileEntry{}, false, fmt.Errorf("failed to add %s to zip: %w", srcPath, err)
	}

	hasher := sha256.New()
	mw := io.MultiWriter(writer, hasher)
	if _, err := io.Copy(mw, file); err != nil {
		return model.BackupFileEntry{}, false, fmt.Errorf("failed to write %s to zip: %w", srcPath, err)
	}

	return model.BackupFileEntry{
		Path:     dstPath,
		Size:     info.Size(),
		Checksum: hex.EncodeToString(hasher.Sum(nil)),
	}, true, nil
}

// verifyArchiveManifest reads the embedded manifest, validates every payload
// entry against its recorded sha256, and returns the parsed manifest. A
// missing, malformed, or mismatched archive is rejected before any file
// touches disk.
func verifyArchiveManifest(zipReader *zip.Reader) (backupMetadata, error) {
	// ponytail: single read of every entry. Acceptable for backups bounded
	// by the per-entry size cap; switch to streaming + selective verification
	// only if backups grow past hundreds of MiB.
	var manifest backupMetadata
	manifestSeen := false
	for _, f := range zipReader.File {
		if f.Name != "backup-metadata.json" {
			continue
		}
		manifestSeen = true
		rc, err := f.Open()
		if err != nil {
			return backupMetadata{}, fmt.Errorf("open manifest: %w", err)
		}
		err = json.NewDecoder(rc).Decode(&manifest)
		_ = rc.Close()
		if err != nil {
			return backupMetadata{}, fmt.Errorf("decode manifest: %w", err)
		}
		break
	}
	if !manifestSeen {
		return backupMetadata{}, errors.New("missing backup-metadata.json")
	}
	if manifest.Version != currentManifestVersion {
		return backupMetadata{}, fmt.Errorf("unsupported manifest version %d", manifest.Version)
	}
	if manifest.Algorithm != defaultChecksumAlgo {
		return backupMetadata{}, fmt.Errorf("unsupported checksum algorithm %q", manifest.Algorithm)
	}

	expected := make(map[string]string, len(manifest.Files))
	expectedSize := make(map[string]int64, len(manifest.Files))
	for _, entry := range manifest.Files {
		expected[entry.Path] = entry.Checksum
		expectedSize[entry.Path] = entry.Size
	}

	for _, f := range zipReader.File {
		if f.FileInfo().IsDir() || f.Name == "backup-metadata.json" {
			continue
		}
		want, ok := expected[f.Name]
		if !ok {
			return backupMetadata{}, fmt.Errorf("unexpected entry %s not in manifest", f.Name)
		}
		if f.UncompressedSize64 != 0 && expectedSize[f.Name] != 0 && int64(f.UncompressedSize64) != expectedSize[f.Name] {
			return backupMetadata{}, fmt.Errorf("size mismatch for %s: manifest=%d zip=%d", f.Name, expectedSize[f.Name], int64(f.UncompressedSize64))
		}
		got, err := checksumZipFile(f)
		if err != nil {
			return backupMetadata{}, fmt.Errorf("checksum %s: %w", f.Name, err)
		}
		if got != want {
			return backupMetadata{}, fmt.Errorf("checksum mismatch for %s", f.Name)
		}
	}

	return manifest, nil
}

// stageRestore extracts the verified archive into a fresh staging directory
// under dataDir. The destination tree is only created if every entry passes
// the per-entry size cap. Only entries listed in the verified manifest are
// extracted, so the staging tree mirrors exactly what swap will publish.
func (s *BackupService) stageRestore(zipReader *zip.Reader, manifest backupMetadata) (string, error) {
	stagingDir, err := os.MkdirTemp(s.dataDir, "restore-staging-")
	if err != nil {
		return "", fmt.Errorf("create staging dir: %w", err)
	}

	// Build a lookup so we skip zip entries that the manifest did not
	// authorize (defense in depth: verifyArchiveManifest already rejected
	// mismatched archives, but filtering here keeps staging clean).
	want := make(map[string]struct{}, len(manifest.Files))
	for _, entry := range manifest.Files {
		want[entry.Path] = struct{}{}
	}

	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() || file.Name == "backup-metadata.json" {
			continue
		}
		if _, ok := want[file.Name]; !ok {
			continue
		}
		if err := s.extractFileFromZip(file, stagingDir); err != nil {
			_ = os.RemoveAll(stagingDir)
			return "", fmt.Errorf("extract %s: %w", file.Name, err)
		}
	}

	if len(manifest.Files) == 0 {
		// No payload files in manifest; nothing to swap. Staging is empty but
		// should not exist as a leftover; remove and return a sentinel path
		// that swapStagingIntoDataDir will treat as a no-op.
		_ = os.RemoveAll(stagingDir)
		return "", nil
	}

	return stagingDir, nil
}

// swapStagingIntoDataDir atomically moves every file extracted under
// stagingDir into s.dataDir. The set of files moved is derived from the
// verified manifest (not from os.ReadDir of stagingDir) so nested paths
// cannot collide with siblings and extra residue in staging can never be
// published. Staging lives on the same filesystem as dataDir by
// construction (os.MkdirTemp on s.dataDir), so os.Rename is atomic per file.
func (s *BackupService) swapStagingIntoDataDir(stagingDir string, manifest backupMetadata) error {
	if stagingDir == "" {
		return nil
	}
	for _, entry := range manifest.Files {
		rel := entry.Path
		if rel == "" || strings.ContainsRune(rel, 0) {
			return fmt.Errorf("manifest entry has invalid path: %q", rel)
		}
		srcPath := filepath.Join(stagingDir, filepath.FromSlash(rel))
		dstPath := filepath.Join(s.dataDir, filepath.FromSlash(rel))
		if _, err := os.Stat(srcPath); err != nil {
			return fmt.Errorf("swap %s: missing in staging: %w", rel, err)
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return fmt.Errorf("swap %s: mkdir: %w", rel, err)
		}
		if err := os.Rename(srcPath, dstPath); err != nil {
			return fmt.Errorf("swap %s: %w", rel, err)
		}
	}
	// Cleanup is unconditional: if swap left residue we still want the
	// staging directory gone so the next MkdirTemp pass can't collide.
	_ = os.RemoveAll(stagingDir)
	return nil
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
	defer func() { _ = reader.Close() }()

	writer, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() { _ = writer.Close() }()

	// The declared size can lie; cap the actual bytes copied. Reading one byte
	// past the limit means the entry is oversized.
	written, err := io.Copy(writer, io.LimitReader(reader, maxBackupEntrySize+1))
	if err != nil {
		return err
	}
	if written > maxBackupEntrySize {
		_ = writer.Close()
		_ = os.Remove(destPath)
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
	value := strings.TrimSpace(strings.ToLower(cfg.Schedule))
	switch value {
	case "disabled", "hourly", "every6h", "daily", "weekly", "custom":
		normalized.Schedule = value
	default:
		normalized.Schedule = defaultBackupConfig.Schedule
	}
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

func readBackupMetadata(file *zip.File) (backupMetadata, error) {
	reader, err := file.Open()
	if err != nil {
		return backupMetadata{}, err
	}
	defer func() { _ = reader.Close() }()

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
	defer func() { _ = reader.Close() }()

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