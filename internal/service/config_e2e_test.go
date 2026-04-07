package service

import (
	"path/filepath"
	"strings"
	"testing"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

// TestConfigService_LoadFullConfig_NoFile tests that LoadFullConfig returns defaults when no file exists
func TestConfigService_LoadFullConfig_NoFile(t *testing.T) {
	t.Parallel()

	service := NewConfigService(t.TempDir(), nil)

	got, err := service.LoadFullConfig()
	if err != nil {
		t.Fatalf("LoadFullConfig() error = %v", err)
	}

	want := model.DefaultFullAppConfig()
	if got.Server.UIPort != want.Server.UIPort {
		t.Fatalf("LoadFullConfig() UIPort = %d, want %d", got.Server.UIPort, want.Server.UIPort)
	}
	if got.Flows.FlowFile != want.Flows.FlowFile {
		t.Fatalf("LoadFullConfig() FlowFile = %s, want %s", got.Flows.FlowFile, want.Flows.FlowFile)
	}
}

// TestConfigService_SaveAndLoad tests that SaveFullConfig and LoadFullConfig round-trip correctly
func TestConfigService_SaveAndLoad(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewConfigService(dataDir, nil)

	original := model.FullAppConfig{
		Server: model.ServerConfig{
			UIPort:        9000,
			UIHost:        "127.0.0.1",
			HTTPAdminRoot: "/admin",
			HTTPNodeRoot:  "/",
			HTTPStatic:    "",
			DisableEditor: false,
		},
		Security: model.SecurityConfig{
			CredentialSecret:  "very-secret-key-12345",
			SessionExpiryTime: 86400,
		},
		Flows: model.FlowsConfig{
			FlowFile:       "custom-flows.json",
			FlowFilePretty: true,
			UserDir:        "",
			NodesDir:       "",
		},
		ContextStorage: model.ContextStorageConfig{
			Default: "default",
			Stores: map[string]model.ContextStoreEntry{
				"default": {Module: "memory"},
			},
		},
		Logging: model.LoggingConfig{
			Console: model.ConsoleLogConfig{
				Level:   "debug",
				Metrics: false,
				Audit:   false,
			},
		},
		Runtime: model.RuntimeConfig{
			FunctionExternalModules:    false,
			FunctionTimeout:            0,
			DebugMaxLength:             1500,
			DiagnosticsEnabled:         true,
			SafeMode:                   false,
			NodeMessageBufferMaxLength: 0,
		},
		HTTPS: model.HTTPSConfig{
			Enabled:  false,
			KeyFile:  "",
			CertFile: "",
			CAFile:   "",
		},
		NodeReconnect: model.NodeReconnectConfig{
			MQTTReconnectTime:   5000,
			SerialReconnectTime: 5000,
			SocketReconnectTime: 10000,
			SocketTimeout:       120000,
		},
		EditorTheme: model.EditorThemeConfig{
			Theme:    "default",
			Tours:    true,
			UserMenu: true,
			Projects: model.EditorProjectsConfig{Enabled: false},
		},
		Palette: model.PaletteConfig{
			Categories: []string{"subflows", "common", "function", "network"},
		},
	}

	if err := service.SaveFullConfig(original); err != nil {
		t.Fatalf("SaveFullConfig() error = %v", err)
	}

	loaded, err := service.LoadFullConfig()
	if err != nil {
		t.Fatalf("LoadFullConfig() error = %v", err)
	}

	if loaded.Server.UIPort != original.Server.UIPort {
		t.Fatalf("UIPort = %d, want %d", loaded.Server.UIPort, original.Server.UIPort)
	}
	if loaded.Server.UIHost != original.Server.UIHost {
		t.Fatalf("UIHost = %s, want %s", loaded.Server.UIHost, original.Server.UIHost)
	}
	if loaded.Flows.FlowFile != original.Flows.FlowFile {
		t.Fatalf("FlowFile = %s, want %s", loaded.Flows.FlowFile, original.Flows.FlowFile)
	}

	// Verify that config.json exists
	if !platform.Exists(filepath.Join(dataDir, "config.json")) {
		t.Fatal("config.json not created")
	}

	// Verify that settings.js exists
	if !platform.Exists(filepath.Join(dataDir, "settings.js")) {
		t.Fatal("settings.js not created")
	}
}

// TestConfigService_ValidateConfig_Valid tests that valid config returns valid=true
func TestConfigService_ValidateConfig_Valid(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewConfigService(dataDir, nil)

	// Save a base config first
	baseConfig := model.DefaultFullAppConfig()
	_ = service.SaveFullConfig(baseConfig)

	// Validate a valid modification
	validConfig := model.DefaultFullAppConfig()
	validConfig.Server.UIPort = 2000
	validConfig.Flows.FlowFile = "custom.json"

	result := service.ValidateConfig(validConfig)
	if !result.Valid {
		t.Fatalf("ValidateConfig() valid = false, errors = %v", result.Errors)
	}
}

// TestConfigService_ValidateConfig_Invalid tests that invalid config returns errors
func TestConfigService_ValidateConfig_Invalid(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewConfigService(dataDir, nil)

	// Save a base config first
	baseConfig := model.DefaultFullAppConfig()
	_ = service.SaveFullConfig(baseConfig)

	// Validate an invalid config (bad port, no flow file extension)
	invalidConfig := model.DefaultFullAppConfig()
	invalidConfig.Server.UIPort = 99999 // out of range
	invalidConfig.Flows.FlowFile = "noextension"

	result := service.ValidateConfig(invalidConfig)
	if result.Valid {
		t.Fatal("ValidateConfig() valid = true, want false")
	}
	if len(result.Errors) == 0 {
		t.Fatal("ValidateConfig() errors empty, want validation errors")
	}

	// Verify we got errors for the invalid fields
	hasPortError := false
	hasFlowFileError := false
	for _, err := range result.Errors {
		if strings.Contains(err.Field, "uiPort") {
			hasPortError = true
		}
		if strings.Contains(err.Field, "flowFile") {
			hasFlowFileError = true
		}
	}
	if !hasPortError {
		t.Fatal("Expected port validation error")
	}
	if !hasFlowFileError {
		t.Fatal("Expected flow file validation error")
	}
}

// TestConfigService_PreviewConfig_Stored tests that preview with nil cfg returns stored config rendered
func TestConfigService_PreviewConfig_Stored(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewConfigService(dataDir, nil)

	baseConfig := model.DefaultFullAppConfig()
	baseConfig.Server.UIPort = 3000
	_ = service.SaveFullConfig(baseConfig)

	preview, err := service.PreviewConfig(nil)
	if err != nil {
		t.Fatalf("PreviewConfig(nil) error = %v", err)
	}

	if !strings.Contains(preview, "uiPort: 3000") {
		t.Fatalf("PreviewConfig() did not contain expected port")
	}
}

// TestConfigService_PreviewConfig_Provided tests that preview with cfg returns that cfg rendered
func TestConfigService_PreviewConfig_Provided(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewConfigService(dataDir, nil)

	// Save default first
	_ = service.SaveFullConfig(model.DefaultFullAppConfig())

	// Create a different config
	customConfig := model.DefaultFullAppConfig()
	customConfig.Server.UIPort = 4000
	customConfig.Server.HTTPAdminRoot = "/custom"

	preview, err := service.PreviewConfig(&customConfig)
	if err != nil {
		t.Fatalf("PreviewConfig(cfg) error = %v", err)
	}

	if !strings.Contains(preview, "uiPort: 4000") {
		t.Fatalf("PreviewConfig() did not contain expected port")
	}
	if !strings.Contains(preview, "httpAdminRoot: \"/custom\"") {
		t.Fatalf("PreviewConfig() did not contain expected httpAdminRoot")
	}
}

// TestConfigService_ImportConfig tests that import of basic settings.js works
func TestConfigService_ImportConfig(t *testing.T) {
	t.Parallel()

	service := NewConfigService(t.TempDir(), nil)

	// A minimal valid settings.js-like string
	content := `
module.exports = {
  uiPort: 1880,
  httpAdminRoot: "/",
  flowFile: "flows.json"
};
`

	cfg, unrecognized, err := service.ImportConfig(content)
	if err != nil {
		t.Fatalf("ImportConfig() error = %v", err)
	}

	// Should have parsed something
	if cfg.Server.UIPort != 1880 {
		t.Fatalf("ImportConfig() UIPort = %d, want 1880", cfg.Server.UIPort)
	}
	if cfg.Flows.FlowFile != "flows.json" {
		t.Fatalf("ImportConfig() FlowFile = %s, want flows.json", cfg.Flows.FlowFile)
	}

	// Check unrecognized (may be empty or have items)
	_ = unrecognized // just verify it's returned
}

// TestConfigService_ApplyConfig_Valid tests that apply valid config writes files
func TestConfigService_ApplyConfig_Valid(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewConfigService(dataDir, nil)

	// Save a base config
	baseConfig := model.DefaultFullAppConfig()
	_ = service.SaveFullConfig(baseConfig)

	// Apply a valid change
	newConfig := model.DefaultFullAppConfig()
	newConfig.Server.UIPort = 5000
	newConfig.Flows.FlowFile = "runtime.json"

	result, err := service.ApplyConfig(newConfig, "testuser")
	if err != nil {
		t.Fatalf("ApplyConfig() error = %v", err)
	}
	if !result.Valid {
		t.Fatalf("ApplyConfig() valid = false, errors = %v", result.Errors)
	}

	// Verify config was written
	loaded, _ := service.LoadFullConfig()
	if loaded.Server.UIPort != 5000 {
		t.Fatalf("ApplyConfig() did not persist UIPort")
	}
	if loaded.Flows.FlowFile != "runtime.json" {
		t.Fatalf("ApplyConfig() did not persist FlowFile")
	}

	// Verify settings.js was written
	if !platform.Exists(filepath.Join(dataDir, "settings.js")) {
		t.Fatal("ApplyConfig() did not write settings.js")
	}
}

// TestConfigService_ApplyConfig_Invalid tests that apply invalid config returns errors without writing
func TestConfigService_ApplyConfig_Invalid(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewConfigService(dataDir, nil)

	// Save a base config
	baseConfig := model.DefaultFullAppConfig()
	_ = service.SaveFullConfig(baseConfig)

	// Try to apply an invalid config
	invalidConfig := model.DefaultFullAppConfig()
	invalidConfig.Server.UIPort = 99999 // out of range
	invalidConfig.Flows.FlowFile = "badfile"

	result, err := service.ApplyConfig(invalidConfig, "testuser")
	if err != nil {
		t.Fatalf("ApplyConfig() error = %v (should return nil error with invalid result)", err)
	}
	if result.Valid {
		t.Fatal("ApplyConfig() valid = true, want false")
	}
	if len(result.Errors) == 0 {
		t.Fatal("ApplyConfig() errors empty, want validation errors")
	}

	// Verify config was NOT changed (still has default)
	loaded, _ := service.LoadFullConfig()
	if loaded.Server.UIPort != baseConfig.Server.UIPort {
		t.Fatal("ApplyConfig() changed config despite validation error")
	}
}

// TestConfigService_ValidateConfig_RestartRequired tests that restart requirement is detected
func TestConfigService_ValidateConfig_RestartRequired(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewConfigService(dataDir, nil)

	// Save a base config
	baseConfig := model.DefaultFullAppConfig()
	_ = service.SaveFullConfig(baseConfig)

	// Change a restart-required field (uiPort)
	newConfig := model.DefaultFullAppConfig()
	newConfig.Server.UIPort = 2000

	result := service.ValidateConfig(newConfig)
	if !result.RestartRequired {
		t.Fatal("ValidateConfig() restartRequired = false, want true for uiPort change")
	}
}

// TestConfigService_ValidateConfig_NoRestartRequired tests that non-critical changes don't require restart
func TestConfigService_ValidateConfig_NoRestartRequired(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	service := NewConfigService(dataDir, nil)

	// Save a base config
	baseConfig := model.DefaultFullAppConfig()
	_ = service.SaveFullConfig(baseConfig)

	// Change a non-restart-required field (debugMaxLength)
	newConfig := model.DefaultFullAppConfig()
	newConfig.Runtime.DebugMaxLength = 2000

	result := service.ValidateConfig(newConfig)
	if result.RestartRequired {
		t.Fatal("ValidateConfig() restartRequired = true, want false for debugMaxLength change")
	}
}

// TestConfigService_GetConfigSchema tests that schema is returned
func TestConfigService_GetConfigSchema(t *testing.T) {
	t.Parallel()

	service := NewConfigService(t.TempDir(), nil)

	schema := service.GetConfigSchema()
	if schema == nil {
		t.Fatal("GetConfigSchema() returned nil")
	}

	// Verify schema has required top-level fields
	if schema["$schema"] == nil {
		t.Fatal("GetConfigSchema() missing $schema")
	}
	if schema["title"] == nil {
		t.Fatal("GetConfigSchema() missing title")
	}
	if schema["type"] == nil {
		t.Fatal("GetConfigSchema() missing type")
	}
	if schema["properties"] == nil {
		t.Fatal("GetConfigSchema() missing properties")
	}

	// Verify schema contains server property
	props := schema["properties"].(map[string]interface{})
	if props["server"] == nil {
		t.Fatal("GetConfigSchema() missing server property")
	}
}
