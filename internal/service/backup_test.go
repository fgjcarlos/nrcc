package service

import (
	"path/filepath"
	"testing"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

func TestBackupServiceCreateListAndRestore(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	if err := platform.EnsureDir(filepath.Join(dataDir, "backups")); err != nil {
		t.Fatalf("EnsureDir(backups) error = %v", err)
	}
	if err := platform.EnsureDir(filepath.Join(dataDir, "manifests")); err != nil {
		t.Fatalf("EnsureDir(manifests) error = %v", err)
	}

	configService := NewConfigService(dataDir, nil)
	if _, err := configService.Apply(model.AppConfig{
		HTTPAdminRoot:      "/ops",
		FlowFile:           "flows.json",
		DiagnosticsEnabled: true,
		ProjectsEnabled:    false,
		CredentialSecret:   "very-secret-123",
	}); err != nil {
		t.Fatalf("Apply(config) error = %v", err)
	}
	if err := platform.WriteFileAtomic(filepath.Join(dataDir, ".env.managed"), []byte("API_KEY=before\n"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic(.env.managed) error = %v", err)
	}
	if err := platform.WriteFileAtomic(filepath.Join(dataDir, "flows.json"), []byte(`{"rev":"one"}`), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic(flows.json) error = %v", err)
	}
	if err := platform.WriteFileAtomic(filepath.Join(dataDir, "flows_cred.json"), []byte(`{"token":"abc"}`), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic(flows_cred.json) error = %v", err)
	}
	if err := platform.WriteFileAtomic(filepath.Join(dataDir, "package.json"), []byte(`{"name":"runtime"}`), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic(package.json) error = %v", err)
	}

	service := NewBackupService(dataDir)

	backup, err := service.Create("manual")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if backup.ID == "" || backup.ArchiveName == "" || backup.ArchiveSHA256 == "" {
		t.Fatalf("Create() = %+v", backup)
	}

	list, err := service.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("List() len = %d, want 1", len(list.Items))
	}

	if err := platform.WriteFileAtomic(filepath.Join(dataDir, ".env.managed"), []byte("API_KEY=after\n"), 0o600); err != nil {
		t.Fatalf("mutate .env.managed error = %v", err)
	}
	if err := platform.WriteFileAtomic(filepath.Join(dataDir, "flows.json"), []byte(`{"rev":"two"}`), 0o644); err != nil {
		t.Fatalf("mutate flows.json error = %v", err)
	}

	preventive, err := service.Restore(backup.ID, nil)
	if err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	if preventive.Reason != "pre_restore" {
		t.Fatalf("preventive backup reason = %q, want pre_restore", preventive.Reason)
	}

	envRaw, err := platform.ReadFile(filepath.Join(dataDir, ".env.managed"))
	if err != nil {
		t.Fatalf("ReadFile(.env.managed) error = %v", err)
	}
	if string(envRaw) != "API_KEY=before\n" {
		t.Fatalf("restored .env.managed = %q", string(envRaw))
	}

	flowsRaw, err := platform.ReadFile(filepath.Join(dataDir, "flows.json"))
	if err != nil {
		t.Fatalf("ReadFile(flows.json) error = %v", err)
	}
	if string(flowsRaw) != `{"rev":"one"}` {
		t.Fatalf("restored flows.json = %q", string(flowsRaw))
	}

	list, err = service.List()
	if err != nil {
		t.Fatalf("List() after restore error = %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("List() after restore len = %d, want 2", len(list.Items))
	}
}

func TestBackupServiceRestoreRejectsTamperedArchive(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	if err := platform.EnsureDir(filepath.Join(dataDir, "backups")); err != nil {
		t.Fatalf("EnsureDir(backups) error = %v", err)
	}
	if err := platform.EnsureDir(filepath.Join(dataDir, "manifests")); err != nil {
		t.Fatalf("EnsureDir(manifests) error = %v", err)
	}
	if err := platform.WriteFileAtomic(filepath.Join(dataDir, "config.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic(config.json) error = %v", err)
	}
	if err := platform.WriteFileAtomic(filepath.Join(dataDir, "settings.js"), []byte("module.exports = {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic(settings.js) error = %v", err)
	}

	service := NewBackupService(dataDir)
	backup, err := service.Create("manual")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := platform.WriteFileAtomic(filepath.Join(dataDir, "backups", backup.ArchiveName), []byte("tampered"), 0o644); err != nil {
		t.Fatalf("tamper backup archive error = %v", err)
	}

	if _, err := service.Restore(backup.ID, nil); err == nil {
		t.Fatal("Restore() error = nil, want tampered archive failure")
	}
}
