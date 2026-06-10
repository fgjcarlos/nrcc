package service

import "testing"

// TestFlowVersionService_StopIsIdempotent is the #291 regression: calling Stop
// more than once must not panic with "close of closed channel".
func TestFlowVersionService_StopIsIdempotent(t *testing.T) {
	svc := NewFlowVersionService(t.TempDir())
	svc.StartPolling()

	svc.Stop()
	svc.Stop() // second call must be a no-op, not a panic
}

// TestFlowVersionService_StopWithoutStart ensures Stop is safe even if polling
// was never started.
func TestFlowVersionService_StopWithoutStart(t *testing.T) {
	svc := NewFlowVersionService(t.TempDir())
	svc.Stop()
	svc.Stop()
}
