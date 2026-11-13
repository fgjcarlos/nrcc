package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

const (
	nodePlainPass   = "node-plain-pass"
	staticPlainPass = "static-plain-pass"
)

func basicAuthBaseConfig(svc *ConfigService) model.NodeRedConfig {
	cfg := svc.GetDefault()
	cfg.HTTPNodeAuth = &model.HTTPBasicAuth{User: "nodes", Pass: nodePlainPass}
	cfg.HTTPStaticAuth = &model.HTTPBasicAuth{User: "static", Pass: staticPlainPass}
	return cfg
}

// assertBasicAuthHashed checks that each surface stores a bcrypt hash of its
// plain password and that no plain password reaches config.json or settings.js.
func assertBasicAuthHashed(t *testing.T, dir string, stored model.NodeRedConfig, settings string) {
	t.Helper()
	for _, tc := range []struct {
		name  string
		auth  *model.HTTPBasicAuth
		plain string
	}{
		{"httpNodeAuth", stored.HTTPNodeAuth, nodePlainPass},
		{"httpStaticAuth", stored.HTTPStaticAuth, staticPlainPass},
	} {
		if tc.auth == nil {
			t.Fatalf("%s missing after save", tc.name)
		}
		if !isBcryptHash(tc.auth.Pass) || !compareHash(t, tc.auth.Pass, tc.plain) {
			t.Errorf("%s pass = %q; want a bcrypt hash of the plain password", tc.name, tc.auth.Pass)
		}
		if !strings.Contains(settings, `pass: "`+tc.auth.Pass+`"`) {
			t.Errorf("settings.js does not carry the %s hash", tc.name)
		}
	}
	// #nosec G304 -- dir is t.TempDir().
	configJSON, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("read config.json: %v", err)
	}
	for _, plain := range []string{nodePlainPass, staticPlainPass} {
		if strings.Contains(settings, plain) {
			t.Errorf("settings.js leaked the plain password %q", plain)
		}
		if strings.Contains(string(configJSON), plain) {
			t.Errorf("config.json leaked the plain password %q", plain)
		}
	}
}

func TestConfigService_Save_HashesPlainBasicAuthPass(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	if err := svc.Save(basicAuthBaseConfig(svc)); err != nil {
		t.Fatalf("save: %v", err)
	}
	stored, err := svc.Get()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	// #nosec G304 -- dir is t.TempDir().
	settings, err := os.ReadFile(filepath.Join(dir, "settings.js"))
	if err != nil {
		t.Fatalf("read settings.js: %v", err)
	}
	assertBasicAuthHashed(t, dir, stored, string(settings))
}

func TestConfigService_CommitStructured_HashesPlainBasicAuthPass(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	staged, err := svc.CommitStructured(basicAuthBaseConfig(svc))
	if err != nil {
		t.Fatalf("commit structured: %v", err)
	}
	assertBasicAuthHashed(t, dir, staged.Committed, staged.Content)
}

func TestConfigService_CommitStructuredPatch_HashesPlainBasicAuthPass(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	if err := svc.Save(svc.GetDefault()); err != nil {
		t.Fatalf("seed save: %v", err)
	}
	staged, err := svc.CommitStructuredPatch(patchFromJSON(t, `{
		"httpNodeAuth": {"user": "nodes", "pass": "`+nodePlainPass+`"},
		"httpStaticAuth": {"user": "static", "pass": "`+staticPlainPass+`"}
	}`))
	if err != nil {
		t.Fatalf("commit structured patch: %v", err)
	}
	assertBasicAuthHashed(t, dir, staged.Committed, staged.Content)
}

func TestConfigService_Save_KeepsBcryptBasicAuthPass(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	cfg := svc.GetDefault()
	cfg.HTTPNodeAuth = &model.HTTPBasicAuth{User: "nodes", Pass: testBcryptHash}
	cfg.HTTPStaticAuth = &model.HTTPBasicAuth{User: "static", Pass: testBcryptHash}
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	stored, err := svc.Get()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.HTTPNodeAuth.Pass != testBcryptHash || stored.HTTPStaticAuth.Pass != testBcryptHash {
		t.Errorf("bcrypt hashes were rewritten: node=%q static=%q", stored.HTTPNodeAuth.Pass, stored.HTTPStaticAuth.Pass)
	}
}

func TestConfigService_Save_RejectsBasicAuthWithoutUser(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	cfg := svc.GetDefault()
	cfg.HTTPNodeAuth = &model.HTTPBasicAuth{User: "", Pass: nodePlainPass}
	err := svc.Save(cfg)
	if err == nil || !strings.Contains(err.Error(), "httpNodeAuth user must not be empty") {
		t.Fatalf("Save() error = %v; want empty-user rejection", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "config.json")); !os.IsNotExist(statErr) {
		t.Errorf("rejected save wrote config.json (stat err = %v)", statErr)
	}
}
