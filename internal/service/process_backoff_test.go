//go:build linux

package service

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestProcessManager_StopCancelsPendingRestart crashes the child, calls Stop
// during the restart backoff and waits past the delay: no new process may
// start (issue #1014).
func TestProcessManager_StopCancelsPendingRestart(t *testing.T) {
	pm, starts, _ := crashingProcessManager(t)
	if err := pm.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForCrash(t, pm)

	if err := pm.Stop(context.Background()); err != nil {
		t.Fatalf("Stop during backoff: %v", err)
	}

	time.Sleep(3 * pm.restartDelay)

	if n := countStarts(t, starts); n != 1 {
		t.Fatalf("Node-RED started %d times, want 1 (restart after Stop)", n)
	}
	if got := pm.Status().Status; got != "stopped" {
		t.Fatalf("Status = %q after Stop, want stopped", got)
	}
}

// TestProcessManager_StartRefusesForeignListener binds the runtime port
// before Start: Start must refuse instead of attaching, Stop must be a no-op,
// and Start must launch the child once the port is free (issue #1018).
func TestProcessManager_StartRefusesForeignListener(t *testing.T) {
	pm, starts, port := crashingProcessManager(t)
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}

	if err := pm.Start(); !errors.Is(err, ErrForeignListener) {
		t.Fatalf("Start over foreign listener = %v, want ErrForeignListener", err)
	}
	if got := pm.Status().Status; got != "stopped" {
		t.Fatalf("Status = %q after refused Start, want stopped", got)
	}
	if _, err := os.Stat(starts); !os.IsNotExist(err) {
		t.Fatalf("Node-RED was launched over a foreign listener (stat err=%v)", err)
	}
	if err := pm.Stop(context.Background()); err != nil {
		t.Fatalf("Stop after refused Start: %v", err)
	}

	_ = ln.Close()
	if err := pm.Start(); err != nil {
		t.Fatalf("Start after the port is free: %v", err)
	}
	waitForCrash(t, pm)
	if err := pm.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if n := countStarts(t, starts); n != 1 {
		t.Fatalf("Node-RED started %d times, want 1", n)
	}
}

// TestProcessManager_StartDuringBackoffRefusesForeignListener binds the port
// during the crash backoff: neither the user Start nor the pending automatic
// restart may adopt or launch over the foreign listener (issues #918, #1018).
func TestProcessManager_StartDuringBackoffRefusesForeignListener(t *testing.T) {
	pm, starts, port := crashingProcessManager(t)
	if err := pm.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForCrash(t, pm)

	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	if err := pm.Start(); !errors.Is(err, ErrForeignListener) {
		t.Fatalf("Start during backoff = %v, want ErrForeignListener", err)
	}

	time.Sleep(3 * pm.restartDelay)

	if n := countStarts(t, starts); n != 1 {
		t.Fatalf("Node-RED started %d times, want 1", n)
	}
	if got := pm.Status().Status; got != "stopped" {
		t.Fatalf("Status = %q, want stopped", got)
	}
	_ = pm.Stop(context.Background())
}

// TestProcessManager_AutoRestartRefusesForeignListenerAndExhausts binds the
// port during the backoff: the automatic restart refuses it and gives up
// (issue #1018).
func TestProcessManager_AutoRestartRefusesForeignListenerAndExhausts(t *testing.T) {
	pm, starts, port := crashingProcessManager(t)
	if err := pm.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForCrash(t, pm)

	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	deadline := time.Now().Add(5 * time.Second)
	for !pm.RestartBudgetExhausted() {
		if time.Now().After(deadline) {
			t.Fatal("RestartBudgetExhausted() = false after auto-restart met a foreign listener")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := pm.Status().Status; got != "stopped" {
		t.Fatalf("Status = %q, want stopped", got)
	}
	if n := countStarts(t, starts); n != 1 {
		t.Fatalf("Node-RED started %d times, want 1", n)
	}
}

// crashingProcessManager returns a manager whose fake Node-RED records each
// start in the returned file and exits 1 at once.
func crashingProcessManager(t *testing.T) (*ProcessManager, string, int) {
	t.Helper()
	dir := t.TempDir()
	starts := filepath.Join(dir, "starts")
	script := filepath.Join(dir, "fake-node-red.sh")
	body := "#!/bin/sh\necho start >> \"" + starts + "\"\nexit 1\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil { //nolint:gosec // G306: the fake Node-RED script must stay executable.
		t.Fatal(err)
	}
	port := unusedRuntimePortForTest(t)
	env := "NODE_RED_PORT=" + strconv.Itoa(port) + "\nNODE_RED_USER_DIR=" + dir + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	pm := NewProcessManager(script, dir)
	pm.restartDelay = 200 * time.Millisecond
	return pm, starts, port
}

// waitForCrash returns once the crash is recorded, before the backoff ends.
func waitForCrash(t *testing.T, pm *ProcessManager) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for len(pm.RestartEvents()) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("child never crashed")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func countStarts(t *testing.T, starts string) int {
	t.Helper()
	data, err := os.ReadFile(starts) //nolint:gosec // G304: path is inside t.TempDir().
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(data), "start")
}
