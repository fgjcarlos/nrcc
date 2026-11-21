package service

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// exitedCmd returns a started command that exits on its own.
func exitedCmd(t *testing.T) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("true")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return cmd
}

func TestCrashBudgetResetsAfterStableUptime(t *testing.T) {
	pm := NewProcessManager("node-red", t.TempDir())
	pm.mu.Lock()
	pm.restartCount = pm.maxRestarts
	pm.startedAt = time.Now().Add(-pm.stableUptime - time.Second)
	pm.mu.Unlock()

	// A closed stopCh skips the restart so the test never spawns Node-RED.
	stopCh := make(chan struct{})
	close(stopCh)
	pm.waitForExit(exitedCmd(t), stopCh, make(chan struct{}))

	if got := pm.Status().ConsecutiveFailures; got != 0 {
		t.Fatalf("ConsecutiveFailures = %d after stable uptime, want 0", got)
	}
}

func TestCrashBudgetExhaustedUntilManualStart(t *testing.T) {
	pm := NewProcessManager("node-red", t.TempDir())
	pm.mu.Lock()
	pm.restartCount = pm.maxRestarts
	pm.startedAt = time.Now()
	pm.mu.Unlock()

	pm.waitForExit(exitedCmd(t), make(chan struct{}), make(chan struct{}))
	if !pm.RestartBudgetExhausted() {
		t.Fatal("RestartBudgetExhausted() = false after giving up")
	}
	if got := pm.Status().ConsecutiveFailures; got != pm.maxRestarts {
		t.Fatalf("ConsecutiveFailures = %d, want %d (crashes close together still count)", got, pm.maxRestarts)
	}

	// A manual start that attaches to a listener on the runtime port succeeds
	// without spawning a child.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	runtimeDir := t.TempDir()
	t.Setenv("NODE_RED_PORT", strconv.Itoa(ln.Addr().(*net.TCPAddr).Port))
	t.Setenv("NODE_RED_USER_DIR", runtimeDir)
	t.Setenv("NODE_RED_SETTINGS", filepath.Join(runtimeDir, "settings.js"))
	if err := pm.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if pm.RestartBudgetExhausted() {
		t.Fatal("RestartBudgetExhausted() = true after a manual start")
	}
}

func TestStatusReportsNoConsecutiveFailuresWhileRunningStably(t *testing.T) {
	pm := NewProcessManager("node-red", t.TempDir())
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	pm.mu.Lock()
	pm.running = true
	pm.currentCmd = cmd
	pm.restartCount = 3
	pm.startedAt = time.Now().Add(-pm.stableUptime - time.Second)
	pm.mu.Unlock()

	if got := pm.Status().ConsecutiveFailures; got != 0 {
		t.Fatalf("ConsecutiveFailures = %d while running stably, want 0", got)
	}
}

// crashingNodeRed writes a stub Node-RED that exits at once and points the
// runtime contract at a free port and a temp user dir.
func crashingNodeRed(t *testing.T) (pm *ProcessManager, script string) {
	t.Helper()
	dir := t.TempDir()
	script = filepath.Join(dir, "fake-node-red")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil { //nolint:gosec // G306 the stub must be executable.
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	userDir := t.TempDir()
	t.Setenv("NODE_RED_PORT", strconv.Itoa(port))
	t.Setenv("NODE_RED_USER_DIR", userDir)
	t.Setenv("NODE_RED_SETTINGS", filepath.Join(userDir, "settings.js"))
	pm = NewProcessManager(script, dir)
	pm.restartDelay = time.Millisecond
	return pm, script
}

func waitExhausted(t *testing.T, pm *ProcessManager) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !pm.RestartBudgetExhausted() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !pm.RestartBudgetExhausted() {
		t.Fatal("restart budget not exhausted")
	}
}

func TestCrashBudgetExhaustedClearedByRestart(t *testing.T) {
	pm, script := crashingNodeRed(t)
	pm.maxRestarts = 1
	if err := pm.Start(); err != nil {
		t.Fatal(err)
	}
	waitExhausted(t, pm)

	if err := os.WriteFile(script, []byte("#!/bin/sh\nexec sleep 30\n"), 0o700); err != nil { //nolint:gosec // G306 the stub must be executable.
		t.Fatal(err)
	}
	if err := pm.Restart(context.Background()); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	t.Cleanup(func() { _ = pm.Stop(context.Background()) })
	if pm.RestartBudgetExhausted() {
		t.Fatal("RestartBudgetExhausted() = true after a successful Restart")
	}
}

func TestFailedAutoRestartLaunchExhaustsBudget(t *testing.T) {
	pm, script := crashingNodeRed(t)
	pm.restartDelay = 200 * time.Millisecond
	if err := pm.Start(); err != nil {
		t.Fatal(err)
	}
	// The running child keeps its open script; the relaunch cannot find it.
	if err := os.Remove(script); err != nil {
		t.Fatal(err)
	}
	waitExhausted(t, pm)
}
