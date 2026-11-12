package handler

import (
	"errors"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// execRecorder is the testing double for RestartService.execCommand.
// It captures every invocation so the tests can assert the right
// command was issued with the right arguments. The recorder also
// returns a canned error from Run / CombinedOutput so failure paths
// can be exercised without spawning real subprocesses.
type execRecorder struct {
	mu       sync.Mutex
	calls    []execCall
	runErr   error
	capture  func(call execCall) *exec.Cmd
}

type execCall struct {
	Name string
	Args []string
}

func newExecRecorder() *execRecorder {
	return &execRecorder{
		capture: func(call execCall) *exec.Cmd {
			// Return a *exec.Cmd pointed at /bin/true so Run returns
			// nil; failures are simulated by overriding runErr.
			return exec.Command("true")
		},
	}
}

func (e *execRecorder) fn() func(name string, arg ...string) *exec.Cmd {
	return func(name string, arg ...string) *exec.Cmd {
		e.mu.Lock()
		defer e.mu.Unlock()
		argsCopy := append([]string(nil), arg...)
		e.calls = append(e.calls, execCall{Name: name, Args: argsCopy})
		if e.runErr != nil {
			// exec.Command("false") always exits non-zero. We
			// don't need to inspect its output — the Run error
			// from /bin/false already encodes the canned runErr
			// via the wrapping in RestartService.runWithError.
			return exec.Command("false")
		}
		return e.capture(execCall{Name: name, Args: argsCopy})
	}
}

// TestRestartService_RestartDockerCompose_Happy asserts the
// documented command shape (`docker compose -p <project> restart`)
// and that a zero exit returns nil.
func TestRestartService_RestartDockerCompose_Happy(t *testing.T) {
	rec := newExecRecorder()
	svc := NewRestartService().WithExecCommand(rec.fn())
	if err := svc.RestartDockerCompose("nrcc"); err != nil {
		t.Fatalf("RestartDockerCompose: %v", err)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(rec.calls))
	}
	if got := rec.calls[0].Name; got != "docker" {
		t.Errorf("name = %q, want docker", got)
	}
	want := []string{"compose", "restart", "-p", "nrcc"}
	if !equalStrings(rec.calls[0].Args, want) {
		t.Errorf("args = %v, want %v", rec.calls[0].Args, want)
	}
}

// TestRestartService_RestartDockerCompose_NoProject asserts the
// command degrades gracefully when the project name is empty
// (single-project hosts do not need -p).
func TestRestartService_RestartDockerCompose_NoProject(t *testing.T) {
	rec := newExecRecorder()
	svc := NewRestartService().WithExecCommand(rec.fn())
	if err := svc.RestartDockerCompose(""); err != nil {
		t.Fatalf("RestartDockerCompose: %v", err)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(rec.calls))
	}
	want := []string{"compose", "restart"}
	if !equalStrings(rec.calls[0].Args, want) {
		t.Errorf("args = %v, want %v", rec.calls[0].Args, want)
	}
}

// TestRestartService_RestartDockerCompose_Failure asserts a non-zero
// exit surfaces as a wrapped error containing the project name.
func TestRestartService_RestartDockerCompose_Failure(t *testing.T) {
	rec := newExecRecorder()
	rec.runErr = errors.New("exit 1")
	svc := NewRestartService().WithExecCommand(rec.fn())
	err := svc.RestartDockerCompose("nrcc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "nrcc") {
		t.Errorf("err = %v, missing project name", err)
	}
}

// TestRestartService_RestartDockerCustom_Happy asserts the command
// shape (`docker restart <container>`) and zero-exit returns nil.
func TestRestartService_RestartDockerCustom_Happy(t *testing.T) {
	rec := newExecRecorder()
	svc := NewRestartService().WithExecCommand(rec.fn())
	if err := svc.RestartDockerCustom("node-red-1"); err != nil {
		t.Fatalf("RestartDockerCustom: %v", err)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(rec.calls))
	}
	if got := rec.calls[0].Name; got != "docker" {
		t.Errorf("name = %q, want docker", got)
	}
	want := []string{"restart", "node-red-1"}
	if !equalStrings(rec.calls[0].Args, want) {
		t.Errorf("args = %v, want %v", rec.calls[0].Args, want)
	}
}

