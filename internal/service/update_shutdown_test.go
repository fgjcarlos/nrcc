package service

import (
	"context"
	"testing"
	"time"
)

// blockingRunner is an execRunner that records it was called, then blocks until
// the context it received is canceled. Used to prove ApplyUpdate's child
// context propagates the service-level shutdown cancellation.
type blockingRunner struct {
	started chan struct{}
	calls   int
}

func (b *blockingRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	b.calls++
	if b.started != nil {
		select {
		case b.started <- struct{}{}:
		default:
		}
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

// TestApplyUpdate_RespectsShutdownCancel is the S-REQ-4.2 regression for HIGH-011:
// `ApplyUpdate` used `context.WithTimeout(context.Background(), ...)` and was
// launched from a goroutine that also used `context.Background()`, so a
// server SIGTERM could not stop an in-flight npm install. The new contract:
// the runner's context is rooted in the service-level applyCtx, and Shutdown
// cancels that root, killing the runner within 1s.
func TestApplyUpdate_RespectsShutdownCancel(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	svc.getInstalledVersionFn = func(ctx context.Context) string { return "4.0.1" }
	svc.getLatestVersionFn = func(ctx context.Context) (string, error) { return "4.0.2", nil }

	// Populate cache so ApplyUpdate proceeds past the version guard.
	forceCtx, forceCancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, _ = svc.ForceCheck(forceCtx)
	forceCancel()

	started := make(chan struct{}, 1)
	runner := &blockingRunner{started: started}
	svc.runner = runner

	// Launch ApplyUpdate in a goroutine using a Background context (mirrors
	// the async PostApply handler in production).
	done := make(chan error, 1)
	go func() {
		done <- svc.ApplyUpdate(context.Background())
	}()

	// Wait for the runner to receive its call.
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("runner.Run was not invoked within 2s")
	}

	// Trigger service shutdown — this must propagate into the runner's context.
	if err := svc.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned unexpected error: %v", err)
	}

	// ApplyUpdate must return within 2s of Shutdown.
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("ApplyUpdate: expected non-nil error after Shutdown, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ApplyUpdate did not return within 2s of Shutdown; child goroutine leaked")
	}
}

// TestShutdownIsIdempotent ensures calling Shutdown multiple times does not
// panic (the cancel func is idempotent in Go 1.20+ but we still guard with
// nil-check + applyMu). A second call after the first should be a no-op.
func TestShutdownIsIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUpdateService(tmpDir)

	for i := 0; i < 3; i++ {
		if err := svc.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown call #%d returned error: %v", i+1, err)
		}
	}
}
