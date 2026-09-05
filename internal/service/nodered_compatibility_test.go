package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestNodeRED5CatalogMatchesFixture(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "nodered-5.0.6-catalog.json"))
	if err != nil {
		t.Fatalf("read Node-RED 5.0.6 catalog fixture: %v", err)
	}

	var expected []struct {
		Key   string `json:"key"`
		Shape string `json:"shape"`
	}
	if err := json.Unmarshal(fixture, &expected); err != nil {
		t.Fatalf("parse Node-RED 5.0.6 catalog fixture: %v", err)
	}

	catalog := NodeRED5Catalog()
	if len(catalog) != len(expected) {
		t.Fatalf("catalog has %d entries, fixture has %d", len(catalog), len(expected))
	}
	for index, want := range expected {
		got := catalog[index]
		if got.Key != want.Key || got.Shape != want.Shape {
			t.Errorf("catalog[%d] = {%q, %q}, want {%q, %q}", index, got.Key, got.Shape, want.Key, want.Shape)
		}
	}
}

func TestCompatibilityPolicy(t *testing.T) {
	settings := model.SettingsDocument{Source: "detected", Writable: true}
	tests := []struct {
		name    string
		version string
		adapter string
		editable bool
	}{
		{name: "Node-RED 4 migration mode", version: "4.0.9", adapter: "nodered-4-read-only", editable: false},
		{name: "Node-RED 5 editable", version: "v5.0.6", adapter: "nodered-5", editable: true},
		{name: "Node-RED 6 unsupported", version: "6.0.0", adapter: "unsupported", editable: false},
		{name: "unknown version", version: "", adapter: "unsupported", editable: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveConfigurationCapabilities(tt.version, settings)
			if got.Adapter != tt.adapter || got.Editable != tt.editable {
				t.Fatalf("ResolveConfigurationCapabilities(%q) = adapter=%q editable=%t, want adapter=%q editable=%t", tt.version, got.Adapter, got.Editable, tt.adapter, tt.editable)
			}
		})
	}
}
