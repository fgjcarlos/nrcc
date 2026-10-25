package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// loggingFixture is a settings.js source with a logging block that carries
// one managed handler, one property NRCC does not expose in the UI
// (`audit`) and one operator-owned handler that NRCC must not touch.
const loggingFixture = `module.exports = {
  uiPort: 1880,
  httpAdminRoot: '/',
  httpNodeRoot: '/',
  logging: {
    console: { level: 'info', metrics: false, audit: false },
    custom: { handler: function (options) { return { log: console.log }; } },
  },
}
`

// TestPatchSettingsJS_RendersSubmittedLoggingValues asserts the renderer
// writes the logging values the operator submitted instead of a constant
// block (issue #882).
func TestPatchSettingsJS_RendersSubmittedLoggingValues(t *testing.T) {
	cfg := model.DefaultNodeRedConfig()
	cfg.Logging = map[string]any{
		"console": map[string]any{"level": "debug", "metrics": true},
	}

	patched := patchSettingsJS(loggingFixture, cfg)

	if !strings.Contains(patched, "level: 'debug'") {
		t.Errorf("patched settings.js did not render the submitted level:\n%s", patched)
	}
	if !strings.Contains(patched, "metrics: true") {
		t.Errorf("patched settings.js did not render the submitted metrics flag:\n%s", patched)
	}
	if strings.Contains(patched, "level: 'info'") {
		t.Errorf("patched settings.js still carries the constant level:\n%s", patched)
	}
}

// TestPatchSettingsJS_RendersSubmittedLoggingHandler asserts the internal
// handler the NRCC UI publishes reaches settings.js as well.
func TestPatchSettingsJS_RendersSubmittedLoggingHandler(t *testing.T) {
	cfg := model.DefaultNodeRedConfig()
	cfg.Logging = map[string]any{
		"console":  map[string]any{"level": "warn"},
		"internal": map[string]any{"level": "trace", "metrics": true},
	}

	patched := patchSettingsJS(loggingFixture, cfg)

	if !strings.Contains(patched, "internal: { level: 'trace', metrics: true }") {
		t.Errorf("patched settings.js did not render the internal handler:\n%s", patched)
	}
}

// TestPatchSettingsJS_PreservesCustomLoggingHandler asserts an
// operator-owned logging entry survives a save that edits the managed
// handlers. The custom handler carries a function the renderer cannot
// reproduce, so it must be left alone.
func TestPatchSettingsJS_PreservesCustomLoggingHandler(t *testing.T) {
	cfg := model.DefaultNodeRedConfig()
	cfg.Logging = map[string]any{
		"console": map[string]any{"level": "debug", "metrics": true},
	}

	patched := patchSettingsJS(loggingFixture, cfg)

	if !strings.Contains(patched, "custom: { handler: function (options) { return { log: console.log }; } }") {
		t.Errorf("patched settings.js dropped the custom logging handler:\n%s", patched)
	}
	if strings.Count(patched, "custom:") != 1 {
		t.Errorf("patched settings.js duplicated the custom logging handler:\n%s", patched)
	}
}

// TestPatchSettingsJS_PreservesUnsubmittedLoggingProperty asserts a
// property of a managed handler the operator did not submit keeps its
// value.
func TestPatchSettingsJS_PreservesUnsubmittedLoggingProperty(t *testing.T) {
	cfg := model.DefaultNodeRedConfig()
	cfg.Logging = map[string]any{
		"console": map[string]any{"level": "debug"},
	}

	patched := patchSettingsJS(loggingFixture, cfg)

	if !strings.Contains(patched, "audit: false") {
		t.Errorf("patched settings.js dropped the audit flag:\n%s", patched)
	}
	if !strings.Contains(patched, "metrics: false") {
		t.Errorf("patched settings.js dropped the metrics flag:\n%s", patched)
	}
}

