package service

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

const backupSchemaVersion = 1

type BackupService struct {
	dataDir       string
	configService ConfigService
}

func NewBackupService(dataDir string) BackupService {
	return BackupService{
		dataDir:       dataDir,
		configService: NewConfigService(dataDir),
	}
}

func (s BackupService) List() (model.BackupList, error) {
	manifestDir := filepath.Join(s.dataDir, "manifests")
	if !platform.Exists(manifestDir) {
		return model.BackupList{Items: []model.BackupSummary{}}, nil
	}

	entries, err := os.ReadDir(manifestDir)
	if err != nil {
		return model.BackupList{}, fmt.Errorf("read manifests dir: %w", err)
	}

	items := make([]model.BackupSummary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		manifest, err := s.loadManifest(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			return model.BackupList{}, err
		}

		items = append(items, model.BackupSummary{
			ID:            manifest.ID,
			Reason:        manifest.Reason,
			CreatedAt:     manifest.CreatedAt,
			ArchiveName:   manifest.ArchiveName,
			ArchiveBytes:  manifest.ArchiveBytes,
			ArchiveSHA256: manifest.ArchiveSHA256,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})

	return model.BackupList{Items: items}, nil
}

func (s BackupService) Create(reason string) (model.BackupSummary, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "manual"
	}

	files, err := s.collectBackupFiles()
	if err != nil {
		return model.BackupSummary{}, err
	}

	id := "bkp_" + time.Now().UTC().Format("20060102T150405.000000000")
	archiveName := id + ".zip"
	archivePath := filepath.Join(s.dataDir, "backups", archiveName)

	archiveBytes, fileDetails, err := s.writeArchive(archivePath, files)
	if err != nil {
		return model.BackupSummary{}, err
	}

	archiveHash, err := hashFile(archivePath)
	if err != nil {
		return model.BackupSummary{}, err
	}

	manifest := model.BackupManifest{
		ID:            id,
		SchemaVersion: backupSchemaVersion,
		Reason:        reason,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		ArchiveName:   archiveName,
		ArchiveSHA256: archiveHash,
		ArchiveBytes:  archiveBytes,
		Files:         fileDetails,
	}

	if err := platform.WriteJSONAtomic(filepath.Join(s.dataDir, "manifests", id+".json"), manifest); err != nil {
		return model.BackupSummary{}, fmt.Errorf("write backup manifest: %w", err)
	}

	return model.BackupSummary{
		ID:            manifest.ID,
		Reason:        manifest.Reason,
		CreatedAt:     manifest.CreatedAt,
		ArchiveName:   manifest.ArchiveName,
		ArchiveBytes:  manifest.ArchiveBytes,
		ArchiveSHA256: manifest.ArchiveSHA256,
	}, nil
}

func (s BackupService) Restore(id string, runtimeManager *ProcessManager) (model.BackupSummary, error) {
	manifest, err := s.loadManifest(id)
	if err != nil {
		return model.BackupSummary{}, err
	}
	if manifest.SchemaVersion != backupSchemaVersion {
		return model.BackupSummary{}, fmt.Errorf("unsupported backup schema version: %d", manifest.SchemaVersion)
	}

	archivePath := filepath.Join(s.dataDir, "backups", manifest.ArchiveName)
	currentHash, err := hashFile(archivePath)
	if err != nil {
		return model.BackupSummary{}, err
	}
	if currentHash != manifest.ArchiveSHA256 {
		return model.BackupSummary{}, fmt.Errorf("backup archive hash mismatch")
	}

	preventiveBackup, err := s.Create("pre_restore")
	if err != nil {
		return model.BackupSummary{}, fmt.Errorf("create preventive backup: %w", err)
	}

	wasRunning := runtimeManager != nil && runtimeManager.Status().Running
	if wasRunning {
		if err := runtimeManager.Stop(); err != nil {
			return model.BackupSummary{}, fmt.Errorf("stop runtime before restore: %w", err)
		}
	}

	if err := s.restoreArchive(manifest, archivePath); err != nil {
		return model.BackupSummary{}, err
	}

	if wasRunning {
		if err := runtimeManager.Start(); err != nil {
			return model.BackupSummary{}, fmt.Errorf("restart runtime after restore: %w", err)
		}
	}

	return preventiveBackup, nil
}

func (s BackupService) loadManifest(id string) (model.BackupManifest, error) {
	var manifest model.BackupManifest
	if err := platform.ReadJSON(filepath.Join(s.dataDir, "manifests", id+".json"), &manifest); err != nil {
		return model.BackupManifest{}, fmt.Errorf("read backup manifest: %w", err)
	}
	return manifest, nil
}

