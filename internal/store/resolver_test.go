package store

import (
	"path/filepath"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestTenantPathResolver_DefaultResolvesToLegacyRoot(t *testing.T) {
	// The whole point of the first slice: the default tenant must resolve to the
	// existing flat DATA_DIR root, so no existing deployment's files move.
	root := t.TempDir()
	r := NewTenantPathResolver(root)

	got, err := r.Resolve(model.DefaultTenantID)
	if err != nil {
		t.Fatalf("Resolve(default) returned error: %v", err)
	}
	if got != root {
		t.Errorf("Resolve(default) = %q, want legacy root %q", got, root)
	}
}

func TestTenantPathResolver_NamedTenantResolvesUnderTenantsDir(t *testing.T) {
	root := t.TempDir()
	r := NewTenantPathResolver(root)

	got, err := r.Resolve("acme")
	if err != nil {
		t.Fatalf("Resolve(acme) returned error: %v", err)
	}
	want := filepath.Join(root, "tenants", "acme")
	if got != want {
		t.Errorf("Resolve(acme) = %q, want %q", got, want)
	}
}

func TestTenantPathResolver_RejectsInvalidTenantIDs(t *testing.T) {
	root := t.TempDir()
	r := NewTenantPathResolver(root)

	invalid := []model.TenantID{"", "..", "../etc", "a/b", ".", ".hidden"}
	for _, id := range invalid {
		got, err := r.Resolve(id)
		if err == nil {
			t.Errorf("Resolve(%q) returned nil error and path %q, want error", id, got)
		}
		if got != "" {
			t.Errorf("Resolve(%q) returned path %q on error, want empty string", id, got)
		}
	}
}
