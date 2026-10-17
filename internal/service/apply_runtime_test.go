package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRuntimeApplyNilResolverFailsClosed(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	coordinator := NewRuntimeApplyCoordinator(f.apply, nil)
	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 4566", 1)
	_, err := coordinator.Apply(context.Background(), ApplyRequest{Path: f.settingsPath, Content: candidate, Expected: f.liveRevision(t), BackupDir: filepath.Join(f.dir, "backups")})
	if !errors.Is(err, ErrUnsupportedAdapter) {
		t.Fatalf("error=%v; want ErrUnsupportedAdapter", err)
	}
	got, readErr := os.ReadFile(f.settingsPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != validSettingsJS(t) {
		t.Fatal("settings changed without runtime authority")
	}
	assertApplySuccessCount(t, f.recorder.snapshot(), 0)
}

func TestRuntimeApplyResolverReturningNilFailsClosed(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	calls := 0
	coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { calls++; return nil, nil })
	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 4565", 1)
	_, err := coordinator.Apply(context.Background(), ApplyRequest{Path: f.settingsPath, Content: candidate, Expected: f.liveRevision(t), BackupDir: filepath.Join(f.dir, "backups")})
	if !errors.Is(err, ErrUnsupportedAdapter) {
		t.Fatalf("error=%v; want ErrUnsupportedAdapter", err)
	}
	if calls != 1 {
		t.Fatalf("resolver calls=%d; want 1", calls)
	}
	got, readErr := os.ReadFile(f.settingsPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != validSettingsJS(t) {
		t.Fatal("settings changed without runtime authority")
	}
	assertApplySuccessCount(t, f.recorder.snapshot(), 0)
}

func TestRuntimeApplyRestartsAndEmitsSuccessOnceAfterReady(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	adapter := &fakeAdapter{name: "runtime"}
	coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { return adapter, nil })
	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 4567", 1)
	_, err := coordinator.Apply(context.Background(), ApplyRequest{Path: f.settingsPath, Content: candidate, Expected: f.liveRevision(t), BackupDir: filepath.Join(f.dir, "backups")})
	if err != nil {
		t.Fatal(err)
	}
	events := f.recorder.snapshot()
	successes := 0
	for _, event := range events {
		if event.Action == "apply.success" {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("success events=%d, want exactly one: %+v", successes, events)
	}
	if adapter.restartCalls != 1 || adapter.readyCalls != 1 {
		t.Fatalf("runtime calls restart=%d ready=%d", adapter.restartCalls, adapter.readyCalls)
	}
}

func TestRuntimeApplyFailureRestoresAndSurfacesRecoveryFailure(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	adapter := &firstRestartFailure{fakeAdapter: &fakeAdapter{name: "runtime"}}
	coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { return adapter, nil })
	_, err := coordinator.Apply(context.Background(), ApplyRequest{Path: f.settingsPath, Content: strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 4568", 1), Expected: f.liveRevision(t), BackupDir: filepath.Join(f.dir, "backups"), RecoveryTimeout: time.Second})
	if err == nil || !strings.Contains(err.Error(), "candidate restart failed") {
		t.Fatalf("error=%v; want original restart failure", err)
	}
	if adapter.calls != 2 || adapter.readyCalls != 1 {
		t.Fatalf("calls restart=%d ready=%d", adapter.calls, adapter.readyCalls)
	}
	assertApplySuccessCount(t, f.recorder.snapshot(), 0)
	got, readErr := os.ReadFile(f.settingsPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != validSettingsJS(t) {
		t.Fatal("original settings not restored")
	}
}

func TestRuntimeApplyRecoveryIgnoresCancelledRequestContext(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	adapter := &fakeAdapter{name: "runtime", restartErr: errors.New("candidate restart failed")}
	coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { return adapter, nil })
	ctx, cancel := context.WithCancel(context.Background())
	adapter.restartErr = errors.New("candidate restart failed")
	adapter.readyErr = nil
	// The fake's first restart cancels the request after the candidate was written.
	cancelAdapter := &cancelOnRestartAdapter{fakeAdapter: adapter, cancel: cancel}
	coordinator = NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { return cancelAdapter, nil })
	_, err := coordinator.Apply(ctx, ApplyRequest{Path: f.settingsPath, Content: strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 4569", 1), Expected: f.liveRevision(t), BackupDir: filepath.Join(f.dir, "backups")})
	if err == nil {
		t.Fatal("cancelled apply should fail")
	}
	if cancelAdapter.recoveryContextErr != nil {
		t.Fatalf("recovery context cancelled: %v", cancelAdapter.recoveryContextErr)
	}
	assertApplySuccessCount(t, f.recorder.snapshot(), 0)
}

func TestRuntimeApplyReadinessFailureRestoresAndSurfacesRollbackRestartError(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	adapter := &recoveryRestartFailure{fakeAdapter: &fakeAdapter{name: "runtime"}}
	coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { return adapter, nil })
	_, err := coordinator.Apply(context.Background(), ApplyRequest{Path: f.settingsPath, Content: strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 4570", 1), Expected: f.liveRevision(t), BackupDir: filepath.Join(f.dir, "backups")})
	if err == nil || !strings.Contains(err.Error(), "candidate readiness failed") || !strings.Contains(err.Error(), "restored restart failed") {
		t.Fatalf("error=%v; want both candidate and recovery failures", err)
	}
	if adapter.calls != 2 || adapter.readyCalls != 1 {
		t.Fatalf("restart calls=%d ready calls=%d", adapter.calls, adapter.readyCalls)
	}
	assertApplySuccessCount(t, f.recorder.snapshot(), 0)
}

func assertApplySuccessCount(t *testing.T, events []recordedAudit, want int) {
	t.Helper()
	count := 0
	for _, event := range events {
		if event.Action == "apply.success" {
			count++
		}
	}
	if count != want {
		t.Fatalf("success event count=%d, want %d: %+v", count, want, events)
	}
}

func TestRuntimeApplyKeepsGuardThroughCandidateAndRecovery(t *testing.T) {
	for _, phase := range []string{"restart", "ready", "recovery"} {
		t.Run(phase, func(t *testing.T) {
			f := newApplyCoordinatorFixture(t)
			adapter := newPhasedApplyAdapter(phase)
			t.Cleanup(adapter.unblock)
			resolverCalls := 0
			coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { resolverCalls++; return adapter, nil })
			request := ApplyRequest{Path: f.settingsPath, Content: strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 4580", 1), Expected: f.liveRevision(t), BackupDir: filepath.Join(f.dir, "backups"), RecoveryTimeout: time.Second}
			finished := make(chan error, 1)
			go func() { _, err := coordinator.Apply(context.Background(), request); finished <- err }()
			select {
			case <-adapter.entered:
			case <-time.After(5 * time.Second):
				t.Fatal("adapter phase did not start")
			}
			if phase == "ready" {
				assertApplySuccessCount(t, f.recorder.snapshot(), 0)
			}
			_, err := coordinator.Apply(context.Background(), request)
			if !errors.Is(err, ErrApplyInFlight) {
				t.Fatalf("concurrent apply error=%v; want ErrApplyInFlight", err)
			}
			adapter.unblock()
			firstErr := <-finished
			if phase == "recovery" {
				if firstErr == nil {
					t.Fatal("candidate failure should remain an error")
				}
			} else if firstErr != nil {
				t.Fatalf("apply failed: %v", firstErr)
			}
			if resolverCalls != 1 {
				t.Fatalf("resolver calls=%d; want one pinned adapter", resolverCalls)
			}
			if phase == "recovery" {
				if adapter.restartCount != 2 || adapter.readyCount != 1 {
					t.Fatalf("pinned adapter calls restart=%d ready=%d; want 2 and 1", adapter.restartCount, adapter.readyCount)
				}
				assertApplySuccessCount(t, f.recorder.snapshot(), 0)
				flight, acquireErr := coordinator.acquire(f.settingsPath)
				if acquireErr != nil {
					t.Fatalf("guard remained held after recovery: %v", acquireErr)
				}
				coordinator.release(f.settingsPath, flight)
			} else {
				assertApplySuccessCount(t, f.recorder.snapshot(), 1)
			}
		})
	}
}

func TestRuntimeApplyRecoveryTimeoutUnwrapsRollbackFailure(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	adapter := &timeoutRecoveryAdapter{fakeAdapter: &fakeAdapter{name: "runtime"}, entered: make(chan struct{}), deadline: make(chan time.Time, 1)}
	coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { return adapter, nil })
	expected := f.liveRevision(t)
	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 4581", 1)
	result := make(chan error, 1)
	go func() {
		_, err := coordinator.Apply(context.Background(), ApplyRequest{Path: f.settingsPath, Content: candidate, Expected: expected, BackupDir: filepath.Join(f.dir, "backups"), RecoveryTimeout: 400 * time.Millisecond})
		result <- err
	}()
	select {
	case <-adapter.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("recovery restart was not attempted")
	}
	select {
	case deadline := <-adapter.deadline:
		if time.Until(deadline) <= 0 {
			t.Fatal("recovery deadline was not in the future")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("recovery context did not carry deadline")
	}
	err := <-result
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v; want recovery deadline in error chain", err)
	}
	var rollbackErr *RollbackError
	if !errors.As(err, &rollbackErr) {
		t.Fatalf("error=%v; want *RollbackError in error chain", err)
	}
	assertApplySuccessCount(t, f.recorder.snapshot(), 0)
}

type phasedApplyAdapter struct {
	*fakeAdapter
	phase        string
	entered      chan struct{}
	release      chan struct{}
	releaseOnce  sync.Once
	restartCount int
	readyCount   int
}

func newPhasedApplyAdapter(phase string) *phasedApplyAdapter {
	return &phasedApplyAdapter{fakeAdapter: &fakeAdapter{name: "phased"}, phase: phase, entered: make(chan struct{}), release: make(chan struct{})}
}
func (a *phasedApplyAdapter) unblock() { a.releaseOnce.Do(func() { close(a.release) }) }

func (a *phasedApplyAdapter) wait(ctx context.Context, stage string) error {
	if stage != a.phase {
		return nil
	}
	select {
	case <-a.entered:
	default:
		close(a.entered)
	}
	select {
	case <-a.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (a *phasedApplyAdapter) Restart(ctx context.Context) error {
	a.restartCount++
	if a.phase == "recovery" {
		if a.restartCount == 1 {
			return errors.New("candidate restart failed")
		}
		return a.wait(ctx, "recovery")
	}
	return a.wait(ctx, "restart")
}
func (a *phasedApplyAdapter) Ready(ctx context.Context) error {
	a.readyCount++
	return a.wait(ctx, "ready")
}

type timeoutRecoveryAdapter struct {
	*fakeAdapter
	entered  chan struct{}
	deadline chan time.Time
	calls    int
}

func (a *timeoutRecoveryAdapter) Restart(ctx context.Context) error {
	a.calls++
	if a.calls == 1 {
		return errors.New("candidate restart failed")
	}
	if ctx == nil {
		return errors.New("nil recovery context")
	}
	select {
	case <-a.entered:
	default:
		close(a.entered)
	}
	if deadline, ok := ctx.Deadline(); ok {
		a.deadline <- deadline
	}
	<-ctx.Done()
	return ctx.Err()
}
func (a *timeoutRecoveryAdapter) Ready(ctx context.Context) error { return nil }

type recoveryRestartFailure struct {
	*fakeAdapter
	calls int
}

func (a *recoveryRestartFailure) Restart(context.Context) error {
	a.calls++
	if a.calls == 2 {
		return errors.New("restored restart failed")
	}
	return nil
}
func (a *recoveryRestartFailure) Ready(context.Context) error {
	a.readyCalls++
	return errors.New("candidate readiness failed")
}

type firstRestartFailure struct {
	*fakeAdapter
	calls int
}

func (a *firstRestartFailure) Restart(ctx context.Context) error {
	a.calls++
	if a.calls == 1 {
		return errors.New("candidate restart failed")
	}
	return nil
}

type cancelOnRestartAdapter struct {
	*fakeAdapter
	cancel             context.CancelFunc
	calls              int
	recoveryContextErr error
}

func (a *cancelOnRestartAdapter) Restart(ctx context.Context) error {
	a.calls++
	if a.calls == 1 {
		a.cancel()
		return nil
	}
	if err := ctx.Err(); err != nil {
		a.recoveryContextErr = err
	}
	return nil
}
