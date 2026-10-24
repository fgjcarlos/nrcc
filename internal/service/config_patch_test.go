package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// patchFromJSON builds a presence-aware patch from a JSON object literal.
func patchFromJSON(t *testing.T, body string) model.ConfigPatch {
	t.Helper()
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(body), &fields); err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	return model.NewConfigPatch(fields)
}

// liveConfiguration is the configuration a patch merges onto in the merge
// table: a running instance with every authentication surface configured.
func liveConfiguration() model.NodeRedConfig {
	return model.NodeRedConfig{
		Port:          1880,
		UIPort:        1880,
		HTTPAdminRoot: "/",
		HTTPNodeRoot:  "/",
		AdminAuth: &model.AdminAuth{
			Type:              "credentials",
			SessionExpiryTime: 14400,
			Users: []model.AdminAuthUser{{
				Username:    "operator",
				Password:    testBcryptHash,
				Permissions: "*",
			}},
		},
		HTTPNodeAuth:   &model.HTTPBasicAuth{User: "nodes", Pass: testBcryptHash},
		HTTPStaticAuth: &model.HTTPBasicAuth{User: "static", Pass: testBcryptHash},
	}
}

// TestMergePatch pins the three operator intents of a patch: an absent
// field keeps the live value, an explicit null disables the surface, and a
// present value replaces the live value.
func TestMergePatch(t *testing.T) {
	for _, tc := range []struct {
		name   string
		patch  string
		verify func(t *testing.T, merged model.NodeRedConfig)
	}{
		{
			name:  "absent field keeps the live value",
			patch: `{"httpStaticAuth": {"user": "static-new", "pass": "` + testBcryptHash + `"}}`,
			verify: func(t *testing.T, merged model.NodeRedConfig) {
				t.Helper()
				if merged.Port != 1880 || merged.HTTPAdminRoot != "/" || merged.HTTPNodeRoot != "/" {
					t.Errorf("the operational settings were dropped: %+v", merged)
				}
				if merged.AdminAuth == nil || merged.AdminAuth.Users[0].Username != "operator" {
					t.Errorf("adminAuth was dropped: %+v", merged.AdminAuth)
				}
				if merged.HTTPNodeAuth == nil || merged.HTTPNodeAuth.User != "nodes" {
					t.Errorf("httpNodeAuth was dropped: %+v", merged.HTTPNodeAuth)
				}
				if merged.HTTPStaticAuth == nil || merged.HTTPStaticAuth.User != "static-new" {
					t.Errorf("httpStaticAuth was not applied: %+v", merged.HTTPStaticAuth)
				}
			},
		},
		{
			name:  "explicit null disables the surface",
			patch: `{"adminAuth": null}`,
			verify: func(t *testing.T, merged model.NodeRedConfig) {
				t.Helper()
				if merged.AdminAuth != nil {
					t.Errorf("adminAuth was not disabled: %+v", merged.AdminAuth)
				}
				if merged.HTTPNodeAuth == nil || merged.HTTPStaticAuth == nil {
					t.Errorf("the other surfaces were disabled: %+v", merged)
				}
				if merged.Port != 1880 {
					t.Errorf("the operational settings were dropped: %+v", merged)
				}
			},
		},
		{
			name:  "nested object merges key by key",
			patch: `{"adminAuth": {"users": [{"username": "second", "password": "", "permissions": "read"}]}}`,
			verify: func(t *testing.T, merged model.NodeRedConfig) {
				t.Helper()
				if merged.AdminAuth == nil {
					t.Fatal("adminAuth was dropped")
				}
				if merged.AdminAuth.Type != "credentials" {
					t.Errorf("adminAuth type = %q, want %q", merged.AdminAuth.Type, "credentials")
				}
				if merged.AdminAuth.SessionExpiryTime != 14400 {
					t.Errorf("adminAuth sessionExpiryTime = %d, want 14400", merged.AdminAuth.SessionExpiryTime)
				}
				if len(merged.AdminAuth.Users) != 1 || merged.AdminAuth.Users[0].Username != "second" {
					t.Errorf("the users array was not replaced: %+v", merged.AdminAuth.Users)
				}
			},
		},
		{
			name:  "explicit zero replaces the live value",
			patch: `{"adminAuth": {"sessionExpiryTime": 0}}`,
			verify: func(t *testing.T, merged model.NodeRedConfig) {
				t.Helper()
				if merged.AdminAuth == nil || merged.AdminAuth.SessionExpiryTime != 0 {
					t.Errorf("the explicit zero was not applied: %+v", merged.AdminAuth)
				}
				if merged.AdminAuth.Type != "credentials" || len(merged.AdminAuth.Users) != 1 {
					t.Errorf("the untouched keys were not preserved: %+v", merged.AdminAuth)
				}
			},
		},
		{
			name:  "the patched uiPort moves the derived port",
			patch: `{"uiPort": 7777}`,
			verify: func(t *testing.T, merged model.NodeRedConfig) {
				t.Helper()
				if merged.UIPort != 7777 {
					t.Errorf("uiPort = %d, want 7777", merged.UIPort)
				}
				if merged.Port != 7777 {
					t.Errorf("port = %d, want 7777 (the port follows the uiPort)", merged.Port)
				}
			},
		},
		{
			name:  "an explicit port stays independent",
			patch: `{"port": 1881, "uiPort": 7777}`,
			verify: func(t *testing.T, merged model.NodeRedConfig) {
				t.Helper()
				if merged.Port != 1881 || merged.UIPort != 7777 {
					t.Errorf("the explicit port was not kept: %+v", merged)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			merged, err := MergePatch(liveConfiguration(), patchFromJSON(t, tc.patch))
			if err != nil {
				t.Fatalf("merge patch: %v", err)
			}
			tc.verify(t, merged)
		})
	}
}

// TestCommitStructuredPatch_MergesOntoPersistedState proves the patch is
// merged onto the configuration NRCC has persisted, not onto the payload.
func TestCommitStructuredPatch_MergesOntoPersistedState(t *testing.T) {
	svc, _ := newSecretService(t)
	if err := svc.Save(secretBearingConfig()); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	state, err := svc.CommitStructuredPatch(patchFromJSON(t, `{"adminAuth": {"type": "credentials", "users": [{"username": "operator", "password": "audit-password-fixture", "permissions": "*"}]}}`))
	if err != nil {
		t.Fatalf("commit patch: %v", err)
	}
	if state.Committed.Port != 1880 || state.Committed.HTTPAdminRoot != "/" {
		t.Errorf("the operational settings were dropped: %+v", state.Committed)
	}
	if state.Committed.AdminAuth == nil || len(state.Committed.AdminAuth.Users) != 1 {
		t.Fatalf("adminAuth was not applied: %+v", state.Committed.AdminAuth)
	}
	if state.Committed.AdminAuth.Users[0].Password == "audit-password-fixture" {
		t.Error("the admin password was stored in cleartext")
	}
	if state.Committed.CredentialSecret != credentialSecretFixture {
		t.Errorf("the credential secret was dropped: %q", state.Committed.CredentialSecret)
	}
}

// TestCommitStructuredPatch_MergesOntoLiveSettingsWhenStoreIsEmpty proves a
// deployment whose store does not exist yet still keeps the port and the
// roots the running Node-RED uses.
func TestCommitStructuredPatch_MergesOntoLiveSettingsWhenStoreIsEmpty(t *testing.T) {
	svc, _ := newSecretService(t)

	state, err := svc.CommitStructuredPatch(patchFromJSON(t, `{"httpNodeAuth": {"user": "nodes", "pass": "`+testBcryptHash+`"}}`))
	if err != nil {
		t.Fatalf("commit patch: %v", err)
	}
	if state.Committed.Port != 1880 {
		t.Errorf("the live port was dropped: %d", state.Committed.Port)
	}
	if state.Committed.HTTPAdminRoot != "/" || state.Committed.HTTPNodeRoot != "/" {
		t.Errorf("the live roots were dropped: %+v", state.Committed)
	}
	if state.Committed.HTTPNodeAuth == nil || state.Committed.HTTPNodeAuth.User != "nodes" {
		t.Errorf("httpNodeAuth was not applied: %+v", state.Committed.HTTPNodeAuth)
	}
}

// TestCommitStructuredPatch_KeepsUnchangedSecrets proves the two "unchanged"
// contracts of a patch survive the merge: a user whose password the operator
// did not retype keeps the stored hash, and a redacted write-only secret
// keeps the stored value.
func TestCommitStructuredPatch_KeepsUnchangedSecrets(t *testing.T) {
	svc, _ := newSecretService(t)
	seed := secretBearingConfig()
	seed.AdminAuth = &model.AdminAuth{
		Type: "credentials",
		Users: []model.AdminAuthUser{{
			Username:    "operator",
			Password:    testBcryptHash,
			Permissions: "*",
		}},
	}
	if err := svc.Save(seed); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	seeded, err := svc.Get()
	if err != nil {
		t.Fatalf("read seeded config: %v", err)
	}
	if seeded.AdminAuth == nil || len(seeded.AdminAuth.Users) == 0 {
		t.Fatalf("the seeded config has no admin user: %+v", seeded.AdminAuth)
	}

	state, err := svc.CommitStructuredPatch(patchFromJSON(t, `{
		"adminAuth": {"type": "credentials", "users": [{"username": "operator", "password": "", "permissions": "read"}]},
		"credentialSecret": "[redacted]"
	}`))
	if err != nil {
		t.Fatalf("commit patch: %v", err)
	}
	if state.Committed.AdminAuth == nil || len(state.Committed.AdminAuth.Users) != 1 {
		t.Fatalf("the admin surface was lost: %+v", state.Committed.AdminAuth)
	}
	if state.Committed.AdminAuth.Users[0].Password != seeded.AdminAuth.Users[0].Password {
		t.Error("the unchanged admin password was replaced")
	}
	if state.Committed.AdminAuth.Users[0].Permissions != "read" {
		t.Errorf("the changed permission was not applied: %q", state.Committed.AdminAuth.Users[0].Permissions)
	}
	if state.Committed.CredentialSecret != credentialSecretFixture {
		t.Errorf("the redacted credential secret was not preserved: %q", state.Committed.CredentialSecret)
	}
}

// TestCommitStructuredPatch_RendersTheMergedSettings proves the settings.js
// the apply writes is the merged configuration and not the patch payload.
func TestCommitStructuredPatch_RendersTheMergedSettings(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(secretBearingSettingsJS()), 0o600); err != nil {
		t.Fatalf("seed settings.js: %v", err)
	}
	svc := NewIsolatedConfigService(dir)

	state, err := svc.CommitStructuredPatch(patchFromJSON(t, `{"httpNodeAuth": {"user": "nodes", "pass": "`+testBcryptHash+`"}}`))
	if err != nil {
		t.Fatalf("commit patch: %v", err)
	}
	if got := state.Content; !containsAll(got, "uiPort: 1880", "httpNodeAuth", "nodes") {
		t.Errorf("the rendered settings.js does not carry the merged configuration:\n%s", got)
	}
}

// containsAll reports whether haystack carries every needle.
func containsAll(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			return false
		}
	}
	return true
}
