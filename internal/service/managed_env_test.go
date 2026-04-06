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
		{Name: "api_key", Value: "secret"},
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

	loaded, err := service.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.Variables) != 2 {
		t.Fatalf("Load() len = %d, want 2", len(loaded.Variables))
	}

	raw, err := platform.ReadFile(filepath.Join(dataDir, ".env.managed"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(raw) != "API_KEY=secret\nFEATURE_FLAG=enabled\n" {
		t.Fatalf("managed env file = %q", string(raw))
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

	if _, err := service.Apply([]model.ManagedEnvVar{{Name: "TOKEN", Value: "abc"}}); err != nil {
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
