package service

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
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

// TestNpmInstallKillsChildOnCancel asserts cancellation reaps the exact
// subprocess started by the npm fixture, without scanning unrelated processes.
func TestNpmInstallKillsChildOnCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping child-PID check in short mode")
	}
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep binary not available")
	}

	workDir := t.TempDir()
	pidFile := filepath.Join(workDir, "child.pid")
	fixture := filepath.Join(workDir, "npm-fixture.sh")
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 1.0.0; exit 0; fi\nprintf '%s' \"$$\" > \"$NRCC_TEST_CHILD_PIDFILE\"\nexec sleep 30\n"
	//nolint:gosec // G306: the npm fixture must stay 0700 to run as an executable.
	if err := os.WriteFile(fixture, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NRCC_TEST_CHILD_PIDFILE", pidFile)
	pm := &NpmPackageManager{WorkDir: workDir, Bin: fixture}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- pm.Install(ctx, "node-red") }()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(pidFile); err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("npm fixture did not publish its child PID")
		}
		time.Sleep(10 * time.Millisecond)
	}
	//nolint:gosec // G304: the path is a test-controlled fixture inside t.TempDir().
	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(pidBytes))
	if err != nil || pid <= 0 {
		cancel()
		t.Fatalf("invalid fixture child PID %q", pidBytes)
	}
	cancel()
	if err := <-done; err == nil {
		t.Fatal("Install: expected error from cancelled context, got nil")
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("find fixture child PID %d: %v", pid, err)
	}
	if err := process.Signal(syscall.Signal(0)); err == nil {
		t.Fatalf("fixture child PID %d is still alive after Install returned", pid)
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
