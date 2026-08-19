package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	// #664: storing an encrypted value while NRCC_ENCRYPTION_KEY is empty
	// is now fail-closed at the service layer. The old behaviour
	// (persisting the value as plaintext while still flagging it
	// Encrypted: true) was the bug — see TestSet_RejectsEncryptedWriteWhenNoKey
	// for the replacement that proves no plaintext reaches config.json.
	configSvc := NewIsolatedConfigService(t.TempDir())
	envSvc := NewEnvService(configSvc)

	if err := envSvc.Set("API_KEY", "super-secret", "secret", "", true); !errors.Is(err, ErrEncryptionKeyRequired) {
		t.Fatalf("Set() error = %v, want ErrEncryptionKeyRequired", err)
	}

	cfg, err := configSvc.Get()
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	for _, ev := range cfg.EnvVars {
		if ev.Key == "API_KEY" {
			t.Errorf("API_KEY was persisted despite the rejected write: %+v", ev)
		}
	}
}

// TestSet_RejectsEncryptedWriteWhenNoKey is the explicit acceptance test
// for #664: calling Set with encrypted=true while the encryption key is
// empty must (a) return ErrEncryptionKeyRequired, (b) leave config.json
// unchanged so no plaintext value reaches disk. The previous test only
// asserts the error; this one also reads the on-disk JSON to prove the
// rejection was atomic and no plaintext slipped through a different code
// path (e.g. the bulk import).
func TestSet_RejectsEncryptedWriteWhenNoKey(t *testing.T) {
	dir := t.TempDir()
	configSvc := NewIsolatedConfigService(dir)
	envSvc := NewEnvService(configSvc) // empty key

	const secretValue = "do-not-leak-me"
	if err := envSvc.Set("API_KEY", secretValue, "secret", "", true); !errors.Is(err, ErrEncryptionKeyRequired) {
		t.Fatalf("Set() error = %v, want ErrEncryptionKeyRequired", err)
	}

	// 1. in-memory state: nothing was committed.
	cfg, err := configSvc.Get()
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if len(cfg.EnvVars) != 0 {
		t.Errorf("config.EnvVars has %d entries, want 0 after rejected write: %#v", len(cfg.EnvVars), cfg.EnvVars)
	}

	// 2. on-disk state: config.json must not contain the secret value.
	configPath := filepath.Join(dir, "config.json")
	// #nosec G304 -- configPath is built from t.TempDir() + the constant
	// filename the test owns; not request-derived.
	raw, err := os.ReadFile(configPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("ReadFile(%q): %v", configPath, err)
		}
		return // no file written — best possible outcome
	}
	if strings.Contains(string(raw), secretValue) {
		t.Fatalf("plaintext secret value %q reached config.json despite the rejected write: %s", secretValue, raw)
	}
	if strings.Contains(string(raw), `"API_KEY"`) {
		t.Errorf("API_KEY was persisted despite the rejected write: %s", raw)
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
