package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func readTestFlows(t *testing.T, dir string) []map[string]json.RawMessage {
	t.Helper()
	// #nosec G304 -- dir is t.TempDir()
	data, err := os.ReadFile(filepath.Join(dir, "flows.json"))
	if err != nil {
		t.Fatal(err)
	}
	var flows []map[string]json.RawMessage
	if err := json.Unmarshal(data, &flows); err != nil {
		t.Fatal(err)
	}
	return flows
}

func readGlobalEnv(t *testing.T, dir string) []nodeRedGlobalEnv {
	t.Helper()
	for _, flow := range readTestFlows(t, dir) {
		var typ string
		_ = json.Unmarshal(flow["type"], &typ)
		if typ == "global-config" {
			var env []nodeRedGlobalEnv
			if err := json.Unmarshal(flow["env"], &env); err != nil {
				t.Fatal(err)
			}
			return env
		}
	}
	return nil
}

func TestEnvServiceSyncsNodeRed5GlobalEnvironment(t *testing.T) {
	dir := t.TempDir()
	initial := `[
    {"id":"tab-1","type":"tab","label":"Main"},
    {"id":"manual-global","type":"global-config","env":[{"name":"MANUAL","value":"keep","type":"str"}],"modules":{"example":"1.0.0"}}
]`
	if err := os.WriteFile(filepath.Join(dir, "flows.json"), []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	svc := NewEnvService(NewIsolatedConfigService(dir))
	for _, envVar := range []model.EnvVar{
		{Key: "TEXT", Value: "hello", Type: "string"},
		{Key: "COUNT", Value: "42", Type: "number"},
		{Key: "ENABLED", Value: "true", Type: "boolean"},
	} {
		if err := svc.Set(envVar.Key, envVar.Value, envVar.Type, "", false); err != nil {
			t.Fatalf("Set(%s): %v", envVar.Key, err)
		}
	}

	env := readGlobalEnv(t, dir)
	want := map[string]nodeRedGlobalEnv{
		"MANUAL":  {Name: "MANUAL", Value: "keep", Type: "str"},
		"TEXT":    {Name: "TEXT", Value: "hello", Type: "str"},
		"COUNT":   {Name: "COUNT", Value: "42", Type: "num"},
		"ENABLED": {Name: "ENABLED", Value: "true", Type: "bool"},
	}
	if len(env) != len(want) {
		t.Fatalf("global env length = %d, want %d: %#v", len(env), len(want), env)
	}
	for _, item := range env {
		if item != want[item.Name] {
			t.Errorf("global env %q = %#v, want %#v", item.Name, item, want[item.Name])
		}
	}

	flows := readTestFlows(t, dir)
	if len(flows) != 2 {
		t.Fatalf("flow nodes changed: got %d, want 2", len(flows))
	}
	var modules map[string]string
	if err := json.Unmarshal(flows[1]["modules"], &modules); err != nil {
		t.Fatal(err)
	}
	if modules["example"] != "1.0.0" {
		t.Fatalf("unrelated global-config fields were lost: %#v", modules)
	}
	info, err := os.Stat(filepath.Join(dir, "flows.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("flows.json permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestEnvServiceUpdatesDeletesAndExcludesSecrets(t *testing.T) {
	dir := t.TempDir()
	svc := NewEnvService(NewIsolatedConfigService(dir), "test-key")

	if err := svc.Set("VALUE", "old", "string", "", false); err != nil {
		t.Fatal(err)
	}
	if err := svc.Set("VALUE", "9", "number", "", false); err != nil {
		t.Fatal(err)
	}
	if err := svc.Set("SECRET", "hidden", "secret", "", true); err != nil {
		t.Fatal(err)
	}

	env := readGlobalEnv(t, dir)
	if len(env) != 1 || env[0] != (nodeRedGlobalEnv{Name: "VALUE", Value: "9", Type: "num"}) {
		t.Fatalf("global env after update/secret = %#v", env)
	}
	// #nosec G304 -- dir is t.TempDir()
	data, _ := os.ReadFile(filepath.Join(dir, "flows.json"))
	if string(data) == "" || strings.Contains(string(data), "hidden") || strings.Contains(string(data), "SECRET") {
		t.Fatalf("secret leaked into flows.json: %s", data)
	}

	if err := svc.Delete("VALUE"); err != nil {
		t.Fatal(err)
	}
	env = readGlobalEnv(t, dir)
	if len(env) != 0 {
		t.Fatalf("global env after delete = %#v", env)
	}
}

func TestEnvServiceReportsPostCommitFailureForMalformedFlows(t *testing.T) {
	dir := t.TempDir()
	flowPath := filepath.Join(dir, "flows.json")
	const malformed = `[{"id":]`
	if err := os.WriteFile(flowPath, []byte(malformed), 0o600); err != nil {
		t.Fatal(err)
	}

	configSvc := NewIsolatedConfigService(dir)
	svc := NewEnvService(configSvc)
	if err := svc.Set("VALUE", "x", "string", "", false); err == nil {
		t.Fatal("Set() error = nil, want malformed-flow error")
	} else if !strings.Contains(err.Error(), "environment JSON committed") {
		t.Fatalf("Set() error = %q, want committed-state context", err)
	}
	// #nosec G304 -- flowPath comes from t.TempDir()
	data, err := os.ReadFile(flowPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != malformed {
		t.Fatalf("malformed flow file changed: %q", data)
	}
	cfg, err := configSvc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.EnvVars) != 1 || cfg.EnvVars[0].Key != "VALUE" || cfg.EnvVars[0].Value != "x" {
		t.Fatalf("env store was not committed before sync failure: %#v", cfg.EnvVars)
	}
}

func TestEnvServiceCreatesSingleGlobalConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "flows.json"), []byte(`[{"id":"tab-1","type":"tab"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewEnvService(NewIsolatedConfigService(dir))
	if err := svc.Set("VALUE", "x", "string", "", false); err != nil {
		t.Fatal(err)
	}

	count := 0
	for _, flow := range readTestFlows(t, dir) {
		var typ string
		_ = json.Unmarshal(flow["type"], &typ)
		if typ == "global-config" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("global-config count = %d, want 1", count)
	}
}

func TestEnvServicePreservesManualGlobalOnSecretCreate(t *testing.T) {
	dir := t.TempDir()
	flow := `[{"id":"manual-global","type":"global-config","env":[{"name":"MANUAL","value":"keep","type":"str"}]}]`
	if err := os.WriteFile(filepath.Join(dir, "flows.json"), []byte(flow), 0o600); err != nil {
		t.Fatal(err)
	}
	// #664: a real encryption key is required for the secret create to
	// succeed. The test asserts the manual global env survives, not that
	// the secret is silently stored as plaintext.
	svc := NewEnvService(NewIsolatedConfigService(dir), "preserve-test-key")
	if err := svc.Set("MANUAL", "ignored", "secret", "", true); err != nil {
		t.Fatal(err)
	}
	env := readGlobalEnv(t, dir)
	if len(env) != 1 || env[0].Value != "keep" {
		t.Fatalf("secret create erased manual global env: %#v", env)
	}
}

func TestEnvServiceRejectsMultipleGlobalConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "flows.json"), []byte(`[{"id":"a","type":"global-config","env":[]},{"id":"b","type":"global-config","env":[]}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	configSvc := NewIsolatedConfigService(dir)
	svc := NewEnvService(configSvc)
	err := svc.Set("VALUE", "x", "string", "", false)
	if err == nil {
		t.Fatal("Set() error = nil, want multiple-global-config error")
	}
	if !strings.Contains(err.Error(), "multiple Node-RED global-config") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSyncNodeRedGlobalEnv_ConcurrentGoroutinesAllKeysPresent proves REQ-1
// and REQ-4: under contention from N goroutines that each set a distinct
// key, the final flows.json must be valid JSON and contain every key.
// Without flock, two concurrent read-modify-write cycles clobber each
// other and the result is corruption (a missing key or invalid JSON).
func TestSyncNodeRedGlobalEnv_ConcurrentGoroutinesAllKeysPresent(t *testing.T) {
	dir := t.TempDir()
	initial := `[{"id":"manual-global","type":"global-config","env":[],"modules":{}}]`
	if err := os.WriteFile(filepath.Join(dir, "flows.json"), []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewEnvService(NewIsolatedConfigService(dir))
	const N = 10

	var wg sync.WaitGroup
	errCh := make(chan error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("KEY_%02d", i)
			val := fmt.Sprintf("v%d", i)
			if err := svc.Set(key, val, "string", "", false); err != nil {
				errCh <- fmt.Errorf("Set %s: %w", key, err)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
	if t.Failed() {
		return
	}

	env := readGlobalEnv(t, dir)
	if len(env) != N {
		t.Fatalf("global env has %d entries, want %d: %#v", len(env), N, env)
	}
	seen := make(map[string]bool, N)
	for _, e := range env {
		seen[e.Name] = true
	}
	for i := 0; i < N; i++ {
		k := fmt.Sprintf("KEY_%02d", i)
		if !seen[k] {
			t.Errorf("missing key %s in final global env", k)
		}
	}

	// File must round-trip as valid JSON.
	// #nosec G304 -- path is t.TempDir() + the test's own flows.json; not request-derived.
	data, err := os.ReadFile(filepath.Join(dir, "flows.json"))
	if err != nil {
		t.Fatalf("read flows.json: %v", err)
	}
	var flows []map[string]json.RawMessage
	if err := json.Unmarshal(data, &flows); err != nil {
		t.Fatalf("flows.json not valid JSON after concurrent edits: %v\n%s", err, data)
	}
}

// TestSyncNodeRedGlobalEnv_HoldsLockDuringRename pre-acquires the
// flockExclusive lock on flows.json from the test, then calls Set
// and asserts that Set fails with EWOULDBLOCK. This proves the lock is
// taken before the read-modify-rename — without the lock fix, Set
// would silently succeed even when the probe holds the flock.
// Polling-based approaches were unreliable because the lock window is
// ~0.6 ms while Set's full duration is ~700 ms.
func TestSyncNodeRedGlobalEnv_HoldsLockDuringRename(t *testing.T) {
	dir := t.TempDir()
	initial := `[{"id":"manual-global","type":"global-config","env":[],"modules":{}}]`
	if err := os.WriteFile(filepath.Join(dir, "flows.json"), []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	probePath := filepath.Join(dir, "flows.json")
	probe, err := flockExclusive(probePath)
	if err != nil {
		t.Fatalf("probe acquire: %v", err)
	}
	defer func() { _ = probe.Close() }()

	svc := NewEnvService(NewIsolatedConfigService(dir))
	err = svc.Set("CONFLICT_KEY", "v", "string", "", false)
	if err == nil {
		t.Fatal("expected Set to fail while another flock holds flows.json.lock")
	}
	if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
		t.Fatalf("expected wrapped EWOULDBLOCK/EAGAIN, got %v", err)
	}
}
