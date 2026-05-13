package model

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalConfigFromFrontend(t *testing.T) {
	// This is the payload that the frontend sends from Configuration.tsx
	frontendPayload := `{
		"uiPort": 1880,
		"uiHost": "0.0.0.0",
		"httpAdminRoot": "/",
		"httpNodeRoot": "/",
		"disableEditor": false,
		"flowFile": "flows.json",
		"userDir": "",
		"nodesDir": "",
		"projectsEnabled": false,
		"logging": {
			"console": {
				"level": "info",
				"metrics": false
			},
			"internal": {
				"level": "info",
				"metrics": false
			}
		},
		"editorTheme": {
			"page": {
				"title": "Node-RED"
			},
			"header": {
				"title": "Node-RED"
			}
		},
		"lang": "en-US",
		"authEnabled": false,
		"editorPageTitle": "Node-RED"
	}`

	var cfg NodeRedConfig
	err := json.Unmarshal([]byte(frontendPayload), &cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal frontend payload: %v", err)
	}

	// Verify key fields were unmarshaled
	if cfg.UIPort != 1880 {
		t.Errorf("Expected UIPort 1880, got %d", cfg.UIPort)
	}
	if cfg.UIHost != "0.0.0.0" {
		t.Errorf("Expected UIHost '0.0.0.0', got '%s'", cfg.UIHost)
	}
	if cfg.HTTPAdminRoot != "/" {
		t.Errorf("Expected HTTPAdminRoot '/', got '%s'", cfg.HTTPAdminRoot)
	}
	if cfg.HTTPNodeRoot != "/" {
		t.Errorf("Expected HTTPNodeRoot '/', got '%s'", cfg.HTTPNodeRoot)
	}
	if cfg.FlowFile != "flows.json" {
		t.Errorf("Expected FlowFile 'flows.json', got '%s'", cfg.FlowFile)
	}
}

func TestCustomUnmarshalPortFallback(t *testing.T) {
	// Test that UnmarshalJSON copies uiPort to port if port is not set
	payload := `{"uiPort": 3000, "httpAdminRoot": "/", "httpNodeRoot": "/"}`

	var cfg NodeRedConfig
	err := json.Unmarshal([]byte(payload), &cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// After UnmarshalJSON, Port should be set from UIPort
	if cfg.Port != 3000 {
		t.Errorf("Expected Port to be set to 3000 (from UIPort), got %d", cfg.Port)
	}
}

func TestBackendPayloadStillWorks(t *testing.T) {
	// Verify backward compatibility with backend-only format
	backendPayload := `{
		"port": 1880,
		"httpAdminRoot": "/",
		"httpNodeRoot": "/",
		"flowFile": "flows.json",
		"userDir": "/data"
	}`

	var cfg NodeRedConfig
	err := json.Unmarshal([]byte(backendPayload), &cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal backend payload: %v", err)
	}

	if cfg.Port != 1880 {
		t.Errorf("Expected Port 1880, got %d", cfg.Port)
	}
}
