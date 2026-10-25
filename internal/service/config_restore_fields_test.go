package service

import (
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// Issue #881: model.NodeRedConfig carries an unexported bookkeeping field
// that records which security surfaces a decoded payload carried. Both
// restore paths walk the struct with reflect, and reflect.Value.Interface
// panics on an unexported field. A panic in a recovery path aborts the
// whole process, so the invariant is pinned by a test.
func TestRestorePaths_IgnoreUnexportedFields(t *testing.T) {
	t.Run("structured restore", func(t *testing.T) {
		svc := NewIsolatedConfigService(t.TempDir())
		previous := model.NodeRedConfig{
			Port:          1880,
			UIPort:        1880,
			HTTPAdminRoot: "/",
			HTTPNodeRoot:  "/",
		}
		if err := svc.Save(previous); err != nil {
			t.Fatalf("seed config: %v", err)
		}
		committed := previous
		committed.HTTPAdminRoot = "/nrcc"

		state := StructuredApplyInput{Previous: previous, Committed: committed}
		if err := svc.RestoreStructured(state); err != nil {
			t.Fatalf("restore structured: %v", err)
		}

		stored, err := svc.Get()
		if err != nil {
			t.Fatalf("read config: %v", err)
		}
		if stored.HTTPAdminRoot != "/" {
			t.Errorf("HTTPAdminRoot = %q, want %q", stored.HTTPAdminRoot, "/")
		}
	})

	t.Run("mirror restore", func(t *testing.T) {
		svc := NewIsolatedConfigService(t.TempDir())
		previous := model.NodeRedConfig{
			Port:          1880,
			UIPort:        1880,
			HTTPAdminRoot: "/",
			HTTPNodeRoot:  "/",
		}
		if err := svc.Save(previous); err != nil {
			t.Fatalf("seed config: %v", err)
		}
		committed := previous
		committed.HTTPAdminRoot = "/nrcc"

		snapshot := MirrorSnapshot{Existed: true, Previous: previous, Committed: committed}
		if err := svc.RestoreMirror(snapshot); err != nil {
			t.Fatalf("restore mirror: %v", err)
		}

		stored, err := svc.Get()
		if err != nil {
			t.Fatalf("read config: %v", err)
		}
		if stored.HTTPAdminRoot != "/" {
			t.Errorf("HTTPAdminRoot = %q, want %q", stored.HTTPAdminRoot, "/")
		}
	})
}
