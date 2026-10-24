package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// reconcileFixture seeds a config service with a mirror and a source file.
func reconcileFixture(t *testing.T) (*ConfigService, string) {
	t.Helper()
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(validSettingsJS(t)), 0o600); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	seed, err := svc.Get()
	if err != nil {
		t.Fatalf("seed read: %v", err)
	}
	if _, err := svc.Update(func(current *model.NodeRedConfig) error {
		*current = seed
		return nil
	}); err != nil {
		t.Fatalf("seed mirror: %v", err)
	}
	return svc, settingsPath
}

// TestReconcileRawSettings_AdoptsDeclaredValues asserts that the source
// wins for the managed values it declares.
func TestReconcileRawSettings_AdoptsDeclaredValues(t *testing.T) {
	svc, _ := reconcileFixture(t)
	content := `module.exports = {
  uiPort: 7777,
  uiHost: '127.0.0.1',
  httpAdminRoot: '/red',
  httpNodeRoot: '/api',
  credentialSecret: 'rotated-secret',
};
`
	if _, err := svc.ReconcileRawSettings(content); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	got, err := svc.Get()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.UIPort != 7777 || got.Port != 7777 {
		t.Fatalf("port=%d uiPort=%d; want 7777", got.Port, got.UIPort)
	}
	if got.UIHost != "127.0.0.1" {
		t.Fatalf("uiHost=%q; want 127.0.0.1", got.UIHost)
	}
	if got.HTTPAdminRoot != "/red" || got.HTTPNodeRoot != "/api" {
		t.Fatalf("roots=%q %q; want /red /api", got.HTTPAdminRoot, got.HTTPNodeRoot)
	}
	if got.CredentialSecret != "rotated-secret" {
		t.Fatalf("credentialSecret=%q; want rotated-secret", got.CredentialSecret)
	}
}

// TestReconcileRawSettings_KeepsUndeclaredValues asserts that a value the
// source is silent about keeps its mirrored value instead of being reset
// to a parser default.
func TestReconcileRawSettings_KeepsUndeclaredValues(t *testing.T) {
	svc, _ := reconcileFixture(t)
	if _, err := svc.Update(func(current *model.NodeRedConfig) error {
		current.FlowFile = "operator-flows.json"
		current.UserDir = "./operator-data"
		current.NodesDir = "./operator-nodes"
		current.Lang = "de"
		current.DisableEditor = true
		return nil
	}); err != nil {
		t.Fatalf("seed mirror: %v", err)
	}
	content := `module.exports = {
  uiPort: 7777,
  httpAdminRoot: '/',
  httpNodeRoot: '/',
};
`
	if _, err := svc.ReconcileRawSettings(content); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	got, err := svc.Get()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.UIPort != 7777 {
		t.Fatalf("uiPort=%d; want 7777", got.UIPort)
	}
	if got.FlowFile != "operator-flows.json" || got.UserDir != "./operator-data" || got.NodesDir != "./operator-nodes" {
		t.Fatalf("undeclared paths were reset: %+v", got)
	}
	if got.Lang != "de" || !got.DisableEditor {
		t.Fatalf("undeclared flags were reset: %+v", got)
	}
}

