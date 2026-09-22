package service

import (
	"reflect"
	"sort"
	"testing"
)

// Test slice 1 of issue #764 — preset contract and registry core.

func TestPresetRegistry_HasExpectedCorePresets(t *testing.T) {
	reg := NewPresetRegistry()
	want := []string{
		"https-tls-preset",
		"functionGlobalContext-strict",
		"functionGlobalContext-relaxed",
		"logging-callback-middleware",
	}
	for _, id := range want {
		if _, ok := reg.Get(id); !ok {
			t.Errorf("expected core preset %q to be registered", id)
		}
	}
}

func TestPresetRegistry_GetUnknown(t *testing.T) {
	reg := NewPresetRegistry()
	p, ok := reg.Get("does-not-exist")
	if ok {
		t.Errorf("unknown preset returned ok=true with %+v", p)
	}
}

func TestPresetRegistry_ListIsStable(t *testing.T) {
	reg := NewPresetRegistry()
	list := reg.List()
	if len(list) < 2 {
		t.Fatalf("expected at least 2 presets, got %d", len(list))
	}
	ids := make([]string, len(list))
	for i, p := range list {
		ids[i] = p.ID
	}
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(ids, sorted) {
		t.Errorf("PresetRegistry.List() is not sorted by ID: %v", ids)
	}
}

func TestPresetSurfaceContract(t *testing.T) {
	reg := NewPresetRegistry()
	for _, p := range reg.List() {
		if p.ID == "" {
			t.Errorf("preset with empty ID: %+v", p)
		}
		if len(p.Surfaces) == 0 {
			t.Errorf("preset %q declares no surfaces", p.ID)
		}
		if p.Trust == "" {
			t.Errorf("preset %q declares no trust tier", p.ID)
		}
		managed := ManagedSettingKeys()
		managedSet := map[string]bool{}
		for _, k := range managed {
			managedSet[k] = true
		}
		for _, key := range p.ManagedKeys {
			if !managedSet[key] {
				t.Errorf("preset %q manages key %q which is not in managedSettingKeys", p.ID, key)
			}
		}
		for _, s := range p.Surfaces {
			switch s {
			case SurfaceEditor, SurfaceHTTP, SurfaceSocketIO:
			default:
				t.Errorf("preset %q declares unknown surface %q", p.ID, s)
			}
		}
	}
}

func TestPresetRegistry_BuildsSourceEdits_Deterministic(t *testing.T) {
	reg := NewPresetRegistry()
	p, ok := reg.Get("https-tls-preset")
	if !ok {
		t.Skip("https-tls-preset not registered")
	}
	current := map[string]string{"https": "{ key: 'old', cert: 'old' }"}
	first, err := p.BuildEdits(current)
	if err != nil {
		t.Fatalf("BuildEdits: %v", err)
	}
	second, err := p.BuildEdits(current)
	if err != nil {
		t.Fatalf("BuildEdits (second): %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("BuildEdits is not deterministic:\n  first=%+v\n  second=%+v", first, second)
	}
}

func TestPresetRegistry_HttpsTlsPresetTouchesHTTPSSurface(t *testing.T) {
	reg := NewPresetRegistry()
	p, ok := reg.Get("https-tls-preset")
	if !ok {
		t.Skip("https-tls-preset not registered")
	}
	if !containsSurface(p.Surfaces, SurfaceHTTP) {
		t.Errorf("https-tls-preset must declare SurfaceHTTP, got %v", p.Surfaces)
	}
	if len(p.ManagedKeys) != 1 || p.ManagedKeys[0] != "https" {
		t.Errorf("https-tls-preset must manage exactly [https], got %v", p.ManagedKeys)
	}
}

func TestPresetRegistry_FGCStrictPreservesFunctionGlobalContext(t *testing.T) {
	reg := NewPresetRegistry()
	p, ok := reg.Get("functionGlobalContext-strict")
	if !ok {
		t.Skip("functionGlobalContext-strict not registered")
	}
	if !containsSurface(p.Surfaces, SurfaceHTTP) && !containsSurface(p.Surfaces, SurfaceEditor) {
		t.Errorf("functionGlobalContext-strict must declare at least HTTP or editor surface, got %v", p.Surfaces)
	}
	if !containsKey(p.ManagedKeys, "functionGlobalContext") {
		t.Errorf("functionGlobalContext-strict must manage functionGlobalContext, got %v", p.ManagedKeys)
	}
}

func TestPresetRegistry_LoggingCallbackMiddleware(t *testing.T) {
	reg := NewPresetRegistry()
	p, ok := reg.Get("logging-callback-middleware")
	if !ok {
		t.Skip("logging-callback-middleware not registered")
	}
	if len(p.ManagedKeys) != 1 || p.ManagedKeys[0] != "logging" {
		t.Errorf("logging-callback-middleware must manage exactly [logging], got %v", p.ManagedKeys)
	}
}

func containsSurface(surfaces []Surface, surface Surface) bool {
	for _, s := range surfaces {
		if s == surface {
			return true
		}
	}
	return false
}

func containsKey(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}
