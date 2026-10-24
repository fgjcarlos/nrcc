package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// The fixtures below are invented test values, never real credentials.
const (
	credentialSecretFixture = "audit-credential-fixture" //nolint:gosec // G101 test fixture, not a credential.
	tlsPassphraseFixture    = "audit-tls-fixture"        //nolint:gosec // G101 test fixture, not a credential.
)

// seedSecretBearingConfig persists a configuration that carries both
// write-only secrets so the redaction tests have a deterministic fixture.
func seedSecretBearingConfig(t *testing.T, svc *service.ConfigService) {
	t.Helper()
	cfg, err := svc.Get()
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	cfg.CredentialSecret = credentialSecretFixture
	cfg.Https = &model.HttpsConfig{
		Key:        "/tmp/key.pem",
		Cert:       "/tmp/cert.pem",
		Port:       1883,
		Passphrase: tlsPassphraseFixture,
	}
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}
}

// requestWithRole builds a request whose context carries the given role.
func requestWithRole(t *testing.T, method, target string, body []byte, role model.UserRole) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, target, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	ctx := context.WithValue(r.Context(), middleware.CtxKeyUser, &model.Claims{
		Username: string(role),
		Role:     role,
	})
	return r.WithContext(ctx), httptest.NewRecorder()
}

// TestConfigHandler_GetConfig_AdminMustNotReceiveTLSOrCredentialSecrets
// covers the admin half of the write-only contract: the configuration
// secrets are redacted for every role, not only for viewers.
func TestConfigHandler_GetConfig_AdminMustNotReceiveTLSOrCredentialSecrets(t *testing.T) {
	f := newApplyHandlerFixture(t)
	seedSecretBearingConfig(t, f.svc)

	req, w := requestWithRole(t, http.MethodGet, "/api/config", nil, model.RoleAdmin)
	NewConfigHandler(f.svc).GetConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	for _, secret := range []string{credentialSecretFixture, tlsPassphraseFixture} {
		if strings.Contains(w.Body.String(), secret) {
			t.Errorf("admin response includes %s", secret)
		}
	}

	var resp model.ApiResponse[model.NodeRedConfig]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response must be valid JSON: %v", err)
	}
	if resp.Data.CredentialSecret != "" {
		t.Errorf("credentialSecret must be redacted; got %q", resp.Data.CredentialSecret)
	}
	if resp.Data.Https == nil {
		t.Fatal("https block must still be returned")
	}
	if resp.Data.Https.Passphrase != "" {
		t.Errorf("https.passphrase must be redacted; got %q", resp.Data.Https.Passphrase)
	}
	// The on-disk PEM paths are not secrets: the configuration UI needs
	// them to render the TLS section.
	if resp.Data.Https.Key != "/tmp/key.pem" || resp.Data.Https.Cert != "/tmp/cert.pem" {
		t.Errorf("non-secret TLS material must be preserved; got %+v", resp.Data.Https)
	}
	if resp.Data.Https.Port != 1883 {
		t.Errorf("https.port must be preserved; got %d", resp.Data.Https.Port)
	}
}

// TestConfigHandler_GetConfig_RedactionKeepsPersistedStateIntact asserts the
// redaction is display-only: neither the stored configuration nor the
// settings.js file changes when a redacted read happens.
func TestConfigHandler_GetConfig_RedactionKeepsPersistedStateIntact(t *testing.T) {
	f := newApplyHandlerFixture(t)
	seedSecretBearingConfig(t, f.svc)

	before, err := f.svc.GetRawSettings()
	if err != nil {
		t.Fatalf("read settings.js: %v", err)
	}

	req, w := requestWithRole(t, http.MethodGet, "/api/config", nil, model.RoleViewer)
	NewConfigHandler(f.svc).GetConfig(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	stored, err := f.svc.Get()
	if err != nil {
		t.Fatalf("read stored config: %v", err)
	}
	if stored.CredentialSecret != credentialSecretFixture {
		t.Errorf("stored credentialSecret = %q, want %q", stored.CredentialSecret, credentialSecretFixture)
	}
	if stored.Https == nil || stored.Https.Passphrase != tlsPassphraseFixture {
		t.Errorf("stored https.passphrase must be preserved; got %+v", stored.Https)
	}

	after, err := f.svc.GetRawSettings()
	if err != nil {
		t.Fatalf("read settings.js: %v", err)
	}
	if after.Content != before.Content {
		t.Errorf("a redacted read must not modify settings.js:\nbefore: %s\nafter: %s", before.Content, after.Content)
	}
}

// TestConfigHandler_SaveConfig_RedactedSecretsKeepStoredValues covers the
// write-back contract: a client that echoes a redacted read back must not
// wipe the stored secrets.
func TestConfigHandler_SaveConfig_RedactedSecretsKeepStoredValues(t *testing.T) {
	f := newApplyHandlerFixture(t)
	seedSecretBearingConfig(t, f.svc)

	payload := model.NodeRedConfig{
		Port:          1880,
		UIPort:        1880,
		HTTPAdminRoot: "/",
		HTTPNodeRoot:  "/",
		// Write-only fields omitted: the client never received them.
		Https: &model.HttpsConfig{
			Key:  "/tmp/key.pem",
			Cert: "/tmp/cert.pem",
			Port: 1883,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req, w := requestWithRole(t, http.MethodPost, "/api/config", body, model.RoleAdmin)
	NewConfigHandler(f.svc).SaveConfig(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	for _, secret := range []string{credentialSecretFixture, tlsPassphraseFixture} {
		if strings.Contains(w.Body.String(), secret) {
			t.Errorf("save response includes %s", secret)
		}
	}

	stored, err := f.svc.Get()
	if err != nil {
		t.Fatalf("read stored config: %v", err)
	}
	if stored.CredentialSecret != credentialSecretFixture {
		t.Errorf("stored credentialSecret = %q, want it preserved", stored.CredentialSecret)
	}
	if stored.Https == nil || stored.Https.Passphrase != tlsPassphraseFixture {
		t.Errorf("stored https.passphrase must be preserved; got %+v", stored.Https)
	}

	doc, err := f.svc.GetRawSettings()
	if err != nil {
		t.Fatalf("read settings.js: %v", err)
	}
	if !strings.Contains(doc.Content, credentialSecretFixture) {
		t.Errorf("settings.js must keep the stored credential secret; got %s", doc.Content)
	}
}

// TestConfigHandler_SaveConfig_ExplicitSecretRotatesStoredValue proves the
// write-only contract does not block a deliberate rotation.
func TestConfigHandler_SaveConfig_ExplicitSecretRotatesStoredValue(t *testing.T) {
	f := newApplyHandlerFixture(t)
	seedSecretBearingConfig(t, f.svc)

	//nolint:gosec // G101 test fixture, not a credential.
	payload := model.NodeRedConfig{
		Port:             1880,
		UIPort:           1880,
		HTTPAdminRoot:    "/",
		HTTPNodeRoot:     "/",
		CredentialSecret: "rotated-credential-fixture",
		Https: &model.HttpsConfig{
			Key:        "/tmp/key.pem",
			Cert:       "/tmp/cert.pem",
			Port:       1883,
			Passphrase: "rotated-tls-fixture",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req, w := requestWithRole(t, http.MethodPost, "/api/config", body, model.RoleAdmin)
	NewConfigHandler(f.svc).SaveConfig(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	for _, secret := range []string{"rotated-credential-fixture", "rotated-tls-fixture"} {
		if strings.Contains(w.Body.String(), secret) {
			t.Errorf("save response includes %s", secret)
		}
	}

	stored, err := f.svc.Get()
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
