package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// newRefreshTestService returns a service whose cache offers 4.0.1 -> 4.0.2
// on an npm-global runtime with a real backup engine.
func newRefreshTestService(t *testing.T) (*UpdateService, string) {
	t.Helper()
	dataDir := t.TempDir()
	svc := newTestUpdateService(t, dataDir)
	pm, _ := npmGlobalProcessManager(t)
	svc.SetProcessManager(pm)
	svc.SetBackupCreator(NewBackupService(dataDir))
	svc.getInstalledVersionFn = func(context.Context) string { return "4.0.1" }
	svc.getLatestVersionFn = func(context.Context) (string, error) { return "4.0.2", nil }
	if _, err := svc.ForceCheck(context.Background()); err != nil {
		t.Fatal(err)
	}
	return svc, filepath.Join(dataDir, updateCacheFile)
}

func installRunner(installedVersion string, extra ...[]byte) *sequenceRunner {
	r := &sequenceRunner{results: []struct {
		output []byte
		err    error
	}{
		{[]byte("installed"), nil},                   // npm install -g
		{[]byte("v" + installedVersion + "\n"), nil}, // managed executable --version
		{[]byte(""), nil},                            // npm install --package-lock-only
	}}
	for _, out := range extra {
		r.results = append(r.results, struct {
			output []byte
			err    error
		}{out, fmt.Errorf("exit status 1")}) // npm audit
	}
	return r
}

// A verified install whose audit then reports criticals has still replaced the
// runtime, so the cache must record the installed version while the flow fails,
// and the failed flow must not block the next update.
func TestApplyUpdateWithBackup_AuditFailureKeepsInstalledVersionCached(t *testing.T) {
	svc, cachePath := newRefreshTestService(t)
	svc.runner = installRunner("4.0.2", []byte(`{"metadata":{"vulnerabilities":{"critical":2}}}`))

	err := svc.ApplyUpdateWithBackup(context.Background())
	if err == nil || !strings.Contains(err.Error(), "critical") {
		t.Fatalf("ApplyUpdateWithBackup error = %v, want the critical audit failure", err)
	}

	state := svc.GetFlowState()
	if state.State != model.StateFailed || state.Error != "update_failed" || state.BackupID == "" {
		t.Fatalf("flow state = %+v, want Failed/update_failed with the backup ID kept", state)
	}
	if history := svc.History(); len(history) != 1 || history[0].ID != state.BackupID {
		t.Fatalf("history = %+v, want the backup %q", history, state.BackupID)
	}
	if status := svc.GetCachedStatus(); status.CurrentVersion != "4.0.2" || status.UpdateAvailable {
		t.Fatalf("in-memory cache = %+v, want currentVersion 4.0.2 and no update available", status)
	}
	persisted, err := svc.store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if persisted.CurrentVersion != "4.0.2" || persisted.UpdateAvailable {
		t.Fatalf("on-disk cache %s = %+v, want currentVersion 4.0.2 and no update available", cachePath, persisted)
	}

	// Reaching the backup step proves the next update started.
	svc.SetBackupCreator(failingBackupCreator{})
	if err := svc.ApplyUpdateWithBackup(context.Background()); err == nil || !strings.Contains(err.Error(), "failed to create pre-update backup") {
		t.Fatalf("next update after the audit failure: got %v, want it to start and reach the backup step", err)
	}
	if got := svc.GetFlowState(); got.State != model.StateFailed || got.Error != "backup_failed" {
		t.Fatalf("next flow state = %+v, want Failed/backup_failed from the new run", got)
	}
}

// An install that does not verify must leave the persisted cache untouched.
// Disk is compared to disk because GetCachedStatus adds Strategy fields.
func TestApplyUpdateWithBackup_VerifyFailureLeavesCacheFileUnchanged(t *testing.T) {
	svc, cachePath := newRefreshTestService(t)
	before, err := os.ReadFile(cachePath) // #nosec G304 -- path is inside t.TempDir().
	if err != nil {
		t.Fatal(err)
	}
	svc.runner = installRunner("4.0.1")

	err = svc.ApplyUpdateWithBackup(context.Background())
	if !errors.Is(err, ErrUpdateVerificationFailed) {
		t.Fatalf("ApplyUpdateWithBackup error = %v, want verification failure", err)
	}
	if got := svc.GetFlowState(); got.State != model.StateFailed || got.Error != "update_failed" {
		t.Fatalf("flow state = %+v, want Failed/update_failed", got)
	}
	after, err := os.ReadFile(cachePath) // #nosec G304 -- path is inside t.TempDir().
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("cache file changed after a failed verification:\nbefore: %s\nafter:  %s", before, after)
	}
}
