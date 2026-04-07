package service

import (
	"testing"

	"nrcc/internal/model"
)

func TestDefaultFullAppConfig(t *testing.T) {
	t.Parallel()

	cfg := model.DefaultFullAppConfig()

	// Verify all defaults are set correctly
	if cfg.Server.UIPort != 1880 {
		t.Errorf("UIPort = %d, want 1880", cfg.Server.UIPort)
	}
	if cfg.Server.UIHost != "0.0.0.0" {
		t.Errorf("UIHost = %s, want 0.0.0.0", cfg.Server.UIHost)
	}
	if cfg.Server.HTTPAdminRoot != "/" {
		t.Errorf("HTTPAdminRoot = %s, want /", cfg.Server.HTTPAdminRoot)
	}
	if cfg.Server.HTTPNodeRoot != "/" {
		t.Errorf("HTTPNodeRoot = %s, want /", cfg.Server.HTTPNodeRoot)
	}
	if cfg.Server.HTTPStatic != "" {
		t.Errorf("HTTPStatic = %s, want empty", cfg.Server.HTTPStatic)
	}
	if cfg.Server.DisableEditor != false {
		t.Errorf("DisableEditor = %v, want false", cfg.Server.DisableEditor)
	}

	if cfg.Security.CredentialSecret != "" {
		t.Errorf("CredentialSecret = %s, want empty", cfg.Security.CredentialSecret)
	}
	if cfg.Security.SessionExpiryTime != 86400 {
		t.Errorf("SessionExpiryTime = %d, want 86400", cfg.Security.SessionExpiryTime)
	}

	if cfg.Flows.FlowFile != "flows.json" {
		t.Errorf("FlowFile = %s, want flows.json", cfg.Flows.FlowFile)
	}

	if cfg.ContextStorage.Default != "default" {
		t.Errorf("ContextStorage.Default = %s, want default", cfg.ContextStorage.Default)
	}
	if _, exists := cfg.ContextStorage.Stores["default"]; !exists {
		t.Error("ContextStorage.Stores missing default entry")
	}

	if cfg.Logging.Console.Level != "info" {
		t.Errorf("Console.Level = %s, want info", cfg.Logging.Console.Level)
	}

	if cfg.HTTPS.Enabled != false {
		t.Errorf("HTTPS.Enabled = %v, want false", cfg.HTTPS.Enabled)
	}

	if cfg.NodeReconnect.MQTTReconnectTime != 5000 {
		t.Errorf("MQTTReconnectTime = %d, want 5000", cfg.NodeReconnect.MQTTReconnectTime)
	}

	if len(cfg.Palette.Categories) == 0 {
		t.Error("Palette.Categories is empty, want non-empty")
	}
}

func TestMergeWithDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  model.FullAppConfig
		want model.FullAppConfig
	}{
		{
			name: "zero_uiport_gets_default",
			cfg: model.FullAppConfig{
				Server: model.ServerConfig{UIPort: 0},
			},
			want: model.FullAppConfig{
				Server: model.ServerConfig{UIPort: 1880},
			},
		},
		{
			name: "zero_sessionexpiryime_gets_default",
			cfg: model.FullAppConfig{
				Security: model.SecurityConfig{SessionExpiryTime: 0},
			},
			want: model.FullAppConfig{
				Security: model.SecurityConfig{SessionExpiryTime: 86400},
			},
		},
		{
			name: "empty_flowfile_gets_default",
			cfg: model.FullAppConfig{
				Flows: model.FlowsConfig{FlowFile: ""},
			},
			want: model.FullAppConfig{
				Flows: model.FlowsConfig{FlowFile: "flows.json"},
			},
		},
		{
			name: "nil_stores_gets_default",
			cfg: model.FullAppConfig{
				ContextStorage: model.ContextStorageConfig{
					Stores: nil,
				},
			},
			want: model.FullAppConfig{
				ContextStorage: model.ContextStorageConfig{
					Stores: map[string]model.ContextStoreEntry{
						"default": {Module: "memory"},
					},
				},
			},
		},
		{
			name: "empty_categories_gets_default",
			cfg: model.FullAppConfig{
				Palette: model.PaletteConfig{Categories: nil},
			},
			want: model.FullAppConfig{
				Palette: model.PaletteConfig{
					Categories: []string{"subflows", "common", "function", "network", "sequence", "parser", "storage"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := model.MergeWithDefaults(tt.cfg)
			if tt.name == "zero_uiport_gets_default" && got.Server.UIPort != 1880 {
				t.Errorf("UIPort = %d, want 1880", got.Server.UIPort)
			}
			if tt.name == "zero_sessionexpiryime_gets_default" && got.Security.SessionExpiryTime != 86400 {
				t.Errorf("SessionExpiryTime = %d, want 86400", got.Security.SessionExpiryTime)
			}
			if tt.name == "empty_flowfile_gets_default" && got.Flows.FlowFile != "flows.json" {
				t.Errorf("FlowFile = %s, want flows.json", got.Flows.FlowFile)
			}
			if tt.name == "nil_stores_gets_default" && got.ContextStorage.Stores == nil {
				t.Error("Stores should be populated with default")
			}
			if tt.name == "empty_categories_gets_default" && len(got.Palette.Categories) == 0 {
				t.Error("Categories should be populated with default")
			}
		})
	}
}

func TestValidateFullAppConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cfg            model.FullAppConfig
		wantErrorCount int
		wantFieldName  string // one of the expected error fields (if checking for specific error)
	}{
		{
			name:           "valid_default_config",
			cfg:            model.DefaultFullAppConfig(),
			wantErrorCount: 0,
		},
		{
			name: "uiport_zero",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.Server.UIPort = 0
				return c
			}(),
			wantErrorCount: 1,
			wantFieldName:  "server.uiPort",
		},
		{
			name: "uiport_too_high",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.Server.UIPort = 65536
				return c
			}(),
			wantErrorCount: 1,
			wantFieldName:  "server.uiPort",
		},
		{
			name: "credentialsecret_too_short",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.Security.CredentialSecret = "short"
				return c
			}(),
			wantErrorCount: 1,
			wantFieldName:  "security.credentialSecret",
		},
		{
			name: "credentialsecret_empty_is_valid",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.Security.CredentialSecret = ""
				return c
			}(),
			wantErrorCount: 0,
		},
		{
			name: "adminauth_zero_users",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.Security.AdminAuth = &model.AdminAuthConfig{
					Type:  "credentials",
					Users: []model.AdminAuthUser{},
				}
				return c
			}(),
			wantErrorCount: 1,
			wantFieldName:  "security.adminAuth.users",
		},
		{
			name: "adminauth_user_empty_username",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.Security.AdminAuth = &model.AdminAuthConfig{
					Type: "credentials",
					Users: []model.AdminAuthUser{
						{Username: "", Permissions: "*"},
					},
				}
				return c
			}(),
			wantErrorCount: 1,
			wantFieldName:  "security.adminAuth.users[0].username",
		},
		{
			name: "flowfile_missing_json",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.Flows.FlowFile = "flows"
				return c
			}(),
			wantErrorCount: 1,
			wantFieldName:  "flows.flowFile",
		},
		{
			name: "flowfile_with_path_separator",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.Flows.FlowFile = "path/to/flows.json"
				return c
			}(),
			wantErrorCount: 1,
			wantFieldName:  "flows.flowFile",
		},
		{
			name: "contextstorage_default_not_in_stores",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.ContextStorage.Default = "nonexistent"
				return c
			}(),
			wantErrorCount: 1,
			wantFieldName:  "contextStorage.default",
		},
		{
			name: "logging_console_level_invalid",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.Logging.Console.Level = "verbose"
				return c
			}(),
			wantErrorCount: 1,
			wantFieldName:  "logging.console.level",
		},
		{
			name: "https_enabled_no_keyfile",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.HTTPS.Enabled = true
				c.HTTPS.KeyFile = ""
				c.HTTPS.CertFile = "cert.pem"
				return c
			}(),
			wantErrorCount: 1,
			wantFieldName:  "https.keyFile",
		},
		{
			name: "https_enabled_no_certfile",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.HTTPS.Enabled = true
				c.HTTPS.KeyFile = "key.pem"
				c.HTTPS.CertFile = ""
				return c
			}(),
			wantErrorCount: 1,
			wantFieldName:  "https.certFile",
		},
		{
			name: "https_disabled_no_files_valid",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.HTTPS.Enabled = false
				c.HTTPS.KeyFile = ""
				c.HTTPS.CertFile = ""
				return c
			}(),
			wantErrorCount: 0,
		},
		{
			name: "codeeditor_invalid_lib",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.EditorTheme.CodeEditor = &model.EditorCodeConfig{Lib: "vim"}
				return c
			}(),
			wantErrorCount: 1,
			wantFieldName:  "editorTheme.codeEditor.lib",
		},
		{
			name: "palette_categories_duplicate",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.Palette.Categories = []string{"common", "common"}
				return c
			}(),
			wantErrorCount: 1,
			wantFieldName:  "palette.categories[1]",
		},
		{
			name: "palette_categories_empty_string",
			cfg: func() model.FullAppConfig {
				c := model.DefaultFullAppConfig()
				c.Palette.Categories = []string{"common", ""}
				return c
			}(),
			wantErrorCount: 1,
			wantFieldName:  "palette.categories[1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateFullAppConfig(tt.cfg)
			if len(errors) != tt.wantErrorCount {
				t.Errorf("got %d errors, want %d", len(errors), tt.wantErrorCount)
				for _, err := range errors {
					t.Logf("  - %s: %s", err.Field, err.Message)
				}
			}

			if tt.wantFieldName != "" {
				found := false
				for _, err := range errors {
					if err.Field == tt.wantFieldName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error for field %s, but not found", tt.wantFieldName)
				}
			}
		})
	}
}

func TestComputeConfigDiff(t *testing.T) {
	t.Parallel()

	current := model.DefaultFullAppConfig()
	proposed := model.DefaultFullAppConfig()
	proposed.Server.UIPort = 3000
	proposed.Flows.FlowFile = "custom-flows.json"

	diff := ComputeConfigDiff(current, proposed)

	if len(diff) == 0 {
		t.Fatal("expected differences, got none")
	}

	foundUIPort := false
	foundFlowFile := false

	for _, entry := range diff {
		if entry.Field == "server.uiPort" {
			foundUIPort = true
			if entry.OldValue != "1880" {
				t.Errorf("UIPort old value = %v, want 1880", entry.OldValue)
			}
			if entry.NewValue != "3000" {
				t.Errorf("UIPort new value = %v, want 3000", entry.NewValue)
			}
		}
		if entry.Field == "flows.flowFile" {
			foundFlowFile = true
			if entry.OldValue != "flows.json" {
				t.Errorf("FlowFile old value = %v, want flows.json", entry.OldValue)
			}
			if entry.NewValue != "custom-flows.json" {
				t.Errorf("FlowFile new value = %v, want custom-flows.json", entry.NewValue)
			}
		}
	}

	if !foundUIPort {
		t.Error("expected UIPort diff, not found")
	}
	if !foundFlowFile {
		t.Error("expected FlowFile diff, not found")
	}
}
