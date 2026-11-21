package service

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func activateLodash(content string) string {
	return activateLibraryInSettings(content, "lodash", "_", true)
}

// While Node-RED runs, a library change goes through the apply coordinator:
// it is written, the runtime adapter restarts once and readiness is probed
// (#1016).
func TestLibrarySettingsUpdaterAppliesThroughCoordinatorWhenRunning(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	adapter := &fakeAdapter{name: "runtime"}
	coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { return adapter, nil })
	update := LibrarySettingsUpdater(coordinator, f.svc, func() bool { return true })

	restarted, err := update(context.Background(), activateLodash)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !restarted {
		t.Fatal("restarted = false, want true after a coordinated apply")
	}
	if adapter.restartCalls != 1 || adapter.readyCalls == 0 {
		t.Fatalf("adapter restart=%d ready=%d, want one restart and a readiness probe", adapter.restartCalls, adapter.readyCalls)
	}
	assertSettingsContain(t, f.settingsPath, `require("lodash")`)
	assertApplySuccessCount(t, f.recorder.snapshot(), 1)
}

// A transform that changes nothing neither writes nor restarts through the
// coordinator: the library service still reloads new nodes itself.
func TestLibrarySettingsUpdaterSkipsUnchangedSettings(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	adapter := &fakeAdapter{name: "runtime"}
	coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { return adapter, nil })
	update := LibrarySettingsUpdater(coordinator, f.svc, func() bool { return true })

	restarted, err := update(context.Background(), func(c string) string { return c })
	if err != nil || restarted {
		t.Fatalf("update = (%t, %v), want (false, nil)", restarted, err)
	}
	if adapter.restartCalls != 0 {
		t.Fatalf("adapter restarted %d times for an unchanged settings.js", adapter.restartCalls)
	}
}

// A stopped Node-RED stays stopped: the change is written without the
// coordinator, so no adapter restart starts it.
func TestLibrarySettingsUpdaterKeepsStoppedRuntimeStopped(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	adapter := &fakeAdapter{name: "runtime"}
	coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { return adapter, nil })
	update := LibrarySettingsUpdater(coordinator, f.svc, func() bool { return false })

	restarted, err := update(context.Background(), activateLodash)
	if err != nil || restarted {
		t.Fatalf("update = (%t, %v), want (false, nil)", restarted, err)
	}
	if adapter.restartCalls != 0 {
		t.Fatalf("adapter restarted %d times while Node-RED was stopped", adapter.restartCalls)
	}
	assertSettingsContain(t, f.settingsPath, `require("lodash")`)
}

// Without an adapter for the environment the coordinator refuses before
// writing; the change still lands through the validated legacy write.
func TestLibrarySettingsUpdaterFallsBackWithoutAdapter(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { return nil, ErrUnsupportedAdapter })
	update := LibrarySettingsUpdater(coordinator, f.svc, func() bool { return true })

	restarted, err := update(context.Background(), activateLodash)
	if err != nil || restarted {
		t.Fatalf("update = (%t, %v), want (false, nil)", restarted, err)
	}
	assertSettingsContain(t, f.settingsPath, `require("lodash")`)
}

// A concurrent apply is surfaced, not bypassed: settings.js is not written
// and the coordinator does not restart.
func TestLibrarySettingsUpdaterSurfacesApplyInFlight(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	adapter := &fakeAdapter{name: "runtime"}
	coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { return adapter, nil })
	if _, err := coordinator.acquire(f.settingsPath); err != nil {
		t.Fatal(err)
	}
	restarted, err := LibrarySettingsUpdater(coordinator, f.svc, func() bool { return true })(context.Background(), activateLodash)
	if !errors.Is(err, ErrApplyInFlight) || restarted || adapter.restartCalls != 0 {
		t.Fatalf("got (%t, %v, restarts=%d), want (false, ErrApplyInFlight, 0)", restarted, err, adapter.restartCalls)
	}
	data, readErr := os.ReadFile(f.settingsPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(data), "lodash") {
		t.Fatal("settings.js was written while another apply was in flight")
	}
}

// When the settings hook already restarted Node-RED, Install and Uninstall
// must not restart it a second time.
func TestLibraryServiceSkipsRestartAfterCoordinatedApply(t *testing.T) {
	svc := NewLibraryServiceWithPackageManager(t.TempDir(), &stubPM{})
	restarts := 0
	svc.SetNodeRedRestart(func() error { restarts++; return nil })
	svc.SetSettingsUpdater(func(context.Context, func(string) string) (bool, error) { return true, nil })

	if err := svc.Install(context.Background(), "lodash", ""); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if err := svc.Uninstall(context.Background(), "lodash"); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if restarts != 0 {
		t.Fatalf("restart hook fired %d times after coordinated applies, want 0", restarts)
	}
}

func assertSettingsContain(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read settings.js: %v", err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("settings.js does not contain %q:\n%s", want, data)
	}
}
