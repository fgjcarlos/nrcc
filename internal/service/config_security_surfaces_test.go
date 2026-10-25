package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// decodeConfigPayload decodes a JSON payload the way the HTTP handler does,
// so the security-surface presence rules (issue #881) are exercised the same
// way production does.
func decodeConfigPayload(t *testing.T, body string) model.NodeRedConfig {
	t.Helper()
	var cfg model.NodeRedConfig
	if err := json.Unmarshal([]byte(body), &cfg); err != nil {
		t.Fatalf("decode config payload: %v", err)
	}
	return cfg
}

// seedAdminAuth stores a two-account adminAuth model the way the security
// boundary cards do: one read-only account and one full-access account.
func seedAdminAuth(t *testing.T, svc *ConfigService) (string, string) {
	t.Helper()
	observerHash := mustBcryptHash(t, "observer-pass")
	adminHash := mustBcryptHash(t, "admin-pass")

	seed := model.NodeRedConfig{
		Port:          1880,
		UIPort:        1880,
		HTTPAdminRoot: "/",
		HTTPNodeRoot:  "/",
		FlowFile:      "flows.json",
		AdminAuth: &model.AdminAuth{
			Type: "credentials",
			Users: []model.AdminAuthUser{
				{Username: "observer", Password: observerHash, Permissions: "read"},
				{Username: "administrator", Password: adminHash, Permissions: "*"},
			},
			SessionExpiryTime: 3600,
		},
	}
	if err := svc.Save(seed); err != nil {
		t.Fatalf("seed save: %v", err)
	}
	return observerHash, adminHash
}

// TestConfigService_Save_PreservesAdminAuthWhenPayloadOmitsIt is the issue
// #881 regression test: an unrelated visual edit must not touch Node-RED's
// authentication model.
func TestConfigService_Save_PreservesAdminAuthWhenPayloadOmitsIt(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	observerHash, adminHash := seedAdminAuth(t, svc)

	payload := decodeConfigPayload(t, `{
		"uiPort": 1880,
		"uiHost": "0.0.0.0",
		"httpAdminRoot": "/",
		"httpNodeRoot": "/",
		"projectsEnabled": false,
		"flowFile": "flows.json",
		"lang": "en-US",
		"editorTheme": {"page": {"title": "New title"}}
	}`)
	if err := svc.Save(payload); err != nil {
		t.Fatalf("visual edit save: %v", err)
	}

	stored, err := svc.Get()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.AdminAuth == nil {
		t.Fatal("adminAuth must survive a payload that omitted it")
	}
	if got := len(stored.AdminAuth.Users); got != 2 {
		t.Fatalf("adminAuth users = %d, want 2 (every account must survive)", got)
	}

	users := map[string]model.AdminAuthUser{}
	for _, user := range stored.AdminAuth.Users {
		users[user.Username] = user
	}
	observer, ok := users["observer"]
	if !ok {
		t.Fatal("the read-only account must survive an unrelated visual edit")
	}
	if observer.Permissions != "read" {
		t.Errorf("observer permissions = %q, want %q", observer.Permissions, "read")
	}
	if observer.Password != observerHash {
		t.Error("the observer password hash must be preserved")
	}
	administrator, ok := users["administrator"]
	if !ok {
		t.Fatal("the full-access account must survive an unrelated visual edit")
	}
	if administrator.Permissions != "*" {
		t.Errorf("administrator permissions = %q, want %q", administrator.Permissions, "*")
	}
	if administrator.Password != adminHash {
		t.Error("the administrator password hash must be preserved")
	}
	if stored.AdminAuth.SessionExpiryTime != 3600 {
		t.Errorf("sessionExpiryTime = %d, want 3600", stored.AdminAuth.SessionExpiryTime)
	}

	// The rendered settings.js is what Node-RED actually loads.
	doc, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("raw settings: %v", err)
	}
	for _, want := range []string{"observer", "administrator", `"read"`, `"*"`, "sessionExpiryTime"} {
		if !strings.Contains(doc.Content, want) {
			t.Errorf("settings.js must keep %q after an unrelated visual edit:\n%s", want, doc.Content)
		}
	}
}

