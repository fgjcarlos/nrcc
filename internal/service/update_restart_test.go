package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// Regression for the T11 follow-up: a finished flow (Completed or Failed) must
// not block the next update until NRCC restarts.
func TestApplyUpdateWithBackup_StartsAgainAfterTerminalState(t *testing.T) {
	for _, terminal := range []model.UpdateState{model.StateCompleted, model.StateFailed} {
		t.Run(string(terminal), func(t *testing.T) {
			svc := updatableService(t)
			svc.SetBackupCreator(failingBackupCreator{})
			svc.setFlowState(model.UpdateFlowState{State: terminal, Phase: "done"})

			err := svc.ApplyUpdateWithBackup(context.Background())

			// Reaching the backup step proves the flow started.
			if err == nil || !strings.Contains(err.Error(), "failed to create pre-update backup") {
				t.Fatalf("expected the new flow to start and reach the backup step, got: %v", err)
			}
			if got := svc.GetFlowState(); got.State != model.StateFailed || got.Error != "backup_failed" {
				t.Fatalf("flow state = %+v, want Failed/backup_failed from the new run", got)
			}
		})
	}
}

// After a verified install the cached status must report the installed
// version, so clients stop offering the update that was just applied.
func TestApplyUpdateWithBackup_RefreshesCachedStatusAfterInstall(t *testing.T) {
	tmpDir := t.TempDir()
	svc := newTestUpdateService(t, tmpDir)
	pm, _ := npmGlobalProcessManager(t)
	svc.SetProcessManager(pm)
	svc.SetBackupCreator(NewBackupService(tmpDir))
	svc.getInstalledVersionFn = func(context.Context) string { return "4.0.1" }
	svc.getLatestVersionFn = func(context.Context) (string, error) { return "4.0.2", nil }
	if _, err := svc.ForceCheck(context.Background()); err != nil {
		t.Fatal(err)
	}
	svc.runner = &mockRunner{output: []byte("v4.0.2\n")}

	if err := svc.ApplyUpdateWithBackup(context.Background()); err != nil {
		t.Fatalf("ApplyUpdateWithBackup: %v", err)
	}

	status := svc.GetCachedStatus()
	if status.CurrentVersion != "4.0.2" || status.UpdateAvailable {
		t.Fatalf("cached status = %+v, want currentVersion 4.0.2 and no update available", status)
	}
	persisted, err := svc.store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if persisted.CurrentVersion != "4.0.2" || persisted.UpdateAvailable {
		t.Fatalf("persisted cache = %+v, want currentVersion 4.0.2 and no update available", persisted)
	}
}

type blockingBackupCreator struct {
	started chan struct{}
	release chan struct{}
}

func (b blockingBackupCreator) CreateTyped(model.BackupType, string) (model.Backup, error) {
	close(b.started)
	<-b.release
	return model.Backup{}, errors.New("released by test")
}

// Risk item 4: an update that is actually running must still be rejected.
func TestApplyUpdateWithBackup_RejectsWhileInProgress(t *testing.T) {
	svc := updatableService(t)
	creator := blockingBackupCreator{started: make(chan struct{}), release: make(chan struct{})}
	svc.SetBackupCreator(creator)
	// A previous run finished; the new run below must start from it.
	svc.setFlowState(model.UpdateFlowState{State: model.StateCompleted, Phase: "completed"})

	done := make(chan error, 1)
	go func() { done <- svc.ApplyUpdateWithBackup(context.Background()) }()
	select {
	case <-creator.started:
	case <-time.After(5 * time.Second):
		t.Fatal("first update never reached the backup step")
	}

	err := svc.ApplyUpdateWithBackup(context.Background())
	if err == nil || err.Error() != "update already in progress" {
		t.Fatalf("second update while the first is running: got %v, want 'update already in progress'", err)
	}
	if got := svc.GetFlowState().State; got != model.StateBackingUp {
		t.Fatalf("rejected start changed state to %s, want BackingUp", got)
	}

	close(creator.release)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("first update did not finish")
	}
}
