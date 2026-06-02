package service

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/store"
)

// ConfigService handles Node-RED configuration
type ConfigService struct {
	store   *store.JSONStore[model.NodeRedConfig]
	dataDir string
	hostSvc *HostService
}

// NewConfigService creates a new config service
func NewConfigService(dataDir string) *ConfigService {
	return NewConfigServiceWithHost(dataDir, NewHostService(dataDir))
}

// NewIsolatedConfigService creates a config service whose settings.js writes are
// confined to dataDir. It is intended for hermetic tests.
func NewIsolatedConfigService(dataDir string) *ConfigService {
	return NewConfigServiceWithHost(dataDir, NewIsolatedHostService(dataDir))
}

// NewConfigServiceWithHost creates a config service with shared host detection.
func NewConfigServiceWithHost(dataDir string, hostSvc *HostService) *ConfigService {
	configPath := filepath.Join(dataDir, "config.json")
	jsonStore := store.NewJSONStore[model.NodeRedConfig](configPath)

	return &ConfigService{
		store:   jsonStore,
		dataDir: dataDir,
		hostSvc: hostSvc,
	}
}

// Get retrieves the current configuration
func (s *ConfigService) Get() (model.NodeRedConfig, error) {
	if s.store.Exists() {
		cfg, err := s.store.Read()
		if err != nil {
			return cfg, err
		}
		s.decorateConfig(&cfg)
		return cfg, nil
	}

	if cfg, ok := s.readFromSettingsFile(); ok {
		s.decorateConfig(&cfg)
		return cfg, nil
	}

	// Return defaults if no config exists
	cfg := s.GetDefault()
	s.decorateConfig(&cfg)
	return cfg, nil
}

// Save saves the configuration
func (s *ConfigService) Save(cfg model.NodeRedConfig) error {
	// If adminAuth is provided, handle password preservation
	if cfg.AdminAuth != nil {
		if err := s.preserveAdminAuthPasswords(&cfg); err != nil {
			return err
		}
	}

	// Validate before saving
	if err := s.Validate(cfg); err != nil {
		return err
	}

	s.decorateConfig(&cfg)
	if err := s.store.Write(cfg); err != nil {
		return err
	}
	_, err := s.writeSettingsFile(renderSettingsJS(cfg), false)
	return err
}

// preserveAdminAuthPasswords preserves existing password hashes when empty password is provided
// This allows frontend to send empty password (meaning "don't change") and backend will use the existing hash.
// It reads ONLY from nrcc's own JSON store — NOT from the live settings.js — to avoid reading from the
// system-wide ~/.node-red/settings.js, which would pollute tests and cause incorrect cross-user behaviour.
func (s *ConfigService) preserveAdminAuthPasswords(cfg *model.NodeRedConfig) error {
	if cfg.AdminAuth == nil || len(cfg.AdminAuth.Users) == 0 {
		return nil
	}

	// Only preserve from nrcc's own persisted store, not from the system settings.js.
	if !s.store.Exists() {
		return nil
	}
	existing, err := s.store.Read()
	if err != nil {
		return nil
	}

	if existing.AdminAuth == nil || len(existing.AdminAuth.Users) == 0 {
		// No existing adminAuth to preserve from
		return nil
	}

	// For each user in the new config, if password is empty, use the existing password hash
	for i, newUser := range cfg.AdminAuth.Users {
		if newUser.Password == "" {
			// Find matching user in existing config by username
			for _, existingUser := range existing.AdminAuth.Users {
				if existingUser.Username == newUser.Username {
					cfg.AdminAuth.Users[i].Password = existingUser.Password
					break
				}
			}
		}
	}

	return nil
}

// GetDefault returns the default configuration
func (s *ConfigService) GetDefault() model.NodeRedConfig {
	return model.DefaultNodeRedConfig()
}

// GetRawSettings loads the active settings.js document.
func (s *ConfigService) GetRawSettings() (model.SettingsDocument, error) {
	status := s.hostSvc.Detect()
	doc := status.Settings
	content, err := os.ReadFile(doc.Path)
	if err != nil {
		if os.IsNotExist(err) {
			doc.Content = renderSettingsJS(s.GetDefault())
			return doc, nil
		}
		return doc, err
	}
	doc.Content = string(content)
	return doc, nil
}

