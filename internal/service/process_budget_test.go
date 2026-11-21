package service

import (
	"net"
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
