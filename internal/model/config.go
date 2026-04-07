package model

import (
	"bytes"
	"encoding/json"
)

// Legacy 5-field config (preserved for backward compatibility)
type AppConfig struct {
	HTTPAdminRoot      string `json:"httpAdminRoot"`
	FlowFile           string `json:"flowFile"`
	DiagnosticsEnabled bool   `json:"diagnosticsEnabled"`
	ProjectsEnabled    bool   `json:"projectsEnabled"`
	CredentialSecret   string `json:"credentialSecret"`
}

// ── Server Configuration ──────────────────────────────────────────────
type ServerConfig struct {
	UIPort        int    `json:"uiPort"`        // default: 1880
	UIHost        string `json:"uiHost"`        // default: "0.0.0.0"
	HTTPAdminRoot string `json:"httpAdminRoot"` // default: "/"
	HTTPNodeRoot  string `json:"httpNodeRoot"`  // default: "/"
	HTTPStatic    string `json:"httpStatic"`    // default: ""
	DisableEditor bool   `json:"disableEditor"` // default: false
}

// ── Security Configuration ────────────────────────────────────────────
type AdminAuthUser struct {
	Username    string `json:"username"`
	Password    string `json:"password"`    // bcrypt hash
	Permissions string `json:"permissions"` // "*" | "read"
}

type AdminAuthDefault struct {
	Permissions string `json:"permissions"` // "read" | "*"
}

type AdminAuthConfig struct {
	Type    string            `json:"type"` // "credentials" | "strategy"
	Users   []AdminAuthUser   `json:"users"`
	Default *AdminAuthDefault `json:"default,omitempty"`
}

type HTTPNodeAuthConfig struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

type SecurityConfig struct {
	AdminAuth         *AdminAuthConfig    `json:"adminAuth,omitempty"`
	HTTPNodeAuth      *HTTPNodeAuthConfig `json:"httpNodeAuth,omitempty"`
	CredentialSecret  string              `json:"credentialSecret"`  // default: ""
	SessionExpiryTime int64               `json:"sessionExpiryTime"` // default: 86400
}

// ── Editor Theme Configuration ────────────────────────────────────────
type EditorPageConfig struct {
	Title   string `json:"title"`
	Favicon string `json:"favicon"`
	CSS     string `json:"css"`
}

type EditorHeaderConfig struct {
	Title string `json:"title"`
	Image string `json:"image"`
	URL   string `json:"url"`
}

type EditorDeployButtonConfig struct {
	Type  string `json:"type"` // "simple" | "confirm"
	Label string `json:"label"`
}

type EditorCodeConfig struct {
	Lib     string            `json:"lib"` // "ace" | "monaco"
	Options map[string]string `json:"options"`
}

type EditorProjectsConfig struct {
	Enabled bool `json:"enabled"` // default: false
}

type EditorThemeConfig struct {
	Theme        string                    `json:"theme"`
	Page         *EditorPageConfig         `json:"page,omitempty"`
	Header       *EditorHeaderConfig       `json:"header,omitempty"`
	DeployButton *EditorDeployButtonConfig `json:"deployButton,omitempty"`
	Tours        bool                      `json:"tours"`    // default: true
	UserMenu     bool                      `json:"userMenu"` // default: true
	Projects     EditorProjectsConfig      `json:"projects"`
	CodeEditor   *EditorCodeConfig         `json:"codeEditor,omitempty"`
}

// ── Flows Configuration ───────────────────────────────────────────────
type FlowsConfig struct {
	FlowFile       string `json:"flowFile"`       // default: "flows.json"
	FlowFilePretty bool   `json:"flowFilePretty"` // default: false
	UserDir        string `json:"userDir"`        // default: ""
	NodesDir       string `json:"nodesDir"`       // default: ""
}

// ── Context Storage Configuration ─────────────────────────────────────
type ContextStoreEntry struct {
	Module string         `json:"module"` // "memory" | "localfilesystem"
	Config map[string]any `json:"config,omitempty"`
}

