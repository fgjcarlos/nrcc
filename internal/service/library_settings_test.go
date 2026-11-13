package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dop251/goja/parser"
)

const librarySettingsFixture = `module.exports = {
  uiPort: 1880,
  functionGlobalContext: {
    os: require('os'),
  },
  externalModules: {}
}
`

func TestActivateLibraryInSettings(t *testing.T) {
	got := activateLibraryInSettings(librarySettingsFixture, "lodash@4", "_", true)

	if !strings.Contains(got, "  functionExternalModules: true,\n") {
		t.Errorf("functionExternalModules not enabled:\n%s", got)
	}
	if !strings.Contains(got, `"_": (function () { try { return require("lodash"); } catch (err) { return undefined; } })(), // nrcc:library "lodash"`) {
		t.Errorf("library not exposed in functionGlobalContext:\n%s", got)
	}
	if !strings.Contains(got, "    os: require('os'),\n") {
		t.Errorf("operator-authored entry was not preserved:\n%s", got)
	}
	if _, err := parser.ParseFile(nil, "settings.js", got, 0); err != nil {
		t.Errorf("rendered settings.js does not parse: %v\n%s", err, got)
	}
	if again := activateLibraryInSettings(got, "lodash", "_", true); again != got {
		t.Errorf("activation is not idempotent:\n%s", again)
	}
}

func TestActivateLibraryInSettings_CreatesGlobalContextBlock(t *testing.T) {
	got := activateLibraryInSettings("module.exports = {\n  uiPort: 1880,\n}\n", "@scope/pkg@^1.2.0", "pkg", true)

	if !strings.Contains(got, "functionGlobalContext: {\n") || !strings.Contains(got, `require("@scope/pkg")`) {
		t.Errorf("functionGlobalContext block not created:\n%s", got)
	}
	if _, err := parser.ParseFile(nil, "settings.js", got, 0); err != nil {
		t.Errorf("rendered settings.js does not parse: %v\n%s", err, got)
	}
}

func TestActivateLibraryInSettings_NodeRedNodeOnlyEnablesExternalModules(t *testing.T) {
	got := activateLibraryInSettings(librarySettingsFixture, "node-red-dashboard", "node-red-dashboard", false)

	if !strings.Contains(got, "functionExternalModules: true") {
		t.Errorf("functionExternalModules not enabled:\n%s", got)
	}
	if strings.Contains(got, "nrcc:library") {
		t.Errorf("a Node-RED node package must not be exposed in functionGlobalContext:\n%s", got)
	}
}

func TestDeactivateLibraryInSettings(t *testing.T) {
	activated := activateLibraryInSettings(librarySettingsFixture, "lodash", "_", true)
	activated = activateLibraryInSettings(activated, "moment", "moment", true)

	got := deactivateLibraryInSettings(activated, "lodash")

	if strings.Contains(got, `"lodash"`) {
		t.Errorf("lodash entry not removed:\n%s", got)
	}
	if !strings.Contains(got, `require("moment")`) || !strings.Contains(got, "os: require('os')") {
		t.Errorf("unrelated entries were removed:\n%s", got)
	}
}

// settingsRecorder is a fake settings updater that applies the transform to
// an in-memory settings.js, the way the server wiring applies it to disk.
type settingsRecorder struct {
	content string
	calls   int
}

func (r *settingsRecorder) update(transform func(string) string) error {
	r.calls++
	r.content = transform(r.content)
	return nil
}

func TestLibraryServiceInstallActivatesLibraryInSettings(t *testing.T) {
	pm := &stubPM{}
	svc := NewLibraryServiceWithPackageManager(t.TempDir(), pm)
	rec := &settingsRecorder{content: librarySettingsFixture}
	svc.SetSettingsUpdater(rec.update)

	if err := svc.Install(context.Background(), "lodash", "_"); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !strings.Contains(rec.content, `"_": (function () { try { return require("lodash")`) {
		t.Errorf("settings.js not updated after install:\n%s", rec.content)
	}

	if err := svc.Uninstall(context.Background(), "lodash"); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if strings.Contains(rec.content, "nrcc:library") {
		t.Errorf("settings.js still exposes the library after uninstall:\n%s", rec.content)
	}
}

