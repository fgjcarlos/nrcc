package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// TestSave_RoundTripsHttpsBlock covers the Slice 1 acceptance for TLS:
// settings.js must contain the configured `https: { ... }` block with
// fs.readFileSync wrapping each path (Node-RED's expected shape) and the
// patcher must NOT duplicate the key on a second Save.
func TestSave_RoundTripsHttpsBlock(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")

	existing := `module.exports = {
  uiPort: 1880,
  httpAdminRoot: "/",
  httpNodeRoot: "/",
  adminAuth: null,
}
`
	if err := os.WriteFile(settingsPath, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := model.DefaultNodeRedConfig()
	cfg.Https = &model.HttpsConfig{
		Key:    "/etc/node-red/tls/key.pem",
		Cert:   "/etc/node-red/tls/cert.pem",
		CA:     "/etc/node-red/tls/ca.pem",
		Port:   8443,
	}
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	after := readFileString(t, settingsPath)
	mustContain := []string{
		"https: {",
		`key: require('fs').readFileSync("/etc/node-red/tls/key.pem")`,
		`cert: require('fs').readFileSync("/etc/node-red/tls/cert.pem")`,
		`ca: require('fs').readFileSync("/etc/node-red/tls/ca.pem")`,
		"port: 8443",
	}
	for _, want := range mustContain {
		if !strings.Contains(after, want) {
			t.Fatalf("Save dropped %q from settings.js:\n%s", want, after)
		}
	}

	// Idempotent: a second Save must not duplicate the https block.
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	twice := readFileString(t, settingsPath)
	if n := strings.Count(twice, "https:"); n != 1 {
		t.Fatalf("https appears %d times after two saves:\n%s", n, twice)
	}
}

// TestSave_RemovesHttpsBlockWhenCleared ensures that flipping the
// structured model back to "no TLS configured" removes the entire
// `https: { ... }` block — operators rely on this to revert Node-RED to
// plain HTTP when their certs expire.
func TestSave_RemovesHttpsBlockWhenCleared(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")

	existing := `module.exports = {
  uiPort: 1880,
  https: {
    key: require('fs').readFileSync("/etc/node-red/key.pem"),
    cert: require('fs').readFileSync("/etc/node-red/cert.pem"),
  },
  httpNodeRoot: "/",
  adminAuth: null,
}
`
	if err := os.WriteFile(settingsPath, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := model.DefaultNodeRedConfig()
	cfg.Https = nil
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	after := readFileString(t, settingsPath)
	if strings.Contains(after, "https:") {
		t.Fatalf("Save kept an https block when cfg.Https was nil:\n%s", after)
	}
	if !strings.Contains(after, "httpNodeRoot: \"/\"") {
		t.Fatalf("Save dropped unrelated scalar httpNodeRoot:\n%s", after)
	}
}

// TestSave_RoundTripsRequireHttps covers the simple boolean slice of
// the TLS acceptance — requireHttps is a top-level boolean and must
// survive Save/Save idempotency.
func TestSave_RoundTripsRequireHttps(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")

	existing := `module.exports = {
  uiPort: 1880,
  httpAdminRoot: "/",
  httpNodeRoot: "/",
  adminAuth: null,
}
`
	if err := os.WriteFile(settingsPath, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := model.DefaultNodeRedConfig()
	cfg.RequireHttps = true
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	after := readFileString(t, settingsPath)
	if !strings.Contains(after, "requireHttps: true,") {
		t.Fatalf("Save did not write requireHttps:\n%s", after)
	}

	if err := svc.Save(cfg); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	twice := readFileString(t, settingsPath)
	if n := strings.Count(twice, "requireHttps:"); n != 1 {
		t.Fatalf("requireHttps appears %d times after two saves:\n%s", n, twice)
	}
}

// TestSave_RoundTripsCredentialSecret covers the rotation path for
// `credentialSecret`. The patcher writes a quoted passphrase on Save
// and leaves any existing setting intact when cfg.CredentialSecret is
// empty (the rotation UI is responsible for always sending the new
// value when the operator confirms the warning).
func TestSave_RoundTripsCredentialSecret(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")

	existing := `module.exports = {
  uiPort: 1880,
  credentialSecret: "old-secret-do-not-leak",
  httpAdminRoot: "/",
  httpNodeRoot: "/",
  adminAuth: null,
}
`
	if err := os.WriteFile(settingsPath, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	// Rotation case: cfg carries a new secret.
	cfg := model.DefaultNodeRedConfig()
	cfg.CredentialSecret = "rotated-secret-on-purpose"
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	after := readFileString(t, settingsPath)
	if !strings.Contains(after, `credentialSecret: "rotated-secret-on-purpose"`) {
		t.Fatalf("Save did not rotate credentialSecret:\n%s", after)
	}

	// Disable case: cfg carries "false" (Node-RED 5 disables credential
	// encryption at rest when credentialSecret is the literal false).
	cfg.CredentialSecret = "false"
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("Save with false: %v", err)
	}
	after = readFileString(t, settingsPath)
	if !strings.Contains(after, "credentialSecret: false,") {
		t.Fatalf("Save did not write credentialSecret: false:\n%s", after)
	}
}

// TestSave_LeavesCredentialSecretUnchangedWhenEmpty documents the
// "rotation UI is the trigger" contract — when the structured config
// leaves CredentialSecret empty the patcher MUST NOT touch the
// existing settings.js line. This is what lets the UI show a Save
// button only when the user actively rotates the secret.
func TestSave_LeavesCredentialSecretUnchangedWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")

	existing := `module.exports = {
  uiPort: 1880,
  credentialSecret: "do-not-touch",
  adminAuth: null,
}
`
	if err := os.WriteFile(settingsPath, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := model.DefaultNodeRedConfig()
	cfg.CredentialSecret = "" // operator did not rotate
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	after := readFileString(t, settingsPath)
	if !strings.Contains(after, `credentialSecret: "do-not-touch"`) {
		t.Fatalf("Save rewrote credentialSecret when cfg was empty:\n%s", after)
	}
}

// TestParseHttpsBlockFromJS verifies the regex-based reader pulls the
// fs.readFileSync paths back out of a freshly rendered settings.js and
// round-trips them into the structured HttpsConfig used by the model.
func TestParseHttpsBlockFromJS(t *testing.T) {
	body := `module.exports = {
  uiPort: 1880,
  https: {
    key: require('fs').readFileSync('/etc/node-red/key.pem'),
    cert: require('fs').readFileSync('/etc/node-red/cert.pem'),
    ca: require('fs').readFileSync('/etc/node-red/ca.pem'),
    port: 443,
    passphrase: "test-pass",
  },
  adminAuth: null,
}
`
	got := parseHttpsBlockFromJS(body)
	if got == nil {
		t.Fatalf("parseHttpsBlockFromJS returned nil for a present https block")
	}
	if got.Key != "/etc/node-red/key.pem" {
		t.Errorf("Key = %q, want /etc/node-red/key.pem", got.Key)
	}
	if got.Cert != "/etc/node-red/cert.pem" {
		t.Errorf("Cert = %q, want /etc/node-red/cert.pem", got.Cert)
	}
	if got.CA != "/etc/node-red/ca.pem" {
		t.Errorf("CA = %q, want /etc/node-red/ca.pem", got.CA)
	}
	if got.Port != 443 {
		t.Errorf("Port = %d, want 443", got.Port)
	}
	if got.Passphrase != "test-pass" {
		t.Errorf("Passphrase = %q, want test-pass", got.Passphrase)
	}
}

// TestParseHttpsBlockFromJS_ReturnsNilWhenAbsent documents the "no
// https block means no TLS configured" contract — the parser must
// return a nil pointer so the renderer can distinguish "operator
// cleared TLS" from "operator never set TLS".
func TestParseHttpsBlockFromJS_ReturnsNilWhenAbsent(t *testing.T) {
	body := `module.exports = { uiPort: 1880, adminAuth: null }`
	if got := parseHttpsBlockFromJS(body); got != nil {
		t.Fatalf("parseHttpsBlockFromJS returned %+v for absent https block", got)
	}
}

// TestParseCredentialSecretFromJS confirms the three legal shapes —
// quoted passphrase, literal false, and absent — round-trip through the
// parser. An absent credentialSecret must come back as the empty
// string so the rotation UI never shows a stale value.
func TestParseCredentialSecretFromJS(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"quoted", `credentialSecret: "abc123"`, "abc123"},
		{"single quoted", `credentialSecret: 'abc123'`, "abc123"},
		{"false", `credentialSecret: false`, "false"},
		{"absent", `uiPort: 1880`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseCredentialSecretFromJS(tc.body); got != tc.want {
				t.Fatalf("parseCredentialSecretFromJS(%q) = %q, want %q", tc.body, got, tc.want)
			}
		})
	}
}