// SaveRawSettings writes the active settings.js document after backing up the previous version.
func (s *ConfigService) SaveRawSettings(content string) (model.SettingsDocument, error) {
	return s.writeSettingsFile(content, true)
}

func (s *ConfigService) writeSettingsFile(content string, syncStore bool) (model.SettingsDocument, error) {
	status := s.hostSvc.Detect()
	doc := status.Settings
	if doc.Path == "" {
		doc.Path = filepath.Join(s.dataDir, "settings.js")
	}
	if err := os.MkdirAll(filepath.Dir(doc.Path), 0755); err != nil {
		return doc, err
	}
	backupPath, err := s.backupSettingsFile(doc.Path)
	if err != nil {
		return doc, err
	}
	if err := os.WriteFile(doc.Path, []byte(content), 0644); err != nil {
		return doc, err
	}
	if syncStore {
		parsed := s.parseConfigFromContent(content)
		s.decorateConfig(&parsed)
		if err := s.store.Write(parsed); err != nil {
			return doc, err
		}
	}
	doc.BackupPath = backupPath
	doc.Content = content
	doc.Writable = true
	return doc, nil
}

// Validate validates a configuration
func (s *ConfigService) Validate(cfg model.NodeRedConfig) error {
	// Use Port if set, otherwise use UIPort
	port := cfg.Port
	if port == 0 && cfg.UIPort > 0 {
		port = cfg.UIPort
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	if cfg.HTTPAdminRoot == "" {
		return fmt.Errorf("httpAdminRoot cannot be empty")
	}

	if cfg.HTTPNodeRoot == "" {
		return fmt.Errorf("httpNodeRoot cannot be empty")
	}

	// Validate Admin Auth if configured
	if cfg.AdminAuth != nil {
		if err := validateAdminAuth(cfg.AdminAuth); err != nil {
			return err
		}
	}

	return nil
}

// validateAdminAuth validates AdminAuth configuration
func validateAdminAuth(auth *model.AdminAuth) error {
	if auth.Type == "" {
		return fmt.Errorf("adminAuth type must not be empty")
	}

	if len(auth.Users) == 0 {
		return fmt.Errorf("adminAuth must have at least one user")
	}

	for i, user := range auth.Users {
		if user.Username == "" {
			return fmt.Errorf("adminAuth user %d: username must not be empty", i)
		}

		if user.Password == "" {
			return fmt.Errorf("adminAuth user %d: password must not be empty", i)
		}

		if len(user.Username) < 3 {
			return fmt.Errorf("adminAuth user %d: username must be at least 3 characters", i)
		}

		if len(user.Password) < 6 {
			return fmt.Errorf("adminAuth user %d: password must be at least 6 characters", i)
		}
	}

	return nil
}

func (s *ConfigService) decorateConfig(cfg *model.NodeRedConfig) {
	if s.hostSvc == nil {
		return
	}
	settings := s.hostSvc.Detect().Settings
	cfg.SettingsPath = settings.Path
	cfg.SettingsSource = settings.Source
}

func (s *ConfigService) readFromSettingsFile() (model.NodeRedConfig, bool) {
	doc, err := s.GetRawSettings()
	if err != nil || doc.Content == "" {
		return model.NodeRedConfig{}, false
	}
	return s.parseConfigFromContent(doc.Content), true
}

func (s *ConfigService) backupSettingsFile(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	backupDir := filepath.Join(s.dataDir, "backups", "settings")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}
	backupPath := filepath.Join(backupDir, fmt.Sprintf("settings-%s.js.bak", time.Now().UTC().Format("20060102-150405")))
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return backupPath, os.WriteFile(backupPath, data, 0644)
}

// renderSettingsJS either patches existing settings.js or generates a new one
func renderSettingsJS(cfg model.NodeRedConfig) string {
	// Note: We don't attempt to read the file from disk here because:
	// 1. renderSettingsJS is called during Save(), and we'd need a file path
	// 2. The patch strategy would be best called from writeSettingsFile() where we have the path
	// For now, we generate from scratch. A future enhancement could implement patching.
	return generateSettingsJS(cfg)
}