// TestReconcileRawSettings_PreservesNRCCOwnedState asserts that the state
// NRCC owns in the mirror survives the reconciliation.
func TestReconcileRawSettings_PreservesNRCCOwnedState(t *testing.T) {
	svc, _ := reconcileFixture(t)
	encrypted := []model.EnvVar{{Key: "TOKEN", Value: "ciphertext", Encrypted: true, Description: "api token"}}
	libraries := []model.FunctionLibrary{{Name: "lib", Alias: "operator-lib"}}
	if _, err := svc.Update(func(current *model.NodeRedConfig) error {
		current.EnvVars = encrypted
		current.FunctionLibraries = libraries
		current.RuntimeState = map[string]any{"marker": "preserve"}
		return nil
	}); err != nil {
		t.Fatalf("seed mirror: %v", err)
	}
	content := `module.exports = {
  uiPort: 7777,
  httpAdminRoot: '/',
  httpNodeRoot: '/',
  runtimeState: { enabled: false },
};
`
	if _, err := svc.ReconcileRawSettings(content); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	got, err := svc.Get()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(got.EnvVars) != 1 || got.EnvVars[0].Key != "TOKEN" || got.EnvVars[0].Value != "ciphertext" || !got.EnvVars[0].Encrypted || got.EnvVars[0].Description != "api token" {
		t.Fatalf("environment metadata was not preserved: %+v", got.EnvVars)
	}
	if len(got.FunctionLibraries) != 1 || got.FunctionLibraries[0].Name != "lib" {
		t.Fatalf("function libraries were not preserved: %+v", got.FunctionLibraries)
	}
	if got.RuntimeState.(map[string]any)["marker"] != "preserve" {
		t.Fatalf("runtime state was not preserved: %+v", got.RuntimeState)
	}
}

// TestReconcileRawSettings_AdoptsSourceEnvValues asserts that the source
// speaks for the environment values it declares while NRCC keeps the
// metadata of the keys it manages.
func TestReconcileRawSettings_AdoptsSourceEnvValues(t *testing.T) {
	svc, _ := reconcileFixture(t)
	if _, err := svc.Update(func(current *model.NodeRedConfig) error {
		current.EnvVars = []model.EnvVar{
			{Key: "PLAIN", Value: "old", Type: "string", Description: "plain value"},
			{Key: "SECRET", Value: "ciphertext", Encrypted: true, Type: "secret"},
		}
		return nil
	}); err != nil {
		t.Fatalf("seed mirror: %v", err)
	}
	content := `module.exports = {
  uiPort: 7777,
  httpAdminRoot: '/',
  httpNodeRoot: '/',
  env: { PLAIN: 'new', ADDED: '42' },
};
`
	if _, err := svc.ReconcileRawSettings(content); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	got, err := svc.Get()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	byKey := map[string]model.EnvVar{}
	for _, envVar := range got.EnvVars {
		byKey[envVar.Key] = envVar
	}
	if byKey["PLAIN"].Value != "new" {
		t.Fatalf("the declared environment value was not adopted: %+v", byKey["PLAIN"])
	}
	if byKey["PLAIN"].Description != "plain value" {
		t.Fatalf("the environment metadata was not preserved: %+v", byKey["PLAIN"])
	}
	if byKey["SECRET"].Value != "ciphertext" || !byKey["SECRET"].Encrypted {
		t.Fatalf("the encrypted environment value was not preserved: %+v", byKey["SECRET"])
	}
	if byKey["ADDED"].Value != "42" {
		t.Fatalf("the new environment key was not adopted: %+v", byKey["ADDED"])
	}
}

// TestReconcileRawSettings_LeavesSourceUntouched asserts that the
// reconciliation never rewrites the source: the operator-owned blocks stay
// verbatim.
func TestReconcileRawSettings_LeavesSourceUntouched(t *testing.T) {
	svc, settingsPath := reconcileFixture(t)
	content := `module.exports = {
  uiPort: 7777,
  httpAdminRoot: '/',
  httpNodeRoot: '/',
  functionGlobalContext: { os: require('os') },
};
`
	if _, err := svc.ReconcileRawSettings(content); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	// The reconciliation never writes the source, so the seeded file is
	// still the live one.
	// #nosec G304 -- test fixture: settingsPath is derived from t.TempDir(), not user input.
	live, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if string(live) != validSettingsJS(t) {
		t.Fatalf("the reconciliation rewrote the source:\n%s", live)
	}
}

