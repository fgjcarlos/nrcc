package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestConfigService_Save_Concurrent(t *testing.T) {
	dir := t.TempDir()
	configSvc := NewIsolatedConfigService(dir)
	envSvc := NewEnvService(configSvc)

	initial := configSvc.GetDefault()
	initial.AdminAuth = &model.AdminAuth{
		Type: "credentials",
		Users: []model.AdminAuthUser{{
			Username:    "admin",
			Password:    "$2a$10$preserved-password-hash",
			Permissions: "*",
		}},
	}
	if err := configSvc.Save(initial); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	staleSave := initial
	staleSave.Lang = "en-US"
	staleSave.AdminAuth.Users[0].Password = ""

	const envCount = 12
	errs := runConcurrent(t, envCount+1, func(i int) error {
		if i == envCount {
			return configSvc.Save(staleSave)
		}
		return envSvc.Set(fmt.Sprintf("ATOMIC_%02d", i), fmt.Sprintf("value-%02d", i), "secret", "", true)
	})
	assertNoError(t, errs)

	got, err := configSvc.Get()
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if got.Lang != "en-US" {
		t.Errorf("Lang = %q, want en-US", got.Lang)
	}
	if got.AdminAuth == nil || got.AdminAuth.Users[0].Password != "$2a$10$preserved-password-hash" {
		t.Fatalf("admin password hash was not preserved: %#v", got.AdminAuth)
	}
	if len(got.EnvVars) != envCount {
		t.Fatalf("EnvVars length = %d, want %d: %#v", len(got.EnvVars), envCount, got.EnvVars)
	}
	seen := make(map[string]string, envCount)
	for _, envVar := range got.EnvVars {
		seen[envVar.Key] = envVar.Value
	}
	for i := 0; i < envCount; i++ {
		key := fmt.Sprintf("ATOMIC_%02d", i)
		if seen[key] != fmt.Sprintf("value-%02d", i) {
			t.Errorf("%s = %q, want value-%02d", key, seen[key], i)
		}
	}
}

func TestConfigService_Update_PreservesOtherFields(t *testing.T) {
	svc := NewIsolatedConfigService(t.TempDir())
	base := svc.GetDefault()
	base.AdminAuth = &model.AdminAuth{
		Type:  "credentials",
		Users: []model.AdminAuthUser{{Username: "admin", Password: "hash", Permissions: "*"}},
	}
	base.EnvVars = []model.EnvVar{{Key: "TOKEN", Value: "secret", Type: "secret", Encrypted: true}}
	if err := svc.store.Write(base); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	afterAdmin, err := svc.Update(func(current *model.NodeRedConfig) error {
		current.AdminAuth.Users[0].Permissions = "read"
		return nil
	})
	if err != nil {
		t.Fatalf("update adminAuth: %v", err)
	}
	if len(afterAdmin.EnvVars) != 1 || afterAdmin.EnvVars[0].Value != "secret" {
		t.Fatalf("adminAuth update changed EnvVars: %#v", afterAdmin.EnvVars)
	}

	afterEnv, err := svc.Update(func(current *model.NodeRedConfig) error {
		current.EnvVars = append(current.EnvVars, model.EnvVar{Key: "PORT", Value: "1880", Type: "number"})
		return nil
	})
	if err != nil {
		t.Fatalf("update EnvVars: %v", err)
	}
	if afterEnv.AdminAuth == nil || afterEnv.AdminAuth.Users[0].Permissions != "read" {
		t.Fatalf("EnvVars update changed adminAuth: %#v", afterEnv.AdminAuth)
	}
	if len(afterEnv.EnvVars) != 2 {
		t.Fatalf("EnvVars length = %d, want 2", len(afterEnv.EnvVars))
	}
}

func TestConfigService_Update_NilCallback(t *testing.T) {
	svc := NewIsolatedConfigService(t.TempDir())
	if err := svc.store.Write(svc.GetDefault()); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	committed, err := svc.Update(nil)
	if err == nil {
		t.Fatal("Update(nil) error = nil, want error")
	}
	if committed.Port != 0 || committed.EnvVars != nil || committed.AdminAuth != nil {
		t.Fatalf("Update(nil) snapshot = %#v, want zero value", committed)
	}
}

func TestConfigService_Save_PostCommitFailure(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "settings.js"), 0o755); err != nil {
		t.Fatalf("create settings.js directory: %v", err)
	}

	svc := NewIsolatedConfigService(dir)
	cfg := svc.GetDefault()
	cfg.Lang = "post-commit"
	err := svc.Save(cfg)
	if err == nil {
		t.Fatal("Save() error = nil, want post-commit settings.js error")
	}
	if !strings.Contains(err.Error(), "config JSON committed") {
		t.Fatalf("Save() error = %q, want committed-state context", err)
	}

	committed, readErr := svc.store.Read()
	if readErr != nil {
		t.Fatalf("read committed JSON: %v", readErr)
	}
	if committed.Lang != "post-commit" {
		t.Fatalf("committed Lang = %q, want post-commit", committed.Lang)
	}
}