// TestPatchSettingsJS_RemovesHttpsBlockWhenNil locks the renderer's
// "clear means delete" behaviour through the patcher directly, without
// touching the disk. This is the unit-level guard the handler-level
// Save_RemovesHttpsBlockWhenCleared relies on.
func TestPatchSettingsJS_RemovesHttpsBlockWhenNil(t *testing.T) {
	body := `module.exports = {
  uiPort: 1880,
  https: {
    key: require('fs').readFileSync('/x'),
    cert: require('fs').readFileSync('/y'),
  },
  adminAuth: null,
}
`
	cfg := model.DefaultNodeRedConfig()
	got := patchSettingsJS(body, cfg)
	if strings.Contains(got, "https:") {
		t.Fatalf("patchSettingsJS kept https block when cfg.Https was nil:\n%s", got)
	}
}

// TestNodeRED5Catalog_TLSEntries are part of the Slice 1 acceptance
// criteria from #762: every control states its meaning and default.
// We assert the canonical strings here so the catalog table stays the
// single source of truth for the docs surfaced in the UI.
func TestNodeRED5Catalog_TLSEntries(t *testing.T) {
	catalog := NodeRED5Catalog()
	entries := make(map[string]SettingCatalogEntry, len(catalog))
	for _, e := range catalog {
		entries[e.Key] = e
	}

	https, ok := entries["https"]
	if !ok {
		t.Fatalf("Node-RED 5 catalog is missing the `https` entry required by #762")
	}
	if !https.RestartRequired {
		t.Errorf("https: RestartRequired = false, want true")
	}
	if !https.Secret {
		t.Errorf("https: Secret = false, want true (paths to cert/key are sensitive)")
	}
	if !https.UIEditable {
		t.Errorf("https: UIEditable = false, want true (slice 1 ships the structured editor)")
	}

	requireHttps, ok := entries["requireHttps"]
	if !ok {
		t.Fatalf("Node-RED 5 catalog is missing the `requireHttps` entry required by #762")
	}
	if !requireHttps.RestartRequired {
		t.Errorf("requireHttps: RestartRequired = false, want true")
	}
	if requireHttps.Secret {
		t.Errorf("requireHttps: Secret = true, want false (boolean toggle)")
	}

	cred, ok := entries["credentialSecret"]
	if !ok {
		t.Fatalf("Node-RED 5 catalog is missing the `credentialSecret` entry required by #762")
	}
	if !cred.Secret {
		t.Errorf("credentialSecret: Secret = false, want true (passphrase material)")
	}
	if !cred.RestartRequired {
		t.Errorf("credentialSecret: RestartRequired = false, want true")
	}
}