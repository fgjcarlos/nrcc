package service

import (
	"context"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

type panickingBackupCreator struct{}

func (panickingBackupCreator) CreateTyped(model.BackupType, string) (model.Backup, error) {
	panic("backup creator exploded")
}

// A panic inside a claimed run must not leave the flow stuck in BackingUp
// (which blocks every later update) nor crash the process: Run recovers,
// reports Failed, and the next claim succeeds.
func TestUpdateRun_PanicMarksFlowFailedAndReleasesClaim(t *testing.T) {
	svc := updatableService(t)
	svc.SetBackupCreator(panickingBackupCreator{})

	run, err := svc.ClaimUpdate()
	if err != nil {
		t.Fatalf("ClaimUpdate: %v", err)
	}
	err = run.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "backup creator exploded") {
		t.Fatalf("Run error = %v, want the recovered panic", err)
	}
	if got := svc.GetFlowState(); got.State != model.StateFailed || got.Error != "internal_error" || got.Phase != "backup" {
		t.Fatalf("flow state = %+v, want Failed/internal_error in phase backup", got)
	}

	svc.SetBackupCreator(failingBackupCreator{})
	next, err := svc.ClaimUpdate()
	if err != nil {
		t.Fatalf("next ClaimUpdate after a panic: %v", err)
	}
	_ = next.Run(context.Background())
}
