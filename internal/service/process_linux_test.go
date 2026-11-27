//go:build linux

package service

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func TestProcessManagerRuntimeConsoleLoggingStatus(t *testing.T) {
	cases := []struct {
		name           string
		disk           string
		launch         *string
		readyStatus    int
		removeSettings bool
		want           string
	}{
		{name: "applied", disk: "debug", launch: stringPointer("debug"), readyStatus: http.StatusUnauthorized, want: "applied"},
		{name: "pending", disk: "warn", launch: stringPointer("debug"), readyStatus: http.StatusUnauthorized, want: "pending"},
		{name: "unobservable launch", disk: "debug", readyStatus: http.StatusUnauthorized, want: "unknown"},
		{name: "not ready", disk: "debug", launch: stringPointer("debug"), readyStatus: http.StatusServiceUnavailable, want: "unknown"},
		{name: "unreadable settings", disk: "debug", launch: stringPointer("debug"), readyStatus: http.StatusUnauthorized, removeSettings: true, want: "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ready := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				code := tc.readyStatus
				if code == 0 {
					code = http.StatusUnauthorized
				}
				w.WriteHeader(code)
			}))
			defer ready.Close()
			settingsPath := filepath.Join(t.TempDir(), "settings.js")
			if err := os.WriteFile(settingsPath, []byte("module.exports = {\n  logging: {\n    console: { level: '"+tc.disk+"' },\n  },\n};\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("sleep", "60")
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
			port := ready.Listener.Addr().(*net.TCPAddr).Port
			pm := &ProcessManager{
				running: true, currentCmd: cmd, runtimePort: port,
				runtimeSettingsPath: settingsPath, launchConsoleLevel: tc.launch,
			}
			if tc.removeSettings {
				if err := os.Remove(settingsPath); err != nil {
					t.Fatal(err)
				}
			}

			got := pm.RuntimeConsoleLoggingStatus()
			if got.Status != tc.want {
				t.Fatalf("status = %q, want %q", got.Status, tc.want)
			}
			if tc.removeSettings {
				if got.ConfiguredLevel != nil {
					t.Fatalf("configured level = %q, want unknown", *got.ConfiguredLevel)
				}
			} else if got.ConfiguredLevel == nil || *got.ConfiguredLevel != tc.disk {
				t.Fatalf("configured level = %v, want %q", got.ConfiguredLevel, tc.disk)
			}
			if tc.want == "unknown" && got.RunningLevel != nil {
				t.Fatalf("unknown running level = %q, want nil", *got.RunningLevel)
			}
			if tc.want != "unknown" && (got.RunningLevel == nil || *got.RunningLevel != "debug") {
				t.Fatalf("running level = %v, want debug", got.RunningLevel)
			}
		})
	}
}

func TestProcessManagerCapturesLevelFromResolvedSettingsPath(t *testing.T) {
	dataDir := t.TempDir()
	settingsPath := filepath.Join(t.TempDir(), "selected.js")
	if err := os.WriteFile(settingsPath, []byte("module.exports = {\n logging: {\n  console: { level: 'trace' },\n },\n};\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NODE_RED_SETTINGS", settingsPath)
	t.Setenv("NODE_RED_PORT", strconv.Itoa(unusedRuntimePortForTest(t)))
	launcher := filepath.Join(t.TempDir(), "node-red-fixture")
	if err := os.WriteFile(launcher, []byte("#!/bin/sh\nexec /bin/sleep 60\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	pm := NewProcessManager(launcher, dataDir)
	if err := pm.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = pm.Stop(context.Background()) })
	if pm.runtimeSettingsPath != settingsPath {
		t.Fatalf("launch settings path = %q, want resolved path %q", pm.runtimeSettingsPath, settingsPath)
	}
	if pm.launchConsoleLevel == nil || *pm.launchConsoleLevel != "trace" {
		t.Fatalf("launch console level = %v, want trace", pm.launchConsoleLevel)
	}
}

func TestReadConsoleLoggingLevelDoesNotEvaluateExpressions(t *testing.T) {
	for _, source := range []string{
		"module.exports = {\n logging: {\n  console: { level: getLevel() },\n },\n};\n",
		"module.exports = {\n logging: {\n  console: { level: `debug` },\n },\n};\n",
	} {
		path := filepath.Join(t.TempDir(), "settings.js")
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := readConsoleLoggingLevel(path); got != nil {
			t.Fatalf("expression %q was treated as a literal level %q", source, *got)
		}
	}
}

func stringPointer(value string) *string { return &value }
