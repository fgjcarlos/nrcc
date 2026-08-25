package service

import (
	"context"
	"errors"
	"testing"
)

// stubPM is a no-op PackageManager used by the library restart tests.
// Each method records that it was called so a test can assert ordering.
type stubPM struct {
	installCalls   int
	uninstallCalls int
	installErr     error
}

func (s *stubPM) Install(ctx context.Context, pkg string) error {
	s.installCalls++
	return s.installErr
}

func (s *stubPM) Uninstall(ctx context.Context, pkg string) error {
	s.uninstallCalls++
	return nil
}

// TestLibraryServiceInstallTriggersRestart proves that a successful install
// fires the restart hook exactly once, while a failed install leaves the
// hook untouched.
func TestLibraryServiceInstallTriggersRestart(t *testing.T) {
	cases := []struct {
		name      string
		installErr error
		wantCalls int
	}{
		{name: "success restarts", installErr: nil, wantCalls: 1},
		{name: "failure does not restart", installErr: errors.New("boom"), wantCalls: 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pm := &stubPM{installErr: c.installErr}
			svc := NewLibraryServiceWithPackageManager(t.TempDir(), pm)
			var restartCalls int
			svc.SetNodeRedRestart(func() error {
				restartCalls++
				return nil
			})
			err := svc.Install(context.Background(), "node-red-dashboard")
			if c.installErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.installErr != nil && err == nil {
				t.Fatal("expected error to propagate")
			}
			if restartCalls != c.wantCalls {
				t.Fatalf("restart calls = %d, want %d", restartCalls, c.wantCalls)
			}
		})
	}
}

// TestLibraryServiceUninstallTriggersRestart is the symmetric assertion for
// the uninstall path.
func TestLibraryServiceUninstallTriggersRestart(t *testing.T) {
	pm := &stubPM{}
	svc := NewLibraryServiceWithPackageManager(t.TempDir(), pm)
	var restartCalls int
	svc.SetNodeRedRestart(func() error {
		restartCalls++
		return nil
	})
	if err := svc.Uninstall(context.Background(), "node-red-dashboard"); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if restartCalls != 1 {
		t.Fatalf("restart calls = %d, want 1", restartCalls)
	}
}

// TestLibraryServiceInstallWithoutHook ensures the no-hook path is the
// silent-no-op documented in fireRestart (external Node-RED, tests, dev).
func TestLibraryServiceInstallWithoutHook(t *testing.T) {
	pm := &stubPM{}
	svc := NewLibraryServiceWithPackageManager(t.TempDir(), pm)
	if err := svc.Install(context.Background(), "node-red-dashboard"); err != nil {
		t.Fatalf("install: %v", err)
	}
	if pm.installCalls != 1 {
		t.Fatalf("pm.installCalls = %d, want 1", pm.installCalls)
	}
}

// TestLibraryServiceInstallReportsRestartError proves the UI cannot receive a
// false success when npm changed the package set but Node-RED failed to reload.
func TestLibraryServiceInstallReportsRestartError(t *testing.T) {
	pm := &stubPM{}
	svc := NewLibraryServiceWithPackageManager(t.TempDir(), pm)
	restartErr := errors.New("node-red already stopped")
	svc.SetNodeRedRestart(func() error { return restartErr })
	err := svc.Install(context.Background(), "node-red-dashboard")
	if err == nil || !errors.Is(err, restartErr) {
		t.Fatalf("install error = %v, want wrapped restart error", err)
	}
}