// TestPatchSettingsJS_KeepsUnconfiguredLoggingBlock asserts a source that
// declares a logging block is not rewritten when the operator configured
// nothing: the operator's own block is not NRCC's to replace.
func TestPatchSettingsJS_KeepsUnconfiguredLoggingBlock(t *testing.T) {
	patched := patchSettingsJS(loggingFixture, model.DefaultNodeRedConfig())

	if !strings.Contains(patched, "console: { level: 'info', metrics: false, audit: false }") {
		t.Errorf("patched settings.js rewrote the operator's logging block:\n%s", patched)
	}
	if !strings.Contains(patched, "custom: { handler: function (options) { return { log: console.log }; } }") {
		t.Errorf("patched settings.js dropped the custom logging handler:\n%s", patched)
	}
}

// TestPatchSettingsJS_AppendsLoggingBlockWhenMissing asserts a source
// without a logging block receives the submitted values.
func TestPatchSettingsJS_AppendsLoggingBlockWhenMissing(t *testing.T) {
	source := "module.exports = {\n  uiPort: 1880,\n}\n"
	cfg := model.DefaultNodeRedConfig()
	cfg.Logging = map[string]any{
		"console": map[string]any{"level": "error", "metrics": true},
	}

	patched := patchSettingsJS(source, cfg)

	if !strings.Contains(patched, "logging: {") {
		t.Fatalf("patched settings.js has no logging block:\n%s", patched)
	}
	if !strings.Contains(patched, "console: { level: 'error', metrics: true }") {
		t.Errorf("patched settings.js did not render the submitted values:\n%s", patched)
	}
}

// TestPatchLoggingSettings_Idempotent asserts a save that changes nothing
// leaves the logging block byte for byte alone.
func TestPatchLoggingSettings_Idempotent(t *testing.T) {
	logging := map[string]any{
		"console": map[string]any{"level": "info", "metrics": false, "audit": false},
	}

	patched := patchLoggingSettings(loggingFixture, logging)

	if patched != loggingFixture {
		t.Errorf("patching the logging block with its own values changed it:\n%s", patched)
	}
}

// TestParseConfigFromContent_ReadsLoggingBlock asserts the structured
// model learns the logging values of the source (issue #882).
func TestParseConfigFromContent_ReadsLoggingBlock(t *testing.T) {
	svc := NewIsolatedConfigService(t.TempDir())

	cfg, err := svc.parseConfigFromContent(loggingFixture)
	if err != nil {
		t.Fatalf("parseConfigFromContent: %v", err)
	}

	logging := newLoggingModel(cfg.Logging)
	console, ok := logging["console"]
	if !ok {
		t.Fatalf("parsed configuration has no console handler: %+v", cfg.Logging)
	}
	if console["level"] != "info" {
		t.Errorf("console level = %v, want info", console["level"])
	}
	if console["metrics"] != false {
		t.Errorf("console metrics = %v, want false", console["metrics"])
	}
	if console["audit"] != false {
		t.Errorf("console audit = %v, want false", console["audit"])
	}
	if _, ok := logging["custom"]; !ok {
		t.Errorf("parsed configuration dropped the custom handler: %+v", cfg.Logging)
	}
}

// TestLoggingRoundTrip_RenderParseRender asserts the values that reach
// settings.js are the values the structured model reads back.
func TestLoggingRoundTrip_RenderParseRender(t *testing.T) {
	svc := NewIsolatedConfigService(t.TempDir())
	cfg := model.DefaultNodeRedConfig()
	cfg.Logging = map[string]any{
		"console":  map[string]any{"level": "debug", "metrics": true, "audit": true},
		"internal": map[string]any{"level": "warn"},
	}

	rendered := generateSettingsJS(cfg)
	if !strings.Contains(rendered, "console: { level: 'debug', metrics: true, audit: true }") {
		t.Fatalf("generated settings.js did not render the submitted values:\n%s", rendered)
	}

	parsed, err := svc.parseConfigFromContent(rendered)
	if err != nil {
		t.Fatalf("parseConfigFromContent: %v", err)
	}

	again := generateSettingsJS(parsed)
	if again != rendered {
		t.Errorf("the logging block did not survive the round trip:\nfirst:\n%s\nsecond:\n%s", rendered, again)
	}
}

