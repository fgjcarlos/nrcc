package service

import (
	"path/filepath"
	"strings"
	"testing"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

func TestConfigServiceLoadDefaultsWhenConfigMissing(t *testing.T) {
	t.Parallel()

	service := NewConfigService(t.TempDir())

	got, err := service.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := DefaultAppConfig()
	if got != want {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}
}

func TestConfigServiceValidateNormalizesAndDiffs(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewConfigService(dataDir)

	if err := platform.WriteJSONAtomic(filepath.Join(dataDir, "config.json"), DefaultAppConfig()); err != nil {
		t.Fatalf("WriteJSONAtomic() error = %v", err)
	}

	result := service.Validate(model.AppConfig{
		HTTPAdminRoot:      "admin",
		FlowFile:           "custom-flows.json",
		DiagnosticsEnabled: false,
		ProjectsEnabled:    true,
		CredentialSecret:   "very-secret-123",
	})

	if !result.Valid {
		t.Fatalf("Validate() valid = false, errors = %v", result.Errors)
	}
	if !result.RestartRequired {
		t.Fatal("Validate() restartRequired = false, want true")
	}
	if len(result.Diff) != 5 {
		t.Fatalf("Validate() diff len = %d, want 5", len(result.Diff))
	}
	if result.Diff[0].Field != "httpAdminRoot" || result.Diff[0].To != "/admin" {
		t.Fatalf("Validate() first diff = %+v, want normalized httpAdminRoot", result.Diff[0])
	}
}

func TestConfigServiceApplyWritesConfigAndSettings(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewConfigService(dataDir)

	candidate := model.AppConfig{
		HTTPAdminRoot:      "/ops",
		FlowFile:           "runtime.json",
		DiagnosticsEnabled: false,
		ProjectsEnabled:    true,
		CredentialSecret:   "very-secret-123",
	}

	result, err := service.Apply(candidate)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !result.Valid {
		t.Fatalf("Apply() valid = false, errors = %v", result.Errors)
	}
	if !result.RestartRequired {
		t.Fatal("Apply() restartRequired = false, want true")
	}

	var stored model.AppConfig
	if err := platform.ReadJSON(filepath.Join(dataDir, "config.json"), &stored); err != nil {
		t.Fatalf("ReadJSON(config.json) error = %v", err)
	}
	if stored != candidate {
		t.Fatalf("stored config = %+v, want %+v", stored, candidate)
	}

	settingsRaw, err := platform.ReadFile(filepath.Join(dataDir, "settings.js"))
	if err != nil {
		t.Fatalf("ReadFile(settings.js) error = %v", err)
	}
	settings := string(settingsRaw)
	mustContain(t, settings, `httpAdminRoot: "/ops"`)
	mustContain(t, settings, `flowFile: "runtime.json"`)
	mustContain(t, settings, `enabled: false`)
	mustContain(t, settings, `enabled: true`)
	mustContain(t, settings, `credentialSecret: "very-secret-123"`)
}

func TestConfigServiceApplyInvalidDoesNotOverwriteExistingState(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewConfigService(dataDir)

	original := model.AppConfig{
		HTTPAdminRoot:      "/ops",
		FlowFile:           "runtime.json",
		DiagnosticsEnabled: true,
		ProjectsEnabled:    false,
		CredentialSecret:   "very-secret-123",
	}

	if _, err := service.Apply(original); err != nil {
		t.Fatalf("Apply(original) error = %v", err)
	}

	settingsBefore, err := platform.ReadFile(filepath.Join(dataDir, "settings.js"))
	if err != nil {
		t.Fatalf("ReadFile(settings.js) before error = %v", err)
	}

	result, err := service.Apply(model.AppConfig{
		HTTPAdminRoot:      "/invalid root",
		FlowFile:           "../escape.json",
		DiagnosticsEnabled: false,
		ProjectsEnabled:    true,
		CredentialSecret:   "short",
	})
	if err != nil {
		t.Fatalf("Apply(invalid) error = %v, want nil with invalid result", err)
	}
	if result.Valid {
		t.Fatal("Apply(invalid) valid = true, want false")
	}
	if len(result.Errors) == 0 {
		t.Fatal("Apply(invalid) errors empty, want validation errors")
	}

	var stored model.AppConfig
	if err := platform.ReadJSON(filepath.Join(dataDir, "config.json"), &stored); err != nil {
		t.Fatalf("ReadJSON(config.json) error = %v", err)
	}
	if stored != original {
		t.Fatalf("stored config after invalid apply = %+v, want %+v", stored, original)
	}

	settingsAfter, err := platform.ReadFile(filepath.Join(dataDir, "settings.js"))
	if err != nil {
		t.Fatalf("ReadFile(settings.js) after error = %v", err)
	}
	if string(settingsAfter) != string(settingsBefore) {
		t.Fatal("settings.js changed after invalid apply")
	}
}

func TestConfigServiceRenderSettingsDisablesCredentialSecretWhenEmpty(t *testing.T) {
	t.Parallel()

	settings, err := renderSettings(model.AppConfig{
		HTTPAdminRoot:      "/",
		FlowFile:           "flows.json",
		DiagnosticsEnabled: true,
		ProjectsEnabled:    false,
		CredentialSecret:   "",
	})
	if err != nil {
		t.Fatalf("renderSettings() error = %v", err)
	}

	mustContain(t, settings, `credentialSecret: false`)
}

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()

	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected %q to contain %q", haystack, needle)
	}
}
