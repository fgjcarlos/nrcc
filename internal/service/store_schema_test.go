package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/store"
)

func TestLegacyUsersWriteSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cc-users.json")
	if err := os.WriteFile(path, []byte(`{"users":[{"username":"legacy"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	users := store.NewVersionedJSONStore[model.CCUsers](path)
	sessions := store.NewJSONStore[model.RefreshSessions](filepath.Join(dir, "refresh_sessions.json"))
	auth := NewAuthService("", users, sessions)
	if auth.GetUserByUsername("legacy") == nil {
		t.Fatal("legacy user is not readable")
	}
	if err := auth.CreateUser(&model.CCUser{Username: "new"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		Version int `json:"schemaVersion"`
		model.CCUsers
	}
	if err := json.Unmarshal(body, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Version != 1 || len(saved.Users) != 2 || saved.Users[0].Username != "legacy" || saved.Users[1].Username != "new" {
		t.Fatal("expected schemaVersion 1 and both users")
	}
}

func TestLegacyConfigWritesSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	backupPath := filepath.Join(dir, "backup_config.json")
	for path, body := range map[string]string{
		configPath: `{"port":1885}`,
		backupPath: `{"retentionManual":5}`,
	} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	config := NewIsolatedConfigService(dir)
	if cfg, err := config.Get(); err != nil || cfg.Port != 1885 {
		t.Fatalf("load legacy config: port %d, err %v", cfg.Port, err)
	}
	if _, err := config.Update(func(cfg *model.NodeRedConfig) error { cfg.Port++; return nil }); err != nil {
		t.Fatal(err)
	}
	backup := NewBackupServiceWithBackupDir(dir, t.TempDir())
	cfg, err := backup.GetConfig()
	if err != nil || cfg.RetentionManual != 5 {
		t.Fatalf("load legacy backup config: retention %d, err %v", cfg.RetentionManual, err)
	}
	cfg.RetentionManual++
	if _, err := backup.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]map[string]int{
		configPath: {"port": 1886, "schemaVersion": 1},
		backupPath: {"retentionManual": 6, "schemaVersion": 1},
	} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var saved map[string]json.RawMessage
		if err := json.Unmarshal(body, &saved); err != nil {
			t.Fatal(err)
		}
		for key, value := range want {
			var got int
			if err := json.Unmarshal(saved[key], &got); err != nil || got != value {
				t.Errorf("%s: %s = %s; want %d", filepath.Base(path), key, saved[key], value)
			}
		}
	}
}

func TestRestoreRejectsFutureStoreSchema(t *testing.T) {
	for _, name := range []string{"config.json", "cc-users.json", "backup_config.json"} {
		t.Run(name, func(t *testing.T) {
			dir, backupDir := t.TempDir(), t.TempDir()
			path := filepath.Join(dir, name)
			const original = `{"schemaVersion":1}`
			if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
				t.Fatal(err)
			}
			writeAtomicRestoreArchive(t, backupDir, "future-schema", map[string]string{name: `{"schemaVersion":2}`, "flows.json": `[]`})
			backup := NewBackupServiceWithBackupDir(dir, backupDir)
			err := backup.Restore("future-schema")
			if err == nil || !strings.Contains(err.Error(), name+" has schemaVersion 2") {
				t.Errorf("restore error = %v; want unsupported schema for %s", err, name)
			}
			if body, err := os.ReadFile(path); err != nil || string(body) != original {
				t.Errorf("existing file changed: %q, %v", body, err)
			}
			if _, err := os.Stat(filepath.Join(dir, "flows.json")); !os.IsNotExist(err) {
				t.Errorf("restore published flows.json: %v", err)
			}
		})
	}
}
