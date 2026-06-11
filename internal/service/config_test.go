package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestConfigService_Get_DefaultsWhenNoConfig(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewIsolatedConfigService(tempDir)

	cfg, err := svc.Get()

	if err != nil {
		t.Errorf("Get should not error: %v", err)
	}

	// Should return some default configuration
	if cfg.HTTPAdminRoot == "" {
		t.Error("HTTPAdminRoot should be populated from defaults")
	}
	if cfg.HTTPNodeRoot == "" {
		t.Error("HTTPNodeRoot should be populated from defaults")
	}
}

func TestConfigService_Save_CreatesConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewIsolatedConfigService(tempDir)

	cfg := model.NodeRedConfig{
		Port:          1880,
		UIPort:        1880,
		HTTPAdminRoot: "/admin",
		HTTPNodeRoot:  "/node",
	}

	err := svc.Save(cfg)

	if err != nil {
		t.Errorf("Save should not error: %v", err)
	}

	// Verify config file exists
	configPath := filepath.Join(tempDir, "config.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("Config file should exist: %v", err)
	}
}

func TestConfigService_Save_UsesTempSettingsWhenNodeRedSettingsEnvIsExternal(t *testing.T) {
	tempDir := t.TempDir()
	externalDir := t.TempDir()
	externalSettings := filepath.Join(externalDir, "settings.js")
	t.Setenv("NODE_RED_SETTINGS", externalSettings)

	svc := NewIsolatedConfigService(tempDir)
	cfg := model.NodeRedConfig{
		Port:          1880,
		UIPort:        1880,
		HTTPAdminRoot: "/admin",
		HTTPNodeRoot:  "/node",
	}

	if err := svc.Save(cfg); err != nil {
		t.Fatalf("Save should not error: %v", err)
	}
	if _, err := os.Stat(externalSettings); !os.IsNotExist(err) {
		t.Fatalf("external NODE_RED_SETTINGS should not be written, stat err=%v", err)
	}
	settingsPath := filepath.Join(tempDir, "settings.js")
	if _, err := os.Stat(settingsPath); err != nil {
		t.Fatalf("settings.js should be written inside tempDir: %v", err)
	}
}

func TestConfigService_SaveAndLoad_RoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewIsolatedConfigService(tempDir)

	original := model.NodeRedConfig{
		Port:          3000,
		UIPort:        3000,
		UIHost:        "127.0.0.1",
		HTTPAdminRoot: "/red",
		HTTPNodeRoot:  "/api",
		FlowFile:      "flows.json",
		Lang:          "es",
	}

	// Save
	if err := svc.Save(original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load
	loaded, err := svc.Get()

	if err != nil {
		t.Errorf("Get after save failed: %v", err)
	}

	// Verify fields match
	if loaded.Port != original.Port {
		t.Errorf("Port mismatch: got %d, want %d", loaded.Port, original.Port)
	}
	if loaded.UIPort != original.UIPort {
		t.Errorf("UIPort mismatch: got %d, want %d", loaded.UIPort, original.UIPort)
	}
	if loaded.UIHost != original.UIHost {
		t.Errorf("UIHost mismatch: got %s, want %s", loaded.UIHost, original.UIHost)
	}
	if loaded.HTTPAdminRoot != original.HTTPAdminRoot {
		t.Errorf("HTTPAdminRoot mismatch: got %s, want %s", loaded.HTTPAdminRoot, original.HTTPAdminRoot)
	}
}

func TestConfigService_GetRawSettings_CreatesDefaultIfNotExists(t *testing.T) {
	tempDir := t.TempDir()
	hostSvc := NewIsolatedHostService(tempDir)
	svc := NewConfigServiceWithHost(tempDir, hostSvc)

	doc, err := svc.GetRawSettings()

	if err != nil {
		t.Errorf("GetRawSettings should not error when file missing: %v", err)
	}

	// Content should be set (either from file or defaults)
	if doc.Path == "" {
		t.Error("Path should be populated")
	}

	// Verify it contains valid JavaScript-like structure
	if doc.Content != "" && !contains(doc.Content, "module.exports") && !contains(doc.Content, "{") {
		t.Error("Content should be valid Node-RED settings format")
	}
}

