package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// Save() used to regenerate settings.js from scratch, silently dropping every
// operator-authored block. Regression guard for #577.
func TestSave_PreservesOperatorBlocksInSettingsJS(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")

	existing := `module.exports = {
  uiPort: 1880,
  httpAdminRoot: "/",
  httpNodeRoot: "/",
  functionGlobalContext: {
    os: require('os'),
  },
  externalModules: {
    autoInstall: true,
  },
  adminAuth: null,
}
`
	if err := os.WriteFile(settingsPath, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := model.DefaultNodeRedConfig()
	cfg.Port = 3000
	cfg.UIPort = 3000
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	after := readFileString(t, settingsPath)
	for _, want := range []string{"functionGlobalContext", "externalModules", "autoInstall", "uiPort: 3000"} {
		if !strings.Contains(after, want) {
			t.Fatalf("Save dropped %q from settings.js:\n%s", want, after)
		}
	}

	// A second Save must not duplicate keys — patch has to be idempotent.
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	twice := readFileString(t, settingsPath)
	for _, key := range []string{"adminAuth:", "editorTheme:", "logging:", "uiPort:"} {
		if n := strings.Count(twice, key); n != 1 {
			t.Fatalf("key %q appears %d times after two saves:\n%s", key, n, twice)
		}
	}
}

// First write, no existing file: generate from scratch.
func TestSave_GeneratesSettingsJSWhenMissing(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)

	if err := svc.Save(model.DefaultNodeRedConfig()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	content := readFileString(t, filepath.Join(dir, "settings.js"))
	if !strings.Contains(content, "module.exports") || !strings.Contains(content, "uiPort:") {
		t.Fatalf("generated settings.js looks wrong:\n%s", content)
	}
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
