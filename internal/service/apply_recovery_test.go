package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestRecoverExternalApplyRuntimeAuditsRestartAndReady(t *testing.T) {
	hook := &recordingRollbackHook{}
	adapter := &fakeAdapter{name: AdapterNameNativeSystemd}
	err := RecoverExternalApplyRuntime(context.Background(), RollbackRequest{
		BackupPath: "/runtime/snapshot.js", ActivePath: "/runtime/settings.js", Stage: RollbackStage("boot"), Adapter: adapter,
		AuditHook: hook.Log, Capabilities: model.ConfigurationCapabilities{Adapter: "nodered-5"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if adapter.restartCalls != 1 || adapter.readyCalls != 1 {
		t.Fatalf("restart/ready calls = %d/%d, want 1/1", adapter.restartCalls, adapter.readyCalls)
	}
	if len(hook.events) != 2 || hook.events[0].Action != "apply.rollback.started" || hook.events[1].Action != "apply.rollback.succeeded" {
		t.Fatalf("events = %+v", hook.events)
	}
	if hook.events[0].Meta["apply_rollback_stage"] != "boot" || hook.events[0].Meta["apply_rollback_backup_path"] != "snapshot.js" {
		t.Fatalf("started meta = %+v", hook.events[0].Meta)
	}
}

func TestRecoverExternalApplyRuntimeReportsRestartFailure(t *testing.T) {
	hook := &recordingRollbackHook{}
	adapter := &fakeAdapter{name: AdapterNameNativeSystemd, restartErr: errors.New("offline")}
	err := RecoverExternalApplyRuntime(context.Background(), RollbackRequest{ActivePath: "/runtime/settings.js", Stage: RollbackStageRestart, Adapter: adapter, AuditHook: hook.Log})
	if err == nil {
		t.Fatal("expected restart failure")
	}
	if len(hook.events) != 2 || hook.events[1].Action != "apply.rollback.failed" {
		t.Fatalf("events = %+v", hook.events)
	}
}
