package service

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"text/template"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

type ConfigService struct {
	dataDir string
	db      *sql.DB
}

func NewConfigService(dataDir string, db *sql.DB) ConfigService {
	return ConfigService{dataDir: dataDir, db: db}
}

func DefaultAppConfig() model.AppConfig {
	return model.AppConfig{
		HTTPAdminRoot:      "/",
		FlowFile:           "flows.json",
		DiagnosticsEnabled: true,
		ProjectsEnabled:    false,
		CredentialSecret:   "",
	}
}

func (s ConfigService) Load() (model.AppConfig, error) {
	path := filepath.Join(s.dataDir, "config.json")
	if !platform.Exists(path) {
		return DefaultAppConfig(), nil
	}

	var cfg model.AppConfig
	if err := platform.ReadJSON(path, &cfg); err != nil {
		return model.AppConfig{}, fmt.Errorf("read config.json: %w", err)
	}
	return normalizeConfig(cfg), nil
}

func (s ConfigService) Validate(candidate model.AppConfig) model.ConfigValidationResult {
	current, err := s.Load()
	if err != nil {
		return model.ConfigValidationResult{
			Valid:  false,
			Errors: []string{err.Error()},
		}
	}

	candidate = normalizeConfig(candidate)
	errors := validateConfig(candidate)
	diff := diffConfig(current, candidate)

	return model.ConfigValidationResult{
		Valid:           len(errors) == 0,
		RestartRequired: len(diff) > 0,
		Errors:          errors,
		Diff:            diff,
	}
}

func (s ConfigService) Apply(candidate model.AppConfig) (model.ConfigValidationResult, error) {
	result := s.Validate(candidate)
	if !result.Valid {
		return result, nil
	}

	candidate = normalizeConfig(candidate)
	if err := platform.WriteJSONAtomic(filepath.Join(s.dataDir, "config.json"), candidate); err != nil {
		return result, fmt.Errorf("write config.json: %w", err)
	}

	settings, err := renderSettings(candidate)
	if err != nil {
		return result, err
	}
	if err := platform.WriteFileAtomic(filepath.Join(s.dataDir, "settings.js"), []byte(settings), 0o644); err != nil {
		return result, fmt.Errorf("write settings.js: %w", err)
	}

	return result, nil
}

func normalizeConfig(cfg model.AppConfig) model.AppConfig {
	defaults := DefaultAppConfig()
	cfg.HTTPAdminRoot = strings.TrimSpace(cfg.HTTPAdminRoot)
	cfg.FlowFile = strings.TrimSpace(cfg.FlowFile)
	cfg.CredentialSecret = strings.TrimSpace(cfg.CredentialSecret)

	if cfg.HTTPAdminRoot == "" {
		cfg.HTTPAdminRoot = defaults.HTTPAdminRoot
	}
	if !strings.HasPrefix(cfg.HTTPAdminRoot, "/") {
		cfg.HTTPAdminRoot = "/" + cfg.HTTPAdminRoot
	}
	if cfg.FlowFile == "" {
		cfg.FlowFile = defaults.FlowFile
	}

	return cfg
}

