package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func TestConfigService_Save_Concurrent(t *testing.T) {
	// Atomicity contract: concurrent Save calls must not tear. Each goroutine
	// performs an independent Save with a distinct Lang value, and after all
	// finish we verify the committed state is exactly one of the proposed
	// values (no partial merge of two writes).
	//
	// Note: this test deliberately does NOT mix env-var writes here because
	// Save honors caller's EnvVars authoritatively — a Save with empty
	// EnvVars would clobber concurrent env-set writes, which is correct
	// behavior but conflates two concerns. Env-var concurrency has its own
	// test (TestEnvService_Set_Concurrent).
	dir := t.TempDir()
	configSvc := NewIsolatedConfigService(dir)
	hashBytes, err := bcrypt.GenerateFromPassword([]byte("preserved-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("generate bcrypt hash: %v", err)
	}
	hash := string(hashBytes)

	seed := configSvc.GetDefault()
	seed.AdminAuth = &model.AdminAuth{
		Type: "credentials",
		Users: []model.AdminAuthUser{{
			Username:    "admin",
			Password:    hash,
			Permissions: "*",
		}},
	}
	if err := configSvc.Save(seed); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	const writers = 8
	proposedLangs := make([]string, writers)
	for i := 0; i < writers; i++ {
		proposedLangs[i] = fmt.Sprintf("lang-%02d", i)
	}
	errs := runConcurrent(t, writers, func(i int) error {
		// Deep-copy seed per goroutine via JSON to avoid racing on shared
		// slices; tweak Lang and clear password to let preserveAdminAuthPasswords
		// keep the seed hash.
		cfgBytes, marshalErr := json.Marshal(seed)
		if marshalErr != nil {
			return marshalErr
		}
		var cfg model.NodeRedConfig
		if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
			return err
		}
		cfg.Lang = proposedLangs[i]
		cfg.AdminAuth.Users[0].Password = ""
		return configSvc.Save(cfg)
	})
	assertNoError(t, errs)

	got, err := configSvc.Get()
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if got.AdminAuth == nil || got.AdminAuth.Users[0].Password != hash {
		t.Fatalf("admin password hash was not preserved: %#v", got.AdminAuth)
	}
	// Lang must be exactly one of the proposed values — not a torn merge.
	found := false
	for _, l := range proposedLangs {
		if got.Lang == l {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Lang = %q, want one of %v (no torn write)", got.Lang, proposedLangs)
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
	if err := os.Mkdir(filepath.Join(dir, "settings.js"), 0o750); err != nil {
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