func (s BackupService) collectBackupFiles() ([]string, error) {
	config, err := s.configService.Load()
	if err != nil {
		return nil, err
	}

	files := []string{
		"config.json",
		"settings.js",
		".env.managed",
		"package.json",
		config.FlowFile,
		credentialFileName(config.FlowFile),
	}

	seen := make(map[string]struct{}, len(files))
	filtered := make([]string, 0, len(files))
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		if _, ok := seen[file]; ok {
			continue
		}
		seen[file] = struct{}{}
		if platform.Exists(filepath.Join(s.dataDir, file)) {
			filtered = append(filtered, file)
		}
	}

	sort.Strings(filtered)
	return filtered, nil
}

func (s BackupService) writeArchive(archivePath string, files []string) (int64, []model.BackupFile, error) {
	if err := platform.EnsureDir(filepath.Dir(archivePath)); err != nil {
		return 0, nil, err
	}

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	details := make([]model.BackupFile, 0, len(files))

	for _, relativePath := range files {
		fullPath := filepath.Join(s.dataDir, relativePath)
		content, err := platform.ReadFile(fullPath)
		if err != nil {
			_ = writer.Close()
			return 0, nil, fmt.Errorf("read backup source file: %w", err)
		}

		entry, err := writer.Create(relativePath)
		if err != nil {
			_ = writer.Close()
			return 0, nil, fmt.Errorf("create backup archive entry: %w", err)
		}
		if _, err := entry.Write(content); err != nil {
			_ = writer.Close()
			return 0, nil, fmt.Errorf("write backup archive entry: %w", err)
		}

		details = append(details, model.BackupFile{
			Path:      relativePath,
			SizeBytes: int64(len(content)),
			SHA256:    hashBytes(content),
		})
	}

	if err := writer.Close(); err != nil {
		return 0, nil, fmt.Errorf("close backup archive: %w", err)
	}
	if err := platform.WriteFileAtomic(archivePath, buffer.Bytes(), 0o644); err != nil {
		return 0, nil, fmt.Errorf("write backup archive: %w", err)
	}

	return int64(buffer.Len()), details, nil
}

func (s BackupService) restoreArchive(manifest model.BackupManifest, archivePath string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open backup archive: %w", err)
	}
	defer reader.Close()

	expected := make(map[string]model.BackupFile, len(manifest.Files))
	for _, file := range manifest.Files {
		expected[file.Path] = file
	}

	found := make([]string, 0, len(reader.File))
	configRestored := false
	for _, file := range reader.File {
		cleanName := filepath.Clean(file.Name)
		if cleanName == "." || strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
			return fmt.Errorf("invalid archive entry path: %s", file.Name)
		}

		expectedFile, ok := expected[cleanName]
		if !ok {
			return fmt.Errorf("unexpected archive entry: %s", cleanName)
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("open archive entry: %w", err)
		}
		content, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return fmt.Errorf("read archive entry: %w", err)
		}
		if hashBytes(content) != expectedFile.SHA256 {
			return fmt.Errorf("file hash mismatch for %s", cleanName)
		}

		targetPath := filepath.Join(s.dataDir, cleanName)
		perm := os.FileMode(0o644)
		if cleanName == ".env.managed" {
			perm = 0o600
		}
		if err := platform.WriteFileAtomic(targetPath, content, perm); err != nil {
			return fmt.Errorf("restore file %s: %w", cleanName, err)
		}

		found = append(found, cleanName)
		if cleanName == "config.json" {
			configRestored = true
		}
	}

	sort.Strings(found)
	expectedNames := make([]string, 0, len(expected))
	for name := range expected {
		expectedNames = append(expectedNames, name)
	}
	sort.Strings(expectedNames)
	if !slices.Equal(found, expectedNames) {
		return fmt.Errorf("backup archive contents do not match manifest")
	}

	if configRestored {
		cfg, err := s.configService.Load()
		if err != nil {
			return fmt.Errorf("reload restored config: %w", err)
		}
		settings, err := renderSettings(cfg)
		if err != nil {
			return err
		}
		if err := platform.WriteFileAtomic(filepath.Join(s.dataDir, "settings.js"), []byte(settings), 0o644); err != nil {
			return fmt.Errorf("rewrite settings.js after restore: %w", err)
		}
	}

	return nil
}

func credentialFileName(flowFile string) string {
	ext := filepath.Ext(flowFile)
	base := strings.TrimSuffix(flowFile, ext)
	if base == "" {
		base = "flows"
	}
	if ext == "" {
		ext = ".json"
	}
	return base + "_cred" + ext
}

func hashBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func hashFile(path string) (string, error) {
	content, err := platform.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file for hash: %w", err)
	}
	return hashBytes(content), nil
}
