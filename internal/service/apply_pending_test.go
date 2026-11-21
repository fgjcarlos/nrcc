package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// killAfterWriteAdapter stops the applying goroutine inside Restart, i.e.
// after the settings write and before readiness. runtime.Goexit runs only
// deferred calls, like a process killed at that point runs none of the
// remaining apply steps.
type killAfterWriteAdapter struct{ *fakeAdapter }

func (killAfterWriteAdapter) Restart(context.Context) error {
	runtime.Goexit()
	return nil
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertNoPendingMarker(t *testing.T, settingsPath string) {
	t.Helper()
	if _, err := os.Stat(applyPendingMarkerPath(settingsPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pending apply marker left behind (stat err=%v)", err)
	}
}

func TestInterruptedApplyIsRolledBackAtNextBoot(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	previous := validSettingsJS(t)
	candidate := strings.Replace(previous, "uiPort: 1880", "uiPort: 4601", 1)
	coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) {
		return killAfterWriteAdapter{&fakeAdapter{name: "killed"}}, nil
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = coordinator.Apply(context.Background(), ApplyRequest{Path: f.settingsPath, Content: candidate, Expected: f.liveRevision(t), BackupDir: filepath.Join(f.dir, "backups")})
	}()
	<-done
	if got := readFile(t, f.settingsPath); got != candidate {
		t.Fatal("precondition: the candidate must be on disk when the process stops")
	}

	// Next start: a fresh service over the same data directory.
	restored, err := NewIsolatedConfigService(f.dir).RecoverInterruptedApply(context.Background(), f.settingsPath)
	if err != nil {
		t.Fatalf("RecoverInterruptedApply: %v", err)
	}
	if restored == "" {
		t.Fatal("RecoverInterruptedApply reported no pending apply")
	}
	if got := readFile(t, f.settingsPath); got != previous {
		t.Fatalf("settings.js after recovery is not the previous file:\n%s", got)
	}
	assertNoPendingMarker(t, f.settingsPath)
}

func TestRecoverInterruptedApplyWithoutMarkerChangesNothing(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	restored, err := f.svc.RecoverInterruptedApply(context.Background(), f.settingsPath)
	if err != nil || restored != "" {
		t.Fatalf("restored=%q err=%v; want no-op", restored, err)
	}
	if got := readFile(t, f.settingsPath); got != validSettingsJS(t) {
		t.Fatal("settings.js changed without a pending apply")
	}
}

func TestRuntimeApplyLeavesNoPendingMarker(t *testing.T) {
	for _, test := range []struct {
		name     string
		readyErr error
		wantErr  bool
	}{
		{name: "ready"},
		{name: "rolled back", readyErr: errors.New("not ready"), wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newApplyCoordinatorFixture(t)
			adapter := &failReadyOnceAdapter{fakeAdapter: &fakeAdapter{name: "fake"}, err: test.readyErr}
			coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { return adapter, nil })
			candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 4602", 1)
			_, err := coordinator.Apply(context.Background(), ApplyRequest{Path: f.settingsPath, Content: candidate, Expected: f.liveRevision(t), BackupDir: filepath.Join(f.dir, "backups")})
			if (err != nil) != test.wantErr {
				t.Fatalf("err=%v; wantErr=%v", err, test.wantErr)
			}
			assertNoPendingMarker(t, f.settingsPath)
		})
	}
}

// failReadyOnceAdapter fails the candidate readiness probe with err and
// lets the post-rollback probe succeed.
type failReadyOnceAdapter struct {
	*fakeAdapter
	err   error
	calls int
}

func (a *failReadyOnceAdapter) Ready(context.Context) error {
	a.calls++
	if a.calls == 1 {
		return a.err
	}
	return nil
}

// alwaysFailReady fails every readiness probe, so the rollback cannot
// confirm recovery.
type alwaysFailReady struct{ *fakeAdapter }

func (alwaysFailReady) Ready(context.Context) error { return errors.New("never ready") }

func TestFailedRecoveryKeepsPendingMarker(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	previous := validSettingsJS(t)
	candidate := strings.Replace(previous, "uiPort: 1880", "uiPort: 4900", 1)
	c := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) {
		return alwaysFailReady{&fakeAdapter{name: "x"}}, nil
	})
	if _, err := c.Apply(context.Background(), ApplyRequest{Path: f.settingsPath, Content: candidate, Expected: f.liveRevision(t), BackupDir: filepath.Join(f.dir, "backups"), RecoveryTimeout: 1}); err == nil {
		t.Fatal("want error")
	}
	if _, err := os.Stat(applyPendingMarkerPath(f.settingsPath)); err != nil {
		t.Fatalf("marker should stay after failed recovery: %v", err)
	}
	restored, err := NewIsolatedConfigService(f.dir).RecoverInterruptedApply(context.Background(), f.settingsPath)
	if err != nil || restored == "" || readFile(t, f.settingsPath) != previous {
		t.Fatalf("restored=%q err=%v", restored, err)
	}
	assertNoPendingMarker(t, f.settingsPath)
}

func TestRecoverInterruptedApplyMissingSnapshotChangesNothing(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	if err := writeApplyPendingMarker(context.Background(), f.settingsPath, filepath.Join(f.dir, "gone.js")); err != nil {
		t.Fatal(err)
	}
	restored, err := f.svc.RecoverInterruptedApply(context.Background(), f.settingsPath)
	if err == nil || restored != "" || readFile(t, f.settingsPath) != validSettingsJS(t) {
		t.Fatalf("restored=%q err=%v", restored, err)
	}
	if _, statErr := os.Stat(applyPendingMarkerPath(f.settingsPath)); statErr != nil {
		t.Fatal("marker should stay for the next boot")
	}
}

// blockMarkerAdapter replaces the marker with a non-empty directory during
// readiness so the success-path removal fails.
type blockMarkerAdapter struct {
	*fakeAdapter
	path string
}

func (a blockMarkerAdapter) Ready(context.Context) error {
	m := applyPendingMarkerPath(a.path)
	_ = os.Remove(m)
	return os.MkdirAll(filepath.Join(m, "x"), 0o750)
}

func TestMarkerRemovalFailureAfterReadyIsAnError(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 5000", 1)
	c := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) {
		return blockMarkerAdapter{&fakeAdapter{name: "x"}, f.settingsPath}, nil
	})
	_, err := c.Apply(context.Background(), ApplyRequest{Path: f.settingsPath, Content: candidate, Expected: f.liveRevision(t), BackupDir: filepath.Join(f.dir, "backups")})
	var ae *ApplyError
	if !errors.As(err, &ae) || ae.Stage != ApplyStageReady {
		t.Fatalf("err=%v", err)
	}
	assertApplySuccessCount(t, f.recorder.snapshot(), 0)
}

func TestNonRuntimeAppliesWriteNoPendingMarker(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 4700", 1)
	if _, err := f.coordinator.Apply(context.Background(), ApplyRequest{Path: f.settingsPath, Content: candidate, Expected: f.liveRevision(t), BackupDir: filepath.Join(f.dir, "backups")}); err != nil {
		t.Fatal(err)
	}
	assertNoPendingMarker(t, f.settingsPath)
	if _, err := f.svc.SaveRawSettings(strings.Replace(candidate, "4700", "4701", 1)); err != nil {
		t.Fatal(err)
	}
	assertNoPendingMarker(t, f.settingsPath)
}