// TestConfigService_Save_PreservesAdminAuthFromLiveSettings covers the
// case where the operator wrote adminAuth by hand before nrcc's own store
// existed: the live settings.js is the fallback source.
func TestConfigService_Save_PreservesAdminAuthFromLiveSettings(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)

	hash := mustBcryptHash(t, "handwritten-pass")
	settings := `module.exports = {
  adminAuth: {
    type: "credentials",
    users: [
      { username: "handwritten", password: ` + strconv.Quote(hash) + `, permissions: "read" }
    ],
    sessionExpiryTime: 900
  }
};
`
	if err := os.WriteFile(filepath.Join(dir, "settings.js"), []byte(settings), 0600); err != nil {
		t.Fatalf("write settings.js: %v", err)
	}

	payload := decodeConfigPayload(t, `{"uiPort":1880,"httpAdminRoot":"/","httpNodeRoot":"/","flowFile":"flows.json","projectsEnabled":false}`)
	if err := svc.Save(payload); err != nil {
		t.Fatalf("save: %v", err)
	}

	stored, err := svc.Get()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.AdminAuth == nil || len(stored.AdminAuth.Users) != 1 {
		t.Fatalf("adminAuth must be seeded from the live settings.js, got %+v", stored.AdminAuth)
	}
	if stored.AdminAuth.Users[0].Username != "handwritten" {
		t.Errorf("adminAuth user = %q, want %q", stored.AdminAuth.Users[0].Username, "handwritten")
	}
	if stored.AdminAuth.Users[0].Permissions != "read" {
		t.Errorf("adminAuth permissions = %q, want %q", stored.AdminAuth.Users[0].Permissions, "read")
	}
	if stored.AdminAuth.SessionExpiryTime != 900 {
		t.Errorf("sessionExpiryTime = %d, want 900", stored.AdminAuth.SessionExpiryTime)
	}
}

// TestConfigService_Save_ClearsAdminAuthWhenPayloadCarriesNull proves the
// rule is "omitted means keep", not "always keep": an explicit null still
// clears the surface.
func TestConfigService_Save_ClearsAdminAuthWhenPayloadCarriesNull(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	seedAdminAuth(t, svc)

	payload := decodeConfigPayload(t, `{"uiPort":1880,"httpAdminRoot":"/","httpNodeRoot":"/","flowFile":"flows.json","projectsEnabled":false,"adminAuth":null}`)
	if err := svc.Save(payload); err != nil {
		t.Fatalf("save: %v", err)
	}

	stored, err := svc.Get()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.AdminAuth != nil {
		t.Errorf("adminAuth must be cleared when the payload carries null, got %+v", stored.AdminAuth)
	}
}

// TestConfigService_Save_KeepsCarriedAdminAuthPasswords covers the security
// cards' flow: the payload carries the complete account list and an empty
// password means "unchanged".
func TestConfigService_Save_KeepsCarriedAdminAuthPasswords(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	observerHash, adminHash := seedAdminAuth(t, svc)

	payload := decodeConfigPayload(t, `{
		"uiPort": 1880,
		"httpAdminRoot": "/",
		"httpNodeRoot": "/",
		"flowFile": "flows.json",
		"projectsEnabled": false,
		"adminAuth": {
			"type": "credentials",
			"users": [
				{"username": "observer", "password": "", "permissions": "read"},
				{"username": "administrator", "password": "", "permissions": "*"}
			],
			"sessionExpiryTime": 3600
		}
	}`)
	if err := svc.Save(payload); err != nil {
		t.Fatalf("save: %v", err)
	}

	stored, err := svc.Get()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.AdminAuth == nil || len(stored.AdminAuth.Users) != 2 {
		t.Fatalf("adminAuth must keep both accounts, got %+v", stored.AdminAuth)
	}
	users := map[string]model.AdminAuthUser{}
	for _, user := range stored.AdminAuth.Users {
		users[user.Username] = user
	}
	if users["observer"].Password != observerHash {
		t.Error("an empty password must keep the stored hash")
	}
	if users["administrator"].Password != adminHash {
		t.Error("an empty password must keep the stored hash")
	}
	if users["observer"].Permissions != "read" {
		t.Errorf("observer permissions = %q, want %q", users["observer"].Permissions, "read")
	}
}