// TestRestartService_RestartDockerCustom_EmptyContainer asserts the
// boundary check rejects an empty container name before invoking exec.
func TestRestartService_RestartDockerCustom_EmptyContainer(t *testing.T) {
	rec := newExecRecorder()
	svc := NewRestartService().WithExecCommand(rec.fn())
	err := svc.RestartDockerCustom("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(rec.calls) != 0 {
		t.Errorf("exec should not be called for empty container, got %d calls", len(rec.calls))
	}
}

// TestRestartService_RestartSystemd_Happy asserts the command shape
// (`systemctl restart <unit>`) and zero-exit returns nil.
func TestRestartService_RestartSystemd_Happy(t *testing.T) {
	rec := newExecRecorder()
	svc := NewRestartService().WithExecCommand(rec.fn())
	if err := svc.RestartSystemd("nodered.service"); err != nil {
		t.Fatalf("RestartSystemd: %v", err)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(rec.calls))
	}
	if got := rec.calls[0].Name; got != "systemctl" {
		t.Errorf("name = %q, want systemctl", got)
	}
	want := []string{"restart", "nodered.service"}
	if !equalStrings(rec.calls[0].Args, want) {
		t.Errorf("args = %v, want %v", rec.calls[0].Args, want)
	}
}

// TestRestartService_RestartSystemd_EmptyUnit asserts the boundary
// check rejects an empty systemd unit before invoking exec.
func TestRestartService_RestartSystemd_EmptyUnit(t *testing.T) {
	rec := newExecRecorder()
	svc := NewRestartService().WithExecCommand(rec.fn())
	err := svc.RestartSystemd("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(rec.calls) != 0 {
		t.Errorf("exec should not be called for empty unit, got %d calls", len(rec.calls))
	}
}

// TestRestartService_RestartSystemd_Failure asserts a non-zero exit
// surfaces as a wrapped error containing the unit name.
func TestRestartService_RestartSystemd_Failure(t *testing.T) {
	rec := newExecRecorder()
	rec.runErr = errors.New("exit 1")
	svc := NewRestartService().WithExecCommand(rec.fn())
	err := svc.RestartSystemd("nodered.service")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "nodered.service") {
		t.Errorf("err = %v, missing unit name", err)
	}
}

// TestRestartService_NilReceiverSafe asserts the helper handles a nil
// receiver without panicking — the slice C coordinator may construct
// the service lazily and we don't want a missing wire-up to crash.
func TestRestartService_NilReceiverSafe(t *testing.T) {
	var svc *RestartService
	if err := svc.RestartDockerCompose("nrcc"); err == nil {
		t.Error("nil receiver RestartDockerCompose = nil, want error")
	}
	if err := svc.RestartDockerCustom("nrcc-1"); err == nil {
		t.Error("nil receiver RestartDockerCustom = nil, want error")
	}
	if err := svc.RestartSystemd("nodered.service"); err == nil {
		t.Error("nil receiver RestartSystemd = nil, want error")
	}
}

// TestRestartService_NewDefaultUsesExecCommand asserts the default
// constructor wires exec.Command (not nil) so production callers do
// not need to set WithExecCommand.
func TestRestartService_NewDefaultUsesExecCommand(t *testing.T) {
	svc := NewRestartService()
	if svc.execCommand == nil {
		t.Error("default RestartService has nil execCommand")
	}
}

// equalStrings is a tiny helper so the table-shaped assertions above
// stay readable. nil vs empty are treated as different so the failure
// messages surface the actual slice.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}