type ContextStorageConfig struct {
	Default string                       `json:"default"` // default: "default"
	Stores  map[string]ContextStoreEntry `json:"stores"`  // default: {"default": {module: "memory"}}
}

// ── Logging Configuration ─────────────────────────────────────────────
type ConsoleLogConfig struct {
	Level   string `json:"level"`   // "fatal"|"error"|"warn"|"info"|"debug"|"trace" default: "info"
	Metrics bool   `json:"metrics"` // default: false
	Audit   bool   `json:"audit"`   // default: false
}

type LoggingConfig struct {
	Console ConsoleLogConfig `json:"console"`
}

// ── Runtime Configuration ─────────────────────────────────────────────
type ExternalModulesPaletteConfig struct {
	AllowInstall bool     `json:"allowInstall"` // default: true
	AllowUpload  bool     `json:"allowUpload"`  // default: false
	AllowList    []string `json:"allowList"`    // default: []
	DenyList     []string `json:"denyList"`     // default: []
}

type ExternalModulesModuleConfig struct {
	AllowInstall bool     `json:"allowInstall"` // default: true
	AllowList    []string `json:"allowList"`    // default: []
	DenyList     []string `json:"denyList"`     // default: []
}

type ExternalModulesConfig struct {
	AutoInstall      bool                         `json:"autoInstall"`      // default: false
	AutoInstallRetry int                          `json:"autoInstallRetry"` // default: 30
	Palette          ExternalModulesPaletteConfig `json:"palette"`
	Modules          ExternalModulesModuleConfig  `json:"modules"`
}

type RuntimeConfig struct {
	FunctionExternalModules    bool                   `json:"functionExternalModules"` // default: false
	FunctionTimeout            int                    `json:"functionTimeout"`         // default: 0
	DebugMaxLength             int                    `json:"debugMaxLength"`          // default: 1000
	ExternalModules            *ExternalModulesConfig `json:"externalModules,omitempty"`
	DiagnosticsEnabled         bool                   `json:"diagnosticsEnabled"`         // default: true
	SafeMode                   bool                   `json:"safeMode"`                   // default: false
	NodeMessageBufferMaxLength int                    `json:"nodeMessageBufferMaxLength"` // default: 0
}

// ── HTTPS Configuration ───────────────────────────────────────────────
type HTTPSConfig struct {
	Enabled  bool   `json:"enabled"` // default: false
	KeyFile  string `json:"keyFile"`
	CertFile string `json:"certFile"`
	CAFile   string `json:"caFile"`
}

// ── Node Reconnect Configuration ──────────────────────────────────────
type NodeReconnectConfig struct {
	MQTTReconnectTime   int `json:"mqttReconnectTime"`   // default: 5000 ms
	SerialReconnectTime int `json:"serialReconnectTime"` // default: 5000 ms
	SocketReconnectTime int `json:"socketReconnectTime"` // default: 10000 ms
	SocketTimeout       int `json:"socketTimeout"`       // default: 120000 ms
}

// ── Palette Configuration ─────────────────────────────────────────────
type PaletteConfig struct {
	Categories []string `json:"categories"`
}

// ── Full App Config ───────────────────────────────────────────────────
type FullAppConfig struct {
	Server         ServerConfig         `json:"server"`
	Security       SecurityConfig       `json:"security"`
	EditorTheme    EditorThemeConfig    `json:"editorTheme"`
	Flows          FlowsConfig          `json:"flows"`
	ContextStorage ContextStorageConfig `json:"contextStorage"`
	Logging        LoggingConfig        `json:"logging"`
	Runtime        RuntimeConfig        `json:"runtime"`
	HTTPS          HTTPSConfig          `json:"https"`
	NodeReconnect  NodeReconnectConfig  `json:"nodeReconnect"`
	Palette        PaletteConfig        `json:"palette"`
}

// ── Validation Result Types ───────────────────────────────────────────
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ConfigDiffEntry struct {
	Field    string `json:"field"`
	OldValue any    `json:"old_value"`
	NewValue any    `json:"new_value"`
}

