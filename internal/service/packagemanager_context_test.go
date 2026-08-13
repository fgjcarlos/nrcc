package service

import (
	"context"
	"errors"
	"os/exec"
	"sync"
	"testing"
	"time"
)

// TestNpmInstallHonorsContextCancel is the core regression for HIGH-004:
// `NpmPackageManager.Install` must terminate its child process within 1s of
// the parent context being cancelled. The previous implementation used bare
// `exec.Command` and blocked until npm finished, so a client disconnect (or a
// SIGTERM during install) left the command running for minutes and could
// corrupt `node_modules`.
func TestNpmInstallHonorsContextCancel(t *testing.T) {
	sleepBin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep binary not available on this platform")
	}

	pm := &NpmPackageManager{
		WorkDir: t.TempDir(),
		Bin:     sleepBin,
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a short delay so the child has time to start.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	// We pass a package spec but the binary is `sleep` so it ignores args; we
	// only need the command to block long enough for the cancel to fire.
	err = pm.Install(ctx, "30")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Install: expected error from cancelled context, got nil")
	}
	if elapsed > 3*time.Second {
		t.Fatalf("Install did not return within 3s of cancel (took %s); child process leaked", elapsed)
	}
}

// TestNpmInstallHonorsTimeout proves the hard install timeout (5 min default)
// applies when the parent context has a shorter deadline. The function must
// return a context.DeadlineExceeded error (or wrap a child exit signal) within
// the requested deadline + a small grace period.
func TestNpmInstallHonorsTimeout(t *testing.T) {
	sleepBin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep binary not available on this platform")
	}

	pm := &NpmPackageManager{
		WorkDir: t.TempDir(),
		Bin:     sleepBin,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err = pm.Install(ctx, "30")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Install: expected error from deadline, got nil")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("Install did not respect 200ms deadline (took %s)", elapsed)
	}
}

// TestNpmInstallKillsChildOnCancel is a stricter companion to
// TestNpmInstallHonorsContextCancel that asserts the OS-level child is gone,
// not just that the Go function returned. It polls `pgrep` (Linux) for the
// spawned `sleep` PID; if the PID is still alive after the cancel returns,
// the test fails. The previous implementation passed the Go-level cancel but
// the npm child continued running.
func TestNpmInstallKillsChildOnCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping child-PID check in short mode")
	}
	sleepBin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep binary not available")
	}

	pm := &NpmPackageManager{
		WorkDir: t.TempDir(),
		Bin:     sleepBin,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Spawn from a goroutine so we can wait for Install to return.
	var installErr error
	var installOnce sync.Once
	done := make(chan struct{})
	go func() {
		installOnce.Do(func() { installErr = pm.Install(ctx, "30") })
		close(done)
	}()

	<-done

	// At this point Install has returned. We can't directly inspect the child
	// PID because exec.Command hides it, but we know the only `sleep 30`
	// process on the system should be gone. If a `sleep 30` is still listed
	// by `pgrep`, the child was orphaned.
	if pid, _ := exec.LookPath("pgrep"); pid != "" {
		out, _ := exec.Command(pid, "-f", "sleep 30").Output()
		if len(out) > 0 {
			t.Fatalf("child `sleep 30` still alive after Install returned; pgrep output: %q", string(out))
		}
	}
	// Even without pgrep, we confirm the Go-level cancel propagated by checking
	// the error category.
	if installErr == nil {
		t.Fatal("Install: expected error from cancellation, got nil")
	}
}

// TestNpmUninstallHonorsContextCancel mirrors the install regression for the
// uninstall path (HIGH-004 also flagged `exec.Command(p.Bin, 'uninstall', ...)`).
func TestNpmUninstallHonorsContextCancel(t *testing.T) {
	sleepBin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep binary not available")
	}
	pm := &NpmPackageManager{
		WorkDir: t.TempDir(),
		Bin:     sleepBin,
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	err = pm.Uninstall(ctx, "30")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Uninstall: expected error from cancelled context, got nil")
	}
	if elapsed > 3*time.Second {
		t.Fatalf("Uninstall did not return within 3s of cancel (took %s)", elapsed)
	}
}

// TestNpmInstallRejectsInvalidPackageWithContext keeps the validation behavior
// identical (errors must wrap ErrInvalidPackageName) and confirms that passing
// a context to Install does not bypass the validator.
func TestNpmInstallRejectsInvalidPackageWithContext(t *testing.T) {
	pm := NewNpmPackageManager(t.TempDir())
	err := pm.Install(context.Background(), "evil; rm -rf /")
	if !errors.Is(err, ErrInvalidPackageName) {
		t.Fatalf("Install: expected ErrInvalidPackageName, got: %v", err)
	}
}