func TestConfigService_SaveRawSettings_CreatesBackup(t *testing.T) {
	tempDir := t.TempDir()
	hostSvc := NewIsolatedHostService(tempDir)
	svc := NewConfigServiceWithHost(tempDir, hostSvc)

	// First save — no backup expected (nothing to back up yet)
	initialContent := "module.exports = { uiPort: 1880, httpAdminRoot: '/' }\n"
	_, err := svc.SaveRawSettings(initialContent)
	if err != nil {
		t.Fatalf("First SaveRawSettings failed: %v", err)
	}

	// Second save — now there is a previous file, so backup MUST be created
	updatedContent := "module.exports = { uiPort: 1881, httpAdminRoot: '/' }\n"
	doc, err := svc.SaveRawSettings(updatedContent)
	if err != nil {
		t.Errorf("Second SaveRawSettings failed: %v", err)
	}

	// Check that backup was created on the second write
	backupDir := filepath.Join(tempDir, "backups", "settings")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Backup directory should exist after second save: %v", err)
	}
	if len(entries) == 0 {
		t.Error("Backup file should exist after second save")
	}
	if doc.BackupPath == "" {
		t.Error("BackupPath should be populated in returned document after second save")
	}
}

func TestConfigService_SaveRawSettings_WritesContent(t *testing.T) {
	tempDir := t.TempDir()
	hostSvc := NewIsolatedHostService(tempDir)
	svc := NewConfigServiceWithHost(tempDir, hostSvc)

	content := "module.exports = { uiPort: 3000 }\n"
	doc, err := svc.SaveRawSettings(content)

	if err != nil {
		t.Errorf("SaveRawSettings failed: %v", err)
	}

	// Read the file back
	savedContent, err := os.ReadFile(doc.Path)

	if err != nil {
		t.Errorf("Failed to read saved file: %v", err)
	}

	if string(savedContent) != content {
		t.Errorf("Content mismatch. Got %q, want %q", string(savedContent), content)
	}
}