// Extended validation result with per-field errors and diff
type ExtendedConfigValidationResult struct {
	Valid           bool              `json:"valid"`
	RestartRequired bool              `json:"restartRequired"`
	Errors          []FieldError      `json:"errors"`
	Diff            []ConfigDiffEntry `json:"diff"`
}

// Legacy validation result (for backward compatibility)
type ConfigValidationResult struct {
	Valid           bool              `json:"valid"`
	RestartRequired bool              `json:"restartRequired"`
	Errors          []string          `json:"errors"`
	Diff            []ConfigDiffEntry `json:"diff"`
}

// ── Config Snapshot Types ─────────────────────────────────────────────
type ConfigSnapshot struct {
	ID         string `json:"id"`
	CreatedAt  string `json:"createdAt"`
	Label      string `json:"label"`
	Reason     string `json:"reason"`
	ConfigJSON string `json:"-"` // stored in DB but not returned in list
	SettingsJS string `json:"-"` // stored in DB but not returned in list
}

type ConfigSnapshotList struct {
	Items []ConfigSnapshot `json:"items"`
}

// ── Backward-Compatible Unmarshaling ──────────────────────────────────

// UnmarshalJSON handles both legacy flat format and new nested format
func (c *FullAppConfig) UnmarshalJSON(data []byte) error {
	// First, try to unmarshal as new nested format
	type Alias FullAppConfig
	var alias Alias
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&alias); err == nil {
		// Successfully decoded as nested format
		*c = FullAppConfig(alias)
		// Apply defaults to fill in any zero-value fields
		*c = MergeWithDefaults(*c)
		return nil
	}

	// Fall back to legacy flat format
	var legacy struct {
		HTTPAdminRoot      string `json:"httpAdminRoot"`
		FlowFile           string `json:"flowFile"`
		DiagnosticsEnabled bool   `json:"diagnosticsEnabled"`
		ProjectsEnabled    bool   `json:"projectsEnabled"`
		CredentialSecret   string `json:"credentialSecret"`
	}
	if err := json.Unmarshal(data, &legacy); err != nil {
		return err
	}

	// Create default config and overlay legacy values
	*c = FullAppConfig{
		Server: ServerConfig{
			UIPort:        1880,
			UIHost:        "0.0.0.0",
			HTTPAdminRoot: legacy.HTTPAdminRoot,
			HTTPNodeRoot:  "/",
			HTTPStatic:    "",
			DisableEditor: false,
		},
		Security: SecurityConfig{
			CredentialSecret:  legacy.CredentialSecret,
			SessionExpiryTime: 86400,
		},
		EditorTheme: EditorThemeConfig{
			Theme:    "",
			Tours:    true,
			UserMenu: true,
			Projects: EditorProjectsConfig{
				Enabled: legacy.ProjectsEnabled,
			},
		},
		Flows: FlowsConfig{
			FlowFile:       legacy.FlowFile,
			FlowFilePretty: false,
			UserDir:        "",
			NodesDir:       "",
		},
		ContextStorage: ContextStorageConfig{
			Default: "default",
			Stores: map[string]ContextStoreEntry{
				"default": {
					Module: "memory",
				},
			},
		},
		Logging: LoggingConfig{
			Console: ConsoleLogConfig{
				Level:   "info",
				Metrics: false,
				Audit:   false,
			},
		},
		Runtime: RuntimeConfig{
			FunctionExternalModules:    false,
			FunctionTimeout:            0,
			DebugMaxLength:             1000,
			DiagnosticsEnabled:         legacy.DiagnosticsEnabled,
			SafeMode:                   false,
			NodeMessageBufferMaxLength: 0,
		},
		HTTPS: HTTPSConfig{
			Enabled: false,
		},
		NodeReconnect: NodeReconnectConfig{
			MQTTReconnectTime:   5000,
			SerialReconnectTime: 5000,
			SocketReconnectTime: 10000,
			SocketTimeout:       120000,
		},
		Palette: PaletteConfig{
			Categories: []string{"subflows", "common", "function", "network", "sequence", "parser", "storage"},
		},
	}

	// Apply defaults to fill in any zero-value fields
	*c = MergeWithDefaults(*c)

	return nil
}