// TestReconcileRawSettings_CreatesMissingMirror asserts that the
// reconciliation publishes a mirror when none exists yet.
func TestReconcileRawSettings_CreatesMissingMirror(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	content := `module.exports = {
  uiPort: 7777,
  httpAdminRoot: '/',
  httpNodeRoot: '/',
};
`
	if _, err := svc.ReconcileRawSettings(content); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json")); err != nil {
		t.Fatalf("the mirror was not created: %v", err)
	}
	got, err := svc.Get()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.UIPort != 7777 {
		t.Fatalf("uiPort=%d; want 7777", got.UIPort)
	}
}

// TestRestoreMirror_RestoresPreviousState asserts that the snapshot a
// reconciliation returns restores the previous mirror state.
func TestRestoreMirror_RestoresPreviousState(t *testing.T) {
	svc, _ := reconcileFixture(t)
	before, err := svc.Get()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := `module.exports = {
  uiPort: 7777,
  httpAdminRoot: '/',
  httpNodeRoot: '/',
};
`
	snapshot, err := svc.ReconcileRawSettings(content)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if snapshot.Existed != true {
		t.Fatal("the snapshot does not report the existing mirror")
	}
	if err := svc.RestoreMirror(snapshot); err != nil {
		t.Fatalf("restore: %v", err)
	}
	after, err := svc.Get()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if after.UIPort != before.UIPort || after.Port != before.Port {
		t.Fatalf("the mirror was not restored: %d/%d; want %d/%d", after.Port, after.UIPort, before.Port, before.UIPort)
	}
}

// TestRestoreMirror_RemovesCreatedMirror asserts that a reconciliation
// that created the mirror is undone by removing it again.
func TestRestoreMirror_RemovesCreatedMirror(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	content := `module.exports = {
  uiPort: 7777,
  httpAdminRoot: '/',
  httpNodeRoot: '/',
};
`
	snapshot, err := svc.ReconcileRawSettings(content)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if snapshot.Existed {
		t.Fatal("the snapshot reports a mirror that did not exist")
	}
	if err := svc.RestoreMirror(snapshot); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json")); !os.IsNotExist(err) {
		t.Fatalf("the mirror was not removed: %v", err)
	}
}

// TestRuntimeRawApplyRestoresMirrorOnReadinessFailure asserts that a raw
// apply whose runtime stage fails rolls the mirror back together with the
// source: both representations have to describe the same configuration.
func TestRuntimeRawApplyRestoresMirrorOnReadinessFailure(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	original := model.DefaultNodeRedConfig()
	original.Port, original.UIPort = 1880, 1880
	original.EnvVars = []model.EnvVar{{Key: "KEEP", Value: "ciphertext", Encrypted: true}}
	if err := f.svc.Save(original); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(f.dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	adapter := &fakeAdapter{name: "runtime", readyErr: errors.New("candidate readiness failed")}
	coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { return adapter, nil })
	content := "module.exports = {\n  uiPort: 7777,\n  httpAdminRoot: '/',\n  httpNodeRoot: '/',\n};\n"
	_, err = coordinator.Apply(context.Background(), ApplyRequest{
		Path:      f.settingsPath,
		Content:   content,
		Expected:  f.liveRevision(t),
		BackupDir: filepath.Join(f.dir, "backups"),
	})
	if err == nil {
		t.Fatal("the readiness failure should reject the raw apply")
	}
	after, readErr := os.ReadFile(filepath.Join(f.dir, "config.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(before) {
		t.Fatalf("the mirror was not restored after the rollback:\n%s", after)
	}
	live, readErr := os.ReadFile(f.settingsPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(live), "uiPort: 1880") {
		t.Fatalf("the source was not restored after the rollback:\n%s", live)
	}
	got, err := f.svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if got.UIPort != 1880 {
		t.Fatalf("the structured read reports uiPort %d; want 1880", got.UIPort)
	}
	if len(got.EnvVars) != 1 || got.EnvVars[0].Key != "KEEP" || !got.EnvVars[0].Encrypted {
		t.Fatalf("the encrypted environment state was not preserved: %+v", got.EnvVars)
	}
}