// TestLoggingRoundTrip_AppliesSubmittedValuesToExistingFile asserts the
// values the operator submits through the API reach the file and are read
// back by the structured model.
func TestLoggingRoundTrip_AppliesSubmittedValuesToExistingFile(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(loggingFixture), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	cfg, err := svc.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	cfg.Logging = map[string]any{
		"console": map[string]any{"level": "debug", "metrics": true},
	}
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	written, err := os.ReadFile(settingsPath) // #nosec G304 -- settingsPath is derived from t.TempDir().
	if err != nil {
		t.Fatalf("read settings.js: %v", err)
	}
	if !strings.Contains(string(written), "console: { level: 'debug', metrics: true, audit: false }") {
		t.Errorf("settings.js does not carry the submitted logging values:\n%s", written)
	}
	if !strings.Contains(string(written), "custom: { handler: function (options) { return { log: console.log }; } }") {
		t.Errorf("settings.js dropped the custom logging handler:\n%s", written)
	}

	reloaded, err := svc.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	logging := newLoggingModel(reloaded.Logging)
	if logging["console"]["level"] != "debug" {
		t.Errorf("the mirror reports console level %v, want debug", logging["console"]["level"])
	}
	if logging["console"]["metrics"] != true {
		t.Errorf("the mirror reports console metrics %v, want true", logging["console"]["metrics"])
	}
}

// TestPatchSettingsJS_KeepsConsoleHandlerInTheBlock asserts the rendered
// block always carries a console handler. Node-RED registers the entries
// named "console" or the ones exposing a handler function, so a block
// without a console entry would leave the runtime completely silent.
func TestPatchSettingsJS_KeepsConsoleHandlerInTheBlock(t *testing.T) {
	source := `module.exports = {
  logging: {
    custom: { handler: function (options) { return { log: console.log }; } },
  },
}
`
	cfg := model.DefaultNodeRedConfig()
	cfg.Logging = map[string]any{
		"internal": map[string]any{"level": "warn"},
	}

	patched := patchSettingsJS(source, cfg)

	if !strings.Contains(patched, "console: { level: 'info', metrics: false, audit: false }") {
		t.Errorf("patched settings.js has no console handler:\n%s", patched)
	}
	if !strings.Contains(patched, "internal: { level: 'warn' }") {
		t.Errorf("patched settings.js did not render the internal handler:\n%s", patched)
	}
	if !strings.Contains(patched, "custom: { handler: function (options) { return { log: console.log }; } }") {
		t.Errorf("patched settings.js dropped the custom logging handler:\n%s", patched)
	}
}

// TestGenerateSettingsJS_KeepsConsoleHandlerInTheBlock asserts a generated
// block carries the console handler even when the model only holds the
// internal handler.
func TestGenerateSettingsJS_KeepsConsoleHandlerInTheBlock(t *testing.T) {
	cfg := model.DefaultNodeRedConfig()
	cfg.Logging = map[string]any{
		"internal": map[string]any{"level": "warn"},
	}

	rendered := generateSettingsJS(cfg)

	if !strings.Contains(rendered, "console: { level: 'info', metrics: false, audit: false }") {
		t.Errorf("generated settings.js has no console handler:\n%s", rendered)
	}
}

// TestQuoteJSString_EscapesOperatorInput asserts a submitted level can
// never terminate the literal and inject JavaScript into settings.js.
func TestQuoteJSString_EscapesOperatorInput(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{name: "plain", value: "debug", want: "'debug'"},
		{name: "quote", value: "de'bug", want: `'de\'bug'`},
		{name: "backslash", value: `a\b`, want: `'a\\b'`},
		{name: "newline", value: "a\nb", want: `'a\nb'`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := quoteJSString(tc.value); got != tc.want {
				t.Errorf("quoteJSString(%q) = %s, want %s", tc.value, got, tc.want)
			}
		})
	}
}

// TestLoggingLiteral_RejectsUnexpectedTypes asserts the renderer never
// writes a value it cannot vouch for.
func TestLoggingLiteral_RejectsUnexpectedTypes(t *testing.T) {
	if _, ok := loggingLiteral("level", 42); ok {
		t.Error("a numeric level must not be rendered")
	}
	if _, ok := loggingLiteral("metrics", "true"); ok {
		t.Error("a string metrics flag must not be rendered")
	}
	if _, ok := loggingLiteral("handler", "function () {}"); ok {
		t.Error("an unsupported property must not be rendered")
	}
}
