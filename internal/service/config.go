package service

import (
	"bytes"
	"database/sql"
	"fmt"
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
