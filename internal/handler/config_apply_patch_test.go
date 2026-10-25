package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// applySecurityPatch posts a security-surface patch through the real apply
// API and returns the recorded response.
func applySecurityPatch(t *testing.T, f *applyHandlerFixture, patch map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	patch["expectedRevision"] = f.liveRevision(t).Fingerprint
	body, err := json.Marshal(patch)
	if err != nil {
		t.Fatalf("marshal patch: %v", err)
	}
	req, w := adminRequest(t, http.MethodPost, "/api/config/apply", body)
	f.handler.ApplyStructured(w, req)
	return w
}

// decodedConfiguration decodes the configuration carried by a successful
// apply response.
func decodedConfiguration(t *testing.T, body []byte) model.NodeRedConfig {
	t.Helper()
	var envelope model.ApiResponse[ApplyStructuredResponse]
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return envelope.Data.Configuration
}

// TestReleaseAuditSecuritySurfacePatchMustApply is the regression
// reproduction from issue #879: a security card sends only the surface it
// owns, and the real apply API must accept it.
func TestReleaseAuditSecuritySurfacePatchMustApply(t *testing.T) {
	f := newApplyHandlerFixture(t)
	body, err := json.Marshal(map[string]interface{}{
		"adminAuth": map[string]interface{}{
			"type": "credentials", "users": []map[string]string{{
				"username": "operator", "password": "audit-password-fixture", "permissions": "*",
			}},
		},
		"expectedRevision": f.liveRevision(t).Fingerprint,
	})
	if err != nil {
		t.Fatal(err)
	}
	req, w := adminRequest(t, http.MethodPost, "/api/config/apply", body)
	f.handler.ApplyStructured(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("UI security patch returned HTTP %d: %s", w.Code, w.Body.String())
	}
}

