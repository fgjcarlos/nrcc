package service

import (
	"strings"
	"testing"

	"nrcc/internal/model"
)

func TestImportSettingsJS_BasicFields(t *testing.T) {
	content := `module.exports = {
		uiPort: 1880,
		uiHost: "0.0.0.0",
		httpAdminRoot: "/",
	};`

	cfg, unrec, err := ImportSettingsJS(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.UIPort != 1880 {
		t.Errorf("expected UIPort=1880, got %d", cfg.Server.UIPort)
	}
	if cfg.Server.UIHost != "0.0.0.0" {
		t.Errorf("expected UIHost='0.0.0.0', got '%s'", cfg.Server.UIHost)
	}
	if cfg.Server.HTTPAdminRoot != "/" {
		t.Errorf("expected HTTPAdminRoot='/', got '%s'", cfg.Server.HTTPAdminRoot)
	}

	if len(unrec) > 0 {
		t.Errorf("expected no unrecognized keys, got %v", unrec)
	}
}

func TestImportSettingsJS_CredentialSecretFalse(t *testing.T) {
	content := `module.exports = {
		credentialSecret: false,
	};`

	cfg, _, err := ImportSettingsJS(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Special case: credentialSecret: false should be stored as empty string
	if cfg.Security.CredentialSecret != "" {
		t.Errorf("expected CredentialSecret='', got '%s'", cfg.Security.CredentialSecret)
	}
}

func TestImportSettingsJS_HTTPNodeRootFalse(t *testing.T) {
	content := `module.exports = {
		httpNodeRoot: false,
	};`

	cfg, _, err := ImportSettingsJS(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Special case: httpNodeRoot: false should be stored as string "false"
	if cfg.Server.HTTPNodeRoot != "false" {
		t.Errorf("expected HTTPNodeRoot='false', got '%s'", cfg.Server.HTTPNodeRoot)
	}
}

func TestImportSettingsJS_HTTPS(t *testing.T) {
	content := `const fs = require('fs');
	
	module.exports = {
		https: {
			key: fs.readFileSync('/path/to/key.pem'),
			cert: fs.readFileSync('/path/to/cert.pem'),
			ca: fs.readFileSync('/path/to/ca.pem'),
		},
	};`

	cfg, _, err := ImportSettingsJS(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.HTTPS.Enabled {
		t.Error("expected HTTPS.Enabled=true")
	}
	if cfg.HTTPS.KeyFile != "/path/to/key.pem" {
		t.Errorf("expected KeyFile='/path/to/key.pem', got '%s'", cfg.HTTPS.KeyFile)
	}
	if cfg.HTTPS.CertFile != "/path/to/cert.pem" {
		t.Errorf("expected CertFile='/path/to/cert.pem', got '%s'", cfg.HTTPS.CertFile)
	}
	if cfg.HTTPS.CAFile != "/path/to/ca.pem" {
		t.Errorf("expected CAFile='/path/to/ca.pem', got '%s'", cfg.HTTPS.CAFile)
	}
}

func TestImportSettingsJS_AdminAuth(t *testing.T) {
	content := `module.exports = {
		adminAuth: {
			type: "credentials",
			users: [
				{
					username: "alice",
					password: "$2a$08$...",
					permissions: "*"
				},
				{
					username: "bob",
					password: "$2a$08$...",
					permissions: "read"
				}
			],
			default: {
				permissions: "read"
			}
		},
	};`

	cfg, _, err := ImportSettingsJS(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Security.AdminAuth == nil {
		t.Fatal("expected AdminAuth to be non-nil")
	}

	if cfg.Security.AdminAuth.Type != "credentials" {
		t.Errorf("expected Type='credentials', got '%s'", cfg.Security.AdminAuth.Type)
	}

	if len(cfg.Security.AdminAuth.Users) != 2 {
		t.Errorf("expected 2 users, got %d", len(cfg.Security.AdminAuth.Users))
	}

	if cfg.Security.AdminAuth.Users[0].Username != "alice" {
		t.Errorf("expected first user username='alice', got '%s'", cfg.Security.AdminAuth.Users[0].Username)
	}

	if cfg.Security.AdminAuth.Users[1].Permissions != "read" {
		t.Errorf("expected second user permissions='read', got '%s'", cfg.Security.AdminAuth.Users[1].Permissions)
	}

	if cfg.Security.AdminAuth.Default == nil || cfg.Security.AdminAuth.Default.Permissions != "read" {
		t.Error("expected AdminAuth.Default.Permissions='read'")
	}
}

func TestImportSettingsJS_PaletteCategories(t *testing.T) {
	content := `module.exports = {
		paletteCategories: ["subflows", "custom", "function", "network"],
	};`

	cfg, _, err := ImportSettingsJS(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Palette.Categories) != 4 {
		t.Errorf("expected 4 categories, got %d", len(cfg.Palette.Categories))
	}

	if cfg.Palette.Categories[0] != "subflows" {
		t.Errorf("expected first category='subflows', got '%s'", cfg.Palette.Categories[0])
	}

	if cfg.Palette.Categories[1] != "custom" {
		t.Errorf("expected second category='custom', got '%s'", cfg.Palette.Categories[1])
	}
}

func TestImportSettingsJS_Logging(t *testing.T) {
	content := `module.exports = {
		logging: {
			console: {
				level: "debug",
				metrics: true,
				audit: false
			}
		},
	};`

	cfg, _, err := ImportSettingsJS(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Logging.Console.Level != "debug" {
		t.Errorf("expected Level='debug', got '%s'", cfg.Logging.Console.Level)
	}

	if !cfg.Logging.Console.Metrics {
		t.Error("expected Metrics=true")
	}

	if cfg.Logging.Console.Audit {
		t.Error("expected Audit=false")
	}
}

func TestImportSettingsJS_Unrecognized(t *testing.T) {
	content := `module.exports = {
		unknownField: "value",
		uiPort: 1880,
		anotherUnknown: true,
	};`

	cfg, unrec, err := ImportSettingsJS(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.UIPort != 1880 {
		t.Errorf("expected UIPort=1880, got %d", cfg.Server.UIPort)
	}

	if len(unrec) != 2 {
		t.Errorf("expected 2 unrecognized keys, got %d: %v", len(unrec), unrec)
	}

	foundUnknownField := false
	foundAnotherUnknown := false
	for _, key := range unrec {
		if key == "unknownField" {
			foundUnknownField = true
		}
		if key == "anotherUnknown" {
			foundAnotherUnknown = true
		}
	}

	if !foundUnknownField {
		t.Error("expected 'unknownField' in unrecognized")
	}
	if !foundAnotherUnknown {
		t.Error("expected 'anotherUnknown' in unrecognized")
	}
}

func TestImportSettingsJS_InvalidContent(t *testing.T) {
	content := `some random content without module.exports`

	_, _, err := ImportSettingsJS(content)
	if err == nil {
		t.Fatal("expected error for invalid content")
	}

	if !strings.Contains(err.Error(), "module.exports") {
		t.Errorf("expected error about module.exports, got: %v", err)
	}
}

func TestImportSettingsJS_WithComments(t *testing.T) {
	content := `// This is a comment
	/* Block comment */
	module.exports = {
		// Server config
		uiPort: 1880, // The port
		uiHost: "127.0.0.1", /* Host */
		// End of server config
	};`

	cfg, _, err := ImportSettingsJS(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.UIPort != 1880 {
		t.Errorf("expected UIPort=1880, got %d", cfg.Server.UIPort)
	}

	if cfg.Server.UIHost != "127.0.0.1" {
		t.Errorf("expected UIHost='127.0.0.1', got '%s'", cfg.Server.UIHost)
	}
}

func TestImportSettingsJS_FullConfig(t *testing.T) {
	// Render a default config to a settings.js string
	defaultCfg := model.DefaultFullAppConfig()
	rendered := RenderSettingsJS(defaultCfg)

	// Import it back
	imported, unrec, err := ImportSettingsJS(rendered)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify key fields match
	if imported.Server.UIPort != defaultCfg.Server.UIPort {
		t.Errorf("UIPort mismatch: expected %d, got %d", defaultCfg.Server.UIPort, imported.Server.UIPort)
	}

	if imported.Server.UIHost != defaultCfg.Server.UIHost {
		t.Errorf("UIHost mismatch: expected '%s', got '%s'", defaultCfg.Server.UIHost, imported.Server.UIHost)
	}

	if imported.Flows.FlowFile != defaultCfg.Flows.FlowFile {
		t.Errorf("FlowFile mismatch: expected '%s', got '%s'", defaultCfg.Flows.FlowFile, imported.Flows.FlowFile)
	}

	if imported.Logging.Console.Level != defaultCfg.Logging.Console.Level {
		t.Errorf("Log level mismatch: expected '%s', got '%s'", defaultCfg.Logging.Console.Level, imported.Logging.Console.Level)
	}

	if len(unrec) > 0 {
		t.Logf("unrecognized keys during roundtrip: %v", unrec)
	}
}

func TestImportSettingsJS_ContextStorage(t *testing.T) {
	content := `module.exports = {
		contextStorage: {
			default: "memory",
			stores: {
				memory: {
					module: "memory"
				},
				file: {
					module: "localfilesystem",
					config: {
						dir: "/data/context"
					}
				}
			}
		},
	};`

	cfg, _, err := ImportSettingsJS(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ContextStorage.Default != "memory" {
		t.Errorf("expected Default='memory', got '%s'", cfg.ContextStorage.Default)
	}

	if len(cfg.ContextStorage.Stores) < 1 {
		t.Errorf("expected at least 1 store, got %d", len(cfg.ContextStorage.Stores))
	}

	if memStore, ok := cfg.ContextStorage.Stores["memory"]; ok {
		if memStore.Module != "memory" {
			t.Errorf("expected memory store module='memory', got '%s'", memStore.Module)
		}
	} else {
		t.Error("expected 'memory' store to exist")
	}
}

func TestImportSettingsJS_EdgeCase_EmptyFlows(t *testing.T) {
	content := `module.exports = {
		flowFile: "flows.json",
		flowFilePretty: false,
		userDir: "",
		nodesDir: "",
	};`

	cfg, _, err := ImportSettingsJS(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Flows.FlowFile != "flows.json" {
		t.Errorf("expected FlowFile='flows.json', got '%s'", cfg.Flows.FlowFile)
	}

	if cfg.Flows.UserDir != "" {
		t.Errorf("expected UserDir='', got '%s'", cfg.Flows.UserDir)
	}
}

func TestImportSettingsJS_HTTPNodeAuth(t *testing.T) {
	content := `module.exports = {
		httpNodeAuth: {
			user: "apiuser",
			pass: "apipass"
		},
	};`

	cfg, _, err := ImportSettingsJS(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Security.HTTPNodeAuth == nil {
		t.Fatal("expected HTTPNodeAuth to be non-nil")
	}

	if cfg.Security.HTTPNodeAuth.User != "apiuser" {
		t.Errorf("expected User='apiuser', got '%s'", cfg.Security.HTTPNodeAuth.User)
	}

	if cfg.Security.HTTPNodeAuth.Pass != "apipass" {
		t.Errorf("expected Pass='apipass', got '%s'", cfg.Security.HTTPNodeAuth.Pass)
	}
}

func TestImportSettingsJS_RuntimeSettings(t *testing.T) {
	content := `module.exports = {
		functionExternalModules: true,
		functionTimeout: 300,
		debugMaxLength: 2000,
		diagnosticsEnabled: false,
		safeMode: true,
		nodeMessageBufferMaxLength: 16,
	};`

	cfg, _, err := ImportSettingsJS(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.Runtime.FunctionExternalModules {
		t.Error("expected FunctionExternalModules=true")
	}

	if cfg.Runtime.FunctionTimeout != 300 {
		t.Errorf("expected FunctionTimeout=300, got %d", cfg.Runtime.FunctionTimeout)
	}

	if cfg.Runtime.DebugMaxLength != 2000 {
		t.Errorf("expected DebugMaxLength=2000, got %d", cfg.Runtime.DebugMaxLength)
	}

	if cfg.Runtime.DiagnosticsEnabled {
		t.Error("expected DiagnosticsEnabled=false")
	}

	if !cfg.Runtime.SafeMode {
		t.Error("expected SafeMode=true")
	}

	if cfg.Runtime.NodeMessageBufferMaxLength != 16 {
		t.Errorf("expected NodeMessageBufferMaxLength=16, got %d", cfg.Runtime.NodeMessageBufferMaxLength)
	}
}

func TestImportSettingsJS_EditorTheme(t *testing.T) {
	content := `module.exports = {
		editorTheme: {
			theme: "dark",
			tours: false,
			userMenu: true,
			projects: {
				enabled: true
			},
			page: {
				title: "My Node-RED",
				favicon: "/icon.png"
			},
			header: {
				title: "Node-RED Control Center"
			},
			deployButton: {
				type: "confirm",
				label: "Deploy"
			}
		},
	};`

	cfg, _, err := ImportSettingsJS(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.EditorTheme.Theme != "dark" {
		t.Errorf("expected Theme='dark', got '%s'", cfg.EditorTheme.Theme)
	}

	if cfg.EditorTheme.Tours {
		t.Error("expected Tours=false")
	}

	if !cfg.EditorTheme.UserMenu {
		t.Error("expected UserMenu=true")
	}

	if !cfg.EditorTheme.Projects.Enabled {
		t.Error("expected Projects.Enabled=true")
	}

	if cfg.EditorTheme.Page == nil || cfg.EditorTheme.Page.Title != "My Node-RED" {
		t.Error("expected Page.Title='My Node-RED'")
	}

	if cfg.EditorTheme.Header == nil || cfg.EditorTheme.Header.Title != "Node-RED Control Center" {
		t.Error("expected Header.Title='Node-RED Control Center'")
	}

	if cfg.EditorTheme.DeployButton == nil || cfg.EditorTheme.DeployButton.Type != "confirm" {
		t.Error("expected DeployButton.Type='confirm'")
	}
}

func TestImportSettingsJS_NodeReconnect(t *testing.T) {
	content := `module.exports = {
		mqttReconnectTime: 10000,
		serialReconnectTime: 15000,
		socketReconnectTime: 20000,
		socketTimeout: 60000,
	};`

	cfg, _, err := ImportSettingsJS(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.NodeReconnect.MQTTReconnectTime != 10000 {
		t.Errorf("expected MQTTReconnectTime=10000, got %d", cfg.NodeReconnect.MQTTReconnectTime)
	}

	if cfg.NodeReconnect.SerialReconnectTime != 15000 {
		t.Errorf("expected SerialReconnectTime=15000, got %d", cfg.NodeReconnect.SerialReconnectTime)
	}

	if cfg.NodeReconnect.SocketReconnectTime != 20000 {
		t.Errorf("expected SocketReconnectTime=20000, got %d", cfg.NodeReconnect.SocketReconnectTime)
	}

	if cfg.NodeReconnect.SocketTimeout != 60000 {
		t.Errorf("expected SocketTimeout=60000, got %d", cfg.NodeReconnect.SocketTimeout)
	}
}
