package service

import "testing"

func TestProcessManagerStatus_UsesCumulativeRestartCount(t *testing.T) {
	pm := NewProcessManager("node-red", t.TempDir(), NewLogBuffer(10))
	pm.mu.Lock()
	pm.restartCount = 2
	pm.cumulativeRestarts = 7
	pm.mu.Unlock()

	status := pm.Status()

	if status.RestartCount != 7 {
		t.Fatalf("Status().RestartCount = %d, want 7 cumulative restarts", status.RestartCount)
	}
	if status.ConsecutiveFailures != 2 {
		t.Fatalf("Status().ConsecutiveFailures = %d, want 2 backoff failures", status.ConsecutiveFailures)
	}
}