// TestApplyStructuredHandler_SecurityPatchPreservesUnrelatedSettings proves
// the patch keeps every setting the patch does not mention: the port, the
// roots and the credential secret of the running instance survive an
// admin-auth-only apply.
func TestApplyStructuredHandler_SecurityPatchPreservesUnrelatedSettings(t *testing.T) {
	f := newApplyHandlerFixture(t)

	w := applySecurityPatch(t, f, map[string]interface{}{
		"adminAuth": map[string]interface{}{
			"type": "credentials", "users": []map[string]string{{
				"username": "operator", "password": "audit-password-fixture", "permissions": "*",
			}},
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	cfg := decodedConfiguration(t, w.Body.Bytes())
	if cfg.Port == 0 || cfg.UIPort == 0 {
		t.Errorf("port was dropped by the patch: port = %d, uiPort = %d", cfg.Port, cfg.UIPort)
	}
	if cfg.HTTPAdminRoot != "/" || cfg.HTTPNodeRoot != "/" {
		t.Errorf("roots were dropped by the patch: admin = %q, node = %q", cfg.HTTPAdminRoot, cfg.HTTPNodeRoot)
	}
	if cfg.AdminAuth == nil || len(cfg.AdminAuth.Users) != 1 {
		t.Fatalf("adminAuth was not applied: %+v", cfg.AdminAuth)
	}
	if cfg.AdminAuth.Users[0].Username != "operator" {
		t.Errorf("adminAuth user = %q, want %q", cfg.AdminAuth.Users[0].Username, "operator")
	}
	if cfg.AdminAuth.Users[0].Password == "audit-password-fixture" {
		t.Error("the admin password was stored in cleartext")
	}

	// The rendered settings.js is the merged configuration, not the patch.
	content, err := os.ReadFile(f.settingsPath)
	if err != nil {
		t.Fatalf("read settings.js: %v", err)
	}
	rendered := string(content)
	for _, want := range []string{"uiPort: 1880", "httpAdminRoot", "httpNodeRoot", "operator"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered settings.js is missing %q:\n%s", want, rendered)
		}
	}
}

// TestApplyStructuredHandler_SurfacePatchesAreIndependent proves the three
// authentication surfaces can be saved one at a time: applying the HTTP-node
// surface leaves the admin surface alone, and applying the static surface
// leaves the HTTP-node surface alone.
func TestApplyStructuredHandler_SurfacePatchesAreIndependent(t *testing.T) {
	f := newApplyHandlerFixture(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("nodes-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("generate bcrypt hash: %v", err)
	}

	w := applySecurityPatch(t, f, map[string]interface{}{
		"adminAuth": map[string]interface{}{
			"type": "credentials", "users": []map[string]string{{
				"username": "operator", "password": "audit-password-fixture", "permissions": "*",
			}},
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("admin patch status = %d; body = %s", w.Code, w.Body.String())
	}

	w = applySecurityPatch(t, f, map[string]interface{}{
		"httpNodeAuth": map[string]interface{}{"user": "nodes", "pass": string(hash)},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("httpNodeAuth patch status = %d; body = %s", w.Code, w.Body.String())
	}
	cfg := decodedConfiguration(t, w.Body.Bytes())
	if cfg.HTTPNodeAuth == nil || cfg.HTTPNodeAuth.User != "nodes" {
		t.Fatalf("httpNodeAuth was not applied: %+v", cfg.HTTPNodeAuth)
	}
	if cfg.HTTPStaticAuth != nil {
		t.Errorf("httpStaticAuth was changed by the httpNodeAuth patch: %+v", cfg.HTTPStaticAuth)
	}
	if cfg.AdminAuth == nil || len(cfg.AdminAuth.Users) != 1 {
		t.Errorf("adminAuth was changed by the httpNodeAuth patch: %+v", cfg.AdminAuth)
	}

	w = applySecurityPatch(t, f, map[string]interface{}{
		"httpStaticAuth": map[string]interface{}{"user": "static", "pass": string(hash)},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("httpStaticAuth patch status = %d; body = %s", w.Code, w.Body.String())
	}
	cfg = decodedConfiguration(t, w.Body.Bytes())
	if cfg.HTTPStaticAuth == nil || cfg.HTTPStaticAuth.User != "static" {
		t.Fatalf("httpStaticAuth was not applied: %+v", cfg.HTTPStaticAuth)
	}
	if cfg.HTTPNodeAuth == nil || cfg.HTTPNodeAuth.User != "nodes" {
		t.Errorf("httpNodeAuth was changed by the httpStaticAuth patch: %+v", cfg.HTTPNodeAuth)
	}
	if cfg.AdminAuth == nil || len(cfg.AdminAuth.Users) != 1 {
		t.Errorf("adminAuth was changed by the httpStaticAuth patch: %+v", cfg.AdminAuth)
	}
}

// TestApplyStructuredHandler_ExplicitNullDisablesSurface proves an explicit
// null is an opt-out and not an absent field: the admin surface is disabled
// while the unrelated settings survive.
func TestApplyStructuredHandler_ExplicitNullDisablesSurface(t *testing.T) {
	f := newApplyHandlerFixture(t)

	w := applySecurityPatch(t, f, map[string]interface{}{
		"adminAuth": map[string]interface{}{
			"type": "credentials", "users": []map[string]string{{
				"username": "operator", "password": "audit-password-fixture", "permissions": "*",
			}},
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("admin patch status = %d; body = %s", w.Code, w.Body.String())
	}

	w = applySecurityPatch(t, f, map[string]interface{}{"adminAuth": nil})
	if w.Code != http.StatusOK {
		t.Fatalf("disable patch status = %d; body = %s", w.Code, w.Body.String())
	}
	cfg := decodedConfiguration(t, w.Body.Bytes())
	if cfg.AdminAuth != nil {
		t.Errorf("adminAuth was not disabled: %+v", cfg.AdminAuth)
	}
	if cfg.Port == 0 || cfg.HTTPAdminRoot != "/" {
		t.Errorf("the disable patch dropped unrelated settings: %+v", cfg)
	}
}

// TestApplyStructuredHandler_PatchKeepsUnchangedSecrets proves the two
// "unchanged" contracts of a patch: a user whose password the operator did
// not touch keeps the stored hash, and a redacted write-only secret keeps
// the stored value.
func TestApplyStructuredHandler_PatchKeepsUnchangedSecrets(t *testing.T) {
	f := newApplyHandlerFixture(t)

	w := applySecurityPatch(t, f, map[string]interface{}{
		"adminAuth": map[string]interface{}{
			"type": "credentials", "users": []map[string]string{{
				"username": "operator", "password": "audit-password-fixture", "permissions": "*",
			}},
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("admin patch status = %d; body = %s", w.Code, w.Body.String())
	}
	first := decodedConfiguration(t, w.Body.Bytes())
	if first.AdminAuth == nil || first.AdminAuth.Users[0].Password == "" {
		t.Fatalf("the admin password was not stored: %+v", first.AdminAuth)
	}

	// The operator re-saves the same user without retyping the password and
	// echoes the redacted credential secret back.
	w = applySecurityPatch(t, f, map[string]interface{}{
		"adminAuth": map[string]interface{}{
			"type": "credentials", "users": []map[string]string{{
				"username": "operator", "password": "", "permissions": "read",
			}},
		},
		"credentialSecret": model.RedactedSecret,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("unchanged-secret patch status = %d; body = %s", w.Code, w.Body.String())
	}
	second := decodedConfiguration(t, w.Body.Bytes())
	if second.AdminAuth == nil || len(second.AdminAuth.Users) != 1 {
		t.Fatalf("the admin surface was lost: %+v", second.AdminAuth)
	}
	if second.AdminAuth.Users[0].Password != first.AdminAuth.Users[0].Password {
		t.Error("the unchanged admin password was replaced")
	}
	if second.AdminAuth.Users[0].Permissions != "read" {
		t.Errorf("the changed permission was not applied: %q", second.AdminAuth.Users[0].Permissions)
	}
	if second.CredentialSecret != first.CredentialSecret {
		t.Errorf("the redacted credential secret was not preserved: %q", second.CredentialSecret)
	}
}
