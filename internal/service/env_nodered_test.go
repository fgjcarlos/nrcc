package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func readTestFlows(t *testing.T, dir string) []map[string]json.RawMessage {
	t.Helper()
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
	if err := os.WriteFile(filepath.Join(dir, "flows.json"), []byte(initial), 0o640); err != nil {
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
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("flows.json permissions = %o, want 640", info.Mode().Perm())
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

func TestEnvServiceLeavesMalformedFlowsAndStoreUntouched(t *testing.T) {
	dir := t.TempDir()
	flowPath := filepath.Join(dir, "flows.json")
	const malformed = `[{"id":]`
	if err := os.WriteFile(flowPath, []byte(malformed), 0o644); err != nil {
		t.Fatal(err)
	}

	configSvc := NewIsolatedConfigService(dir)
	svc := NewEnvService(configSvc)
	if err := svc.Set("VALUE", "x", "string", "", false); err == nil {
		t.Fatal("Set() error = nil, want malformed-flow error")
	}
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
	if len(cfg.EnvVars) != 0 {
		t.Fatalf("env store changed despite sync failure: %#v", cfg.EnvVars)
	}
}

func TestEnvServiceCreatesSingleGlobalConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "flows.json"), []byte(`[{"id":"tab-1","type":"tab"}]`), 0o644); err != nil {
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