// generateSettingsJS builds settings.js from scratch (fallback for first-run)
func generateSettingsJS(cfg model.NodeRedConfig) string {
	var builder strings.Builder
	builder.WriteString("module.exports = {\n")
	builder.WriteString(fmt.Sprintf("  uiPort: %d,\n", cfg.Port))
	if cfg.UIHost != "" {
		builder.WriteString(fmt.Sprintf("  uiHost: %q,\n", cfg.UIHost))
	}
	builder.WriteString(fmt.Sprintf("  httpAdminRoot: %q,\n", cfg.HTTPAdminRoot))
	builder.WriteString(fmt.Sprintf("  httpNodeRoot: %q,\n", cfg.HTTPNodeRoot))
	if cfg.FlowFile != "" {
		builder.WriteString(fmt.Sprintf("  flowFile: %q,\n", cfg.FlowFile))
	}
	if cfg.UserDir != "" {
		builder.WriteString(fmt.Sprintf("  userDir: %q,\n", cfg.UserDir))
	}
	if cfg.NodesDir != "" {
		builder.WriteString(fmt.Sprintf("  nodesDir: %q,\n", cfg.NodesDir))
	}
	builder.WriteString(fmt.Sprintf("  disableEditor: %t,\n", cfg.DisableEditor))
	builder.WriteString(fmt.Sprintf("  projectsEnabled: %t,\n", cfg.ProjectsEnabled))
	if cfg.Lang != "" {
		builder.WriteString(fmt.Sprintf("  lang: %q,\n", cfg.Lang))
	}
	if block := renderEnvBlock(cfg); block != "" {
		builder.WriteString(block)
		builder.WriteString("\n")
	}
	if cfg.AdminAuth != nil && len(cfg.AdminAuth.Users) > 0 {
		user := cfg.AdminAuth.Users[0]
		builder.WriteString("  adminAuth: {\n")
		builder.WriteString(fmt.Sprintf("    type: %q,\n", cfg.AdminAuth.Type))
		builder.WriteString("    users: [{\n")
		builder.WriteString(fmt.Sprintf("      username: %q,\n", user.Username))
		builder.WriteString(fmt.Sprintf("      password: %q,\n", user.Password))
		builder.WriteString(fmt.Sprintf("      permissions: %q,\n", user.Permissions))
		builder.WriteString("    }],\n")
		builder.WriteString("  },\n")
	} else {
		builder.WriteString("  adminAuth: null,\n")
	}
	builder.WriteString("  editorTheme: {\n")
	builder.WriteString(fmt.Sprintf("    projects: { enabled: %t },\n", cfg.ProjectsEnabled))
	builder.WriteString("  },\n")
	builder.WriteString("  logging: {\n")
	builder.WriteString("    console: { level: 'info', metrics: false, audit: false },\n")
	builder.WriteString("  },\n")
	builder.WriteString("}\n")
	return builder.String()
}

// patchSettingsJS applies targeted regex substitutions to preserve unknown blocks
func patchSettingsJS(existingContent string, cfg model.NodeRedConfig) string {
	content := existingContent

	// Scalar keys: use line-level regex (handles comments, varying spacing)
	content = replaceScalarKey(content, "uiPort", fmt.Sprintf("%d", cfg.Port))
	if cfg.UIHost != "" {
		content = replaceScalarKey(content, "uiHost", fmt.Sprintf("%q", cfg.UIHost))
	}
	content = replaceScalarKey(content, "httpAdminRoot", fmt.Sprintf("%q", cfg.HTTPAdminRoot))
	content = replaceScalarKey(content, "httpNodeRoot", fmt.Sprintf("%q", cfg.HTTPNodeRoot))
	if cfg.FlowFile != "" {
		content = replaceScalarKey(content, "flowFile", fmt.Sprintf("%q", cfg.FlowFile))
	}
	if cfg.UserDir != "" {
		content = replaceScalarKey(content, "userDir", fmt.Sprintf("%q", cfg.UserDir))
	}
	if cfg.NodesDir != "" {
		content = replaceScalarKey(content, "nodesDir", fmt.Sprintf("%q", cfg.NodesDir))
	}
	content = replaceScalarKey(content, "disableEditor", fmt.Sprintf("%t", cfg.DisableEditor))
	content = replaceScalarKey(content, "projectsEnabled", fmt.Sprintf("%t", cfg.ProjectsEnabled))
	if cfg.Lang != "" {
		content = replaceScalarKey(content, "lang", fmt.Sprintf("%q", cfg.Lang))
	}

	// Block keys: use brace-aware replacement
	if block := renderEnvBlock(cfg); block != "" {
		content = replaceArrayKey(content, "env", block)
	} else {
		content = removeArrayKey(content, "env")
	}
	content = replaceBlockKey(content, "adminAuth", renderAdminAuthBlock(cfg))
	content = replaceBlockKey(content, "editorTheme", renderEditorThemeBlock(cfg))
	content = replaceBlockKey(content, "logging", renderLoggingBlock())

	return content
}

