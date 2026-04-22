package service

import (
	"path/filepath"
	"strings"
	"testing"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

func TestManagedEnvServiceLoadDefaults(t *testing.T) {
	t.Parallel()

	state, err := NewManagedEnvService(t.TempDir()).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(state.Variables) != 0 || state.RestartRequired {
		t.Fatalf("Load() = %+v, want empty state", state)
	}
}

func TestManagedEnvServiceApplyAndLoad(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewManagedEnvService(dataDir)

	state, err := service.Apply([]model.ManagedEnvVar{
		{Name: "api_key", Value: "secret", Secret: true},
		{Name: "FEATURE_FLAG", Value: "enabled"},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !state.RestartRequired || len(state.Variables) != 2 {
		t.Fatalf("Apply() state = %+v", state)
	}
	if state.Variables[0].Name != "API_KEY" {
		t.Fatalf("Apply() normalized name = %q, want API_KEY", state.Variables[0].Name)
	}
	if !state.Variables[0].Secret || state.Variables[0].Value != "" || !state.Variables[0].HasValue {
		t.Fatalf("Apply() secret variable = %+v, want masked secret metadata", state.Variables[0])
	}

	loaded, err := service.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.Variables) != 2 {
		t.Fatalf("Load() len = %d, want 2", len(loaded.Variables))
	}
	if !loaded.Variables[0].Secret || loaded.Variables[0].Value != "" || !loaded.Variables[0].HasValue {
		t.Fatalf("Load() secret variable = %+v, want masked secret metadata", loaded.Variables[0])
	}

	raw, err := platform.ReadFile(filepath.Join(dataDir, ".env.managed"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(raw)
	if strings.Contains(content, "secret") {
		t.Fatalf("managed env file leaked plaintext secret: %q", content)
	}
	if !strings.Contains(content, "API_KEY="+managedEnvEncryptedPrefix) || !strings.Contains(content, "FEATURE_FLAG=enabled\n") {
		t.Fatalf("managed env file = %q", content)
	}

	keyRaw, err := platform.ReadFile(filepath.Join(dataDir, managedEnvKeyFile))
	if err != nil {
		t.Fatalf("ReadFile(key) error = %v", err)
	}
	if strings.TrimSpace(string(keyRaw)) == "" {
		t.Fatal("managed env key file is empty")
	}

	runtimeLines, err := service.RuntimeLines()
	if err != nil {
		t.Fatalf("RuntimeLines() error = %v", err)
	}
	joined := strings.Join(runtimeLines, "\n")
	for _, expected := range []string{"API_KEY=secret", "FEATURE_FLAG=enabled"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("RuntimeLines() missing %q in %q", expected, joined)
		}
	}
}

func TestManagedEnvServiceValidateReservedAndInvalidNames(t *testing.T) {
	t.Parallel()

	service := NewManagedEnvService(t.TempDir())
	errors := service.Validate([]model.ManagedEnvVar{
		{Name: "PORT", Value: "1880"},
		{Name: "NRCC_SECRET", Value: "x"},
		{Name: "bad-name", Value: "x"},
		{Name: "DUP", Value: "one"},
		{Name: "DUP", Value: "two"},
		{Name: "HAS_NEWLINE", Value: "first\nsecond"},
	})

	if len(errors) < 5 {
		t.Fatalf("Validate() errors = %v, want several", errors)
	}
	joined := strings.Join(errors, " | ")
	for _, expected := range []string{"PORT is reserved", "NRCC_SECRET is reserved", "BAD-NAME", "DUP is duplicated", "HAS_NEWLINE contains a newline"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("Validate() errors missing %q in %q", expected, joined)
		}
	}
}

func TestManagedEnvServiceApplyInvalidDoesNotOverwrite(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewManagedEnvService(dataDir)

	if _, err := service.Apply([]model.ManagedEnvVar{{Name: "TOKEN", Value: "abc", Secret: true}}); err != nil {
		t.Fatalf("Apply(valid) error = %v", err)
	}

	before, err := platform.ReadFile(filepath.Join(dataDir, ".env.managed"))
	if err != nil {
		t.Fatalf("ReadFile(before) error = %v", err)
	}

	if _, err := service.Apply([]model.ManagedEnvVar{{Name: "PORT", Value: "1880"}}); err != nil {
		t.Fatalf("Apply(invalid) error = %v, want nil", err)
	}

	after, err := platform.ReadFile(filepath.Join(dataDir, ".env.managed"))
	if err != nil {
		t.Fatalf("ReadFile(after) error = %v", err)
	}
	if string(before) != string(after) {
		t.Fatal(".env.managed changed after invalid apply")
	}
}

func TestManagedEnvServiceLoadBackwardCompatiblePlaintext(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	if err := platform.WriteFileAtomic(filepath.Join(dataDir, ".env.managed"), []byte("API_KEY=legacy\n"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic(.env.managed) error = %v", err)
	}

	state, err := NewManagedEnvService(dataDir).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(state.Variables) != 1 {
		t.Fatalf("Load() len = %d, want 1", len(state.Variables))
	}
	if state.Variables[0].Secret {
		t.Fatalf("Load() secret = true for legacy plaintext variable: %+v", state.Variables[0])
	}
	if state.Variables[0].Value != "legacy" {
		t.Fatalf("Load() value = %q, want legacy", state.Variables[0].Value)
	}
}

func TestManagedEnvServiceApplyPreservesExistingMaskedSecret(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewManagedEnvService(dataDir)

	if _, err := service.Apply([]model.ManagedEnvVar{{Name: "API_KEY", Value: "first-secret", Secret: true}}); err != nil {
		t.Fatalf("Apply(initial) error = %v", err)
	}
	if _, err := service.Apply([]model.ManagedEnvVar{{Name: "API_KEY", Value: "", Secret: true, HasValue: true}}); err != nil {
		t.Fatalf("Apply(masked preserve) error = %v", err)
	}

	runtimeLines, err := service.RuntimeLines()
	if err != nil {
		t.Fatalf("RuntimeLines() error = %v", err)
	}
	if len(runtimeLines) != 1 || runtimeLines[0] != "API_KEY=first-secret" {
		t.Fatalf("RuntimeLines() = %v, want preserved secret", runtimeLines)
	}
}

func TestManagedEnvServiceApplyClearsSecretWhenRequested(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewManagedEnvService(dataDir)

	if _, err := service.Apply([]model.ManagedEnvVar{{Name: "API_KEY", Value: "first-secret", Secret: true}}); err != nil {
		t.Fatalf("Apply(initial) error = %v", err)
	}
	if _, err := service.Apply([]model.ManagedEnvVar{{Name: "API_KEY", Value: "", Secret: true, HasValue: false}}); err != nil {
		t.Fatalf("Apply(clear) error = %v", err)
	}

	runtimeLines, err := service.RuntimeLines()
	if err != nil {
		t.Fatalf("RuntimeLines() error = %v", err)
	}
	if len(runtimeLines) != 1 || runtimeLines[0] != "API_KEY=" {
		t.Fatalf("RuntimeLines() = %v, want cleared secret", runtimeLines)
	}
}