// TestConfigService_Save_PreservesBasicAuthSurfaces proves the node-http and
// static basic-auth surfaces survive a payload that omits them, and that an
// explicit null still clears them.
func TestConfigService_Save_PreservesBasicAuthSurfaces(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)

	nodeHash := mustBcryptHash(t, "node-pass")
	staticHash := mustBcryptHash(t, "static-pass")
	seed := model.NodeRedConfig{
		Port:          1880,
		UIPort:        1880,
		HTTPAdminRoot: "/",
		HTTPNodeRoot:  "/",
		FlowFile:      "flows.json",
		HTTPNodeAuth:  &model.HTTPBasicAuth{User: "node-user", Pass: nodeHash},
		HTTPStaticAuth: &model.HTTPBasicAuth{
			User: "static-user",
			Pass: staticHash,
		},
	}
	if err := svc.Save(seed); err != nil {
		t.Fatalf("seed save: %v", err)
	}

	payload := decodeConfigPayload(t, `{"uiPort":1880,"httpAdminRoot":"/","httpNodeRoot":"/","flowFile":"flows.json","projectsEnabled":false}`)
	if err := svc.Save(payload); err != nil {
		t.Fatalf("save: %v", err)
	}

	stored, err := svc.Get()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.HTTPNodeAuth == nil || stored.HTTPNodeAuth.User != "node-user" {
		t.Errorf("httpNodeAuth must survive a payload that omitted it, got %+v", stored.HTTPNodeAuth)
	}
	if stored.HTTPStaticAuth == nil || stored.HTTPStaticAuth.User != "static-user" {
		t.Errorf("httpStaticAuth must survive a payload that omitted it, got %+v", stored.HTTPStaticAuth)
	}

	clearing := decodeConfigPayload(t, `{"uiPort":1880,"httpAdminRoot":"/","httpNodeRoot":"/","flowFile":"flows.json","projectsEnabled":false,"httpNodeAuth":null}`)
	if err := svc.Save(clearing); err != nil {
		t.Fatalf("clear save: %v", err)
	}
	stored, err = svc.Get()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.HTTPNodeAuth != nil {
		t.Errorf("httpNodeAuth must be cleared when the payload carries null, got %+v", stored.HTTPNodeAuth)
	}
	if stored.HTTPStaticAuth == nil {
		t.Error("httpStaticAuth must survive a payload that only cleared httpNodeAuth")
	}
}

// TestConfigService_Save_KeepsProgrammaticAdminAuth proves that a config
// built in Go (not decoded from JSON) stays authoritative: the presence
// rules only apply to decoded payloads.
func TestConfigService_Save_KeepsProgrammaticAdminAuth(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	seedAdminAuth(t, svc)

	cfg := model.NodeRedConfig{
		Port:          1880,
		UIPort:        1880,
		HTTPAdminRoot: "/",
		HTTPNodeRoot:  "/",
		FlowFile:      "flows.json",
		AdminAuth: &model.AdminAuth{
			Type: "credentials",
			Users: []model.AdminAuthUser{{
				Username:    "replacement",
				Password:    mustBcryptHash(t, "replacement-pass"),
				Permissions: "*",
			}},
		},
	}
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	stored, err := svc.Get()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.AdminAuth == nil || len(stored.AdminAuth.Users) != 1 {
		t.Fatalf("a programmatic config must stay authoritative, got %+v", stored.AdminAuth)
	}
	if stored.AdminAuth.Users[0].Username != "replacement" {
		t.Errorf("adminAuth user = %q, want %q", stored.AdminAuth.Users[0].Username, "replacement")
	}
}