// replaceScalarKey replaces or appends a scalar key:value pair
func replaceScalarKey(content, key, value string) string {
	// Try to match and replace existing line: key: value, (with optional trailing comma)
	rePattern := fmt.Sprintf(`(?m)^\s*%s\s*:\s*[^,\n]*,?\s*$`, regexp.QuoteMeta(key))
	re := regexp.MustCompile(rePattern)

	if re.MatchString(content) {
		// Replace existing line, preserving indentation
		replacement := fmt.Sprintf("  %s: %s,", key, value)
		return re.ReplaceAllString(content, replacement)
	}

	// Key not found, append before closing brace
	appendPattern := `\n\s*\}\s*$`
	appendRe := regexp.MustCompile(appendPattern)
	if appendRe.MatchString(content) {
		newEntry := fmt.Sprintf("  %s: %s,\n", key, value)
		return appendRe.ReplaceAllString(content, "\n"+newEntry+"}\n")
	}

	return content
}

// replaceBlockKey replaces a block (multi-line key: { ... }) or appends it
func replaceBlockKey(content, key, blockContent string) string {
	// Find and replace the entire block, using brace-depth tracking
	rePattern := fmt.Sprintf(`(?ms)^\s*%s\s*:\s*\{[^{}]*(?:\{[^{}]*\}[^{}]*)*\},?\s*$`, regexp.QuoteMeta(key))
	re := regexp.MustCompile(rePattern)

	if re.MatchString(content) {
		// Replace existing block
		return re.ReplaceAllString(content, blockContent)
	}

	// Block not found, append before closing brace
	appendPattern := `\n\s*\}\s*$`
	appendRe := regexp.MustCompile(appendPattern)
	if appendRe.MatchString(content) {
		newEntry := "\n" + blockContent + "\n"
		return appendRe.ReplaceAllString(content, newEntry+"}\n")
	}

	return content
}

// replaceArrayKey replaces or appends a multi-line array key.
func replaceArrayKey(content, key, arrayContent string) string {
	rePattern := fmt.Sprintf(`(?ms)^\s*%s\s*:\s*\[[^\]]*\],?\s*$`, regexp.QuoteMeta(key))
	re := regexp.MustCompile(rePattern)
	if re.MatchString(content) {
		return re.ReplaceAllString(content, arrayContent)
	}

	appendPattern := `\n\s*\}\s*$`
	appendRe := regexp.MustCompile(appendPattern)
	if appendRe.MatchString(content) {
		newEntry := "\n" + arrayContent + "\n"
		return appendRe.ReplaceAllString(content, newEntry+"}\n")
	}
	return content
}

func removeArrayKey(content, key string) string {
	rePattern := fmt.Sprintf(`(?ms)^\s*%s\s*:\s*\[[^\]]*\],?\s*\n?`, regexp.QuoteMeta(key))
	return regexp.MustCompile(rePattern).ReplaceAllString(content, "")
}

func nodeRedEnvType(typ string) string {
	switch typ {
	case "number":
		return "num"
	case "boolean":
		return "bool"
	default:
		return "str"
	}
}