func TestLibraryServiceInstallSkipsGlobalContextForNodeRedNodes(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "node-red-dashboard")
	if err := os.MkdirAll(pkgDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"node-red-dashboard","node-red":{"nodes":{}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewLibraryServiceWithPackageManager(dir, &stubPM{})
	rec := &settingsRecorder{content: librarySettingsFixture}
	svc.SetSettingsUpdater(rec.update)

	if err := svc.Install(context.Background(), "node-red-dashboard", ""); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if strings.Contains(rec.content, "nrcc:library") || !strings.Contains(rec.content, "functionExternalModules: true") {
		t.Errorf("unexpected settings.js for a Node-RED node package:\n%s", rec.content)
	}
}

func TestLibraryServiceInstallRejectsInvalidAliasBeforeNpm(t *testing.T) {
	pm := &stubPM{}
	svc := NewLibraryServiceWithPackageManager(t.TempDir(), pm)
	rec := &settingsRecorder{content: librarySettingsFixture}
	svc.SetSettingsUpdater(rec.update)

	err := svc.Install(context.Background(), "lodash", `x"); process.exit(1); ("`)
	if !errors.Is(err, ErrInvalidLibraryAlias) {
		t.Fatalf("Install error = %v, want ErrInvalidLibraryAlias", err)
	}
	if pm.installCalls != 0 || rec.calls != 0 {
		t.Errorf("npm calls = %d, settings calls = %d; want 0 and 0", pm.installCalls, rec.calls)
	}
}

// TestTransformRawSettingsPersistsLibraryActivation drives the real disk path
// the server wires into LibraryService: the guarded require() must pass the
// raw-save validation and land in settings.js.
func TestTransformRawSettingsPersistsLibraryActivation(t *testing.T) {
	dir := t.TempDir()
	svc := NewConfigServiceWithHost(dir, NewIsolatedHostService(dir))
	if _, err := svc.SaveRawSettings(librarySettingsFixture); err != nil {
		t.Fatalf("seed settings.js: %v", err)
	}

	err := svc.TransformRawSettings(func(content string) string {
		return activateLibraryInSettings(content, "lodash", "_", true)
	})
	if err != nil {
		t.Fatalf("TransformRawSettings: %v", err)
	}
	doc, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("GetRawSettings: %v", err)
	}
	if !strings.Contains(doc.Content, `require("lodash")`) || !strings.Contains(doc.Content, "functionExternalModules: true") {
		t.Errorf("settings.js on disk was not activated:\n%s", doc.Content)
	}
}

// A one-line functionGlobalContext must still render valid JavaScript: the
// entry's trailing line comment must not swallow the rest of the line.
func TestActivateLibraryInSettings_OneLineGlobalContext(t *testing.T) {
	for _, in := range []string{
		"module.exports = {\n  uiPort: 1880,\n  functionGlobalContext: {},\n}\n",
		"module.exports = {\n  uiPort: 1880,\n  functionGlobalContext: { os: require('os') },\n}\n",
	} {
		got := activateLibraryInSettings(in, "lodash", "_", true)
		if _, err := parser.ParseFile(nil, "settings.js", got, 0); err != nil {
			t.Errorf("does not parse: %v\n%s", err, got)
		}
		if strings.Contains(in, "os:") && !strings.Contains(got, "os: require('os')") {
			t.Errorf("operator entry lost:\n%s", got)
		}
	}
}

// An invalid package name keeps reporting INVALID_PACKAGE_NAME even when the
// alias defaults to that name.
func TestLibraryServiceInstallInvalidPackageBeatsAlias(t *testing.T) {
	pm := &stubPM{}
	svc := NewLibraryServiceWithPackageManager(t.TempDir(), pm)
	err := svc.Install(context.Background(), "lodash;rm", "")
	if !errors.Is(err, ErrInvalidPackageName) {
		t.Fatalf("err = %v, want ErrInvalidPackageName", err)
	}
	if pm.installCalls != 0 {
		t.Errorf("npm ran %d times for an invalid package name", pm.installCalls)
	}
}

// Read-only configuration is never written, and must not turn a successful
// install into a failure (baseline behaviour before settings activation).
func TestTransformRawSettingsSkipsReadOnlyConfiguration(t *testing.T) {
	dir := t.TempDir()
	svc := NewConfigServiceWithHost(dir, nil) // no host: version unknown, not editable
	called := false
	if err := svc.TransformRawSettings(func(c string) string { called = true; return c + "x" }); err != nil {
		t.Fatalf("TransformRawSettings on read-only configuration = %v, want nil", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "settings.js")); !os.IsNotExist(err) || called {
		t.Errorf("read-only configuration was touched (stat err = %v, transform called = %t)", err, called)
	}
}
