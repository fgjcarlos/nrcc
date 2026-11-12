//go:build linux

package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// newDeadlineTestProcessManager builds a ProcessManager whose child is a
// shell script that installs the given trap before sleeping.
func newDeadlineTestProcessManager(t *testing.T, trap string) (*ProcessManager, string) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-node-red.sh")
	body := fmt.Sprintf("#!/bin/sh\ntrap '' %s\nsleep 30\n", trap)
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	// Pick a free port so startLocked does not collide with a real listener.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	envFile := filepath.Join(dir, ".env")
	if err := os.WriteFile(envFile, []byte("NODE_RED_PORT="+strconv.Itoa(port)+"\nNODE_RED_USER_DIR="+dir+"\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}
	return NewProcessManager(script, dir), dir
}

// TestProcessManager_StopHonoursContext proves Stop returns before the
// context deadline even when the child ignores SIGTERM and SIGKILL-level
// escalation would still be pending (issue #918, S1).
func TestProcessManager_StopHonoursContext(t *testing.T) {
	pm, _ := newDeadlineTestProcessManager(t, "TERM")
	if err := pm.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := pm.Stop(ctx)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Stop with expired ctx = nil error, want deadline error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("Stop err = %v, want context error", err)
	}
	if elapsed > 2*time.Second {
		t.Errorf("Stop took %v, want it to return near the deadline", elapsed)
	}
	_ = pm.Stop(context.Background())
}

// TestProcessManager_RestartRefusesForeignListener proves the restart path
// never attaches to a listener it does not own (issue #918, S2): when the
// runtime port is bound by a foreign process, Restart fails instead of
// silently adopting the foreign runtime.
func TestProcessManager_RestartRefusesForeignListener(t *testing.T) {
	pm, _ := newDeadlineTestProcessManager(t, "TERM")
	if err := pm.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	port := pm.EffectiveRuntimePort()
	if port == 0 {
		t.Fatal("effective runtime port not recorded")
	}
	// Bind a foreign listener on the port before restart so the strict
	// start path sees an occupied port after our child exits.
	foreign, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("foreign listen: %v", err)
	}
	t.Cleanup(func() { _ = foreign.Close() })

	err = pm.Restart(context.Background())
	if err == nil {
		t.Fatal("Restart over foreign listener = nil error, want refusal")
	}
	if !errors.Is(err, ErrForeignListener) {
		t.Errorf("Restart err = %v, want ErrForeignListener", err)
	}
	_ = pm.Stop(context.Background())
}

// TestProcessManager_StopNormalStillWorks is the regression case: a child
// that honours SIGTERM still stops cleanly with a generous context.
func TestProcessManager_StopNormalStillWorks(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "sleep.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	envFile := filepath.Join(dir, ".env")
	if err := os.WriteFile(envFile, []byte("NODE_RED_PORT="+strconv.Itoa(port)+"\nNODE_RED_USER_DIR="+dir+"\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}
	pm := NewProcessManager(script, dir)
	if err := pm.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pm.Stop(ctx); err != nil {
		t.Errorf("Stop: %v", err)
	}
}

// ensure syscall import stays used if traps evolve.
var _ = syscall.SIGTERM
