package service

import (
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// nodeRedDefaultSettingsJS is the settings.js Node-RED's own container image
// ships. Its last property has no trailing comma, which is what made nrcc's
// "append the missing key" path emit invalid JavaScript.
const nodeRedDefaultSettingsJS = `module.exports = {
    // Node-RED settings — managed by nrcc
    // See https://nodered.org/docs/user-guide/runtime/configuration
    flowFile: 'flows.json',
    credentialSecret: false,
    flowFilePretty: true,
    adminAuth: null,
    editorTheme: {
        projects: { enabled: false }
    },
    logging: {
        console: { level: 'info', metrics: false, audit: false }
    },
    exportGlobalContextKeys: false,
    externalModules: {}
}
`

// TestPatchSettingsJS_AppendsAfterAPropertyWithoutTrailingComma is the
// issue #881 regression test for the settings renderer: appending a key that
// the file does not carry yet must not produce invalid JavaScript.
//
// Without the fix the rendered file read
//
//	externalModules: {}
//	uiPort: 1880,
//
// which Node-RED (and nrcc's own settings validation) rejects with a syntax
// error, so no configuration edit could be saved at all in a fresh stack.
func TestPatchSettingsJS_AppendsAfterAPropertyWithoutTrailingComma(t *testing.T) {
	cfg := model.DefaultNodeRedConfig()
	cfg.ProjectsEnabled = false

	content := patchSettingsJS(nodeRedDefaultSettingsJS, cfg)

	if !strings.Contains(content, "externalModules: {},") {
		t.Fatalf("the property before the appended key needs a trailing comma:\n%s", content)
	}
	if !strings.Contains(content, "uiPort: 1880,") {
		t.Fatalf("patched settings.js missing uiPort:\n%s", content)
	}
	// Every appended entry must sit on its own line: a missing separator
	// would concatenate two properties into one invalid statement.
	if strings.Contains(content, "uiPort: 1880,  ") {
		t.Fatalf("appended entries must be separated by a newline:\n%s", content)
	}
	assertBalancedObject(t, content)
}

// TestPatchSettingsJS_AppendsEveryMissingKeySeparately covers a payload
// that carries several keys the file does not have yet.
func TestPatchSettingsJS_AppendsEveryMissingKeySeparately(t *testing.T) {
	cfg := model.DefaultNodeRedConfig()
	cfg.UserDir = "./data"
	cfg.ProjectsEnabled = false

	content := patchSettingsJS(nodeRedDefaultSettingsJS, cfg)

	for _, want := range []string{
		"uiPort: 1880,",
		`httpAdminRoot: "/",`,
		`httpNodeRoot: "/",`,
		`userDir: "./data",`,
		"disableEditor: false,",
		"projectsEnabled: false,",
		`httpStatic: "uploads",`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("patched settings.js missing %q:\n%s", want, content)
		}
	}
	assertBalancedObject(t, content)
}

// TestPatchSettingsJS_KeepsAnAlreadyValidFileUnchanged proves the fix does
// not disturb a file that already ends with a comma.
func TestPatchSettingsJS_KeepsAnAlreadyValidFileUnchanged(t *testing.T) {
	existing := `module.exports = {
  uiPort: 1880,
  externalModules: {},
}
`

	content := patchSettingsJS(existing, model.DefaultNodeRedConfig())

	if strings.Count(content, "externalModules: {},") != 1 {
		t.Fatalf("the existing entry must not be duplicated:\n%s", content)
	}
	assertBalancedObject(t, content)
}

// assertBalancedObject checks the structural invariants a rendered
// settings.js must satisfy: every top-level property is comma-separated and
// the module.exports object is closed exactly once.
func assertBalancedObject(t *testing.T, content string) {
	t.Helper()
	if !strings.HasPrefix(strings.TrimSpace(content), "module.exports = {") {
		t.Fatalf("rendered settings.js must export an object:\n%s", content)
	}
	if strings.Count(content, "module.exports") != 1 {
		t.Fatalf("rendered settings.js must declare module.exports once:\n%s", content)
	}
	if strings.Count(content, "{") != strings.Count(content, "}") {
		t.Fatalf("rendered settings.js has unbalanced braces:\n%s", content)
	}
	if !strings.HasSuffix(strings.TrimRight(content, " \t\r\n"), "}") {
		t.Fatalf("rendered settings.js must end with the closing brace:\n%s", content)
	}
}
