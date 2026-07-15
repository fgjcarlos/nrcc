package service

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveNodeRedRuntime_Defaults covers the contract when no env
// overrides are set: port 1880, userDir = dataDir, settings under userDir.
func TestResolveNodeRedRuntime_Defaults(t *testing.T) {
	dataDir := t.TempDir()
	rt, err := resolveNodeRedRuntime(map[string]string{}, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Port != "1880" {
		t.Errorf("Port = %q, want %q", rt.Port, "1880")
	}
	if rt.UserDir == "" {
		t.Error("UserDir must not be empty")
	}
	if !filepath.IsAbs(rt.UserDir) {
		t.Errorf("UserDir must be absolute, got %q", rt.UserDir)
	}
	if rt.SettingsPath != filepath.Join(rt.UserDir, "settings.js") {
		t.Errorf("SettingsPath = %q, want %q", rt.SettingsPath, filepath.Join(rt.UserDir, "settings.js"))
	}
}

// TestResolveNodeRedRuntime_EnvOverrides verifies that NODE_RED_PORT,
// NODE_RED_USER_DIR, and NODE_RED_SETTINGS are honored.
func TestResolveNodeRedRuntime_EnvOverrides(t *testing.T) {
	dataDir := t.TempDir()
	overrideDir := t.TempDir()
	settingsDir := t.TempDir()

	rt, err := resolveNodeRedRuntime(map[string]string{
		"NODE_RED_PORT":      "2880",
		"NODE_RED_USER_DIR":  overrideDir,
		"NODE_RED_SETTINGS":  filepath.Join(settingsDir, "alt-settings.js"),
	}, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Port != "2880" {
		t.Errorf("Port = %q, want %q", rt.Port, "2880")
	}
	if !filepath.IsAbs(rt.UserDir) || filepath.Clean(rt.UserDir) != filepath.Clean(overrideDir) {
		t.Errorf("UserDir = %q, want %q", rt.UserDir, overrideDir)
	}
	if rt.SettingsPath != filepath.Join(settingsDir, "alt-settings.js") {
		t.Errorf("SettingsPath = %q, want %q", rt.SettingsPath, filepath.Join(settingsDir, "alt-settings.js"))
	}
}

// TestResolveNodeRedRuntime_InvalidPort rejects non-integer and out-of-range ports.
func TestResolveNodeRedRuntime_InvalidPort(t *testing.T) {
	cases := map[string]string{
		"non-numeric": "abc",
		"zero":        "0",
		"negative":    "-1",
		"too-large":   "70000",
	}
	for name, port := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := resolveNodeRedRuntime(map[string]string{
				"NODE_RED_PORT": port,
			}, t.TempDir())
			if err == nil {
				t.Fatalf("expected error for NODE_RED_PORT=%q", port)
			}
			if !strings.Contains(err.Error(), "NODE_RED_PORT") {
				t.Errorf("error %q must mention NODE_RED_PORT", err)
			}
		})
	}
}

// TestResolveNodeRedRuntime_SettingsRelative verifies that a relative
// NODE_RED_SETTINGS path resolves under the resolved UserDir.
func TestResolveNodeRedRuntime_SettingsRelative(t *testing.T) {
	dataDir := t.TempDir()
	rt, err := resolveNodeRedRuntime(map[string]string{
		"NODE_RED_SETTINGS": "custom/my-settings.js",
	}, dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(rt.UserDir, "custom", "my-settings.js")
	if rt.SettingsPath != want {
		t.Errorf("SettingsPath = %q, want %q", rt.SettingsPath, want)
	}
}

// TestResolveNodeRedRuntime_PortBoundary accepts port 1 and 65535.
func TestResolveNodeRedRuntime_PortBoundary(t *testing.T) {
	for _, port := range []string{"1", "65535"} {
		rt, err := resolveNodeRedRuntime(map[string]string{
			"NODE_RED_PORT": port,
		}, t.TempDir())
		if err != nil {
			t.Errorf("port %q should be accepted, got error: %v", port, err)
			continue
		}
		if rt.Port != port {
			t.Errorf("Port = %q, want %q", rt.Port, port)
		}
	}
}