func validateConfig(cfg model.AppConfig) []string {
	var errors []string

	if !strings.HasPrefix(cfg.HTTPAdminRoot, "/") {
		errors = append(errors, "httpAdminRoot must start with /")
	}
	if strings.Contains(cfg.HTTPAdminRoot, " ") {
		errors = append(errors, "httpAdminRoot must not contain spaces")
	}
	if cfg.FlowFile == "" {
		errors = append(errors, "flowFile is required")
	}
	if strings.Contains(cfg.FlowFile, "/") || strings.Contains(cfg.FlowFile, `\`) {
		errors = append(errors, "flowFile must be a file name, not a path")
	}
	if cfg.CredentialSecret != "" && len(cfg.CredentialSecret) < 12 {
		errors = append(errors, "credentialSecret must be at least 12 characters when set")
	}

	return errors
}

func diffConfig(from, to model.AppConfig) []model.ConfigDiffEntry {
	var diff []model.ConfigDiffEntry

	if from.HTTPAdminRoot != to.HTTPAdminRoot {
		diff = append(diff, model.ConfigDiffEntry{Field: "httpAdminRoot", OldValue: from.HTTPAdminRoot, NewValue: to.HTTPAdminRoot})
	}
	if from.FlowFile != to.FlowFile {
		diff = append(diff, model.ConfigDiffEntry{Field: "flowFile", OldValue: from.FlowFile, NewValue: to.FlowFile})
	}
	if from.DiagnosticsEnabled != to.DiagnosticsEnabled {
		diff = append(diff, model.ConfigDiffEntry{Field: "diagnosticsEnabled", OldValue: boolString(from.DiagnosticsEnabled), NewValue: boolString(to.DiagnosticsEnabled)})
	}
	if from.ProjectsEnabled != to.ProjectsEnabled {
		diff = append(diff, model.ConfigDiffEntry{Field: "projectsEnabled", OldValue: boolString(from.ProjectsEnabled), NewValue: boolString(to.ProjectsEnabled)})
	}
	if from.CredentialSecret != to.CredentialSecret {
		diff = append(diff, model.ConfigDiffEntry{Field: "credentialSecret", OldValue: secretState(from.CredentialSecret), NewValue: secretState(to.CredentialSecret)})
	}

	return diff
}

func renderSettings(cfg model.AppConfig) (string, error) {
	const tpl = `module.exports = {
  uiPort: process.env.PORT || 1880,
  httpAdminRoot: "{{ .HTTPAdminRoot }}",
  flowFile: "{{ .FlowFile }}",
  diagnostics: {
    enabled: {{ .DiagnosticsEnabled }},
    ui: {{ .DiagnosticsEnabled }},
  },
  editorTheme: {
    projects: {
      enabled: {{ .ProjectsEnabled }},
    },
  },
  credentialSecret: {{ .CredentialSecretValue }},
}
`

	data := struct {
		HTTPAdminRoot         string
		FlowFile              string
		DiagnosticsEnabled    string
		ProjectsEnabled       string
		CredentialSecretValue string
	}{
		HTTPAdminRoot:         cfg.HTTPAdminRoot,
		FlowFile:              cfg.FlowFile,
		DiagnosticsEnabled:    boolString(cfg.DiagnosticsEnabled),
		ProjectsEnabled:       boolString(cfg.ProjectsEnabled),
		CredentialSecretValue: "false",
	}
	if cfg.CredentialSecret != "" {
		data.CredentialSecretValue = fmt.Sprintf("%q", cfg.CredentialSecret)
	}

	var buf bytes.Buffer
	if err := template.Must(template.New("settings").Parse(tpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render settings.js: %w", err)
	}

	return buf.String(), nil
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func secretState(value string) string {
	if value == "" {
		return "not set"
	}
	return "set"
}

// ── Phase E Methods ───────────────────────────────────────────────────

// LoadFullConfig reads config.json from dataDir and returns a FullAppConfig.
// If the file doesn't exist, returns DefaultFullAppConfig() with no error.
// If a JSON parse error occurs, returns an error.
// After loading, applies defaults via MergeWithDefaults.
func (s ConfigService) LoadFullConfig() (model.FullAppConfig, error) {
	path := filepath.Join(s.dataDir, "config.json")
	if !platform.Exists(path) {
		return model.DefaultFullAppConfig(), nil
	}

	var cfg model.FullAppConfig
	if err := platform.ReadJSON(path, &cfg); err != nil {
		return model.FullAppConfig{}, fmt.Errorf("read config.json: %w", err)
	}
	return model.MergeWithDefaults(cfg), nil
}

// SaveFullConfig writes the config to config.json and also renders and writes settings.js.
// Uses atomic writes for both files.
// After writing, updates the in-memory config cache.
func (s ConfigService) SaveFullConfig(cfg model.FullAppConfig) error {
	cfg = model.MergeWithDefaults(cfg)

	// Write config.json atomically
	if err := platform.WriteJSONAtomic(filepath.Join(s.dataDir, "config.json"), cfg); err != nil {
		return fmt.Errorf("write config.json: %w", err)
	}

	// Render and write settings.js atomically
	settings := RenderSettingsJS(cfg)
	if err := platform.WriteFileAtomic(filepath.Join(s.dataDir, "settings.js"), []byte(settings), 0o644); err != nil {
		return fmt.Errorf("write settings.js: %w", err)
	}

	return nil
}

// ValidateConfig validates a FullAppConfig against its current stored version.
// Returns ExtendedConfigValidationResult with per-field errors, diff, and restart requirement.
// RestartRequired is true if any of these fields changed:
// - server.uiPort, server.uiHost, server.httpAdminRoot, server.disableEditor
// - https.enabled, https.keyFile, https.certFile
func (s ConfigService) ValidateConfig(cfg model.FullAppConfig) model.ExtendedConfigValidationResult {
	// Validate the proposed config
	errors := ValidateFullAppConfig(cfg)

	// Load current config to compute diff
	current, err := s.LoadFullConfig()
	if err != nil {
		// If we can't load current config, treat as if empty (use defaults)
		current = model.DefaultFullAppConfig()
	}

	// Compute diff between current and proposed
	diff := ComputeConfigDiff(current, cfg)

	// Determine if restart is required by checking if any restart-required fields changed
	restartRequired := isRestartRequired(diff)

	return model.ExtendedConfigValidationResult{
		Valid:           len(errors) == 0,
		RestartRequired: restartRequired,
		Errors:          errors,
		Diff:            diff,
	}
}

// ApplyConfig applies a new FullAppConfig:
// 1. Validates the config — if invalid, returns validation result without writing
// 2. Creates a pre-apply snapshot of the current config (if db is available)
// 3. Writes the new config atomically (SaveFullConfig)
// 4. Logs the apply event to stdout
// 5. Returns the validation result (valid=true, diff, restartRequired)
func (s ConfigService) ApplyConfig(cfg model.FullAppConfig, username string) (model.ExtendedConfigValidationResult, error) {
	// Step 1: Validate
	result := s.ValidateConfig(cfg)
	if !result.Valid {
		// Return validation result without writing
		return result, nil
	}

	// Step 2: Create snapshot of current config if db is available
	if s.db != nil {
		current, _ := s.LoadFullConfig()
		currentJSON, _ := json.Marshal(current)
		settingsJS := RenderSettingsJS(current)

		snapService := NewSnapshotService(s.db)
		_, _ = snapService.CreateSnapshot(
			"Pre-apply snapshot",
			fmt.Sprintf("Applied by %s", username),
			string(currentJSON),
			settingsJS,
		)
	}

	// Step 3: Write the new config
	if err := s.SaveFullConfig(cfg); err != nil {
		return result, err
	}

	// Step 4: Log the apply event
	log.Printf("[CONFIG] Config applied by user: %s", username)

	// Step 5: Return result
	return result, nil
}

// PreviewConfig renders the given config as settings.js.
// If cfg is nil, uses the currently stored config.
// Does NOT write any files.
func (s ConfigService) PreviewConfig(cfg *model.FullAppConfig) (string, error) {
	var config model.FullAppConfig
	if cfg == nil {
		var err error
		config, err = s.LoadFullConfig()
		if err != nil {
			return "", err
		}
	} else {
		config = *cfg
	}

	return RenderSettingsJS(config), nil
}

// ImportConfig imports a settings.js file content and returns the parsed FullAppConfig.
// This is a pass-through to ImportSettingsJS but keeps the import logic centralized in the service layer.
func (s ConfigService) ImportConfig(content string) (model.FullAppConfig, []string, error) {
	return ImportSettingsJS(content)
}

// GetConfigSchema returns a JSON Schema (draft-07) describing FullAppConfig.
// Returns a map[string]interface{} that can be marshaled to JSON.
func (s ConfigService) GetConfigSchema() map[string]interface{} {
	return map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title":   "Node-RED Full Configuration",
		"type":    "object",
		"properties": map[string]interface{}{
			"server": map[string]interface{}{
				"type":        "object",
				"title":       "Server Configuration",
				"description": "Server address and port settings",
				"properties": map[string]interface{}{
					"uiPort": map[string]interface{}{
						"type":              "integer",
						"description":       "Port for the Node-RED editor UI",
						"default":           1880,
						"minimum":           1,
						"maximum":           65535,
						"x-restartRequired": true,
					},
					"uiHost": map[string]interface{}{
						"type":              "string",
						"description":       "Host for the Node-RED editor UI (0.0.0.0 = all interfaces)",
						"default":           "0.0.0.0",
						"x-restartRequired": true,
					},
					"httpAdminRoot": map[string]interface{}{
						"type":              "string",
						"description":       "Root path for admin UI (must start with /)",
						"default":           "/",
						"x-restartRequired": true,
					},
					"httpNodeRoot": map[string]interface{}{
						"type":        "string",
						"description": "Root path for nodes (set to \"false\" to disable)",
						"default":     "/",
					},
					"httpStatic": map[string]interface{}{
						"type":        "string",
						"description": "Serve static files from a directory",
						"default":     "",
					},
					"disableEditor": map[string]interface{}{
						"type":              "boolean",
						"description":       "Disable the visual editor",
						"default":           false,
						"x-restartRequired": true,
					},
				},
			},
			"security": map[string]interface{}{
				"type":        "object",
				"title":       "Security Configuration",
				"description": "Authentication and credential settings",
				"properties": map[string]interface{}{
					"credentialSecret": map[string]interface{}{
						"type":        "string",
						"description": "Secret used to encrypt node credentials (12+ chars if set)",
						"default":     "",
						"x-sensitive": true,
					},
					"sessionExpiryTime": map[string]interface{}{
						"type":        "integer",
						"description": "Session expiration time in seconds (300-2592000)",
						"default":     86400,
						"minimum":     300,
						"maximum":     2592000,
					},
					"adminAuth": map[string]interface{}{
						"type":        "object",
						"description": "Admin authentication configuration",
						"x-sensitive": true,
					},
					"httpNodeAuth": map[string]interface{}{
						"type":        "object",
						"description": "HTTP node authentication configuration",
						"x-sensitive": true,
					},
				},
			},
			"flows": map[string]interface{}{
				"type":        "object",
				"title":       "Flows Configuration",
				"description": "Flow storage and directory settings",
				"properties": map[string]interface{}{
					"flowFile": map[string]interface{}{
						"type":        "string",
						"description": "File name for storing flows (must end with .json)",
						"default":     "flows.json",
					},
					"flowFilePretty": map[string]interface{}{
						"type":        "boolean",
						"description": "Pretty-print flow file JSON",
						"default":     false,
					},
					"userDir": map[string]interface{}{
						"type":        "string",
						"description": "User directory path",
						"default":     "",
					},
					"nodesDir": map[string]interface{}{
						"type":        "string",
						"description": "Custom nodes directory path",
						"default":     "",
					},
				},
			},
			"contextStorage": map[string]interface{}{
				"type":        "object",
				"title":       "Context Storage Configuration",
				"description": "In-memory or persistent context storage",
				"properties": map[string]interface{}{
					"default": map[string]interface{}{
						"type":        "string",
						"description": "Default context storage name",
						"default":     "default",
					},
					"stores": map[string]interface{}{
						"type":        "object",
						"description": "Context storage stores (default: {\"default\": {\"module\": \"memory\"}})",
						"default": map[string]interface{}{
							"default": map[string]interface{}{
								"module": "memory",
							},
						},
					},
				},
			},
			"logging": map[string]interface{}{
				"type":        "object",
				"title":       "Logging Configuration",
				"description": "Logging levels and output",
				"properties": map[string]interface{}{
					"console": map[string]interface{}{
						"type":        "object",
						"description": "Console logging settings",
						"properties": map[string]interface{}{
							"level": map[string]interface{}{
								"type":        "string",
								"description": "Log level (fatal, error, warn, info, debug, trace)",
								"default":     "info",
								"enum":        []string{"fatal", "error", "warn", "info", "debug", "trace"},
							},
							"metrics": map[string]interface{}{
								"type":        "boolean",
								"description": "Enable metrics logging",
								"default":     false,
							},
							"audit": map[string]interface{}{
								"type":        "boolean",
								"description": "Enable audit logging",
								"default":     false,
							},
						},
					},
				},
			},
			"runtime": map[string]interface{}{
				"type":        "object",
				"title":       "Runtime Configuration",
				"description": "Runtime behavior and external modules",
				"properties": map[string]interface{}{
					"diagnosticsEnabled": map[string]interface{}{
						"type":        "boolean",
						"description": "Enable diagnostics",
						"default":     true,
					},
					"functionExternalModules": map[string]interface{}{
						"type":        "boolean",
						"description": "Allow external modules in function nodes",
						"default":     false,
					},
					"functionTimeout": map[string]interface{}{
						"type":        "integer",
						"description": "Function node timeout in seconds (0=no timeout)",
						"default":     0,
						"minimum":     0,
						"maximum":     3600,
					},
					"debugMaxLength": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum debug message length",
						"default":     1000,
						"minimum":     100,
						"maximum":     100000,
					},
					"nodeMessageBufferMaxLength": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum node message buffer length",
						"default":     0,
						"minimum":     0,
						"maximum":     10000,
					},
					"safeMode": map[string]interface{}{
						"type":        "boolean",
						"description": "Enable safe mode (disable all flows on start)",
						"default":     false,
					},
				},
			},
			"https": map[string]interface{}{
				"type":        "object",
				"title":       "HTTPS Configuration",
				"description": "HTTPS/TLS settings",
				"properties": map[string]interface{}{
					"enabled": map[string]interface{}{
						"type":              "boolean",
						"description":       "Enable HTTPS",
						"default":           false,
						"x-restartRequired": true,
					},
					"keyFile": map[string]interface{}{
						"type":              "string",
						"description":       "Path to TLS key file (required if HTTPS enabled)",
						"default":           "",
						"x-restartRequired": true,
					},
					"certFile": map[string]interface{}{
						"type":              "string",
						"description":       "Path to TLS certificate file (required if HTTPS enabled)",
						"default":           "",
						"x-restartRequired": true,
					},
					"caFile": map[string]interface{}{
						"type":        "string",
						"description": "Path to CA certificate file (optional)",
						"default":     "",
					},
				},
			},
			"nodeReconnect": map[string]interface{}{
				"type":        "object",
				"title":       "Node Reconnect Configuration",
				"description": "Reconnection timing for node connections",
				"properties": map[string]interface{}{
					"mqttReconnectTime": map[string]interface{}{
						"type":        "integer",
						"description": "MQTT reconnection delay (ms)",
						"default":     5000,
						"minimum":     100,
						"maximum":     300000,
					},
					"serialReconnectTime": map[string]interface{}{
						"type":        "integer",
						"description": "Serial reconnection delay (ms)",
						"default":     5000,
						"minimum":     100,
						"maximum":     300000,
					},
					"socketReconnectTime": map[string]interface{}{
						"type":        "integer",
						"description": "Socket reconnection delay (ms)",
						"default":     10000,
						"minimum":     100,
						"maximum":     300000,
					},
					"socketTimeout": map[string]interface{}{
						"type":        "integer",
						"description": "Socket timeout (ms)",
						"default":     120000,
						"minimum":     1000,
						"maximum":     600000,
					},
				},
			},
			"editorTheme": map[string]interface{}{
				"type":        "object",
				"title":       "Editor Theme Configuration",
				"description": "UI theme and appearance settings",
				"properties": map[string]interface{}{
					"theme": map[string]interface{}{
						"type":        "string",
						"description": "Theme name",
						"default":     "",
					},
					"tours": map[string]interface{}{
						"type":        "boolean",
						"description": "Enable editor tours",
						"default":     true,
					},
					"userMenu": map[string]interface{}{
						"type":        "boolean",
						"description": "Show user menu",
						"default":     true,
					},
					"projects": map[string]interface{}{
						"type":        "object",
						"description": "Projects configuration",
						"properties": map[string]interface{}{
							"enabled": map[string]interface{}{
								"type":        "boolean",
								"description": "Enable projects",
								"default":     false,
							},
						},
					},
				},
			},
			"palette": map[string]interface{}{
				"type":        "object",
				"title":       "Palette Configuration",
				"description": "Node palette categories",
				"properties": map[string]interface{}{
					"categories": map[string]interface{}{
						"type":        "array",
						"description": "Palette category names",
						"default":     []string{"subflows", "common", "function", "network", "sequence", "parser", "storage"},
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
		},
		"required": []string{"server", "security", "flows", "contextStorage", "logging", "runtime", "https", "nodeReconnect", "editorTheme", "palette"},
	}
}

// isRestartRequired checks if any restart-required field changed.
// Restart-required fields:
// - server.uiPort, server.uiHost, server.httpAdminRoot, server.disableEditor
// - https.enabled, https.keyFile, https.certFile
func isRestartRequired(diff []model.ConfigDiffEntry) bool {
	restartRequiredFields := map[string]bool{
		"server.uiPort":        true,
		"server.uiHost":        true,
		"server.httpAdminRoot": true,
		"server.disableEditor": true,
		"https.enabled":        true,
		"https.keyFile":        true,
		"https.certFile":       true,
	}

	for _, entry := range diff {
		if restartRequiredFields[entry.Field] {
			return true
		}
	}
	return false
}