func renderEnvBlock(cfg model.NodeRedConfig) string {
	vars := make([]model.EnvVar, 0, len(cfg.EnvVars))
	for _, envVar := range cfg.EnvVars {
		typ := envVar.Type
		if typ == "" || typ == "plain" {
			typ = "string"
		}
		if envVar.Key == "" || envVar.Encrypted || typ == "secret" {
			continue
		}
		vars = append(vars, model.EnvVar{Key: envVar.Key, Value: envVar.Value, Type: nodeRedEnvType(typ)})
	}
	if len(vars) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("  env: [\n")
	for i, envVar := range vars {
		builder.WriteString("    {\n")
		builder.WriteString(fmt.Sprintf("      name: %q,\n", envVar.Key))
		builder.WriteString(fmt.Sprintf("      value: %q,\n", envVar.Value))
		builder.WriteString(fmt.Sprintf("      type: %q\n", envVar.Type))
		if i == len(vars)-1 {
			builder.WriteString("    }\n")
		} else {
			builder.WriteString("    },\n")
		}
	}
	builder.WriteString("  ],")
	return builder.String()
}

// renderAdminAuthBlock renders adminAuth as a block string
func renderAdminAuthBlock(cfg model.NodeRedConfig) string {
	if cfg.AdminAuth == nil || len(cfg.AdminAuth.Users) == 0 {
		return "  adminAuth: null,"
	}

	var builder strings.Builder
	builder.WriteString("  adminAuth: {\n")
	user := cfg.AdminAuth.Users[0]
	builder.WriteString(fmt.Sprintf("    type: %q,\n", cfg.AdminAuth.Type))
	builder.WriteString("    users: [{\n")
	builder.WriteString(fmt.Sprintf("      username: %q,\n", user.Username))
	builder.WriteString(fmt.Sprintf("      password: %q,\n", user.Password))
	builder.WriteString(fmt.Sprintf("      permissions: %q,\n", user.Permissions))
	builder.WriteString("    }],\n")
	builder.WriteString("  },")
	return builder.String()
}

// renderEditorThemeBlock renders editorTheme as a block string
func renderEditorThemeBlock(cfg model.NodeRedConfig) string {
	var builder strings.Builder
	builder.WriteString("  editorTheme: {\n")
	builder.WriteString(fmt.Sprintf("    projects: { enabled: %t },\n", cfg.ProjectsEnabled))
	builder.WriteString("  },")
	return builder.String()
}

// renderLoggingBlock renders logging as a block string
func renderLoggingBlock() string {
	return "  logging: {\n    console: { level: 'info', metrics: false, audit: false },\n  },"
}

func (s *ConfigService) parseConfigFromContent(content string) model.NodeRedConfig {
	cfg := model.DefaultNodeRedConfig()
	cfg.Port = parseIntFromJS(content, "uiPort", parseIntFromJS(content, "port", cfg.Port))
	cfg.UIPort = cfg.Port
	cfg.UIHost = parseStringFromJS(content, "uiHost", cfg.UIHost)
	cfg.HTTPAdminRoot = parseStringFromJS(content, "httpAdminRoot", cfg.HTTPAdminRoot)
	cfg.HTTPNodeRoot = parseStringFromJS(content, "httpNodeRoot", cfg.HTTPNodeRoot)
	cfg.FlowFile = parseStringFromJS(content, "flowFile", cfg.FlowFile)
	cfg.UserDir = parseStringFromJS(content, "userDir", cfg.UserDir)
	cfg.NodesDir = parseStringFromJS(content, "nodesDir", cfg.NodesDir)
	cfg.Lang = parseStringFromJS(content, "lang", cfg.Lang)
	cfg.DisableEditor = parseBoolFromJS(content, "disableEditor", cfg.DisableEditor)
	cfg.ProjectsEnabled = parseProjectsEnabledFromJS(content, cfg.ProjectsEnabled)
	cfg.AdminAuth = parseAdminAuthFromJS(content)
	return cfg
}

func parseStringFromJS(content, key, fallback string) string {
	re := regexp.MustCompile(key + `\s*:\s*['"]([^'"]+)['"]`)
	matches := re.FindStringSubmatch(content)
	if len(matches) == 2 {
		return matches[1]
	}
	return fallback
}

