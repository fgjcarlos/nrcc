//go:build linux

package service

import (
	"context"
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

// TestProcessManager_StartDuringBackoffWins attaches to a listener during the
// backoff: the stale restart must not relaunch over the attached runtime.
func TestProcessManager_StartDuringBackoffWins(t *testing.T) {
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
	if err := pm.Start(); err != nil {
		t.Fatalf("Start during backoff: %v", err)
	}

	time.Sleep(3 * pm.restartDelay)

	if n := countStarts(t, starts); n != 1 {
		t.Fatalf("Node-RED started %d times, want 1", n)
	}
	if !pm.IsExternalMode() || pm.RestartBudgetExhausted() {
		t.Fatalf("external=%v exhausted=%v, want attached and not exhausted", pm.IsExternalMode(), pm.RestartBudgetExhausted())
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
