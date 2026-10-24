package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// The fixtures below are invented test values, never real credentials.
const (
	credentialSecretFixture = "audit-credential-fixture"
	tlsPassphraseFixture    = "audit-tls-fixture"
	// redactedSecretMarker mirrors model.RedactedSecret: a client that
	// echoes a redacted read back must not wipe the stored secret.
	redactedSecretMarker = "[redacted]"
)

// secretBearingSettingsJS is a settings.js document that already carries
// the TLS block, mirroring an operator's file after TLS was configured.
func secretBearingSettingsJS() string {
	return `module.exports = {
  uiPort: 1880,
  httpAdminRoot: "/",
  httpNodeRoot: "/",
  credentialSecret: 'initial-secret-value',
  adminAuth: null,
  https: {
    key: require('fs').readFileSync("/tmp/key.pem"),
    cert: require('fs').readFileSync("/tmp/cert.pem"),
    passphrase: "audit-tls-fixture",
  },
}
`
}

// secretBearingConfig returns a valid configuration that carries both
// write-only secrets.
func secretBearingConfig() model.NodeRedConfig {
	return model.NodeRedConfig{
		Port:             1880,
		UIPort:           1880,
		HTTPAdminRoot:    "/",
		HTTPNodeRoot:     "/",
		CredentialSecret: credentialSecretFixture,
		Https: &model.HttpsConfig{
			Key:        "/tmp/key.pem",
			Cert:       "/tmp/cert.pem",
			Port:       1883,
			Passphrase: tlsPassphraseFixture,
		},
	}
}

// newSecretService returns an isolated config service whose settings.js
// already carries the TLS block.
func newSecretService(t *testing.T) (*ConfigService, string) {
	t.Helper()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(secretBearingSettingsJS()), 0o600); err != nil {
		t.Fatalf("seed settings.js: %v", err)
	}
	return NewIsolatedConfigService(dir), settingsPath
}

// TestConfigService_Save_PreservesCredentialSecretWhenOmitted covers the
// write-back contract for the credential-encryption secret: an omitted
// field means "keep the stored value", never "clear the secret".
func TestConfigService_Save_PreservesCredentialSecretWhenOmitted(t *testing.T) {
	svc, _ := newSecretService(t)
	if err := svc.Save(secretBearingConfig()); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	candidate := secretBearingConfig()
	candidate.CredentialSecret = ""
	if err := svc.Save(candidate); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stored, err := svc.Get()
	if err != nil {
		t.Fatalf("read stored config: %v", err)
	}
	if stored.CredentialSecret != credentialSecretFixture {
		t.Errorf("stored credentialSecret = %q, want %q", stored.CredentialSecret, credentialSecretFixture)
	}
}

// TestConfigService_Save_PreservesCredentialSecretWhenRedacted covers a
// client that echoes a redaction placeholder back to the API.
func TestConfigService_Save_PreservesCredentialSecretWhenRedacted(t *testing.T) {
	svc, _ := newSecretService(t)
	if err := svc.Save(secretBearingConfig()); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	candidate := secretBearingConfig()
	candidate.CredentialSecret = redactedSecretMarker
	if err := svc.Save(candidate); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stored, err := svc.Get()
	if err != nil {
		t.Fatalf("read stored config: %v", err)
	}
	if stored.CredentialSecret != credentialSecretFixture {
		t.Errorf("stored credentialSecret = %q, want %q", stored.CredentialSecret, credentialSecretFixture)
	}
}

// TestConfigService_Save_PreservesHTTPSPassphraseWhenOmitted covers the
// write-back contract for the TLS private-key passphrase.
func TestConfigService_Save_PreservesHTTPSPassphraseWhenOmitted(t *testing.T) {
	svc, _ := newSecretService(t)
	if err := svc.Save(secretBearingConfig()); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	candidate := secretBearingConfig()
	candidate.Https = &model.HttpsConfig{
		Key:  "/tmp/key.pem",
		Cert: "/tmp/cert.pem",
		Port: 1883,
	}
	if err := svc.Save(candidate); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stored, err := svc.Get()
	if err != nil {
		t.Fatalf("read stored config: %v", err)
	}
	if stored.Https == nil || stored.Https.Passphrase != tlsPassphraseFixture {
		t.Fatalf("stored https.passphrase = %+v, want %q", stored.Https, tlsPassphraseFixture)
	}
}

