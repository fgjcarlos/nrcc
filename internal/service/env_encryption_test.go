package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestSet_EncryptsSecretWhenKeyProvided(t *testing.T) {
	configSvc := NewIsolatedConfigService(t.TempDir())
	envSvc := NewEnvService(configSvc, "test-encryption-key")

	if err := envSvc.Set("API_KEY", "super-secret", "secret", "", true); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	cfg, _ := configSvc.Get()
	stored := cfg.EnvVars[0]

	if !IsEncrypted(stored.Value) {
		t.Errorf("stored value should be encrypted, got %q", stored.Value)
	}
	if stored.Value == "super-secret" {
		t.Error("stored value should NOT be plaintext")
	}
}

func TestSet_PlaintextWhenNoKey(t *testing.T) {
	configSvc := NewIsolatedConfigService(t.TempDir())
	envSvc := NewEnvService(configSvc)

	if err := envSvc.Set("API_KEY", "super-secret", "secret", "", true); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	cfg, _ := configSvc.Get()
	if cfg.EnvVars[0].Value != "super-secret" {
		t.Errorf("without encryption key, value should remain plaintext, got %q", cfg.EnvVars[0].Value)
	}
}

func TestSet_DoesNotEncryptNonSecrets(t *testing.T) {
	configSvc := NewIsolatedConfigService(t.TempDir())
	envSvc := NewEnvService(configSvc, "test-key")

	if err := envSvc.Set("PORT", "8080", "number", "", false); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	cfg, _ := configSvc.Get()
	if cfg.EnvVars[0].Value != "8080" {
		t.Errorf("non-secret value should not be encrypted, got %q", cfg.EnvVars[0].Value)
	}
}

func TestGetAll_DecryptsSecrets(t *testing.T) {
	configSvc := NewIsolatedConfigService(t.TempDir())
	envSvc := NewEnvService(configSvc, "test-key")

	_ = envSvc.Set("DB_PASS", "my-password", "secret", "", true)
	_ = envSvc.Set("PORT", "3000", "number", "", false)

	all, err := envSvc.GetAll()
	if err != nil {
		t.Fatalf("GetAll() error: %v", err)
	}

	if all["DB_PASS"] != "my-password" {
		t.Errorf("decrypted value = %q, want %q", all["DB_PASS"], "my-password")
	}
	if all["PORT"] != "3000" {
		t.Errorf("PORT = %q, want %q", all["PORT"], "3000")
	}
}

func TestGetAll_PlaintextPassthroughWhenNoKey(t *testing.T) {
	configSvc := NewIsolatedConfigService(t.TempDir())

	cfg, _ := configSvc.Get()
	cfg.EnvVars = []model.EnvVar{
		{Key: "OLD_SECRET", Value: "plaintext-value", Type: "secret", Encrypted: true},
	}
	_ = configSvc.Save(cfg)

	envSvc := NewEnvService(configSvc)
	all, err := envSvc.GetAll()
	if err != nil {
		t.Fatalf("GetAll() error: %v", err)
	}

	if all["OLD_SECRET"] != "plaintext-value" {
		t.Errorf("without key, plaintext value should pass through, got %q", all["OLD_SECRET"])
	}
}

func TestList_MigratesPlaintextToEncrypted(t *testing.T) {
	configSvc := NewIsolatedConfigService(t.TempDir())

	cfg, _ := configSvc.Get()
	cfg.EnvVars = []model.EnvVar{
		{Key: "OLD_SECRET", Value: "plaintext-password", Type: "secret", Encrypted: true},
	}
	_ = configSvc.Save(cfg)

	envSvc := NewEnvService(configSvc, "migration-key")
	_, err := envSvc.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	cfg, _ = configSvc.Get()
	if !IsEncrypted(cfg.EnvVars[0].Value) {
		t.Error("after migration, stored value should be encrypted")
	}

	decrypted, err := Decrypt(cfg.EnvVars[0].Value, "migration-key")
	if err != nil {
		t.Fatalf("Decrypt() error: %v", err)
	}
	if decrypted != "plaintext-password" {
		t.Errorf("decrypted migrated value = %q, want %q", decrypted, "plaintext-password")
	}
}

func TestSet_PreservesEncryptedValueOnEmptyUpdate(t *testing.T) {
	configSvc := NewIsolatedConfigService(t.TempDir())
	envSvc := NewEnvService(configSvc, "test-key")

	_ = envSvc.Set("SECRET", "original-value", "secret", "", true)

	cfg, _ := configSvc.Get()
	originalStored := cfg.EnvVars[0].Value

	_ = envSvc.Set("SECRET", "", "secret", "updated desc", true)

	cfg, _ = configSvc.Get()
	if cfg.EnvVars[0].Value != originalStored {
		t.Error("empty value update should preserve existing encrypted value")
	}
}

func TestWriteDotenv_Permissions(t *testing.T) {
	dir := t.TempDir()
	if err := WriteDotenv(dir, "KEY=value"); err != nil {
		t.Fatalf("WriteDotenv() error: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf(".env permissions = %o, want 0600", perm)
	}
}
