package service

import (
	"fmt"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestEnvService_ListMigration_Concurrent(t *testing.T) {
	configSvc := NewIsolatedConfigService(t.TempDir())
	base := configSvc.GetDefault()
	base.EnvVars = []model.EnvVar{{
		Key:       "LEGACY_SECRET",
		Value:     "plaintext",
		Type:      "plain",
		Encrypted: true,
	}}
	if err := configSvc.store.Write(base); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	envSvc := NewEnvService(configSvc, "migration-key")
	const callers = 16
	results := make([][]model.EnvVar, callers)
	errs := runConcurrent(t, callers, func(i int) error {
		var err error
		results[i], err = envSvc.List()
		return err
	})
	assertNoError(t, errs)

	for i, result := range results {
		if len(result) != 1 || result[0].Key != "LEGACY_SECRET" || result[0].Type != "secret" || result[0].Value != "" {
			t.Errorf("List result %d = %#v, want one masked migrated secret", i, result)
		}
	}
	committed, err := configSvc.store.Read()
	if err != nil {
		t.Fatalf("read committed config: %v", err)
	}
	if len(committed.EnvVars) != 1 || committed.EnvVars[0].Type != "secret" || !IsEncrypted(committed.EnvVars[0].Value) {
		t.Fatalf("committed migration = %#v", committed.EnvVars)
	}
	plaintext, err := Decrypt(committed.EnvVars[0].Value, "migration-key")
	if err != nil || plaintext != "plaintext" {
		t.Fatalf("decrypt migrated value = %q, %v; want plaintext", plaintext, err)
	}
}

func TestEnvService_ListMigration_EncryptedTypeOnly(t *testing.T) {
	dir := t.TempDir()
	configSvc := NewIsolatedConfigService(dir)
	ciphertext, err := Encrypt("already-encrypted", "migration-key")
	if err != nil {
		t.Fatalf("encrypt fixture: %v", err)
	}
	base := configSvc.GetDefault()
	base.EnvVars = []model.EnvVar{{
		Key:       "LEGACY_TYPE",
		Value:     ciphertext,
		Type:      "string",
		Encrypted: true,
	}}
	if err := configSvc.store.Write(base); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	result, err := NewEnvService(configSvc, "migration-key").List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result) != 1 || result[0].Type != "secret" || result[0].Value != "" {
		t.Fatalf("List result = %#v, want masked secret", result)
	}
	committed, err := configSvc.store.Read()
	if err != nil {
		t.Fatalf("read committed config: %v", err)
	}
	if committed.EnvVars[0].Type != "secret" || committed.EnvVars[0].Value != ciphertext {
		t.Fatalf("committed migration = %#v, want type-only migration", committed.EnvVars[0])
	}
}

func TestEnvService_Set_Concurrent(t *testing.T) {
	configSvc := NewIsolatedConfigService(t.TempDir())
	if err := configSvc.store.Write(configSvc.GetDefault()); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	// #664: the original test used "secret" type so the concurrency
	// assertion could compare plaintext against the caller's input. With
	// fail-closed semantics the test now uses a real key and decrypts
	// each committed value to verify the assertion; "secret" type is
	// kept so the JSONStore.Update lock is exercised on the same
	// encrypted branch as the production hot path.
	envSvc := NewEnvService(configSvc, "concurrent-test-key")

	const count = 24
	errs := runConcurrent(t, count, func(i int) error {
		return envSvc.Set(fmt.Sprintf("KEY_%02d", i), fmt.Sprintf("value-%02d", i), "secret", "", true)
	})
	assertNoError(t, errs)

	config, err := configSvc.Get()
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if len(config.EnvVars) != count {
		t.Fatalf("EnvVars length = %d, want %d", len(config.EnvVars), count)
	}
	seen := make(map[string]string, count)
	for _, envVar := range config.EnvVars {
		seen[envVar.Key] = envVar.Value
	}
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("KEY_%02d", i)
		plaintext, derr := Decrypt(seen[key], "concurrent-test-key")
		if derr != nil {
			t.Errorf("%s = %q, failed to decrypt: %v", key, seen[key], derr)
			continue
		}
		if plaintext != fmt.Sprintf("value-%02d", i) {
			t.Errorf("%s decrypted = %q, want value-%02d", key, plaintext, i)
		}
	}
}

func TestEnvService_Set_SameKey_Concurrent(t *testing.T) {
	configSvc := NewIsolatedConfigService(t.TempDir())
	if err := configSvc.store.Write(configSvc.GetDefault()); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	envSvc := NewEnvService(configSvc, "concurrent-test-key")

	const count = 20
	errs := runConcurrent(t, count, func(i int) error {
		return envSvc.Set("SHARED", fmt.Sprintf("value-%02d", i), "secret", "", true)
	})
	assertNoError(t, errs)

	config, err := configSvc.Get()
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if len(config.EnvVars) != 1 || config.EnvVars[0].Key != "SHARED" {
		t.Fatalf("EnvVars = %#v, want one SHARED entry", config.EnvVars)
	}
	plaintext, derr := Decrypt(config.EnvVars[0].Value, "concurrent-test-key")
	if derr != nil {
		t.Fatalf("SHARED decrypt: %v", derr)
	}
	valid := false
	for i := 0; i < count; i++ {
		if plaintext == fmt.Sprintf("value-%02d", i) {
			valid = true
			break
		}
	}
	if !valid {
		t.Fatalf("SHARED decrypted = %q, want one committed caller value", plaintext)
	}
}

func TestEnvService_Delete_Concurrent(t *testing.T) {
	configSvc := NewIsolatedConfigService(t.TempDir())
	base := configSvc.GetDefault()
	base.EnvVars = []model.EnvVar{{Key: "DELETE_ME", Value: "value", Type: "secret", Encrypted: true}}
	if err := configSvc.store.Write(base); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	envSvc := NewEnvService(configSvc)

	errs := runConcurrent(t, 16, func(_ int) error {
		return envSvc.Delete("DELETE_ME")
	})
	assertNoError(t, errs)

	config, err := configSvc.Get()
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if len(config.EnvVars) != 0 {
		t.Fatalf("EnvVars = %#v, want DELETE_ME removed exactly once effectively", config.EnvVars)
	}
}