// TestConfigService_Save_PreservesHTTPSPassphraseInSettingsJS proves the
// preserved passphrase also survives the settings.js render: dropping it
// would break the Node-RED TLS listener on the next restart.
func TestConfigService_Save_PreservesHTTPSPassphraseInSettingsJS(t *testing.T) {
	svc, settingsPath := newSecretService(t)
	if err := svc.Save(secretBearingConfig()); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	candidate := secretBearingConfig()
	candidate.Https = &model.HttpsConfig{
		Key:  "/tmp/key.pem",
		Cert: "/tmp/cert.pem",
		Port: 1883,
	}
	if err := svc.Save(candidate); err != nil {
		t.Fatalf("save config: %v", err)
	}

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings.js: %v", err)
	}
	if !strings.Contains(string(content), tlsPassphraseFixture) {
		t.Errorf("settings.js must keep the stored passphrase; got %s", content)
	}
}

// TestConfigService_Save_RotatesExplicitSecrets proves the write-only
// contract does not block a deliberate rotation.
func TestConfigService_Save_RotatesExplicitSecrets(t *testing.T) {
	svc, _ := newSecretService(t)
	if err := svc.Save(secretBearingConfig()); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	candidate := secretBearingConfig()
	candidate.CredentialSecret = "rotated-credential-fixture"
	candidate.Https = &model.HttpsConfig{
		Key:        "/tmp/key.pem",
		Cert:       "/tmp/cert.pem",
		Port:       1883,
		Passphrase: "rotated-tls-fixture",
	}
	if err := svc.Save(candidate); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stored, err := svc.Get()
	if err != nil {
		t.Fatalf("read stored config: %v", err)
	}
	if stored.CredentialSecret != "rotated-credential-fixture" {
		t.Errorf("stored credentialSecret = %q, want the rotated value", stored.CredentialSecret)
	}
	if stored.Https == nil || stored.Https.Passphrase != "rotated-tls-fixture" {
		t.Errorf("stored https.passphrase = %+v, want the rotated value", stored.Https)
	}
}

// TestConfigService_Save_CredentialSecretFalseDisablesEncryption proves the
// documented "false" opt-out is not mistaken for a redacted value.
func TestConfigService_Save_CredentialSecretFalseDisablesEncryption(t *testing.T) {
	svc, _ := newSecretService(t)
	if err := svc.Save(secretBearingConfig()); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	candidate := secretBearingConfig()
	candidate.CredentialSecret = "false"
	if err := svc.Save(candidate); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stored, err := svc.Get()
	if err != nil {
		t.Fatalf("read stored config: %v", err)
	}
	if stored.CredentialSecret != "false" {
		t.Errorf("stored credentialSecret = %q, want %q", stored.CredentialSecret, "false")
	}
}

// TestConfigService_CommitStructured_PreservesSecretsWhenOmitted covers the
// slice C apply path, which commits the configuration through a different
// entry point than Save.
func TestConfigService_CommitStructured_PreservesSecretsWhenOmitted(t *testing.T) {
	svc, _ := newSecretService(t)
	if err := svc.Save(secretBearingConfig()); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	candidate := secretBearingConfig()
	candidate.CredentialSecret = ""
	candidate.Https = &model.HttpsConfig{
		Key:  "/tmp/key.pem",
		Cert: "/tmp/cert.pem",
		Port: 1883,
	}
	staged, err := svc.CommitStructured(candidate)
	if err != nil {
		t.Fatalf("commit structured config: %v", err)
	}
	if staged.Committed.CredentialSecret != credentialSecretFixture {
		t.Errorf("committed credentialSecret = %q, want %q", staged.Committed.CredentialSecret, credentialSecretFixture)
	}
	if staged.Committed.Https == nil || staged.Committed.Https.Passphrase != tlsPassphraseFixture {
		t.Errorf("committed https.passphrase = %+v, want %q", staged.Committed.Https, tlsPassphraseFixture)
	}
	if !strings.Contains(staged.Content, tlsPassphraseFixture) {
		t.Errorf("rendered settings.js must keep the stored passphrase; got %s", staged.Content)
	}
}