func TestConfigService_Validate_RequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		cfg     model.NodeRedConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: model.NodeRedConfig{
				Port:          1880,
				UIPort:        1880,
				HTTPAdminRoot: "/",
				HTTPNodeRoot:  "/",
			},
			wantErr: false,
		},
		{
			name: "invalid port zero",
			cfg: model.NodeRedConfig{
				Port:          0,
				UIPort:        0,
				HTTPAdminRoot: "/",
				HTTPNodeRoot:  "/",
			},
			wantErr: true,
		},
		{
			name: "invalid port too high",
			cfg: model.NodeRedConfig{
				Port:          99999,
				UIPort:        99999,
				HTTPAdminRoot: "/",
				HTTPNodeRoot:  "/",
			},
			wantErr: true,
		},
		{
			name: "missing HTTPAdminRoot",
			cfg: model.NodeRedConfig{
				Port:         1880,
				UIPort:       1880,
				HTTPNodeRoot: "/",
			},
			wantErr: true,
		},
		{
			name: "missing HTTPNodeRoot",
			cfg: model.NodeRedConfig{
				Port:          1880,
				UIPort:        1880,
				HTTPAdminRoot: "/",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewIsolatedConfigService(t.TempDir())
			err := svc.Validate(tt.cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigService_ValidateAdminAuth_Valid(t *testing.T) {
	auth := &model.AdminAuth{
		Type: "credentials",
		Users: []model.AdminAuthUser{
			{
				Username:    "admin",
				Password:    "password123",
				Permissions: "*",
			},
		},
	}

	err := validateAdminAuth(auth)

	if err != nil {
		t.Errorf("validateAdminAuth should not error for valid input: %v", err)
	}
}

func TestConfigService_ValidateAdminAuth_InvalidUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"empty username", "", true},
		{"too short", "ab", true},
		{"valid", "admin", false},
		{"valid longer", "user123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &model.AdminAuth{
				Type: "credentials",
				Users: []model.AdminAuthUser{
					{
						Username:    tt.username,
						Password:    "password123",
						Permissions: "*",
					},
				},
			}

			err := validateAdminAuth(auth)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateAdminAuth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigService_ValidateAdminAuth_InvalidPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"empty password", "", true},
		{"too short", "pass", true},
		{"valid", "password123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &model.AdminAuth{
				Type: "credentials",
				Users: []model.AdminAuthUser{
					{
						Username:    "admin",
						Password:    tt.password,
						Permissions: "*",
					},
				},
			}

			err := validateAdminAuth(auth)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateAdminAuth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigService_ParseConfigFromContent_ExtractsPort(t *testing.T) {
	svc := NewIsolatedConfigService(t.TempDir())

	content := `module.exports = {
  uiPort: 3000,
  httpAdminRoot: '/admin',
  httpNodeRoot: '/api'
}`

	cfg := svc.parseConfigFromContent(content)

	if cfg.Port != 3000 {
		t.Errorf("Expected Port 3000, got %d", cfg.Port)
	}
	if cfg.HTTPAdminRoot != "/admin" {
		t.Errorf("Expected HTTPAdminRoot '/admin', got %s", cfg.HTTPAdminRoot)
	}
}

func TestConfigService_ParseConfigFromContent_WithLangAndFlowFile(t *testing.T) {
	svc := NewIsolatedConfigService(t.TempDir())

	content := `module.exports = {
  uiPort: 1880,
  httpAdminRoot: '/',
  httpNodeRoot: '/',
  lang: 'es',
  flowFile: 'my-flows.json',
  disableEditor: true
}`

	cfg := svc.parseConfigFromContent(content)

	if cfg.Lang != "es" {
		t.Errorf("Expected Lang 'es', got %s", cfg.Lang)
	}
	if cfg.FlowFile != "my-flows.json" {
		t.Errorf("Expected FlowFile 'my-flows.json', got %s", cfg.FlowFile)
	}
	if !cfg.DisableEditor {
		t.Error("Expected DisableEditor to be true")
	}
}

func TestConfigService_RenderSettingsJS_StructureValid(t *testing.T) {
	cfg := model.NodeRedConfig{
		Port:          1880,
		UIHost:        "0.0.0.0",
		HTTPAdminRoot: "/",
		HTTPNodeRoot:  "/",
		FlowFile:      "flows.json",
	}

	rendered := renderSettingsJS(cfg)

	// Verify basic structure
	if !contains(rendered, "module.exports") {
		t.Error("Should start with module.exports")
	}
	if !contains(rendered, "uiPort:") {
		t.Error("Should contain uiPort")
	}
	if !contains(rendered, "1880") {
		t.Error("Should contain port number")
	}
	if !contains(rendered, "httpAdminRoot") {
		t.Error("Should contain httpAdminRoot")
	}
}

func TestConfigService_RenderSettingsJS_WithAdminAuth(t *testing.T) {
	cfg := model.NodeRedConfig{
		Port:          1880,
		HTTPAdminRoot: "/",
		HTTPNodeRoot:  "/",
		AdminAuth: &model.AdminAuth{
			Type: "credentials",
			Users: []model.AdminAuthUser{
				{
					Username:    "admin",
					Password:    "hashedpwd123",
					Permissions: "*",
				},
			},
		},
	}

	rendered := renderSettingsJS(cfg)

	if !contains(rendered, "adminAuth") {
		t.Error("Should contain adminAuth")
	}
	if !contains(rendered, "admin") {
		t.Error("Should contain username")
	}
	if !contains(rendered, "hashedpwd123") {
		t.Error("Should contain password")
	}
}

func TestParseStringFromJS_Valid(t *testing.T) {
	content := `module.exports = { httpAdminRoot: '/admin', uiHost: '0.0.0.0' }`

	result := parseStringFromJS(content, "httpAdminRoot", "default")

	if result != "/admin" {
		t.Errorf("Expected '/admin', got %q", result)
	}
}

func TestParseStringFromJS_Fallback(t *testing.T) {
	content := `module.exports = { port: 1880 }`

	result := parseStringFromJS(content, "notfound", "fallback")

	if result != "fallback" {
		t.Errorf("Expected 'fallback', got %q", result)
	}
}

func TestParseIntFromJS_Valid(t *testing.T) {
	content := `module.exports = { uiPort: 3000 }`

	result := parseIntFromJS(content, "uiPort", 1880)

	if result != 3000 {
		t.Errorf("Expected 3000, got %d", result)
	}
}

func TestParseIntFromJS_Fallback(t *testing.T) {
	content := `module.exports = { port: 'abc' }`

	result := parseIntFromJS(content, "port", 1880)

	if result != 1880 {
		t.Errorf("Expected 1880, got %d", result)
	}
}

func TestParseBoolFromJS_True(t *testing.T) {
	content := `module.exports = { disableEditor: true }`

	result := parseBoolFromJS(content, "disableEditor", false)

	if !result {
		t.Error("Expected true")
	}
}

func TestParseBoolFromJS_False(t *testing.T) {
	content := `module.exports = { projectsEnabled: false }`

	result := parseBoolFromJS(content, "projectsEnabled", true)

	if result {
		t.Error("Expected false")
	}
}

func TestConfigService_GetDefault_HasRequiredFields(t *testing.T) {
	svc := NewIsolatedConfigService(t.TempDir())

	cfg := svc.GetDefault()

	if cfg.HTTPAdminRoot == "" {
		t.Error("Default config should have HTTPAdminRoot")
	}
	if cfg.HTTPNodeRoot == "" {
		t.Error("Default config should have HTTPNodeRoot")
	}
	if cfg.Port == 0 {
		t.Error("Default config should have non-zero Port")
	}
}

func TestConfigService_PreserveAdminAuthPasswords(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewIsolatedConfigService(tempDir)

	// Save initial config with password
	initial := model.NodeRedConfig{
		Port:          1880,
		UIPort:        1880,
		HTTPAdminRoot: "/",
		HTTPNodeRoot:  "/",
		AdminAuth: &model.AdminAuth{
			Type: "credentials",
			Users: []model.AdminAuthUser{
				{
					Username:    "admin",
					Password:    "hashedpwd123",
					Permissions: "*",
				},
			},
		},
	}
	svc.Save(initial)

	// Now create update with empty password (meaning don't change)
	updated := model.NodeRedConfig{
		Port:          1880,
		UIPort:        1880,
		HTTPAdminRoot: "/",
		HTTPNodeRoot:  "/",
		AdminAuth: &model.AdminAuth{
			Type: "credentials",
			Users: []model.AdminAuthUser{
				{
					Username:    "admin",
					Password:    "", // Empty - should preserve
					Permissions: "*",
				},
			},
		},
	}

	// This should preserve the password
	if err := svc.preserveAdminAuthPasswords(&updated); err != nil {
		t.Errorf("preserveAdminAuthPasswords failed: %v", err)
	}

	if updated.AdminAuth.Users[0].Password != "hashedpwd123" {
		t.Errorf("Password should be preserved, got %s", updated.AdminAuth.Users[0].Password)
	}
}

func TestConfigService_DecorateConfig_SetsSettingsPath(t *testing.T) {
	tempDir := t.TempDir()
	hostSvc := NewIsolatedHostService(tempDir)
	svc := NewConfigServiceWithHost(tempDir, hostSvc)

	cfg := model.NodeRedConfig{
		Port:          1880,
		HTTPAdminRoot: "/",
		HTTPNodeRoot:  "/",
	}

	svc.decorateConfig(&cfg)

	if cfg.SettingsPath == "" {
		t.Error("SettingsPath should be set")
	}
	if cfg.SettingsSource == "" {
		t.Error("SettingsSource should be set")
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i < len(s); i++ {
		if len(s[i:]) < len(substr) {
			return false
		}
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// ============ NEW TESTS FOR ROUND-TRIP PRESERVATION ============

// TestConfigService_RoundTripPreservation_WithCustomContent verifies that unknown blocks
// are preserved when patching settings.js (Fix 1: Patch Strategy)
func TestConfigService_RoundTripPreservation_WithCustomContent(t *testing.T) {
	tempDir := t.TempDir()

	// Create a settings.js fixture with custom blocks that should be preserved
	fixtureContent := `module.exports = {
  uiPort: 1880,
  httpAdminRoot: "/",
  httpNodeRoot: "/",
  projectsEnabled: true,
  disableEditor: false,

  // Custom function global context - should be preserved
  functionGlobalContext: {
    lodash: require("lodash"),
    moment: require("moment"),
  },

  // Export custom context keys - should be preserved
  exportGlobalContextKeys: ["lodash", "moment"],

  // Custom middleware - should be preserved
  httpMiddleware: function(req, res, next) {
    if (req.path === "/custom-route") {
      res.send("custom");
    } else {
      next();
    }
  },

  adminAuth: null,
  editorTheme: {
    projects: { enabled: true },
  },
  logging: {
    console: { level: "info", metrics: false, audit: false },
  },
}
`

	// Save the fixture
	settingsPath := filepath.Join(tempDir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(fixtureContent), 0644); err != nil {
		t.Fatalf("Failed to write fixture: %v", err)
	}

	// Now patch it (simulate Save with visual editor changes)
	newCfg := model.NodeRedConfig{
		Port:            3000,
		UIPort:          3000,
		HTTPAdminRoot:   "/admin",
		HTTPNodeRoot:    "/api",
		ProjectsEnabled: true,
		DisableEditor:   true,
	}

	// Use patchSettingsJS to update the content
	patchedContent := patchSettingsJS(fixtureContent, newCfg)

	// Verify modelled keys were updated
	if !contains(patchedContent, "uiPort: 3000") {
		t.Error("uiPort should be updated to 3000")
	}
	if !contains(patchedContent, `httpAdminRoot: "/admin"`) {
		t.Error("httpAdminRoot should be updated to /admin")
	}
	if !contains(patchedContent, `httpNodeRoot: "/api"`) {
		t.Error("httpNodeRoot should be updated to /api")
	}
	if !contains(patchedContent, "disableEditor: true") {
		t.Error("disableEditor should be updated to true")
	}

	// Verify unknown blocks are still present
	if !contains(patchedContent, "functionGlobalContext") {
		t.Error("functionGlobalContext should be preserved (unknown block)")
	}
	if !contains(patchedContent, "exportGlobalContextKeys") {
		t.Error("exportGlobalContextKeys should be preserved (unknown block)")
	}
	if !contains(patchedContent, "httpMiddleware") {
		t.Error("httpMiddleware should be preserved (unknown block)")
	}
	if !contains(patchedContent, `require("lodash")`) {
		t.Error("require() calls should be preserved")
	}
	if !contains(patchedContent, `require("moment")`) {
		t.Error("require() calls should be preserved")
	}
}

// TestParseProjectsEnabledFromJS_FixedRegex verifies that projectsEnabled is parsed correctly
// and not confused by logging.console.metrics.enabled (Fix 2: projectsEnabled Regex)
func TestParseProjectsEnabledFromJS_FixedRegex(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "standalone projectsEnabled true",
			content: `module.exports = {
  projectsEnabled: true,
  logging: { console: { metrics: false, audit: false } }
}`,
			expected: true,
		},
		{
			name: "standalone projectsEnabled false with logging.metrics before it",
			content: `module.exports = {
  logging: {
    console: {
      level: "info",
      metrics: false,
      audit: false
    }
  },
  projectsEnabled: true
}`,
			expected: true,
		},
		{
			name: "projectsEnabled in editorTheme.projects.enabled",
			content: `module.exports = {
  editorTheme: {
    projects: { enabled: true }
  }
}`,
			expected: true,
		},
		{
			name: "projectsEnabled false both places",
			content: `module.exports = {
  projectsEnabled: false,
  editorTheme: {
    projects: { enabled: false }
  }
}`,
			expected: false,
		},
		{
			name: "no projectsEnabled, fallback",
			content: `module.exports = {
  uiPort: 1880,
  httpAdminRoot: "/"
}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseProjectsEnabledFromJS(tt.content, false)
			if result != tt.expected {
				t.Errorf("parseProjectsEnabledFromJS() got %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestParseAdminAuthFromJS_FromRawSettings verifies that adminAuth is correctly parsed
// from raw settings.js content (Fix 3: Parse adminAuth from raw)
func TestParseAdminAuthFromJS_FromRawSettings(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		expectNil  bool
		expectType string
		expectUser model.AdminAuthUser
	}{
		{
			name: "valid adminAuth with one user",
			content: `module.exports = {
  adminAuth: {
    type: "credentials",
    users: [{
      username: "admin",
      password: "hashedpwd123",
      permissions: "*"
    }]
  }
}`,
			expectNil:  false,
			expectType: "credentials",
			expectUser: model.AdminAuthUser{
				Username:    "admin",
				Password:    "hashedpwd123",
				Permissions: "*",
			},
		},
		{
			name: "adminAuth null",
			content: `module.exports = {
  adminAuth: null,
  uiPort: 1880
}`,
			expectNil: true,
		},
		{
			name: "adminAuth missing entirely",
			content: `module.exports = {
  uiPort: 1880,
  httpAdminRoot: "/"
}`,
			expectNil: true,
		},
		{
			name: "adminAuth with special characters in password",
			content: `module.exports = {
  adminAuth: {
    type: "credentials",
    users: [{
      username: "admin",
      password: "p@ssw0rd!#$%",
      permissions: "*"
    }]
  }
}`,
			expectNil:  false,
			expectType: "credentials",
			expectUser: model.AdminAuthUser{
				Username:    "admin",
				Password:    "p@ssw0rd!#$%",
				Permissions: "*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAdminAuthFromJS(tt.content)

			if tt.expectNil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Error("Expected non-nil AdminAuth, got nil")
				return
			}

			if result.Type != tt.expectType {
				t.Errorf("Type mismatch: got %s, want %s", result.Type, tt.expectType)
			}

			if len(result.Users) == 0 {
				t.Error("Expected at least one user")
				return
			}

			user := result.Users[0]
			if user.Username != tt.expectUser.Username {
				t.Errorf("Username mismatch: got %s, want %s", user.Username, tt.expectUser.Username)
			}
			if user.Password != tt.expectUser.Password {
				t.Errorf("Password mismatch: got %s, want %s", user.Password, tt.expectUser.Password)
			}
			if user.Permissions != tt.expectUser.Permissions {
				t.Errorf("Permissions mismatch: got %s, want %s", user.Permissions, tt.expectUser.Permissions)
			}
		})
	}
}

// TestConfigService_SaveRawSettings_SyncsAdminAuth verifies that adminAuth from raw settings.js
// is correctly synced to config.json (part of Fix 3)
func TestConfigService_SaveRawSettings_SyncsAdminAuth(t *testing.T) {
	tempDir := t.TempDir()
	hostSvc := NewIsolatedHostService(tempDir)
	svc := NewConfigServiceWithHost(tempDir, hostSvc)

	// Raw settings with adminAuth
	rawSettings := `module.exports = {
  uiPort: 1880,
  httpAdminRoot: "/",
  httpNodeRoot: "/",
  adminAuth: {
    type: "credentials",
    users: [{
      username: "admin",
      password: "hashedpwd456",
      permissions: "*"
    }]
  }
}`

	// Save raw settings with syncStore=true
	_, err := svc.SaveRawSettings(rawSettings)
	if err != nil {
		t.Fatalf("SaveRawSettings failed: %v", err)
	}

	// Load config and verify adminAuth was synced
	cfg, err := svc.Get()
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if cfg.AdminAuth == nil {
		t.Error("AdminAuth should be populated from raw settings.js")
	} else {
		if cfg.AdminAuth.Type != "credentials" {
			t.Errorf("AdminAuth Type should be 'credentials', got %s", cfg.AdminAuth.Type)
		}
		if len(cfg.AdminAuth.Users) != 1 {
			t.Errorf("AdminAuth should have 1 user, got %d", len(cfg.AdminAuth.Users))
		} else {
			user := cfg.AdminAuth.Users[0]
			if user.Username != "admin" {
				t.Errorf("Username should be 'admin', got %s", user.Username)
			}
			if user.Password != "hashedpwd456" {
				t.Errorf("Password should be 'hashedpwd456', got %s", user.Password)
			}
		}
	}
}

func TestGenerateSettingsJS_RendersNodeRedGlobalEnvAndSkipsSecrets(t *testing.T) {
	cfg := model.DefaultNodeRedConfig()
	cfg.EnvVars = []model.EnvVar{
		{Key: "PUBLIC_API_URL", Value: "https://example.test", Type: "string"},
		{Key: "FEATURE_FLAG", Value: "true", Type: "boolean"},
		{Key: "API_TOKEN", Value: "super-secret", Type: "secret", Encrypted: true},
	}

	content := generateSettingsJS(cfg)

	for _, want := range []string{
		`env: [`,
		`name: "PUBLIC_API_URL"`,
		`value: "https://example.test"`,
		`type: "str"`,
		`name: "FEATURE_FLAG"`,
		`type: "bool"`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("settings.js missing %q in:\n%s", want, content)
		}
	}
	if strings.Contains(content, "API_TOKEN") || strings.Contains(content, "super-secret") {
		t.Fatalf("secret env vars must not be written to Node-RED settings.js env array:\n%s", content)
	}
}

func TestPatchSettingsJS_ReplacesExistingEnvArray(t *testing.T) {
	cfg := model.DefaultNodeRedConfig()
	cfg.EnvVars = []model.EnvVar{{Key: "NEW_VALUE", Value: "42", Type: "number"}}
	existing := `module.exports = {
  uiPort: 1880,
  env: [
    {
      name: "OLD_VALUE",
      value: "old",
      type: "str"
    }
  ],
  logging: { console: { level: 'info' } },
}`

	content := patchSettingsJS(existing, cfg)

	if !strings.Contains(content, `name: "NEW_VALUE"`) || !strings.Contains(content, `value: "42"`) {
		t.Fatalf("patched settings.js missing new env value:\n%s", content)
	}
	if strings.Contains(content, "OLD_VALUE") {
		t.Fatalf("patched settings.js should remove stale env values:\n%s", content)
	}
	if !strings.Contains(content, "logging:") {
		t.Fatalf("patched settings.js should preserve unrelated blocks:\n%s", content)
	}
}
