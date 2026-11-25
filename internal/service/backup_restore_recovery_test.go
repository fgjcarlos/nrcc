package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// restoreRecoveryProbe records the post-restore steps in call order.
type restoreRecoveryProbe struct {
	calls      []string
	depsErr    error
	restartErr error
	readyErr   error
}

func (p *restoreRecoveryProbe) hooks() (func(context.Context) error, func(context.Context) error) {
	deps := func(context.Context) error {
		p.calls = append(p.calls, "deps")
		return p.depsErr
	}
	ready := func(context.Context) error {
		p.calls = append(p.calls, "ready")
		return p.readyErr
	}
	return deps, ready
}

// TestRestoreReinstallsDependenciesAndWaitsReady is the acceptance test for S5:
// a restore that replaced package.json reinstalls node_modules before the
// restart, and only reports success once Node-RED answers again.
func TestRestoreReinstallsDependenciesAndWaitsReady(t *testing.T) {
	dataDir := t.TempDir()
	writeTestFile(t, filepath.Join(dataDir, "flows.json"), `[{"id":"1"}]`)
	writeTestFile(t, filepath.Join(dataDir, "package.json"), `{"name":"nrcc-test"}`)

	svc := NewBackupService(dataDir)
	backup, err := svc.CreateTyped(model.BackupTypeManual, "snap")
	if err != nil {
		t.Fatalf("CreateTyped: %v", err)
	}

	probe := &restoreRecoveryProbe{}
	deps, ready := probe.hooks()
	svc.SetRestoreHooks(
		func() error { probe.calls = append(probe.calls, "quiesce"); return nil },
		func() error { probe.calls = append(probe.calls, "restart"); return probe.restartErr },
	)
	svc.SetRestoreRecoveryHooks(deps, ready)

	if _, err := svc.RestoreWithSafetyBackup(backup.ID); err != nil {
		t.Fatalf("RestoreWithSafetyBackup: %v", err)
	}

	want := []string{"quiesce", "deps", "restart", "ready"}
	if strings.Join(probe.calls, ",") != strings.Join(want, ",") {
		t.Fatalf("post-restore calls = %v, want %v", probe.calls, want)
	}
}

// TestRestoreSkipsDependencySyncWithoutPackageJSON proves the reinstall only
// runs when the archive actually replaced package.json.
func TestRestoreSkipsDependencySyncWithoutPackageJSON(t *testing.T) {
	dataDir := t.TempDir()
	writeTestFile(t, filepath.Join(dataDir, "flows.json"), `[{"id":"1"}]`)

	svc := NewBackupService(dataDir)
	backup, err := svc.CreateTyped(model.BackupTypeManual, "snap")
	if err != nil {
		t.Fatalf("CreateTyped: %v", err)
	}
	probe := &restoreRecoveryProbe{}
	deps, ready := probe.hooks()
	svc.SetRestoreHooks(
		func() error { probe.calls = append(probe.calls, "quiesce"); return nil },
		func() error { probe.calls = append(probe.calls, "restart"); return nil },
	)
	svc.SetRestoreRecoveryHooks(deps, ready)

	if _, err := svc.RestoreWithSafetyBackup(backup.ID); err != nil {
		t.Fatalf("RestoreWithSafetyBackup: %v", err)
	}
	if strings.Join(probe.calls, ",") != "quiesce,restart,ready" {
		t.Fatalf("post-restore calls = %v, want no dependency sync", probe.calls)
	}
}

// TestRestoreReadinessFailureIsReported proves a restore whose Node-RED never
// comes back is reported, not swallowed, while the files stay restored.
func TestRestoreReadinessFailureIsReported(t *testing.T) {
	dataDir := t.TempDir()
	writeTestFile(t, filepath.Join(dataDir, "flows.json"), `[{"id":"1"}]`)

	svc := NewBackupService(dataDir)
	backup, err := svc.CreateTyped(model.BackupTypeManual, "snap")
	if err != nil {
		t.Fatalf("CreateTyped: %v", err)
	}
	probe := &restoreRecoveryProbe{readyErr: errors.New("admin endpoint never answered")}
	deps, ready := probe.hooks()
	svc.SetRestoreHooks(
		func() error { probe.calls = append(probe.calls, "quiesce"); return nil },
		func() error { probe.calls = append(probe.calls, "restart"); return nil },
	)
	svc.SetRestoreRecoveryHooks(deps, ready)

	_, err = svc.RestoreWithSafetyBackup(backup.ID)
	if err == nil {
		t.Fatal("RestoreWithSafetyBackup returned nil, want a readiness error")
	}
	if !strings.Contains(err.Error(), "did not become ready") {
		t.Fatalf("error = %v, want it to name the readiness failure", err)
	}
	// #nosec G304 -- dataDir is t.TempDir().
	// #nosec G304 -- dataDir is t.TempDir().
	if got, readErr := os.ReadFile(filepath.Join(dataDir, "flows.json")); readErr != nil || string(got) != `[{"id":"1"}]` {
		t.Fatalf("files were not left restored: got=%q err=%v", got, readErr)
	}
}

// TestRestoreRestartFailureIsReported proves a restart failure reaches the
// caller, skips the readiness wait, and leaves the restored files in place.
func TestRestoreRestartFailureIsReported(t *testing.T) {
	dataDir := t.TempDir()
	writeTestFile(t, filepath.Join(dataDir, "flows.json"), `[{"id":"1"}]`)

	svc := NewBackupService(dataDir)
	backup, err := svc.CreateTyped(model.BackupTypeManual, "snap")
	if err != nil {
		t.Fatalf("CreateTyped: %v", err)
	}
	probe := &restoreRecoveryProbe{restartErr: errors.New("node-red refused to start")}
	deps, ready := probe.hooks()
	svc.SetRestoreHooks(
		func() error { probe.calls = append(probe.calls, "quiesce"); return nil },
		func() error { probe.calls = append(probe.calls, "restart"); return probe.restartErr },
	)
	svc.SetRestoreRecoveryHooks(deps, ready)

	_, err = svc.RestoreWithSafetyBackup(backup.ID)
	if err == nil {
		t.Fatal("RestoreWithSafetyBackup returned nil, want a restart error")
	}
	if !strings.Contains(err.Error(), "restart Node-RED") {
		t.Fatalf("error = %v, want it to name the restart failure", err)
	}
	if strings.Contains(strings.Join(probe.calls, ","), "ready") {
		t.Fatalf("readiness ran after a failed restart: %v", probe.calls)
	}
	// #nosec G304 -- dataDir is t.TempDir().
	if got, readErr := os.ReadFile(filepath.Join(dataDir, "flows.json")); readErr != nil || string(got) != `[{"id":"1"}]` {
		t.Fatalf("files were not left restored: got=%q err=%v", got, readErr)
	}
}
