package service

import (
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

const testBcryptHash = "$2b$08$wuAqPiKJlVN27eF5qJp.RuQYuy6ZYONW7a/UWYxDTtwKFCdB8F19y"

func TestRenderAdminAuthMultipleUsers(t *testing.T) {
	cfg := authSurfaceConfig()
	rendered := renderSettingsJS(cfg)

	for _, want := range []string{
		"adminAuth:", `username: "admin"`, `permissions: "*"`,
		`username: "reader"`, `permissions: "read"`, "sessionExpiryTime: 86400",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered settings missing %q", want)
		}
	}
	parsed, err := (&ConfigService{}).parseConfigFromContent(rendered)
	if err != nil {
		t.Fatalf("parse rendered settings: %v", err)
	}
	if parsed.AdminAuth == nil || len(parsed.AdminAuth.Users) != 2 {
		t.Fatalf("adminAuth users = %#v; want two users", parsed.AdminAuth)
	}
	if parsed.AdminAuth.SessionExpiryTime != 86400 {
		t.Errorf("sessionExpiryTime = %d; want 86400", parsed.AdminAuth.SessionExpiryTime)
	}
	if parsed.AdminAuth.Users[1].Permissions != "read" {
		t.Errorf("second user permissions = %q; want read", parsed.AdminAuth.Users[1].Permissions)
	}
}

func TestRenderHTTPAuthCanonicalKeys(t *testing.T) {
	rendered := renderSettingsJS(authSurfaceConfig())
	for _, want := range []string{"httpNodeAuth: { user:", "httpStaticAuth: { user:", `pass: "` + testBcryptHash + `"`} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered settings missing %q", want)
		}
	}
	for _, forbidden := range []string{"nodeHttpAuth", "staticAuth"} {
		if strings.Contains(rendered, forbidden) {
			t.Errorf("rendered settings contain non-canonical HTTP auth field %q", forbidden)
		}
	}
	surfaces, err := ParseAuthenticationSurfacesViaSandbox(rendered)
	if err != nil {
		t.Fatalf("parse authentication surfaces: %v", err)
	}
	if surfaces.HTTPNodeAuth == nil || surfaces.HTTPStaticAuth == nil {
		t.Fatalf("sandbox did not parse HTTP auth surfaces: %#v", surfaces)
	}
	parsed, err := (&ConfigService{}).parseConfigFromContent(rendered)
	if err != nil {
		t.Fatalf("parse rendered settings: %v", err)
	}
	if parsed.HTTPNodeAuth == nil || parsed.HTTPStaticAuth == nil {
		t.Fatalf("HTTP auth surfaces not parsed: %#v", parsed)
	}
}

func TestAuthSurfaceRoundTripPreservesUnmanagedSource(t *testing.T) {
	source := `module.exports = {
  adminAuth: { type: "strategy", strategy: function() { return "unchanged"; } },
  httpNodeAuth: { user: "nodes", pass: "` + testBcryptHash + `" },
  httpMiddleware: function(req, res, next) { next(); }
}`
	parsed, err := (&ConfigService{}).parseConfigFromContent(source)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}
	patched := patchSettingsJS(source, parsed)
	for _, want := range []string{"type: \"strategy\"", "return \"unchanged\"", "httpMiddleware"} {
		if !strings.Contains(patched, want) {
			t.Errorf("patched source lost unmanaged content %q", want)
		}
	}
}

func TestHTTPAuthValidationRedactsPasswords(t *testing.T) {
	secret := "not-a-bcrypt-password"
	err := (&ConfigService{}).Validate(model.NodeRedConfig{
		Port: 1880, HTTPAdminRoot: "/", HTTPNodeRoot: "/",
		HTTPNodeAuth: &model.HTTPBasicAuth{User: "nodes", Pass: secret},
	})
	if err == nil {
		t.Fatal("expected invalid bcrypt password to be rejected")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("validation error leaked password: %v", err)
	}
}

func authSurfaceConfig() model.NodeRedConfig {
	return model.NodeRedConfig{
		Port: 1880, HTTPAdminRoot: "/", HTTPNodeRoot: "/",
		AdminAuth: &model.AdminAuth{
			Type: "credentials", SessionExpiryTime: 86400,
			Users: []model.AdminAuthUser{
				{Username: "admin", Password: testBcryptHash, Permissions: "*"},
				{Username: "reader", Password: testBcryptHash, Permissions: "read"},
			},
		},
		HTTPNodeAuth:   &model.HTTPBasicAuth{User: "nodes", Pass: testBcryptHash},
		HTTPStaticAuth: &model.HTTPBasicAuth{User: "static", Pass: testBcryptHash},
	}
}
