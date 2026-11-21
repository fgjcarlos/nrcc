//go:build linux

package service

import (
	"context"
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
	if err := pm.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// The crash is recorded before the backoff starts.
	deadline := time.Now().Add(5 * time.Second)
	for len(pm.RestartEvents()) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("child never crashed")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := pm.Stop(context.Background()); err != nil {
		t.Fatalf("Stop during backoff: %v", err)
	}

	time.Sleep(3 * pm.restartDelay)

	data, err := os.ReadFile(starts) //nolint:gosec // G304: path is inside t.TempDir().
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(data), "start"); n != 1 {
		t.Fatalf("Node-RED started %d times, want 1 (restart after Stop)", n)
	}
	if got := pm.Status().Status; got != "stopped" {
		t.Fatalf("Status = %q after Stop, want stopped", got)
	}
}
