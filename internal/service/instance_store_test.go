package service

import (
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestNewInstanceStore_SeedsDefaultFromDataDir(t *testing.T) {
	const dataDir = "/var/lib/nrcc/data"
	store := NewInstanceStore(dataDir)

	list := store.List()
	if len(list) != 1 {
		t.Fatalf("List() returned %d instances, want 1 (default only)", len(list))
	}

	def := list[0]
	if def.ID != model.DefaultInstanceID {
		t.Errorf("default ID = %q, want %q", def.ID, model.DefaultInstanceID)
	}
	if def.Kind != model.InstanceKindLocal {
		t.Errorf("default Kind = %q, want %q", def.Kind, model.InstanceKindLocal)
	}
	if def.DataDir != dataDir {
		t.Errorf("default DataDir = %q, want %q (the existing DATA_DIR)", def.DataDir, dataDir)
	}
	if def.Name == "" {
		t.Errorf("default Name must not be empty")
	}
	if def.Health == "" {
		t.Errorf("default Health must be set")
	}
}

func TestNewInstanceStore_SeededDefaultIsValid(t *testing.T) {
	// The seeded default must satisfy the same model validation as any other
	// instance — it is not a special-cased malformed record.
	store := NewInstanceStore("./data")
	if err := store.List()[0].Validate(); err != nil {
		t.Errorf("seeded default failed Validate(): %v", err)
	}
}

func TestNewInstanceStore_StampsTimestamps(t *testing.T) {
	store := NewInstanceStore("./data")
	def := store.List()[0]
	if def.CreatedAt.IsZero() {
		t.Errorf("default CreatedAt is zero, want a real timestamp")
	}
	if def.UpdatedAt.IsZero() {
		t.Errorf("default UpdatedAt is zero, want a real timestamp")
	}
}