// parseProjectsEnabledFromJS looks for projectsEnabled key specifically (not just "enabled")
// This avoids matching "enabled:" inside logging.console.metrics or other nested properties
func parseProjectsEnabledFromJS(content string, fallback bool) bool {
	// First try to find standalone "projectsEnabled: true|false"
	re := regexp.MustCompile(`projectsEnabled\s*:\s*(true|false)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) == 2 {
		return matches[1] == "true"
	}

	// Fallback to editorTheme.projects.enabled pattern (nested)
	// This handles: editorTheme: { projects: { enabled: true } }
	nestedRe := regexp.MustCompile(`editorTheme\s*:\s*\{[^}]*projects\s*:\s*\{[^}]*enabled\s*:\s*(true|false)`)
	if nestedMatches := nestedRe.FindStringSubmatch(content); len(nestedMatches) == 2 {
		return nestedMatches[1] == "true"
	}

	return fallback
}

// parseAdminAuthFromJS extracts the adminAuth block from raw settings.js content
// It handles the multiline adminAuth: { type: "...", users: [{ username, password, permissions }] } structure
func parseAdminAuthFromJS(content string) *model.AdminAuth {
	// Match the entire adminAuth block using a pattern that captures the JSON-like structure
	// Pattern: adminAuth: { type: "...", users: [{ username: "...", password: "...", permissions: "..." }] }
	adminAuthRe := regexp.MustCompile(`(?ms)adminAuth\s*:\s*\{([^{}]*(?:\{[^{}]*\}[^{}]*)*)\}`)
	adminAuthMatches := adminAuthRe.FindStringSubmatch(content)

	if len(adminAuthMatches) == 0 {
		return nil
	}

	blockContent := adminAuthMatches[1]

	// Check if it's the null case
	nullRe := regexp.MustCompile(`(?i)adminAuth\s*:\s*null`)
	if nullRe.MatchString(content) {
		return nil
	}

	// Extract type
	typeRe := regexp.MustCompile(`type\s*:\s*['"]([^'"]+)['"]`)
	typeMatches := typeRe.FindStringSubmatch(blockContent)
	if len(typeMatches) == 0 {
		return nil
	}
	authType := typeMatches[1]

	// Extract users array
	usersRe := regexp.MustCompile(`users\s*:\s*\[\s*(\{[^}]+\}(?:\s*,\s*\{[^}]+\})*)\s*\]`)
	usersMatches := usersRe.FindStringSubmatch(blockContent)
	if len(usersMatches) == 0 {
		return nil
	}

	usersContent := usersMatches[1]

	// Extract individual user objects
	userRe := regexp.MustCompile(`\{([^}]+)\}`)
	userMatches := userRe.FindAllStringSubmatch(usersContent, -1)

	var users []model.AdminAuthUser
	for _, userMatch := range userMatches {
		if len(userMatch) == 0 {
			continue
		}
		userContent := userMatch[1]

		// Extract username, password, permissions
		username := extractQuotedValue(userContent, "username")
		password := extractQuotedValue(userContent, "password")
		permissions := extractQuotedValue(userContent, "permissions")

		if username != "" && password != "" {
			users = append(users, model.AdminAuthUser{
				Username:    username,
				Password:    password,
				Permissions: permissions,
			})
		}
	}

	if len(users) == 0 {
		return nil
	}

	return &model.AdminAuth{
		Type:  authType,
		Users: users,
	}
}

// extractQuotedValue extracts a quoted value from a key:value pair in a string
func extractQuotedValue(content, key string) string {
	re := regexp.MustCompile(key + `\s*:\s*['"]([^'"]+)['"]`)
	matches := re.FindStringSubmatch(content)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func parseIntFromJS(content, key string, fallback int) int {
	re := regexp.MustCompile(key + `\s*:\s*(\d+)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) == 2 {
		value, err := strconv.Atoi(matches[1])
		if err == nil {
			return value
		}
	}
	return fallback
}

func parseBoolFromJS(content, key string, fallback bool) bool {
	re := regexp.MustCompile(key + `\s*:\s*(true|false)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) == 2 {
		return matches[1] == "true"
	}
	return fallback
}
