package model

// DefaultFullAppConfig returns a FullAppConfig with all default values populated.
func DefaultFullAppConfig() FullAppConfig {
	return FullAppConfig{
		Server: ServerConfig{
			UIPort:        1880,
			UIHost:        "0.0.0.0",
			HTTPAdminRoot: "/",
			HTTPNodeRoot:  "/",
			HTTPStatic:    "",
			DisableEditor: false,
		},
		Security: SecurityConfig{
			AdminAuth:         nil,
			HTTPNodeAuth:      nil,
			CredentialSecret:  "",
			SessionExpiryTime: 86400,
		},
		EditorTheme: EditorThemeConfig{
			Theme:        "",
			Page:         nil,
			Header:       nil,
			DeployButton: nil,
			Tours:        true,
			UserMenu:     true,
			Projects:     EditorProjectsConfig{Enabled: false},
			CodeEditor:   nil,
		},
		Flows: FlowsConfig{
			FlowFile:       "flows.json",
			FlowFilePretty: false,
			UserDir:        "",
			NodesDir:       "",
		},
		ContextStorage: ContextStorageConfig{
			Default: "default",
			Stores: map[string]ContextStoreEntry{
				"default": {Module: "memory"},
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
			ExternalModules:            nil,
			DiagnosticsEnabled:         true,
			SafeMode:                   false,
			NodeMessageBufferMaxLength: 0,
		},
		HTTPS: HTTPSConfig{
			Enabled:  false,
			KeyFile:  "",
			CertFile: "",
			CAFile:   "",
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
}

// MergeWithDefaults takes a partially-set config and returns a version with defaults
// applied for zero values. It fills in missing values intelligently:
// - For struct fields: if zero-valued, use the default
// - For slices: if nil or empty and the default is non-empty, use the default
// - For maps: if nil, use the default
// - For int fields: if zero and the default is non-zero, use the default
// - For string fields: use provided value (empty string is valid)
// - For bool fields: use provided value (zero value false is valid)
func MergeWithDefaults(cfg FullAppConfig) FullAppConfig {
	defaults := DefaultFullAppConfig()

	// Server section
	if cfg.Server.UIPort == 0 {
		cfg.Server.UIPort = defaults.Server.UIPort
	}
	if cfg.Server.UIHost == "" {
		cfg.Server.UIHost = defaults.Server.UIHost
	}
	if cfg.Server.HTTPAdminRoot == "" {
		cfg.Server.HTTPAdminRoot = defaults.Server.HTTPAdminRoot
	}
	if cfg.Server.HTTPNodeRoot == "" {
		cfg.Server.HTTPNodeRoot = defaults.Server.HTTPNodeRoot
	}
	// HTTPStatic: empty string is valid
	// DisableEditor: false is valid

	// Security section
	// AdminAuth: nil is valid
	// HTTPNodeAuth: nil is valid
	// CredentialSecret: empty string is valid
	if cfg.Security.SessionExpiryTime == 0 {
		cfg.Security.SessionExpiryTime = defaults.Security.SessionExpiryTime
	}

	// EditorTheme section
	// Theme: empty string is valid
	// Page: nil is valid
	// Header: nil is valid
	// DeployButton: nil is valid
	// Tours: false is valid (we don't override bool fields)
	// UserMenu: false is valid (we don't override bool fields)
	// For projects, the struct is not nil so don't override individual fields
	// CodeEditor: nil is valid

	// Flows section
	if cfg.Flows.FlowFile == "" {
		cfg.Flows.FlowFile = defaults.Flows.FlowFile
	}
	// FlowFilePretty: false is valid
	// UserDir: empty string is valid
	// NodesDir: empty string is valid

	// ContextStorage section
	if cfg.ContextStorage.Default == "" {
		cfg.ContextStorage.Default = defaults.ContextStorage.Default
	}
	if cfg.ContextStorage.Stores == nil || len(cfg.ContextStorage.Stores) == 0 {
		cfg.ContextStorage.Stores = defaults.ContextStorage.Stores
	}

	// Logging section
	if cfg.Logging.Console.Level == "" {
		cfg.Logging.Console.Level = defaults.Logging.Console.Level
	}
	// Metrics and Audit: false is valid

	// Runtime section
	if cfg.Runtime.FunctionTimeout == 0 {
		// 0 is a valid value (no timeout), don't override
	}
	if cfg.Runtime.DebugMaxLength == 0 {
		cfg.Runtime.DebugMaxLength = defaults.Runtime.DebugMaxLength
	}
	// ExternalModules: nil is valid
	// DiagnosticsEnabled: false is valid (but defaults say true)
	// SafeMode: false is valid
	// NodeMessageBufferMaxLength: 0 is valid

	// HTTPS section
	// Enabled: false is valid
	// KeyFile, CertFile, CAFile: empty strings are valid

	// NodeReconnect section
	if cfg.NodeReconnect.MQTTReconnectTime == 0 {
		cfg.NodeReconnect.MQTTReconnectTime = defaults.NodeReconnect.MQTTReconnectTime
	}
	if cfg.NodeReconnect.SerialReconnectTime == 0 {
		cfg.NodeReconnect.SerialReconnectTime = defaults.NodeReconnect.SerialReconnectTime
	}
	if cfg.NodeReconnect.SocketReconnectTime == 0 {
		cfg.NodeReconnect.SocketReconnectTime = defaults.NodeReconnect.SocketReconnectTime
	}
	if cfg.NodeReconnect.SocketTimeout == 0 {
		cfg.NodeReconnect.SocketTimeout = defaults.NodeReconnect.SocketTimeout
	}

	// Palette section
	if cfg.Palette.Categories == nil || len(cfg.Palette.Categories) == 0 {
		cfg.Palette.Categories = defaults.Palette.Categories
	}

	return cfg
}
