package service

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/store"
)

func TestUpdateService_AppendBackup_Concurrent(t *testing.T) {
	const appends = 5

	dataDir := filepath.Join(t.TempDir(), "nested", "data")
	svc := &UpdateService{
		dataDir: dataDir,
		backupStore: store.NewJSONStore[[]model.BackupEntry](
			filepath.Join(dataDir, updateBackupsFile),
		),
	}

	errs := runConcurrent(t, appends, func(i int) error {
		return svc.AppendBackup(model.BackupEntry{ID: fmt.Sprintf("concurrent-%d", i)})
	})
	assertNoError(t, errs)

	entries, err := svc.backupStore.Read()
	if err != nil {
		t.Fatalf("Read backup history: %v", err)
	}
	if len(entries) > appends {
		t.Fatalf("backup history length = %d; want at most %d", len(entries), appends)
	}
	if len(entries) != appends {
		t.Fatalf("backup history length = %d; want %d concurrent appends", len(entries), appends)
	}

	seen := make(map[string]bool, appends)
	for _, entry := range entries {
		seen[entry.ID] = true
	}
	for i := 0; i < appends; i++ {
		id := fmt.Sprintf("concurrent-%d", i)
		if !seen[id] {
			t.Errorf("backup %q was lost", id)
		}
	}

	if err := svc.AppendBackup(model.BackupEntry{ID: "newest"}); err != nil {
		t.Fatalf("AppendBackup at capacity: %v", err)
	}
	entries, err = svc.backupStore.Read()
	if err != nil {
		t.Fatalf("Read backup history after capacity append: %v", err)
	}
	if len(entries) != appends {
		t.Fatalf("backup history length after capacity append = %d; want %d", len(entries), appends)
	}
	if entries[len(entries)-1].ID != "newest" {
		t.Errorf("latest backup ID = %q; want newest", entries[len(entries)-1].ID)
	}
}
