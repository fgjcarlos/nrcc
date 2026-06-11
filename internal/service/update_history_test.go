package service

import (
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// TestUpdateService_History is part of the #292 fix: update history must reflect
// the real applied-update backup catalog, not a hardcoded empty list.
func TestUpdateService_History(t *testing.T) {
	svc := NewUpdateService(t.TempDir())

	if h := svc.History(); len(h) != 0 {
		t.Fatalf("empty catalog should yield no history, got %d", len(h))
	}

	if err := svc.AppendBackup(model.BackupEntry{ID: "b1", FromVersion: "1.0.0", Status: "completed"}); err != nil {
		t.Fatalf("AppendBackup: %v", err)
	}

	h := svc.History()
	if len(h) != 1 || h[0].ID != "b1" {
		t.Fatalf("expected 1 history entry b1, got %+v", h)
	}
}
